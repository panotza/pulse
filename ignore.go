package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
)

func readGitIgnore() []string {
	f, err := os.Open(".gitignore")
	if err != nil {
		return nil
	}
	defer f.Close()

	patterns, err := readLines(f)
	if err != nil {
		slog.Error("failed to read .gitignore", slog.Any("error", err))
		return nil
	}
	slog.Debug("read .gitignore", slog.Any("patterns", patterns))
	return patterns
}

func readPulseIgnore() []string {
	f, err := os.Open(".pulseignore")
	if err != nil {
		return nil
	}
	defer f.Close()

	patterns, err := readLines(f)
	if err != nil {
		slog.Error("failed to read .gitignore", slog.Any("error", err))
		return nil
	}
	slog.Debug("read .pulseignore", slog.Any("patterns", patterns))
	return patterns
}

func mergeIgnorePatterns(patterns ...[]string) []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0)
	for i := range patterns {
		for _, p := range patterns[i] {
			if p == "" {
				continue // Skip empty patterns
			}
			if _, ok := seen[p]; ok {
				continue // Skip if already seen
			}

			merged = append(merged, p)
			seen[p] = struct{}{}
		}
	}
	return merged
}

// readLines reads lines from an io.Reader and strips UTF-8 BOM characters.
func readLines(reader io.Reader) ([]string, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	scanner := bufio.NewScanner(reader)
	var lines []string
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}

	for lineNumber := 0; scanner.Scan(); lineNumber++ {
		line := scanner.Bytes()
		if lineNumber == 0 {
			line = bytes.TrimPrefix(line, utf8BOM)
		}
		lines = append(lines, string(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines: %w", err)
	}

	return lines, nil
}
