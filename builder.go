package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Builder struct {
	rootPath   string
	dir        string
	outBinPath string
	buildArgs  []string

	buildSignal chan struct{}
}

func NewBuilder(rootPath, outBinPath string, buildArgs []string) *Builder {
	var dir string
	if filepath.IsAbs(rootPath) {
		dir = rootPath
	}
	return &Builder{
		rootPath:    rootPath,
		dir:         dir,
		outBinPath:  outBinPath,
		buildArgs:   buildArgs,
		buildSignal: make(chan struct{}),
	}
}

func (b *Builder) BuildSignal() <-chan struct{} {
	return b.buildSignal
}

func (b *Builder) Build(ctx context.Context) (err error) {
	args := append([]string{"go", "build", "-o", b.outBinPath}, b.buildArgs...)
	args = append(args, ".")

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = b.dir

	log.Println("[Pulse] Building...")
	start := time.Now()
	defer func() {
		if err == nil {
			log.Printf("[Pulse] Successfully Build. (%s)\n", time.Since(start))
			b.buildSignal <- struct{}{}
		}
	}()
	err = cmd.Run()
	return
}
