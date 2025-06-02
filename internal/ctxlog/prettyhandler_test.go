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
			if handler == nil {
				t.Error("NewPrettyHandler() returned nil")
			}
			if handler.h == nil {
				t.Error("NewPrettyHandler() created handler with nil inner handler")
			}
			if handler.b == nil {
				t.Error("NewPrettyHandler() created handler with nil buffer")
			}
			if handler.m == nil {
				t.Error("NewPrettyHandler() created handler with nil mutex")
			}
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
			if got != tt.want {
				t.Errorf("PrettyHandler.Enabled() = %v, want %v", got, tt.want)
			}
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
	if !ok {
		t.Error("WithAttrs() did not return *PrettyHandler")
	}

	// Should share the same buffer and mutex
	if prettyHandler.b != handler.b {
		t.Error("WithAttrs() should share the same buffer")
	}
	if prettyHandler.m != handler.m {
		t.Error("WithAttrs() should share the same mutex")
	}
}

func TestPrettyHandler_WithGroup(t *testing.T) {
	handler := NewPrettyHandler(&slog.HandlerOptions{})
	groupName := "test_group"

	newHandler := handler.WithGroup(groupName)
	if newHandler == nil {
		t.Error("WithGroup() returned nil")
	}

	prettyHandler, ok := newHandler.(*PrettyHandler)
	if !ok {
		t.Error("WithGroup() did not return *PrettyHandler")
	}

	// Should share the same buffer and mutex
	if prettyHandler.b != handler.b {
		t.Error("WithGroup() should share the same buffer")
	}
	if prettyHandler.m != handler.m {
		t.Error("WithGroup() should share the same mutex")
	}
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
			if err != nil {
				t.Errorf("Handle() returned error: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got: %s", expected, output)
				}
			}

			// Should end with newline
			if !strings.HasSuffix(output, "\n") {
				t.Error("Output should end with newline")
			}
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
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	output := buf.String()

	// Should contain redacted secret
	if !strings.Contains(output, "[REDACTED]") {
		t.Error("Expected secret to be redacted")
	}

	// Should not contain original password
	if strings.Contains(output, "password123") {
		t.Error("Original password should not appear in output")
	}

	// Should contain public data
	if !strings.Contains(output, "public") {
		t.Error("Public data should appear in output")
	}
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

	if err == nil {
		t.Error("computeAttrs() should return error when inner handler fails")
	}
}

func TestFunctionalOptions(t *testing.T) {
	t.Run("WithDestinationWriter", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(nil, WithDestinationWriter(&buf))

		if handler.writer != &buf {
			t.Error("WithDestinationWriter() did not set writer correctly")
		}
	})

	t.Run("WithColour", func(t *testing.T) {
		handler := NewPrettyHandler(nil, WithColour())

		if !handler.colour {
			t.Error("WithColour() did not enable colour")
		}
	})

	t.Run("WithAutoColour", func(t *testing.T) {
		_ = NewPrettyHandler(nil, WithAutoColour())

		// The value depends on the color.Enabled() function
		// We just test that it sets the field
		// (the value could be true or false depending on environment)
	})

	t.Run("WithOutputEmptyAttrs", func(t *testing.T) {
		handler := NewPrettyHandler(nil, WithOutputEmptyAttrs())

		if !handler.outputEmptyAttrs {
			t.Error("WithOutputEmptyAttrs() did not enable outputEmptyAttrs")
		}
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
			if !got.Equal(tt.want) {
				t.Errorf("suppressDefaults() = %v, want %v", got, tt.want)
			}
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
			if !got.Equal(tt.want) {
				t.Errorf("suppressDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	if ErrMarshalAttribute == nil {
		t.Error("ErrMarshalAttribute should not be nil")
	}

	if ErrIoWrite == nil {
		t.Error("ErrIoWrite should not be nil")
	}

	if ErrMarshalAttribute.Error() == "" {
		t.Error("ErrMarshalAttribute should have non-empty error message")
	}

	if ErrIoWrite.Error() == "" {
		t.Error("ErrIoWrite should have non-empty error message")
	}
}

// Helper types for testing

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

type invalidJSONMarshaller struct{}

func (m invalidJSONMarshaller) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

func TestTimeFormat(t *testing.T) {
	if TimeFormat != "[15:04:05.000]" {
		t.Errorf("timeFormat = %q, want %q", TimeFormat, "[15:04:05.000]")
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
		if err != nil {
			t.Errorf("Handle() returned error for level %v: %v", level, err)
		}

		output := buf.String()
		if output == "" {
			t.Errorf("No output for level %v", level)
		}
	}
}
