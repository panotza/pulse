package work

import (
	"context"
	"fmt"
	"io"
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
}

func NewBuilder(packagePath, outBinPath string, buildArgs []string, prebuildCmd string) *Builder {
	return &Builder{
		packagePath: packagePath,
		outBinPath:  outBinPath,
		buildArgs:   buildArgs,
		prebuildCmd: prebuildCmd,
	}
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
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stdout, stderr)

	log.Printf("[Pulse] %s\n", b.prebuildCmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("prebuild command failed: %w", err)
	}

	return nil
}

func (b *Builder) build(ctx context.Context) (err error) {
	args := append([]string{"go", "build", "-o", b.outBinPath}, b.buildArgs...)
	args = append(args, b.packagePath)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stdout, stderr)

	log.Println("[Pulse] Building...")
	start := time.Now()
	defer func() {
		if err == nil {
			log.Printf("[Pulse] Successfully Build. (%s)\n", time.Since(start))
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed for package %s: %w", b.packagePath, err)
	}

	return nil
}
