// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachproviders

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// IncludeHidden is a type that indicates whether to include hidden files and directories.
type IncludeHidden bool

var (
	// HiddenInclude tells the foreach command to include hidden files and directories.
	HiddenInclude = IncludeHidden(true)
	// HiddenExclude tells the foreach command to exclude hidden files and directories.
	HiddenExclude = IncludeHidden(false)
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

// ListDirectoriesDepth is an item provider that lists all directories in a path at given depth.
func ListDirectoriesDepth(depth int, includeHidden IncludeHidden) func(context.Context, string) ([]string, error) {
	return func(ctx context.Context, workingDirectory string) ([]string, error) {
		// Find all directories in the given path
		var dirs []string

		err := filepath.WalkDir(workingDirectory, func(path string, d fs.DirEntry, err error) error {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Continue execution
			}

			if err != nil {
				return err
			}

			if !d.IsDir() {
				return nil
			}

			// Check if the directory is hidden
			isHidden := path != workingDirectory && filepath.Base(path)[0] == '.'

			// Skip hidden directories if includeHidden is false
			if !bool(includeHidden) && isHidden {
				return filepath.SkipDir
			}

			// check depth
			relPath, err := filepath.Rel(workingDirectory, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path for %s: %w", path, err)
			}

			if depth > 0 && strings.Count(relPath, string(os.PathSeparator)) > depth-1 {
				return filepath.SkipDir // Skip directories deeper than specified depth
			}

			if path != workingDirectory {
				path, err = filepath.Rel(workingDirectory, path)
				if err != nil {
					return fmt.Errorf("failed to get relative path for %s: %w", path, err)
				}

				dirs = append(dirs, path)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list directories in %s: %w", workingDirectory, err)
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
