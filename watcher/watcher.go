package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/codeglyph/go-dotignore"
)

type FileNotifier interface {
	Add(path string) error
	Listen(ctx context.Context) <-chan struct{}
}

type FileWatcher struct {
	ignorePatterns []string
	logger         *slog.Logger
	notifier       FileNotifier
	ignoreMatcher  *dotignore.PatternMatcher
}

// FileWatcherOption defines a function type for configuring FileWatcher
type FileWatcherOption func(*FileWatcher)

// WithIgnorePatterns sets the exclude paths for the file watcher
func WithIgnorePatterns(patterns []string) FileWatcherOption {
	return func(fw *FileWatcher) {
		fw.ignorePatterns = patterns
	}
}

// WithLogger sets the logger for the file watcher
func WithLogger(logger *slog.Logger) FileWatcherOption {
	return func(fw *FileWatcher) {
		fw.logger = logger
	}
}

// WithNotifier sets the file notifier for the file watcher
func WithNotifier(notifier FileNotifier) FileWatcherOption {
	return func(fw *FileWatcher) {
		fw.notifier = notifier
	}
}

// NewFileWatcher creates a new FileWatcher with optional configuration
func NewFileWatcher(options ...FileWatcherOption) (*FileWatcher, error) {
	fw := &FileWatcher{
		ignorePatterns: nil,
		logger:         slog.New(slog.DiscardHandler),
		notifier:       nil,
	}

	// Apply all provided options
	for _, option := range options {
		option(fw)
	}

	matcher, err := dotignore.NewPatternMatcher(fw.ignorePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern matcher: %w", err)
	}
	fw.ignoreMatcher = matcher

	if fw.notifier == nil {
		fsNotify, err := NewFSNotify(fw.logger, fw.ignoreMatcher)
		if err != nil {
			return nil, err
		}
		fw.notifier = fsNotify
	}
	return fw, nil
}

func (fw *FileWatcher) isPathIgnored(path string) (bool, error) {
	return fw.ignoreMatcher.Matches(path)
}

func (fw *FileWatcher) AddDirectory(ctx context.Context, path string) error {
	// Walk through all subdirectories and add them
	return filepath.WalkDir(path, func(walkPath string, d os.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			// Log the error but continue walking
			fw.logger.ErrorContext(ctx, "failed walking directory", slog.String("path", walkPath), slog.Any("error", err))
			return nil
		}

		ignored, err := fw.isPathIgnored(walkPath)
		if err != nil {
			return err
		}
		if ignored {
			if d.IsDir() {
				fw.logger.DebugContext(ctx, "skipping ignored directory path", slog.String("path", walkPath))
				return filepath.SkipDir // Skip this directory if it's ignored
			}
		}

		// Only add directories
		if d.IsDir() {
			if err := fw.notifier.Add(walkPath); err != nil {
				fw.logger.ErrorContext(ctx, "failed to add directory to watcher", slog.String("path", walkPath), slog.Any("error", err))
				// Continue walking even if we can't add this directory
			} else {
				fw.logger.DebugContext(ctx, "added directory to watcher", slog.String("path", walkPath))
			}
		}

		return nil
	})
}

func (fw *FileWatcher) Listen(ctx context.Context) <-chan struct{} {
	return fw.notifier.Listen(ctx)
}
