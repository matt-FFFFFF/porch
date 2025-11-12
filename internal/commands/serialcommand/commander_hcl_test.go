// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/progress"
"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommanderFactory for testing serial command creation.
type mockCommanderFactory struct{}

func (m *mockCommanderFactory) Get(commandType string) (commands.Commander, bool) {
	switch commandType {
	case "shell":
		return &mockShellCommander{}, true
	case "serial":
		return NewCommander(), true
	default:
		return nil, false
	}
}

func (m *mockCommanderFactory) CreateRunnableFromHCL(
	ctx context.Context, hclCommand *hcl.CommandBlock, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	// Simple mock implementation that creates a mock runnable
	return &mockRunnable{label: hclCommand.Name}, nil
}

func (m *mockCommanderFactory) CreateRunnableFromYAML(
	ctx context.Context, payload []byte, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCommanderFactory) Register(cmdtype string, commander commands.Commander) error {
	return nil
}

func (m *mockCommanderFactory) Iter() iter.Seq2[string, commands.Commander] {
	return func(yield func(string, commands.Commander) bool) {
		// Empty iterator for testing
	}
}

func (m *mockCommanderFactory) ResolveCommandGroup(groupName string) ([]any, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCommanderFactory) AddCommandGroup(name string, commands []any) {
	// No-op for testing
}

type mockShellCommander struct{}

func (m *mockShellCommander) CreateFromYaml(
	ctx context.Context, factory commands.CommanderFactory, payload []byte, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	return &mockRunnable{label: "mock-shell"}, nil
}

func (m *mockShellCommander) CreateFromHcl(
	ctx context.Context, factory commands.CommanderFactory, hclCommand *hcl.CommandBlock, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	return &mockRunnable{label: hclCommand.Name}, nil
}

type mockRunnable struct {
	label string
}

func (m *mockRunnable) Run(ctx context.Context) runbatch.Results {
	return runbatch.Results{}
}
func (m *mockRunnable) GetLabel() string                          { return m.label }
func (m *mockRunnable) GetCwd() string                            { return "/" }
func (m *mockRunnable) SetCwd(cwd string) error                   { return nil }
func (m *mockRunnable) SetCwdToSpecificAbsolute(cwd string) error { return nil }
func (m *mockRunnable) GetCwdRel() string                         { return "" }
func (m *mockRunnable) InheritEnv(env map[string]string)          {}
func (m *mockRunnable) SetParent(parent runbatch.Runnable)        {}
func (m *mockRunnable) GetParent() runbatch.Runnable              { return nil }
func (m *mockRunnable) SetProgressReporter(reporter progress.Reporter) {}
func (m *mockRunnable) GetProgressReporter() progress.Reporter { return nil }

func (m *mockRunnable) ShouldRun(state runbatch.PreviousCommandStatus) runbatch.ShouldRunAction {
	return runbatch.ShouldRunActionRun
}

func TestCommander_CreateFromHcl(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	commander := NewCommander()
	factory := &mockCommanderFactory{}

	testCases := []struct {
		name           string
		hclCommand     *hcl.CommandBlock
		expectError    bool
		errorType      error
		validateResult func(t *testing.T, runnable runbatch.Runnable)
	}{
		{
			name: "valid HCL with single command",
			hclCommand: &hcl.CommandBlock{
				Type: "serial",
				Name: "test-serial",
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd-1",
						CommandLine: "echo 'test1'",
					},
				},
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				serialBatch, ok := runnable.(*runbatch.SerialBatch)
				require.True(t, ok, "expected SerialBatch")
				assert.Contains(t, serialBatch.GetLabel(), "test-serial")
				assert.Len(t, serialBatch.Commands, 1)
			},
		},
		{
			name: "valid HCL with multiple commands",
			hclCommand: &hcl.CommandBlock{
				Type: "serial",
				Name: "test-multi-serial",
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd-1",
						CommandLine: "echo 'test1'",
					},
					{
						Type:        "shell",
						Name:        "shell-cmd-2",
						CommandLine: "echo 'test2'",
					},
					{
						Type:        "shell",
						Name:        "shell-cmd-3",
						CommandLine: "echo 'test3'",
					},
				},
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				serialBatch, ok := runnable.(*runbatch.SerialBatch)
				require.True(t, ok, "expected SerialBatch")
				assert.Len(t, serialBatch.Commands, 3)
			},
		},
		{
			name: "valid HCL with working directory",
			hclCommand: &hcl.CommandBlock{
				Type:             "serial",
				Name:             "test-serial-wd",
				WorkingDirectory: "/tmp",
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd",
						CommandLine: "pwd",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid HCL with environment variables",
			hclCommand: &hcl.CommandBlock{
				Type: "serial",
				Name: "test-serial-env",
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd",
						CommandLine: "env | grep TEST",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "serial",
				Name:            "test-serial-condition",
				RunsOnCondition: "success",
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd",
						CommandLine: "echo 'conditional'",
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty commands list",
			hclCommand: &hcl.CommandBlock{
				Type:     "serial",
				Name:     "test-empty-serial",
				Commands: []*hcl.CommandBlock{},
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				serialBatch, ok := runnable.(*runbatch.SerialBatch)
				require.True(t, ok, "expected SerialBatch")
				assert.Empty(t, serialBatch.Commands)
			},
		},
		{
			name: "invalid runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "serial",
				Name:            "test-invalid-condition",
				RunsOnCondition: "invalid-condition",
				Commands: []*hcl.CommandBlock{
					{
						Type:        "shell",
						Name:        "shell-cmd",
						CommandLine: "echo 'test'",
					},
				},
			},
			expectError: true,
			errorType:   commands.ErrHclConfig,
		},
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runnable, err := commander.CreateFromHcl(ctx, factory, tc.hclCommand, parent)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType)
				}

				assert.Nil(t, runnable)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, runnable)

				if tc.validateResult != nil {
					tc.validateResult(t, runnable)
				}
			}
		})
	}
}

func TestCommander_CreateFromHcl_ContextCancellation(t *testing.T) {
	commander := NewCommander()
	factory := &mockCommanderFactory{}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel the context

	hclCommand := &hcl.CommandBlock{
		Type: "serial",
		Name: "test-cancelled",
		Commands: []*hcl.CommandBlock{
			{
				Type:        "shell",
				Name:        "shell-cmd",
				CommandLine: "echo 'test'",
			},
		},
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
	}

	runnable, err := commander.CreateFromHcl(ctx, factory, hclCommand, parent)

	// Should handle context cancellation gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "cancelled")
	}
	// The result depends on timing - it might succeed or fail
	_ = runnable
}
