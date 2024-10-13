package work

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Builder struct {
	packagePath string
	outBinPath  string
	buildArgs   []string
	prebuildCmd string

	buildSignal chan struct{}
}

func NewBuilder(packagePath, outBinPath string, buildArgs []string, prebuildCmd string) *Builder {
	return &Builder{
		packagePath: packagePath,
		outBinPath:  outBinPath,
		buildArgs:   buildArgs,
		prebuildCmd: prebuildCmd,
		buildSignal: make(chan struct{}),
	}
}

func (b *Builder) BuildSignal() <-chan struct{} {
	return b.buildSignal
}

func (b *Builder) Build(ctx context.Context) error {
	err := b.prebuild(ctx)
	if err != nil {
		return err
	}
	err = b.build(ctx)
	return err
}

func (b *Builder) prebuild(ctx context.Context) error {
	if b.prebuildCmd == "" {
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C")
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c")
	}
	cmd.Args = append(cmd.Args, b.prebuildCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[Pulse] %s\n", b.prebuildCmd)
	return cmd.Run()
}

func (b *Builder) build(ctx context.Context) (err error) {
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
	return err
}
