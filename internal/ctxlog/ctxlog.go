// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package ctxlog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type loggerKey struct{}

// DefaultLogger is a text logger with prettified output that is used if no logger is provided.
var DefaultLogger = slog.New(NewPrettyHandler(&slog.HandlerOptions{
	Level:     LevelVar,
	AddSource: false,
},
	WithDestinationWriter(os.Stdout),
	WithAutoColour(),
))

// DebugLogger is a text logger with prettified output that is used for debug logging.
// It includes source information and is used for debugging purposes.
var DebugLogger = slog.New(NewPrettyHandler(&slog.HandlerOptions{
	Level:     LevelVar,
	AddSource: true,
},
	WithDestinationWriter(os.Stdout),
	WithAutoColour(),
))

// JSONLogger is a JSON logger that can be substituted for the default logger.
var JSONLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: LevelVar,
}))

// LevelVar is a variable that holds the log level.
// Initially this is set to the log level from the environment variable.
// It can me modified at runtime to change the log level.
var LevelVar = &slog.LevelVar{}

func init() {
	// Set the default log level based on the environment variable
	LevelVar.Set(logLevelFromEnv())
}

// New creates a new context with the given logger.
// If logger is nil, it uses the default logger.
// The log level is set based on the environment variable.
// The variable name for the log level is derived from the executable name.
// For example, if the executable is named "myapp", the environment variable
// for the log level would be "MYAPP_LOG_LEVEL". The log level can be set to
// "DEBUG", "INFO", "WARN", "ERROR", or any other value will default to "WARN".
func New(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		logger = DefaultLogger
	}

	if LevelVar.Level() == slog.LevelDebug {
		logger = DebugLogger
	}

	return context.WithValue(ctx, loggerKey{}, logger)
}

// NewForTUI creates a new context with a TUI-compatible logger that won't interfere with display.
func NewForTUI(ctx context.Context, w io.Writer) context.Context {
	// TUILogger is a logger that has a selectable output write to avoid interfering with TUI display.
	// Used when TUI mode is active to prevent log messages from corrupting the interface.
	tuiLogger := slog.New(NewPrettyHandler(&slog.HandlerOptions{
		Level: LevelVar,
	},
		WithDestinationWriter(w),
		WithAutoColour(),
	))

	return context.WithValue(ctx, loggerKey{}, tuiLogger)
}

// Logger returns the logger from the context, or the default logger if not found.
func Logger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return DefaultLogger
	}

	return logger
}

// Info logs an info message with the given context.
func Info(ctx context.Context, msg string, args ...any) {
	Logger(ctx).Info(msg, args...)
}

// Debug logs a debug message with the given context.
func Debug(ctx context.Context, msg string, args ...any) {
	Logger(ctx).Debug(msg, args...)
}

// Warn logs a warning message with the given context.
func Warn(ctx context.Context, msg string, args ...any) {
	Logger(ctx).Warn(msg, args...)
}

// Error logs a warning message with the given context.
func Error(ctx context.Context, msg string, args ...any) {
	Logger(ctx).Error(msg, args...)
}

func logLevelFromEnv() slog.Level {
	exec, _ := os.Executable()
	exec = filepath.Base(exec)
	ext := filepath.Ext(exec)

	if ext == ".exe" {
		exec = exec[:len(exec)-len(ext)]
	}

	exec = strings.ToUpper(exec)
	envName := strings.ToUpper(exec + "_LOG_LEVEL")

	// Check the environment variable for the log level
	levelStr := os.Getenv(envName)
	switch levelStr {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
