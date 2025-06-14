package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/panotza/pulse/watcher"
	"github.com/panotza/pulse/work"
)

func run(packagePath string, args []string) error {
	// Configure slog with the specified log level
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		var lv slog.Level
		err := lv.UnmarshalText([]byte(s))
		if err != nil {
			return err
		}

		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: lv,
		})
		slog.SetDefault(slog.New(handler))
	}

	ctx, shutdown := signal.NotifyContext(context.Background(), os.Interrupt)
	defer shutdown()

	fsWatcher, err := watcher.NewFileWatcher(
		watcher.WithIgnorePatterns(excludes),
		watcher.WithLogger(slog.Default()),
	)
	if err != nil {
		return err
	}
	fsSignal := fsWatcher.Listen(ctx)

	outBinPath := genOutBinPath(packagePath)
	defer os.Remove(outBinPath)
	fmt.Println("Pulse bin:", outBinPath)

	var runArgs []string
	if i := slices.Index(args, "--"); i >= 0 {
		runArgs = args[i+1:]
	}

	if len(watchDirs) == 0 {
		watchDirs = append(watchDirs, ".")
	}
	for _, watchDir := range watchDirs {
		fi, err := os.Stat(watchDir)
		if err != nil {
			return fmt.Errorf("stat watch path %s: %w", watchDir, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("watch path %s is not a directory", watchDir)
		}

		err = fsWatcher.AddDirectory(ctx, watchDir)
		if err != nil {
			return err
		}
	}

	// Create runner and builder
	runner := work.NewRunner(*workingDir, outBinPath, runArgs)
	go runner.Listen(ctx)

	buildCtx, cancelBuild := context.WithCancel(ctx)
	builder := work.NewBuilder(packagePath, outBinPath, buildArgs, *prebuildCmd)

	// Main loop to handle file system events and build process
	for {
		select {
		case <-ctx.Done():
			runner.Stop()
			cancelBuild()
			return nil
		case _, ok := <-fsSignal:
			if !ok {
				// Channel closed, watcher stopped
				runner.Stop()
				cancelBuild()
				return nil
			}

			runner.Stop()
			cancelBuild()

			buildCtx, cancelBuild = context.WithCancel(ctx)
			go func() {
				err := builder.Build(buildCtx)
				if err == nil {
					runner.Refresh()
				}
			}()
		}
	}
}

func genOutBinPath(packagePath string) string {
	hash := md5.Sum([]byte(packagePath))
	name := filepath.Base(packagePath)
	name += hex.EncodeToString(hash[:])[:4]
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}

	dir := filepath.Join(os.TempDir(), "pulse")
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, name)
}
