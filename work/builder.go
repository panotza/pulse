package work

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"
)

type Builder struct {
	packagePath string
	outBinPath  string
	buildArgs   []string

	buildSignal chan struct{}
}

func NewBuilder(packagePath, outBinPath string, buildArgs []string) *Builder {
	return &Builder{
		packagePath: packagePath,
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
	args = append(args, b.packagePath)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
