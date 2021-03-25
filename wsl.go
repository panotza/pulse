package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func clientMode(ctx context.Context, wd string) (bool, error) {
	// inside WSL but try to watch files on Windows
	if runtime.GOOS == "linux" {
		b, err := ioutil.ReadFile("/proc/version")
		if err != nil {
			log.Fatal(err)
		}

		insideWSL := strings.Contains(string(b), "microsoft")
		if !insideWSL {
			return false, nil
		}

		if !strings.HasPrefix(wd, "/mnt") {
			return false, nil
		}
		return true, nil
	}

	// inside Windows but try to watch files on WSL
	if runtime.GOOS == "windows" && strings.HasPrefix(wd, `\\wsl$`) {
		return false, fmt.Errorf("watch files on WSL from Windows is not supported yet")
	}
	return false, nil
}

func execPipe(ctx context.Context, w io.Writer, name string, arg ...string) error {
	command := exec.Command(name, arg...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	err = command.Start()
	if err != nil {
		return err
	}

	mw := io.MultiWriter(os.Stdout, w)
	go io.Copy(mw, stdout)
	go io.Copy(os.Stdout, stderr)

	waitDone := make(chan struct{})
	go func() {
		command.Wait()
		close(waitDone)
	}()

	select {
	case <-ctx.Done():
		err := killServer(command)
		if err != nil {
			fmt.Println(err)
		}
	case <-waitDone:
	}
	return nil
}

func killServer(command *exec.Cmd) error {
	if command == nil {
		return nil
	}
	if command.Process == nil {
		return nil
	}

	done := make(chan error)
	go func() {
		command.Wait()
		close(done)
	}()

	err := command.Process.Signal(os.Interrupt)
	if err != nil {
		err = command.Process.Kill()
		if err != nil {
			return err
		}
	}

	// wait for our process to die before we return or hard kill after 3 sec
	select {
	case <-time.After(3 * time.Second):
		if err := command.Process.Kill(); err != nil {
			log.Println("failed to kill: ", err)
		}
	case <-done:
	}
	command = nil

	return nil
}

func pathToWindows(path string) string {
	if !strings.HasPrefix(path, "/mnt/") {
		return path
	}

	path = strings.TrimPrefix(path, "/mnt/")
	drive := path[0:1]
	path = strings.ReplaceAll(path[1:], "/", "\\")
	return drive + ":" + path
}
