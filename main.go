package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	osSignal "os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	w "github.com/panotza/pulse/watcher"
	"github.com/panotza/pulse/work"
)

type excludeFlag []string

func (f *excludeFlag) String() string { return "" }

func (f *excludeFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

type buildArgFlag []string

func (f *buildArgFlag) String() string { return "" }

func (f *buildArgFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

var (
	excludes      excludeFlag
	buildArgs     buildArgFlag
	onlyGo        = flag.Bool("go", false, "Reload only when .go file changes.")
	disablePreset = flag.Bool("xp", false, "Disable built-in preset.")
	watchDir      = flag.String("wd", ".", "Watching directory.")
	workingDir    = flag.String("cwd", ".", "Working directory of the executable.")
)

func main() {
	flag.Var(&excludes, "x", "Exclude a directory or a file. can be set multiple times.")
	flag.Var(&buildArgs, "buildArgs", "Additional go build arguments.")
	flag.Parse()
	args := flag.Args()

	var err error

	packagePath := "."
	if len(args) > 0 {
		packagePath = args[0]
	}

	packagePath, err = filepath.Abs(packagePath)
	if err != nil {
		log.Fatal(err)
	}

	if !*disablePreset {
		excludes = append(excludes,
			".git",
			".idea",
			".yarn",
			".vscode",
			".github",
			"node_modules",
		)
	}

	ctx, shutdown := osSignal.NotifyContext(context.Background(), os.Interrupt)
	defer shutdown()

	{
		fi, err := os.Stat(*watchDir)
		if err != nil {
			log.Fatal("stat watch path:", err)
		}

		if !fi.IsDir() {
			log.Fatal("watch path should be a directory")
		}
	}

	watcher := w.NewFSNotify(*watchDir, excludes, *onlyGo)
	signal := watcher.Watch(ctx)

	outBinPath := getOutBinPath(packagePath)
	defer os.Remove(outBinPath)
	fmt.Println("Pulse bin:", outBinPath)

	builder := work.NewBuilder(packagePath, outBinPath, buildArgs)

	var runArgs []string
	if i := slices.Index(args, "--"); i >= 0 {
		runArgs = args[i+1:]
	}
	runner := work.NewRunner(*workingDir, outBinPath, builder.BuildSignal(), runArgs)
	go runner.Listen(ctx)

	err = filepath.WalkDir(*watchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			for _, ex := range excludes {
				if strings.HasSuffix(path, ex) {
					return filepath.SkipDir
				}
			}
			if err := watcher.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		shutdown()
		log.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	for {
		// this thread handle watch signal. should not contain any blocking code
		select {
		case <-ctx.Done():
			cancel()
			return
		case _, ok := <-signal:
			if !ok {
				if err := watcher.Error(); err != nil {
					log.Println("[Pulse] watcher error:", err)
				}
				return
			}

			cancel()
			runCtx, cancel = context.WithCancel(ctx)
			go builder.Build(runCtx)
		}
	}
}

func getOutBinPath(packagePath string) string {
	hash := md5.Sum([]byte(packagePath))
	name := filepath.Base(packagePath)
	name += hex.EncodeToString(hash[:])[:4]
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}

	dir := filepath.Join(os.TempDir(), "pulse")
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, name)
}
