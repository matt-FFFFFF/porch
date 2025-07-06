// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package hcl

import "github.com/spf13/afero"

// FsFactory is a function that returns an afero filesystem.
var FsFactory = func() afero.Fs {
	return afero.NewOsFs()
}
