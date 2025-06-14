package watcher

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockFileNotifier implements FileNotifier for testing
type mockFileNotifier struct {
	addedPaths []string
	addError   error
	addFunc    func(path string) error
}

func (m *mockFileNotifier) Add(path string) error {
	if m.addFunc != nil {
		return m.addFunc(path)
	}
	if m.addError != nil {
		return m.addError
	}
	m.addedPaths = append(m.addedPaths, path)
	return nil
}

func (m *mockFileNotifier) Listen(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// setupTestDir creates a temporary directory structure for testing
func setupTestDir(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"dir1",
		"dir1/subdir1",
		"dir1/subdir2",
		"dir2",
		"dir2/subdir3",
		"ignored_dir",
		"ignored_dir/nested",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create some files
	files := []string{
		"dir1/file1.txt",
		"dir1/subdir1/file2.txt",
		"dir2/file3.txt",
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
		f.Close()
	}

	return tempDir
}

func TestFileWatcher_AddDirectory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tempDir := setupTestDir(t)

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("AddDirectory failed: %v", err)
		}

		// Verify all directories were added (not files)
		expectedDirs := []string{
			tempDir,
			filepath.Join(tempDir, "dir1"),
			filepath.Join(tempDir, "dir1", "subdir1"),
			filepath.Join(tempDir, "dir1", "subdir2"),
			filepath.Join(tempDir, "dir2"),
			filepath.Join(tempDir, "dir2", "subdir3"),
			filepath.Join(tempDir, "ignored_dir"),
			filepath.Join(tempDir, "ignored_dir", "nested"),
		}

		if len(mock.addedPaths) != len(expectedDirs) {
			t.Errorf("Expected %d directories to be added, got %d", len(expectedDirs), len(mock.addedPaths))
		}

		// Verify each expected directory was added
		addedMap := make(map[string]bool)
		for _, path := range mock.addedPaths {
			addedMap[path] = true
		}

		for _, expectedDir := range expectedDirs {
			if !addedMap[expectedDir] {
				t.Errorf("Expected directory %s was not added", expectedDir)
			}
		}
	})

	t.Run("WithIgnorePatterns", func(t *testing.T) {
		tempDir := setupTestDir(t)

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		// Create FileWatcher with ignore patterns
		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
			WithIgnorePatterns([]string{"ignored_dir", "*/subdir2"}),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("AddDirectory failed: %v", err)
		}

		// Verify ignored directories were not added
		for _, path := range mock.addedPaths {
			if filepath.Base(path) == "ignored_dir" {
				t.Errorf("Ignored directory %s was added", path)
			}
			if filepath.Base(path) == "subdir2" {
				t.Errorf("Ignored directory %s was added", path)
			}
			if strings.Contains(path, "ignored_dir") {
				t.Errorf("Directory under ignored path %s was added", path)
			}
		}
	})

	t.Run("NonexistentPath", func(t *testing.T) {
		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, "/nonexistent/path")
		// Should not return error, but should log the error and continue
		if err != nil {
			t.Errorf("AddDirectory should not fail for nonexistent path, got: %v", err)
		}

		// No directories should be added
		if len(mock.addedPaths) != 0 {
			t.Errorf("Expected no directories to be added for nonexistent path, got %d", len(mock.addedPaths))
		}
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		tempDir := setupTestDir(t)

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = fw.AddDirectory(ctx, tempDir)

		// Should return context canceled error
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("ContextTimeout", func(t *testing.T) {
		tempDir := setupTestDir(t)

		mock := &mockFileNotifier{
			// Simulate slow Add operation
			addFunc: func(path string) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		err = fw.AddDirectory(ctx, tempDir)

		// Should return context deadline exceeded error
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
		}
	})

	t.Run("NotifierAddError", func(t *testing.T) {
		tempDir := setupTestDir(t)

		addError := errors.New("failed to add path")
		mock := &mockFileNotifier{
			addError: addError,
		}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		// Should not return error even if notifier.Add fails (it just logs and continues)
		if err != nil {
			t.Errorf("AddDirectory should not fail when notifier.Add fails, got: %v", err)
		}
	})

	t.Run("FilePermissionError", func(t *testing.T) {
		// Create a directory that we can't read
		tempDir := t.TempDir()

		// Create a subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		// Remove read permissions from the subdirectory
		err = os.Chmod(subDir, 0o000)
		if err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}

		// Restore permissions for cleanup
		defer func() {
			os.Chmod(subDir, 0o755)
		}()

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		// Should not return error (it logs the error and continues)
		if err != nil {
			t.Errorf("AddDirectory should not fail for permission errors, got: %v", err)
		}

		// The root directory should still be added
		if len(mock.addedPaths) == 0 {
			t.Errorf("Expected at least root directory to be added")
		}
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		tempDir := t.TempDir()

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("AddDirectory failed: %v", err)
		}

		// Should add the root directory
		if len(mock.addedPaths) != 1 {
			t.Errorf("Expected 1 directory to be added, got %d", len(mock.addedPaths))
		}

		if mock.addedPaths[0] != tempDir {
			t.Errorf("Expected root directory %s to be added, got %s", tempDir, mock.addedPaths[0])
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		tempDir := setupTestDir(t)

		// Create a logger that captures output
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, nil))

		mock := &mockFileNotifier{
			addError: errors.New("mock add error"),
		}

		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("AddDirectory failed: %v", err)
		}

		// Verify that error was logged
		if logOutput.Len() == 0 {
			t.Error("Expected error to be logged")
		}
	})

	t.Run("IgnoredRootDirectory", func(t *testing.T) {
		tempDir := setupTestDir(t)

		mock := &mockFileNotifier{}
		logger := slog.New(slog.DiscardHandler)

		// Create FileWatcher that ignores the root directory itself
		// This should cause filepath.SkipDir to be returned
		rootDirName := filepath.Base(tempDir)
		fw, err := NewFileWatcher(
			WithNotifier(mock),
			WithLogger(logger),
			WithIgnorePatterns([]string{rootDirName}),
		)
		if err != nil {
			t.Fatalf("Failed to create FileWatcher: %v", err)
		}

		ctx := context.Background()
		err = fw.AddDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("AddDirectory failed: %v", err)
		}

		// No directories should be added because the root directory is ignored
		if len(mock.addedPaths) != 0 {
			t.Errorf("Expected no directories to be added due to ignore pattern, got %d", len(mock.addedPaths))
		}
	})
}
