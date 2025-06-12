// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getUrl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		url       string
		wantErr   error
		wantBytes []byte
	}{
		{
			name:    "empty url returns error",
			url:     "",
			wantErr: ErrGetConfigFile,
		},
		{
			name:    "getter.GetFile fails",
			url:     "git::http://notexist//file.yaml",
			wantErr: ErrGetConfigFile,
		},
		{
			name:      "getter.GetFile success with URL",
			url:       "git::https://github.com/matt-FFFFFF/porch//cmd/porch/run/testdata/test.txt?ref=main",
			wantErr:   nil,
			wantBytes: []byte("this is a test file\n"),
		},
		{
			name:      "getter.GetFile succeeds",
			url:       "./testdata/test.txt",
			wantErr:   nil,
			wantBytes: []byte("this is a test file\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			bytes, err := getURL(ctx, tc.url)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, bytes)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantBytes, bytes)
			}
		})
	}
}

func Test_splitFileNameFromGetterUrl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		url      string
		wantURL  string
		wantFile string
	}{
		{
			name:     "standard getter url with file",
			url:      "git::https://github.com/user/repo//path/to/file.yaml",
			wantURL:  "git::https://github.com/user/repo//path/to",
			wantFile: "file.yaml",
		},
		{
			name:     "standard getter url with file in root",
			url:      "git::https://github.com/user/repo//file.yaml",
			wantURL:  "git::https://github.com/user/repo",
			wantFile: "file.yaml",
		},
		{
			name:     "standard getter url with file in root and ref",
			url:      "git::https://github.com/user/repo//file.yaml?ref=main",
			wantURL:  "git::https://github.com/user/repo?ref=main",
			wantFile: "file.yaml",
		},
		{
			name:     "getter url with ref query",
			url:      "git::https://github.com/user/repo//path/to/file.yaml?ref=main",
			wantURL:  "git::https://github.com/user/repo//path/to?ref=main",
			wantFile: "file.yaml",
		},
		{
			name:     "no double slash in url",
			url:      "https://github.com/user/repo/path/to/file.yaml",
			wantURL:  "",
			wantFile: "",
		},
		{
			name:     "empty string",
			url:      "",
			wantURL:  "",
			wantFile: "",
		},
		{
			name:     "trailing slash after double slash",
			url:      "git::https://github.com/user/repo//path/to/",
			wantURL:  "",
			wantFile: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotFile := splitFileNameFromGetterURL(tc.url)
			assert.Equal(t, tc.wantURL, gotURL, "url")
			assert.Equal(t, tc.wantFile, gotFile, "file name")
		})
	}
}
