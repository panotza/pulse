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
)

func main() {
	flag.Var(&excludes, "x", "Exclude a directory or a file. can be set multiple times.")
	flag.Var(&buildArgs, "buildArgs", "Additional go build arguments.")
	flag.Parse()
	args := flag.Args()

	rootPath := "."
	if len(args) > 0 {
		rootPath = args[0]
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
		fi, err := os.Stat(rootPath)
		if err != nil {
			log.Fatal("stat watch path:", err)
		}

		if !fi.IsDir() {
			log.Fatal("watch path should be a directory")
		}
	}

	watcher := w.NewFSNotify(rootPath, excludes, *onlyGo)
	signal := watcher.Watch(ctx)

	outBinPath := getOutBinPath(rootPath)
	defer os.Remove(outBinPath)
	fmt.Println("Pulse bin:", outBinPath)

	builder := work.NewBuilder(rootPath, outBinPath, buildArgs)

	var runArgs []string
	if i := slices.Index(args, "--"); i >= 0 {
		runArgs = args[i+1:]
	}
	runner := work.NewRunner(rootPath, outBinPath, builder.BuildSignal(), runArgs)
	go runner.Listen(ctx)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
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

func getOutBinPath(rootPath string) string {
	if !filepath.IsAbs(rootPath) {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		rootPath = filepath.Join(wd, rootPath)
	}
	name := filepath.Base(rootPath)

	hash := md5.Sum([]byte(rootPath))
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
