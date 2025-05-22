// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"os"
	"time"

	"github.com/spf13/afero"
)

// errorFS is a filesystem wrapper that returns errors for specific operations.
type errorFS struct {
	fs afero.Fs
	// Path that should generate an error
	errorPath string
}

// Create implements afero.Fs.
func (e *errorFS) Create(name string) (afero.File, error) {
	if name == e.errorPath {
		return nil, os.ErrPermission
	}

	return e.fs.Create(name)
}

// Mkdir implements afero.Fs.
func (e *errorFS) Mkdir(name string, perm os.FileMode) error {
	if name == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Mkdir(name, perm)
}

// MkdirAll implements afero.Fs.
func (e *errorFS) MkdirAll(path string, perm os.FileMode) error {
	if path == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.MkdirAll(path, perm)
}

// Open implements afero.Fs.
func (e *errorFS) Open(name string) (afero.File, error) {
	if name == e.errorPath {
		return nil, os.ErrPermission
	}

	return e.fs.Open(name)
}

// OpenFile implements afero.Fs.
func (e *errorFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if name == e.errorPath {
		return nil, os.ErrPermission
	}

	return e.fs.OpenFile(name, flag, perm)
}

// Remove implements afero.Fs.
func (e *errorFS) Remove(name string) error {
	if name == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Remove(name)
}

// RemoveAll implements afero.Fs.
func (e *errorFS) RemoveAll(path string) error {
	if path == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.RemoveAll(path)
}

// Rename implements afero.Fs.
func (e *errorFS) Rename(oldname, newname string) error {
	if oldname == e.errorPath || newname == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Rename(oldname, newname)
}

// Stat implements afero.Fs.
func (e *errorFS) Stat(name string) (os.FileInfo, error) {
	if name == e.errorPath {
		return nil, os.ErrPermission
	}

	return e.fs.Stat(name)
}

// Name implements afero.Fs.
func (e *errorFS) Name() string {
	return "errorFS"
}

// Chmod implements afero.Fs.
func (e *errorFS) Chmod(name string, mode os.FileMode) error {
	if name == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Chmod(name, mode) //nolint:wrapcheck
}

// Chown implements afero.Fs.
func (e *errorFS) Chown(name string, uid, gid int) error {
	if name == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Chown(name, uid, gid)
}

// Chtimes implements afero.Fs.
func (e *errorFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	if name == e.errorPath {
		return os.ErrPermission
	}

	return e.fs.Chtimes(name, atime, mtime)
}
