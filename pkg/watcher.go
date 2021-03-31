package pkg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type fsnotifyWatcher struct {
	e        *fsnotify.Watcher
	excludes []*regexp.Regexp
}

func (w *fsnotifyWatcher) Watch(ctx context.Context) <-chan string {
	c := make(chan string)
	debouncer := newDebouncer(300 * time.Millisecond)

	go func() {
		defer func() {
			w.e.Close()
			close(c)
		}()

	Loop:
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

				name := filepath.Clean(event.Name)

				for _, x := range w.excludes {
					if x.MatchString(name) {
						continue Loop
					}
				}

				switch event.Op {
				case fsnotify.Create:
					err := w.e.Add(event.Name)
					if err != nil {
						log.Print("failed to watch ", event.Name)
					}
				case fsnotify.Remove:
					w.e.Remove(event.Name)
				}

				debouncer.fire(c, name)
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

func NewFSNotifyWatcher(root string, excludes []string) (*fsnotifyWatcher, error) {
	var reExcludes []*regexp.Regexp
	for _, x := range excludes {
		reExcludes = append(reExcludes, regexp.MustCompile(x))
	}

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

			for _, x := range excludes {
				if x == path {
					return filepath.SkipDir
				}
			}

			return w.Add(path)
		})
		done <- err
	}()

	// sometimes windows hang when adding watching path
	select {
	case <-time.After(time.Second * 5):
		return nil, fmt.Errorf("init fs notify watcher failed")
	case err := <-done:
		if err != nil {
			return nil, err
		}
	}

	return &fsnotifyWatcher{e: w, excludes: reExcludes}, nil
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
