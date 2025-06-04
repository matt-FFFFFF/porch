// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package ctxlog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrettyHandler(t *testing.T) {
	tests := []struct {
		name    string
		options *slog.HandlerOptions
		opts    []Option
	}{
		{
			name:    "with nil options",
			options: nil,
			opts:    []Option{},
		},
		{
			name: "with custom options",
			options: &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			},
			opts: []Option{},
		},
		{
			name:    "with functional options",
			options: &slog.HandlerOptions{},
			opts: []Option{
				WithColour(),
				WithOutputEmptyAttrs(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPrettyHandler(tt.options, tt.opts...)
			require.NotNil(t, handler, "NewPrettyHandler() should not return nil")

			assert.NotNil(t, handler.h, "NewPrettyHandler() created handler with nil inner handler")
			assert.NotNil(t, handler.b, "NewPrettyHandler() created handler with nil buffer")
			assert.NotNil(t, handler.m, "NewPrettyHandler() created handler with nil mutex")
		})
	}
}

func TestPrettyHandler_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		level   slog.Level
		options *slog.HandlerOptions
		want    bool
	}{
		{
			name:    "debug level with debug handler",
			level:   slog.LevelDebug,
			options: &slog.HandlerOptions{Level: slog.LevelDebug},
			want:    true,
		},
		{
			name:    "debug level with info handler",
			level:   slog.LevelDebug,
			options: &slog.HandlerOptions{Level: slog.LevelInfo},
			want:    false,
		},
		{
			name:    "info level with debug handler",
			level:   slog.LevelInfo,
			options: &slog.HandlerOptions{Level: slog.LevelDebug},
			want:    true,
		},
		{
			name:    "error level with warn handler",
			level:   slog.LevelError,
			options: &slog.HandlerOptions{Level: slog.LevelWarn},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPrettyHandler(tt.options)

			got := handler.Enabled(context.Background(), tt.level)
			assert.Equal(t, tt.want, got, "PrettyHandler.Enabled() = %v, want %v", got, tt.want)
		})
	}
}

func TestPrettyHandler_WithAttrs(t *testing.T) {
	handler := NewPrettyHandler(&slog.HandlerOptions{})
	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	newHandler := handler.WithAttrs(attrs)
	if newHandler == nil {
		t.Error("WithAttrs() returned nil")
	}

	prettyHandler, ok := newHandler.(*PrettyHandler)
	assert.True(t, ok, "WithAttrs() did not return *PrettyHandler")

	// Should share the same buffer and mutex
	assert.Equal(t, prettyHandler.b, handler.b, "WithAttrs() should share the same buffer")
	assert.Equal(t, prettyHandler.m, handler.m, "WithAttrs() should share the same mutex")
}

func TestPrettyHandler_WithGroup(t *testing.T) {
	handler := NewPrettyHandler(&slog.HandlerOptions{})
	groupName := "test_group"

	newHandler := handler.WithGroup(groupName)
	assert.NotNil(t, newHandler, "WithGroup() returned nil")

	prettyHandler, ok := newHandler.(*PrettyHandler)
	assert.True(t, ok, "WithGroup() did not return *PrettyHandler")

	// Should share the same buffer and mutex
	assert.Equal(t, handler.b, prettyHandler.b, "WithGroup() should share the same buffer")

	assert.Equal(t, handler.m, prettyHandler.m, "WithGroup() should share the same mutex")
}

func TestPrettyHandler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		level          slog.Level
		message        string
		attrs          []any
		options        []Option
		expectInOutput []string
	}{
		{
			name:    "basic info message",
			level:   slog.LevelInfo,
			message: "test message",
			attrs:   []any{},
			expectInOutput: []string{
				"INFO:",
				"test message",
			},
		},
		{
			name:    "debug message with attributes",
			level:   slog.LevelDebug,
			message: "debug message",
			attrs:   []any{"key", "value", "number", 42},
			expectInOutput: []string{
				"DEBUG:",
				"debug message",
				"key",
				"value",
				"42",
			},
		},
		{
			name:    "warning message",
			level:   slog.LevelWarn,
			message: "warning message",
			attrs:   []any{},
			expectInOutput: []string{
				"WARN:",
				"warning message",
			},
		},
		{
			name:    "error message",
			level:   slog.LevelError,
			message: "error message",
			attrs:   []any{},
			expectInOutput: []string{
				"ERROR:",
				"error message",
			},
		},
		{
			name:    "message with empty attrs output enabled",
			level:   slog.LevelInfo,
			message: "test message",
			attrs:   []any{},
			options: []Option{WithOutputEmptyAttrs()},
			expectInOutput: []string{
				"INFO:",
				"test message",
				"{}", // empty JSON object should be output
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := append([]Option{WithDestinationWriter(&buf)}, tt.options...)
			handler := NewPrettyHandler(&slog.HandlerOptions{
				Level: slog.LevelDebug, // Enable all levels for testing
			}, opts...)

			// Create a record
			record := slog.NewRecord(time.Now(), tt.level, tt.message, 0)
			record.Add(tt.attrs...)

			err := handler.Handle(context.Background(), record)
			require.NoError(t, err, "Handle() should not return an error")

			output := buf.String()
			for _, expected := range tt.expectInOutput {
				assert.Contains(t, output, expected, "Expected output to contain %q", expected)
			}

			// Should end with newline
			assert.True(t, strings.HasSuffix(output, "\n"), "Output should end with newline")
		})
	}
}

func TestPrettyHandler_Handle_WithReplaceAttr(t *testing.T) {
	var buf bytes.Buffer

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{} // Remove time
		}

		if a.Key == "secret" {
			return slog.String("secret", "[REDACTED]")
		}

		return a
	}

	handler := NewPrettyHandler(&slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	}, WithDestinationWriter(&buf))

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.Add("secret", "password123", "public", "data")

	err := handler.Handle(context.Background(), record)
	require.NoError(t, err, "Handle() should not return an error")

	output := buf.String()

	// Should contain redacted secret
	assert.Contains(t, output, "[REDACTED]", "Expected secret to be redacted")

	// Should not contain original password
	assert.NotContains(t, output, "password123", "Original password should not appear in output")

	// Should contain public data
	assert.Contains(t, output, "public", "Public data should appear in output")
}

func TestPrettyHandler_computeAttrs_Error(t *testing.T) {
	// Create a handler that will fail during inner handle
	handler := &PrettyHandler{
		h: &failingHandler{},
		b: &bytes.Buffer{},
		m: &sync.Mutex{},
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	_, err := handler.computeAttrs(context.Background(), record)

	assert.Error(t, err, "computeAttrs() should return an error when inner handler fails")
}

func TestFunctionalOptions(t *testing.T) {
	t.Run("WithDestinationWriter", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(nil, WithDestinationWriter(&buf))

		assert.Equal(t, &buf, handler.writer, "WithDestinationWriter() did not set writer correctly")
	})

	t.Run("WithColour", func(t *testing.T) {
		handler := NewPrettyHandler(nil, WithColour())

		assert.True(t, handler.colour, "WithColour() did not enable colour output")
	})

	t.Run("WithOutputEmptyAttrs", func(t *testing.T) {
		handler := NewPrettyHandler(nil, WithOutputEmptyAttrs())

		assert.True(t, handler.outputEmptyAttrs, "WithOutputEmptyAttrs() did not enable outputEmptyAttrs")
	})
}

func TestSuppressDefaults(t *testing.T) {
	suppressFunc := suppressDefaults(nil)

	tests := []struct {
		name string
		attr slog.Attr
		want slog.Attr
	}{
		{
			name: "time key should be suppressed",
			attr: slog.Time(slog.TimeKey, time.Now()),
			want: slog.Attr{},
		},
		{
			name: "level key should be suppressed",
			attr: slog.Any(slog.LevelKey, slog.LevelInfo),
			want: slog.Attr{},
		},
		{
			name: "message key should be suppressed",
			attr: slog.String(slog.MessageKey, "test"),
			want: slog.Attr{},
		},
		{
			name: "custom key should not be suppressed",
			attr: slog.String("custom", "value"),
			want: slog.String("custom", "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suppressFunc([]string{}, tt.attr)
			assert.Truef(t, got.Equal(tt.want), "suppressDefaults() = %v, want %v", got, tt.want)
		})
	}
}

func TestSuppressDefaults_WithNext(t *testing.T) {
	nextFunc := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "transform" {
			return slog.String("transform", "transformed")
		}

		return a
	}

	suppressFunc := suppressDefaults(nextFunc)

	tests := []struct {
		name string
		attr slog.Attr
		want slog.Attr
	}{
		{
			name: "time key should still be suppressed",
			attr: slog.Time(slog.TimeKey, time.Now()),
			want: slog.Attr{},
		},
		{
			name: "transform key should be transformed",
			attr: slog.String("transform", "original"),
			want: slog.String("transform", "transformed"),
		},
		{
			name: "other key should pass through",
			attr: slog.String("other", "value"),
			want: slog.String("other", "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suppressFunc([]string{}, tt.attr)
			assert.Truef(t, got.Equal(tt.want), "suppressDefaults() with next = %v, want %v", got, tt.want)
		})
	}
}

func TestErrorConstants(t *testing.T) {
	require.Error(t, ErrMarshalAttribute, "ErrMarshalAttribute should not be nil")
	require.Error(t, ErrIoWrite, "ErrIoWrite should not be nil")

	require.Error(t, ErrIoWrite, "ErrIoWrite should have non-empty error message")
	require.Error(t, ErrMarshalAttribute, "ErrMarshalAttribute should have non-empty error message")
}

// Helper types for testing.
type failingHandler struct{}

func (h *failingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *failingHandler) Handle(ctx context.Context, r slog.Record) error {
	return errors.New("failing handler error")
}

func (h *failingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *failingHandler) WithGroup(name string) slog.Handler {
	return h
}

type failingWriter struct{}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

func TestPrettyHandler_Handle_WriteError(t *testing.T) {
	handler := NewPrettyHandler(&slog.HandlerOptions{
		Level: slog.LevelDebug,
	}, WithDestinationWriter(&failingWriter{}))

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(context.Background(), record)

	if err == nil {
		t.Error("Handle() should return error when writer fails")
	}

	if !errors.Is(err, ErrIoWrite) {
		t.Errorf("Handle() should return ErrIoWrite, got: %v", err)
	}
}

func TestPrettyHandler_LevelColors(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&slog.HandlerOptions{
		Level: slog.LevelDebug,
	}, WithDestinationWriter(&buf), WithColour())

	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
		slog.LevelError + 2, // Higher than error
	}

	for _, level := range levels {
		buf.Reset()

		record := slog.NewRecord(time.Now(), level, "test message", 0)

		err := handler.Handle(context.Background(), record)
		require.NoError(t, err, "Handle() should not return error for level %v", level)

		output := buf.String()
		assert.NotEmpty(t, output, "Output should not be empty for level %v", level)
	}
}
