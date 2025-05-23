// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch_test

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCommand struct {
	label      string
	cwd        string
	executedOn []string
}

func (t *testCommand) SetCwd(cwd string) {
	t.cwd = cwd
}

// InheritEnv sets the environment variables for the batch.
func (t *testCommand) InheritEnv(_ map[string]string) {
	// No-op for the test command
}

func (t *testCommand) Run(ctx context.Context) runbatch.Results {
	return runbatch.Results{
		{
			Label:    t.label,
			ExitCode: 0,
		},
	}
}

func TestForEachCommandSerial(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	provider := func(ctx context.Context, _ string) ([]string, error) {
		return items, nil
	}

	testCmd1 := &testCommand{label: "Test Command 1"}
	testCmd2 := &testCommand{label: "Test Command 2"}

	cmd := runbatch.NewForEachCommand(
		"Test ForEach",
		provider,
		runbatch.ForEachSerial,
		testCmd1, testCmd2,
	)

	results := cmd.Run(context.Background())

	assert.Len(t, results, 1)
	require.NotNil(t, results[0])

	// In serial mode, we should have 3 child results (one for each item)
	assert.Equal(t, "Test ForEach", results[0].Label)
	assert.Equal(t, 0, results[0].ExitCode)
	assert.NoError(t, results[0].Error)
	assert.NotEmpty(t, results[0].Children)
}

func TestForEachCommandParallel(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	provider := func(ctx context.Context, _ string) ([]string, error) {
		return items, nil
	}

	testCmd1 := &testCommand{label: "Test Command 1"}

	cmd := runbatch.NewForEachCommand(
		"Test ForEach Parallel",
		provider,
		runbatch.ForEachParallel,
		testCmd1,
	)

	results := cmd.Run(context.Background())

	assert.Len(t, results, 1)
	require.NotNil(t, results[0])

	// In parallel mode, we should have results for each item
	assert.Equal(t, "Test ForEach Parallel", results[0].Label)
	assert.Equal(t, 0, results[0].ExitCode)
	assert.NoError(t, results[0].Error)
	assert.NotEmpty(t, results[0].Children)
}

func TestForEachCommandEmptyList(t *testing.T) {
	provider := func(ctx context.Context, _ string) ([]string, error) {
		return []string{}, nil
	}

	testCmd := &testCommand{label: "Test Command"}

	cmd := runbatch.NewForEachCommand(
		"Test Empty ForEach",
		provider,
		runbatch.ForEachSerial,
		testCmd,
	)

	results := cmd.Run(context.Background())

	assert.Len(t, results, 1)
	require.NotNil(t, results[0])

	// With an empty list, we should have no errors and no child results
	assert.Equal(t, "Test Empty ForEach", results[0].Label)
	assert.Equal(t, 0, results[0].ExitCode)
	assert.NoError(t, results[0].Error)
	assert.Empty(t, results[0].Children)
}

func TestForEachCommandFailingProvider(t *testing.T) {
	expectedErr := assert.AnError
	provider := func(ctx context.Context, _ string) ([]string, error) {
		return nil, expectedErr
	}

	testCmd := &testCommand{label: "Test Command"}

	cmd := runbatch.NewForEachCommand(
		"Test Failing Provider",
		provider,
		runbatch.ForEachSerial,
		testCmd,
	)

	results := cmd.Run(context.Background())

	assert.Len(t, results, 1)
	require.NotNil(t, results[0])

	// Provider error should be propagated
	assert.Equal(t, "Test Failing Provider", results[0].Label)
	assert.Equal(t, -1, results[0].ExitCode)
	assert.ErrorIs(t, results[0].Error, runbatch.ErrItemsProviderFailed)
	assert.Empty(t, results[0].Children)
}
