package pkg

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

type runner struct {
	bin     string
	args    []string
	writer  io.Writer
	command *exec.Cmd
}

// NewRunner creates new runner
func NewRunner(bin string, args ...string) *runner {
	return &runner{
		bin:    bin,
		args:   args,
		writer: ioutil.Discard,
	}
}

func (r *runner) Run(ctx context.Context) error {
	if r.command == nil || r.Exited() {
		err := r.runBin(ctx)
		if err != nil {
			log.Print("Error running: ", err)
			return err
		}
	}
	return nil
}

func (r *runner) SetWriter(writer io.Writer) {
	r.writer = writer
}

func (r *runner) Kill() error {
	if r.command == nil {
		return nil
	}
	if r.command.Process == nil {
		return nil
	}

	done := make(chan error)
	go func() {
		r.command.Wait()
		close(done)
	}()

	err := r.command.Process.Signal(os.Interrupt)
	if err != nil {
		if err = r.command.Process.Kill(); err != nil {
			return err
		}
	}

	// wait for our process to die before we return or hard kill after 3 sec
	select {
	case <-time.After(3 * time.Second):
		if err := r.command.Process.Kill(); err != nil {
			log.Println("failed to kill: ", err)
		}
	case <-done:
	}
	r.command = nil

	return nil
}

func (r *runner) Exited() bool {
	return r.command != nil && r.command.ProcessState != nil && r.command.ProcessState.Exited()
}

func (r *runner) runBin(ctx context.Context) error {
	r.command = exec.CommandContext(ctx, r.bin, r.args...)
	stdout, err := r.command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := r.command.StderrPipe()
	if err != nil {
		return err
	}

	err = r.command.Start()
	if err != nil {
		return err
	}

	go io.Copy(r.writer, stdout)
	go io.Copy(r.writer, stderr)

	done := make(chan struct{})
	go func() {
		err := r.command.Wait()
		if err != nil {
			log.Print(err)
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		r.Kill()
	case <-done:
	}
	return nil
}
