// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"encoding/gob"
	"errors"
	"io"
)

var (
	// ErrWriteGob is returned when writing the results to a binary format fails.
	ErrWriteGob = errors.New("failed to write binary results")
)

func writeResultGob(w io.Writer, results Results) error {
	enc := gob.NewEncoder(w)
	if err := enc.Encode(results); err != nil {
		return errors.Join(ErrWriteGob, err)
	}

	return nil
}
