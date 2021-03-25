package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/panotza/pulse/pkg"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "pulse"
	app.Usage = "A live reload utility for Go web applications."
	app.Action = mainAction
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "bin",
			Aliases: []string{"b"},
			Value:   ".pulse",
			Usage:   "name of generated binary file",
		},
		&cli.StringFlag{
			Name:    "path",
			Aliases: []string{"t"},
			Value:   ".",
			Usage:   "Path to watch files from",
		},
		&cli.StringFlag{
			Name:    "build",
			Aliases: []string{"d"},
			Value:   "",
			Usage:   "Path to build files from (defaults to same value as --path)",
		},
		&cli.StringSliceFlag{
			Name:    "excludeDir",
			Aliases: []string{"x"},
			Value:   &cli.StringSlice{},
			Usage:   "Relative directories to exclude",
		},
		&cli.BoolFlag{ // for backward compatible
			Name:  "all",
			Usage: "reloads whenever any file changes, as opposed to reloading only on .go file change",
		},
		&cli.BoolFlag{
			Name:    "watcher",
			Aliases: []string{"w"},
			Value:   false,
			Usage:   "only watch files and send events to stdout",
		},
		// &cli.StringFlag{
		// 	Name:  "buildArgs",
		// 	Usage: "Additional go build arguments",
		// },
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func mainAction(c *cli.Context) error {
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// TODO: impl
	// buildArgs, err := shellwords.Parse(c.String("buildArgs"))
	// if err != nil {
	// 	return err
	// }

	watcherMode := c.Bool("watcher")

	buildPath := c.String("build")
	if buildPath == "" {
		buildPath = c.String("path")
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		<-exit
		shutdown()
	}()

	var wg sync.WaitGroup
	var watcher interface {
		Watch(ctx context.Context) <-chan string
	}
	var executor interface {
		Do(ctx context.Context, file string)
	}

	isClient, err := clientMode(ctx, wd)
	if err != nil {
		return err
	}

	builder := pkg.NewBuilder(buildPath, c.String("bin"), wd, []string{})
	runner := pkg.NewRunner(filepath.Join(wd, builder.Binary()))
	runner.SetWriter(os.Stdout)

	if watcherMode {
		log.Printf("[%s]: running as watcher mode", runtime.GOOS)
		executor = pkg.NewPrintStdoutExecutor()
	} else {
		executor = pkg.NewExecutor(&wg, builder, runner)
	}

	if isClient {
		r, w := io.Pipe()
		defer r.Close()
		defer w.Close()

		go func(ctx context.Context) {
			_ = execPipe(ctx, w, "pulse.exe", append([]string{"-w"}, os.Args[1:]...)...)
			log.Print("host exited")
			shutdown()
		}(ctx)

		log.Print("[wsl]: start running in client mode connecting to Windows")
		watcher = pkg.NewStreamWatcher(r)
	} else {
		watcher, err = pkg.NewWatcher(
			c.String("path"),
			append(c.StringSlice("excludeDir"), ".git"),
		)
		if err != nil {
			return fmt.Errorf("[%s]: %v", runtime.GOOS, err)
		}
	}

	// build on start
	executor.Do(ctx, "init")

	for fc := range watcher.Watch(ctx) {
		// skip pulse binary
		if strings.HasPrefix(filepath.Clean(fc), builder.Binary()) {
			continue
		}

		executor.Do(ctx, fc)
	}

	wg.Wait()
	return nil
}
