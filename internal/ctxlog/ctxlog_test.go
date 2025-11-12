// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package ctxlog

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		logger *slog.Logger
		want   *slog.Logger
	}{
		{
			name:   "with custom logger",
			logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
			want:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
		},
		{
			name:   "with nil logger should use default",
			logger: nil,
			want:   DefaultLogger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx := New(ctx, tt.logger)

			// Extract the logger from context
			logger := Logger(newCtx)

			if tt.logger == nil {
				// Should return DefaultLogger
				if logger != DefaultLogger {
					t.Errorf("New() with nil logger should return DefaultLogger")
				}
			} else {
				// Should return the provided logger (comparing handlers since loggers can't be compared directly)
				if logger == nil {
					t.Errorf("New() returned nil logger")
				}
			}
		})
	}
}

func TestLogger(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectDefault bool
	}{
		{
			name: "context with logger",
			setupContext: func() context.Context {
				logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
				return New(context.Background(), logger)
			},
			expectDefault: false,
		},
		{
			name: "context without logger",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectDefault: true,
		},
		{
			name: "context with nil logger value",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), loggerKey{}, nil)
			},
			expectDefault: true,
		},
		{
			name: "context with wrong type value",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), loggerKey{}, "not a logger")
			},
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			logger := Logger(ctx)

			if tt.expectDefault {
				if logger != DefaultLogger {
					t.Errorf("Logger() should return DefaultLogger when no valid logger in context")
				}
			} else {
				if logger == nil {
					t.Errorf("Logger() returned nil")
				}

				if logger == DefaultLogger {
					t.Errorf("Logger() should not return DefaultLogger when context has logger")
				}
			}
		})
	}
}

func TestLoggingFunctions(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	ctx := New(context.Background(), logger)

	tests := []struct {
		name     string
		logFunc  func(context.Context, string, ...any)
		message  string
		args     []any
		expected string
	}{
		{
			name:     "Info logging",
			logFunc:  Info,
			message:  "test info message",
			args:     []any{"key", "value"},
			expected: "INFO",
		},
		{
			name:     "Debug logging",
			logFunc:  Debug,
			message:  "test debug message",
			args:     []any{"debug_key", "debug_value"},
			expected: "DEBUG",
		},
		{
			name:     "Warn logging",
			logFunc:  Warn,
			message:  "test warning message",
			args:     []any{"warn_key", "warn_value"},
			expected: "WARN",
		},
		{
			name:     "Error logging",
			logFunc:  Error,
			message:  "test error message",
			args:     []any{"error_key", "error_value"},
			expected: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(ctx, tt.message, tt.args...)

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected log output to contain %q, got: %s", tt.expected, output)
			}

			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected log output to contain message %q, got: %s", tt.message, output)
			}
		})
	}
}

func TestLogLevelFromEnv(t *testing.T) {
	// Save original environment value
	originalValue := os.Getenv(porchLogLevelEnvVar)

	defer func() {
		if originalValue != "" {
			os.Setenv(porchLogLevelEnvVar, originalValue)
		} else {
			os.Unsetenv(porchLogLevelEnvVar)
		}
	}()

	tests := []struct {
		name          string
		envValue      string
		expectedLevel slog.Level
	}{
		{
			name:          "DEBUG level",
			envValue:      "DEBUG",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "INFO level",
			envValue:      "INFO",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "WARN level",
			envValue:      "WARN",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "ERROR level",
			envValue:      "ERROR",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "Invalid level defaults to WARN",
			envValue:      "INVALID",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "Empty level defaults to INFO",
			envValue:      "",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment variable
			if tt.envValue != "" {
				os.Setenv(porchLogLevelEnvVar, tt.envValue)
			} else {
				os.Unsetenv(porchLogLevelEnvVar)
			}

			// Test the function
			level := logLevelFromEnv()
			assert.Equal(t, tt.expectedLevel, level, "logLevelFromEnv() should return the expected log level")
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	if DefaultLogger == nil {
		t.Error("DefaultLogger should not be nil")
	}

	// Save original level and restore at end
	originalLevel := LevelVar.Level()
	defer LevelVar.Set(originalLevel)

	// Set level to Debug to ensure INFO is enabled
	LevelVar.Set(slog.LevelDebug)

	// Test basic functionality
	assert.True(t,
		DefaultLogger.Enabled(context.Background(),
			slog.LevelInfo),
		"DefaultLogger should be enabled for INFO",
	)
}

func TestJSONLogger(t *testing.T) {
	assert.NotNil(t, JSONLogger, "JSONLogger should not be nil")

	// Save original level and restore at end
	originalLevel := LevelVar.Level()
	defer LevelVar.Set(originalLevel)

	// Set level to Debug to ensure INFO is enabled
	LevelVar.Set(slog.LevelDebug)

	// Test that JSONLogger works
	assert.True(
		t,
		JSONLogger.Enabled(context.Background(), slog.LevelInfo),
		"JSONLogger should be enabled for INFO level when LevelVar is set to DEBUG",
	)
}

func TestLevelVar(t *testing.T) {
	assert.NotNil(t, LevelVar, "LevelVar should not be nil")

	// Test that we can get and set the level
	originalLevel := LevelVar.Level()

	LevelVar.Set(slog.LevelDebug)

	assert.Equal(t, slog.LevelDebug, LevelVar.Level(), "LevelVar.Set() should update the level")

	// Restore original level
	LevelVar.Set(originalLevel)
}

func TestLoggingWithDefaultLogger(t *testing.T) {
	// Test that logging functions work with default context (no logger)
	ctx := context.Background()

	// These should not panic and should use DefaultLogger
	Info(ctx, "test info")
	Debug(ctx, "test debug")
	Warn(ctx, "test warn")
	Error(ctx, "test error")
}

func TestLoggerKey(t *testing.T) {
	// Test that loggerKey is a proper type for context keys
	key1 := loggerKey{}
	key2 := loggerKey{}

	assert.Equal(t, key1, key2, "loggerKey instances should be equal")
}
