package work

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Runner struct {
	binPath    string
	workingDir string
	args       []string

	refreshSignal chan struct{}
	stopSignal    chan struct{}
}

func NewRunner(workingDir string, binPath string, args []string) *Runner {
	return &Runner{
		binPath:       binPath,
		workingDir:    workingDir,
		refreshSignal: make(chan struct{}),
		stopSignal:    make(chan struct{}),
		args:          args,
	}
}

func (r *Runner) Refresh() {
	select {
	case r.refreshSignal <- struct{}{}:
	default:
	}
}

func (r *Runner) Stop() {
	select {
	case r.stopSignal <- struct{}{}:
	default:
	}
}

func (r *Runner) Listen(ctx context.Context) {
	var stopProcess context.CancelFunc = func() {}

	for {
		select {
		case <-ctx.Done():
			stopProcess()
			return
		case <-r.stopSignal:
			stopProcess()
		case <-r.refreshSignal:
			stopProcess()

			// Start a new process.
			var processCtx context.Context
			processCtx, stopProcess = context.WithCancel(ctx)
			go func() {
				if err := r.startProcess(processCtx); err != nil {
					log.Printf("[Runner] failed to start process: %v\n", err)
				}
			}()
		}
	}
}

func (r *Runner) startProcess(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.binPath, r.args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = r.workingDir
	cmd.WaitDelay = 3 * time.Second
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		var xe *exec.ExitError
		switch {
		case errors.As(err, &xe):
			log.Printf("[Runner] process exited with code: %d\n", xe.ExitCode())
		case errors.Is(err, context.Canceled):
			return nil
		default:
			return err
		}
	}
	return nil
}
