package pkg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type watcher struct {
	e *fsnotify.Watcher
}

func (w *watcher) Watch(ctx context.Context) <-chan string {
	c := make(chan string)
	debouncer := newDebouncer(300 * time.Millisecond)
	go func() {
		defer func() {
			w.e.Close()
			close(c)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.e.Events:
				if !ok {
					return
				}

				if event.Op == fsnotify.Chmod {
					continue
				}

				debouncer.fire(c, event.Name)
			case err, ok := <-w.e.Errors:
				if !ok {
					return
				}
				log.Println("watcher:", err)
			}
		}
	}()

	return c
}

func NewWatcher(root string, excludeDirs []string) (*watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	done := make(chan error)
	go func() {
		err = filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				return err
			}

			for _, x := range excludeDirs {
				if x == path {
					return filepath.SkipDir
				}
			}

			return w.Add(path)
		})
		done <- err
	}()

	select {
	case <-time.After(time.Second * 5):
		return nil, fmt.Errorf("init watcher failed")
	case err := <-done:
		if err != nil {
			return nil, err
		}
	}
	return &watcher{e: w}, nil
}

type streamWatcher struct {
	reader io.ReadCloser
}

func NewStreamWatcher(reader io.ReadCloser) *streamWatcher {
	return &streamWatcher{reader: reader}
}

func (w *streamWatcher) Watch(ctx context.Context) <-chan string {
	c := make(chan string)

	go func() {
		<-ctx.Done()
		w.reader.Close()
	}()

	go func() {
		scanner := bufio.NewScanner(w.reader)
		for scanner.Scan() {
			l := scanner.Text()
			if strings.HasPrefix(l, "[event]") {
				c <- strings.TrimPrefix(l, "[event]: ")
			}
		}
		w.reader.Close()
		close(c)
	}()

	return c
}

type debouncer struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
}

func newDebouncer(after time.Duration) *debouncer {
	return &debouncer{
		after: after,
	}
}

func (d *debouncer) fire(c chan<- string, file string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.after, func() {
		c <- file
	})
}
