package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type builder struct {
	dir       string
	binary    string
	wd        string
	buildArgs []string
}

// NewBuilder creates new builder
func NewBuilder(dir string, bin string, wd string, buildArgs []string) *builder {
	if len(bin) == 0 {
		bin = "bin"
	}

	// does not work on Windows without the ".exe" extension
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(bin, ".exe") {
			bin += ".exe"
		}
	}

	if dir == "" {
		dir = "."
	} else {
		dir = filepath.Join(wd, dir)
	}

	return &builder{
		dir:       dir,
		binary:    bin,
		wd:        wd,
		buildArgs: buildArgs,
	}
}

func (b *builder) Binary() string {
	return b.binary
}

func (b *builder) Build(ctx context.Context) error {
	args := append([]string{"go", "build", "-o", filepath.Join(b.wd, b.binary)}, b.buildArgs...)
	args = append(args, b.dir)

	command := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s%s", output, err)
	}
	if !command.ProcessState.Success() {
		return fmt.Errorf("%s", output)
	}
	return nil
}
