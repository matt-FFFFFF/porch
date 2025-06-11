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
			name:      "getter.GetFile succeeds",
			url:       "./testdata/test.txt",
			wantErr:   nil,
			wantBytes: []byte("this is a test file\n"),
		},
		// Add more cases as needed, e.g., success case with a local file
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			bytes, err := getUrl(ctx, tc.url)
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, bytes)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantBytes, bytes)
			}
		})
	}
}
