package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FSNotify struct {
	root         string
	excludePaths []string
	onlyGo       bool
	w            *fsnotify.Watcher
	err          error
}

func NewFSNotify(root string, excludePath []string, onlyGo bool) *FSNotify {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	return &FSNotify{
		root:         root,
		excludePaths: excludePath,
		onlyGo:       onlyGo,
		w:            watcher,
	}
}

func (fsw *FSNotify) Watch(ctx context.Context) <-chan struct{} {
	signal := make(chan struct{})

	fire, cancelFire := NewDebounce(time.Millisecond*150, func() {
		signal <- struct{}{}
	})

	// fire once at the start to trigger build
	go func() {
		fire()
	}()

	go func() {
		defer fsw.w.Close()
		defer func() {
			cancelFire()
			close(signal)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-fsw.w.Events:
				if !ok {
					fsw.err = fmt.Errorf("fsnotify watcher channel closed")
					return
				}

				for _, ex := range fsw.excludePaths {
					if strings.HasSuffix(event.Name, ex) {
						continue
					}
				}

				if fsw.onlyGo && filepath.Ext(event.Name) != ".go" {
					continue
				}

				if event.Has(fsnotify.Create) {
					fi, err := os.Stat(event.Name)
					if err == nil && fi.IsDir() {
						if err := fsw.w.Add(event.Name); err != nil {
							fsw.err = err
							return
						}
					}
					fire()
				} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					fire()
				}
			case err, ok := <-fsw.w.Errors:
				if !ok {
					fsw.err = fmt.Errorf("fsnotify watcher channel closed")
				} else {
					fsw.err = err
				}
				return
			}
		}
	}()
	return signal
}

func (fsw *FSNotify) Add(path string) error {
	return fsw.w.Add(path)
}

func (fsw *FSNotify) Error() error {
	return fsw.err
}
