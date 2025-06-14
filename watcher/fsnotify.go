package watcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/codeglyph/go-dotignore"
	"github.com/fsnotify/fsnotify"
)

// FSNotify watches the filesystem for changes with configurable filters
type FSNotify struct {
	*fsnotify.Watcher
	logger        *slog.Logger
	ignoreMatcher *dotignore.PatternMatcher
}

// NewFSNotify creates a new filesystem watcher with specified filters
func NewFSNotify(logger *slog.Logger, ignoreMatcher *dotignore.PatternMatcher) (*FSNotify, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &FSNotify{
		Watcher:       watcher,
		logger:        logger,
		ignoreMatcher: ignoreMatcher,
	}, nil
}

func (fsw *FSNotify) Listen(ctx context.Context) <-chan struct{} {
	signal := make(chan struct{}, 1)

	go func() {
		defer fsw.Close()
		defer close(signal)

		fire, stopFire := newDebounce(100*time.Millisecond, func() {
			signal <- struct{}{}
		})
		defer stopFire()

		// Initial build trigger
		fire()

		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil && !errors.Is(err, context.Canceled) {
					fsw.logger.ErrorContext(ctx, "fsnotify exit with error", slog.Any("error", err))
				}
				return
			case event, ok := <-fsw.Events:
				if !ok {
					return
				}

				shouldFire, err := fsw.handleEvent(ctx, event)
				if err != nil {
					fsw.logger.WarnContext(ctx, "fsnotify event handling warning", slog.Any("error", err))
				}

				if shouldFire {
					fire()
				}
			case err, ok := <-fsw.Errors:
				if !ok {
					return
				}

				fsw.logger.ErrorContext(ctx, "fsnotify error", slog.Any("error", err))
				return
			}
		}
	}()

	return signal
}

// handleEvent processes fsnotify events and determines whether to trigger file system notifications.
// It automatically adds newly created files/directories to the watcher and returns true for
// events that should trigger notifications (Create, Remove, Write, Rename operations).
func (fsw *FSNotify) handleEvent(ctx context.Context, event fsnotify.Event) (bool, error) {
	if event.Has(fsnotify.Create) {
		ignored, err := fsw.ignoreMatcher.Matches(event.Name)
		if err != nil {
			return false, err
		}
		if ignored {
			fsw.logger.DebugContext(ctx, "fsnotify create event ignored", slog.String("path", event.Name))
		} else {
			fsw.logger.DebugContext(ctx, "fsnotify create event", slog.String("path", event.Name))

			err = fsw.Watcher.Add(event.Name)
			if err != nil {
				return false, fmt.Errorf("failed to add path %s to fsnotify watcher: %w", event.Name, err)
			}
		}
		return true, nil
	}
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
		return true, nil
	}

	return false, nil
}

func (fsw *FSNotify) Add(path string) error {
	return fsw.Watcher.Add(path)
}

// Ensure that FSNotify implements the FileNotifier interface
var _ FileNotifier = (*FSNotify)(nil)
