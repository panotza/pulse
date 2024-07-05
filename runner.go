package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sync/singleflight"
)

type Runner struct {
	binPath  string
	rootPath string
	dir      string
	args     []string

	refreshSig <-chan struct{}
}

func NewRunner(rootPath string, binPath string, refreshSig <-chan struct{}, args []string) *Runner {
	var dir string
	if filepath.IsAbs(rootPath) {
		dir = rootPath
	}

	return &Runner{
		binPath:    binPath,
		rootPath:   rootPath,
		dir:        dir,
		refreshSig: refreshSig,
		args:       args,
	}
}

func (r *Runner) Listen(ctx context.Context) {
	var (
		kill func()
		err  error
	)
	defer func() {
		if kill != nil {
			kill()
		}
	}()

	g := singleflight.Group{}

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-r.refreshSig:
			if !ok {
				continue
			}

			// new signal can be received while waiting kill()
			// so it should ignore trailing signal and only run once after killed
			g.Do("exec", func() (any, error) {
				if kill != nil {
					kill() // kill can take upto 3 secs
				}
				kill, err = r.exec()
				if err != nil {
					log.Println("start process failed:", err)
				}
				return nil, nil
			})
		}
	}
}

func (r *Runner) exec() (func(), error) {
	cmd := exec.Command(r.binPath, r.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = r.dir
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	kill := func() {
		done := make(chan struct{})
		go func() {
			cmd.Wait()
			close(done)
		}()

		// trying a "soft" kill first
		if runtime.GOOS == "windows" {
			cmd.Process.Kill()
		} else {
			cmd.Process.Signal(os.Interrupt)
		}

		// wait for our process to die before we return or hard kill after 3 sec
		select {
		case <-time.After(3 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				log.Println("[Pulse] failed to kill: ", err)
			}
		case <-done:
		}
	}
	return kill, nil
}
