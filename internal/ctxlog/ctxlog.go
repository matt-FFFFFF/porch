// Package ctxlog provides a context-based logger that can be used to log messages
// with different log levels. It uses the slog package for structured logging.
// The log level is set based on the environment variable, which allows for
// dynamic configuration of the log level at runtime.
// The variable name for the log level is derived from the executable name.
// For example, if the executable is named "myapp", the environment variable
// for the log level would be "MYAPP_LOG_LEVEL". The log level can be set to
// "DEBUG", "INFO", "WARN", "ERROR", or any other value will default to "WARN".
package ctxlog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type loggerKey struct{}

// DefaultLogger is a text logger that is used if no logger is provided.
var DefaultLogger = slog.New(NewPretty(&slog.HandlerOptions{
	Level: LevelVar,
},
	WithColor(),
	WithDestinationWriter(os.Stdout),
))

var JsonLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: LevelVar,
}))

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

	return context.WithValue(ctx, loggerKey{}, logger)
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
		return slog.LevelWarn
	}
}
