// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles is an item provider that lists files in a directory matching a pattern.
// It returns the full paths to the files.
func ListFiles(pattern string) func(ctx context.Context, workingDirectory string) ([]string, error) {
	return func(ctx context.Context, workingDirectory string) ([]string, error) {
		// Use the working directory as base if pattern is not absolute
		searchPattern := pattern
		if !filepath.IsAbs(pattern) {
			searchPattern = filepath.Join(workingDirectory, pattern)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue execution
		}

		// Find files matching the pattern
		matches, err := filepath.Glob(searchPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to list files with pattern %s: %w", pattern, err)
		}

		return matches, nil
	}
}

// ListDirectories is an item provider that lists all directories in a path.
// It returns the full paths to the directories.
func ListDirectories(basePath string) func(ctx context.Context, workingDirectory string) ([]string, error) {
	return func(ctx context.Context, workingDirectory string) ([]string, error) {
		// Use the working directory as base if pattern is not absolute
		searchPath := basePath
		if !filepath.IsAbs(basePath) {
			searchPath = filepath.Join(workingDirectory, basePath)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue execution
		}

		// Find all directories in the given path
		var dirs []string

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != searchPath {
				dirs = append(dirs, path)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list directories in %s: %w", basePath, err)
		}

		return dirs, nil
	}
}

// SplitString is an item provider that splits a string by a delimiter.
// It returns the list of substrings.
func SplitString(s string, delimiter string) func(ctx context.Context, _ string) ([]string, error) {
	return func(ctx context.Context, _ string) ([]string, error) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue execution
		}

		return strings.Split(s, delimiter), nil
	}
}
