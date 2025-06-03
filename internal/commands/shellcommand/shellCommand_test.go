// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Success(t *testing.T) {
	t.Run("basic command", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo hello", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, base, cmd.BaseCommand)
		assert.Equal(t, defaultShell(ctx), cmd.Path)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo hello"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo hello"}, cmd.Args)
		}
	})

	t.Run("command with spaces", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo hello world", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo hello world"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo hello world"}, cmd.Args)
		}
	})

	t.Run("command with double quotes", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, `echo "hello world"`, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, `echo "hello world"`}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, `echo "hello world"`}, cmd.Args)
		}
	})

	t.Run("command with single quotes", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo 'hello world'", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo 'hello world'"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo 'hello world'"}, cmd.Args)
		}
	})

	t.Run("command with pipes", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo hello | grep hello", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo hello | grep hello"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo hello | grep hello"}, cmd.Args)
		}
	})

	t.Run("command with redirection", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo hello > output.txt", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo hello > output.txt"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo hello > output.txt"}, cmd.Args)
		}
	})

	t.Run("command with environment variables", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "echo $HOME", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, "echo $HOME"}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, "echo $HOME"}, cmd.Args)
		}
	})

	t.Run("complex command with multiple features", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		complexCmd := `find /tmp -name "*.txt" | grep "test" | head -5 > results.txt && echo "Done"`
		cmd, err := New(ctx, base, complexCmd, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, complexCmd}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, complexCmd}, cmd.Args)
		}
	})

	t.Run("command with special characters", func(t *testing.T) {
		ctx := context.Background()
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

		specialCmd := `echo "Special chars: !@#$%^&*()[]{}|\\;:'\",.<>?/~"`
		cmd, err := New(ctx, base, specialCmd, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		if runtime.GOOS == GOOSWindows {
			assert.Equal(t, []string{commandSwitchWindows, specialCmd}, cmd.Args)
		} else {
			assert.Equal(t, []string{commandSwitchUnix, specialCmd}, cmd.Args)
		}
	})
}

func TestNew_EmptyCommand(t *testing.T) {
	ctx := context.Background()
	base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

	cmd, err := New(ctx, base, "", nil, nil)
	assert.Nil(t, cmd)
	require.ErrorIs(t, err, ErrCommandNotFound)
}

func TestNew_WhitespaceOnlyCommand(t *testing.T) {
	ctx := context.Background()
	base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

	// Test various whitespace-only strings
	whitespaceCommands := []string{" ", "\t", "\n", "\r", "   ", "\t\n\r   "}

	for _, wsCmd := range whitespaceCommands {
		t.Run("whitespace: "+wsCmd, func(t *testing.T) {
			cmd, err := New(ctx, base, wsCmd, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			// Whitespace commands should be passed through as-is
			if runtime.GOOS == GOOSWindows {
				assert.Equal(t, []string{commandSwitchWindows, wsCmd}, cmd.Args)
			} else {
				assert.Equal(t, []string{commandSwitchUnix, wsCmd}, cmd.Args)
			}
		})
	}
}

func TestDefaultShell(t *testing.T) {
	ctx := context.Background()

	if runtime.GOOS == GOOSWindows {
		t.Run("windows default shell", func(t *testing.T) {
			// Test with no SystemRoot env var
			originalSystemRoot := os.Getenv(winSystemRootEnv)
			os.Unsetenv(winSystemRootEnv)

			defer func() {
				if originalSystemRoot != "" {
					os.Setenv(winSystemRootEnv, originalSystemRoot)
				}
			}()

			shell := defaultShell(ctx)
			expected := `C:\Windows\System32\cmd.exe`
			assert.Equal(t, expected, shell)
		})

		t.Run("windows with custom SystemRoot", func(t *testing.T) {
			originalSystemRoot := os.Getenv(winSystemRootEnv)
			customRoot := `D:\CustomWindows`
			os.Setenv(winSystemRootEnv, customRoot)

			defer func() {
				if originalSystemRoot != "" {
					os.Setenv(winSystemRootEnv, originalSystemRoot)
				} else {
					os.Unsetenv(winSystemRootEnv)
				}
			}()

			shell := defaultShell(ctx)
			expected := `D:\CustomWindows\System32\cmd.exe`
			assert.Equal(t, expected, shell)
		})
	} else {
		t.Run("unix with SHELL env var", func(t *testing.T) {
			originalShell := os.Getenv("SHELL")
			customShell := "/usr/bin/zsh"
			os.Setenv("SHELL", customShell)

			defer func() {
				if originalShell != "" {
					os.Setenv("SHELL", originalShell)
				} else {
					os.Unsetenv("SHELL")
				}
			}()

			shell := defaultShell(ctx)
			assert.Equal(t, customShell, shell)
		})

		t.Run("unix without SHELL env var", func(t *testing.T) {
			originalShell := os.Getenv("SHELL")
			os.Unsetenv("SHELL")

			defer func() {
				if originalShell != "" {
					os.Setenv("SHELL", originalShell)
				}
			}()

			shell := defaultShell(ctx)
			assert.Equal(t, binSh, shell)
		})

		t.Run("unix with empty SHELL env var", func(t *testing.T) {
			originalShell := os.Getenv("SHELL")
			os.Setenv("SHELL", "")

			defer func() {
				if originalShell != "" {
					os.Setenv("SHELL", originalShell)
				} else {
					os.Unsetenv("SHELL")
				}
			}()

			shell := defaultShell(ctx)
			assert.Equal(t, binSh, shell)
		})
	}
}

func TestNew_WithDifferentBaseCommands(t *testing.T) {
	ctx := context.Background()

	t.Run("with custom working directory", func(t *testing.T) {
		base := runbatch.NewBaseCommand("test", "/tmp", runbatch.RunOnSuccess, nil, nil)

		cmd, err := New(ctx, base, "pwd", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, "/tmp", cmd.Cwd)
	})

	t.Run("with environment variables", func(t *testing.T) {
		env := map[string]string{"TEST_VAR": "test_value"}
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, env)

		cmd, err := New(ctx, base, "echo $TEST_VAR", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, env, cmd.Env)
	})

	t.Run("with custom run condition", func(t *testing.T) {
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnError, nil, nil)

		cmd, err := New(ctx, base, "echo error handler", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, runbatch.RunOnError, cmd.RunsOnCondition)
	})

	t.Run("with custom exit codes", func(t *testing.T) {
		exitCodes := []int{1, 2, 3}
		base := runbatch.NewBaseCommand("test", "", runbatch.RunOnExitCodes, exitCodes, nil)

		cmd, err := New(ctx, base, "echo custom exit codes", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, exitCodes, cmd.RunsOnExitCodes)
	})
}

func TestCommandConstants(t *testing.T) {
	t.Run("verify constants", func(t *testing.T) {
		assert.Equal(t, "windows", GOOSWindows)
		assert.Equal(t, "/C", commandSwitchWindows)
		assert.Equal(t, "-c", commandSwitchUnix)
		assert.Equal(t, "System32", winSystem32)
		assert.Equal(t, "cmd.exe", cmdExe)
		assert.Equal(t, "/bin/sh", binSh)
		assert.Equal(t, "SystemRoot", winSystemRootEnv)
	})
}

func TestNew_CommandLineEdgeCases(t *testing.T) {
	ctx := context.Background()
	base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

	testCases := []struct {
		name    string
		command string
	}{
		{
			name:    "command with nested quotes",
			command: `echo "He said 'Hello World'"`,
		},
		{
			name:    "command with escaped quotes",
			command: `echo "\"Hello World\""`,
		},
		{
			name:    "command with backslashes",
			command: `echo "C:\\Program Files\\Test"`,
		},
		{
			name:    "command with multiple pipes",
			command: `cat file.txt | grep "test" | sort | uniq`,
		},
		{
			name:    "command with logical operators",
			command: `test -f file.txt && echo "exists" || echo "not found"`,
		},
		{
			name:    "command with input redirection",
			command: `sort < input.txt > output.txt`,
		},
		{
			name:    "command with append redirection",
			command: `echo "log entry" >> logfile.txt`,
		},
		{
			name:    "command with stderr redirection",
			command: `command 2>&1 | tee output.log`,
		},
		{
			name:    "command with background process",
			command: `long_running_command &`,
		},
		{
			name:    "command with subshell",
			command: `echo $(date) > timestamp.txt`,
		},
		{
			name:    "command with variable assignment",
			command: `VAR=value command`,
		},
		{
			name:    "command with here document",
			command: `cat << EOF > file.txt\nHello\nWorld\nEOF`,
		},
		{
			name:    "command with glob patterns",
			command: `ls *.txt | wc -l`,
		},
		{
			name:    "command with unicode characters",
			command: `echo "Hello 世界 🌍"`,
		},
		{
			name:    "very long command",
			command: `echo "` + string(make([]byte, 1000)) + `"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := New(ctx, base, tc.command, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			assert.Equal(t, base, cmd.BaseCommand)
			assert.Equal(t, defaultShell(ctx), cmd.Path)

			expectedArgs := []string{commandSwitchUnix, tc.command}
			if runtime.GOOS == GOOSWindows {
				expectedArgs = []string{commandSwitchWindows, tc.command}
			}

			assert.Equal(t, expectedArgs, cmd.Args)
		})
	}
}
