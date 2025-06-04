// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package ctxlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/TylerBrock/colorjson"
	"github.com/matt-FFFFFF/porch/internal/color"
	"golang.org/x/term"
)

var (
	// ErrMarshalAttribute is returned when an error occurs while marshaling an attribute.
	ErrMarshalAttribute = errors.New("error when marshaling attribute")
	// ErrIoWrite is returned when an error occurs while writing to the output.
	ErrIoWrite = errors.New("error when writing to output")
)

const (
	// TimeFormat is the format used for timestamps in log messages.
	TimeFormat = "[15:04:05.000]"
)

var jsonFormatter = colorjson.NewFormatter()

func init() {
	jsonFormatter.Indent = 2
	jsonFormatter.DisabledColor = !term.IsTerminal(int(os.Stdout.Fd()))
}

// PrettyHandler is a custom slog handler that formats log messages to the console in a pretty way.
type PrettyHandler struct {
	h                slog.Handler
	r                func([]string, slog.Attr) slog.Attr
	b                *bytes.Buffer
	m                *sync.Mutex
	writer           io.Writer
	colour           bool
	outputEmptyAttrs bool
}

// Enabled checks if the handler is enabled for the given level.
func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// WithAttrs creates a new handler with the given attributes.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{h: h.h.WithAttrs(attrs), b: h.b, r: h.r, m: h.m, writer: h.writer, colour: h.colour}
}

// WithGroup creates a new handler with the given group name.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{h: h.h.WithGroup(name), b: h.b, r: h.r, m: h.m, writer: h.writer, colour: h.colour}
}

func (h *PrettyHandler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()

	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any

	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}

	return attrs, nil
}

// Handle implements the slog.Handler interface for PrettyHandler.
func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	var level string

	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}
	if h.r != nil {
		levelAttr = h.r([]string{}, levelAttr)
	}

	if !levelAttr.Equal(slog.Attr{}) {
		level = levelAttr.Value.String() + ":"

		switch {
		case r.Level <= slog.LevelDebug:
			level = color.Colorize(level, color.FgWhite)
		case r.Level <= slog.LevelInfo:
			level = color.Colorize(level, color.FgCyan)
		case r.Level < slog.LevelWarn:
			level = color.Colorize(level, color.FgBlue)
		case r.Level < slog.LevelError:
			level = color.Colorize(level, color.FgYellow)
		case r.Level <= slog.LevelError+1:
			level = color.Colorize(level, color.FgRed)
		default: // r.Level > slog.LevelError+1
			level = color.Colorize(level, color.FgHiMagenta)
		}
	}

	var timestamp string

	timeAttr := slog.Attr{
		Key:   slog.TimeKey,
		Value: slog.StringValue(r.Time.Format(TimeFormat)),
	}
	if h.r != nil {
		timeAttr = h.r([]string{}, timeAttr)
	}

	if !timeAttr.Equal(slog.Attr{}) {
		timestamp = color.Colorize(timeAttr.Value.String(), color.FgWhite)
	}

	var msg string

	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}
	if h.r != nil {
		msgAttr = h.r([]string{}, msgAttr)
	}

	if !msgAttr.Equal(slog.Attr{}) {
		msg = color.Colorize(msgAttr.Value.String(), color.FgHiWhite)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	var attrsAsBytes []byte

	if h.outputEmptyAttrs || len(attrs) > 0 {
		attrsAsBytes, err = jsonFormatter.Marshal(attrs)
		if err != nil {
			return errors.Join(ErrMarshalAttribute, err)
		}
	}

	out := strings.Builder{}
	if len(timestamp) > 0 {
		out.WriteString(timestamp)
		out.WriteString(" ")
	}

	if len(level) > 0 {
		out.WriteString(level)
		out.WriteString(" ")
	}

	if len(msg) > 0 {
		out.WriteString(msg)
		out.WriteString(" ")
	}

	if len(attrsAsBytes) > 0 {
		out.WriteString(color.Colorize(string(attrsAsBytes), color.FgHiWhite))
	}

	out.WriteString("\n")

	_, err = io.WriteString(h.writer, out.String())
	if err != nil {
		return errors.Join(ErrIoWrite, err)
	}

	return nil
}

func suppressDefaults(next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}

		if next == nil {
			return a
		}

		return next(groups, a)
	}
}

// NewPrettyHandler creates a new PrettyHandler with the given options.
func NewPrettyHandler(handlerOptions *slog.HandlerOptions, options ...Option) *PrettyHandler {
	if handlerOptions == nil {
		handlerOptions = &slog.HandlerOptions{}
	}

	buf := &bytes.Buffer{}
	handler := &PrettyHandler{
		b: buf,
		h: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level:       handlerOptions.Level,
			AddSource:   handlerOptions.AddSource,
			ReplaceAttr: suppressDefaults(handlerOptions.ReplaceAttr),
		}),
		r: handlerOptions.ReplaceAttr,
		m: &sync.Mutex{},
	}

	for _, opt := range options {
		opt(handler)
	}

	return handler
}

// Option implements a functional options pattern for PrettyHandler.
type Option func(h *PrettyHandler)

// WithDestinationWriter sets the destination writer for the PrettyHandler.
func WithDestinationWriter(writer io.Writer) Option {
	return func(h *PrettyHandler) {
		h.writer = writer
	}
}

// WithColour enables color output for the PrettyHandler.
func WithColour() Option {
	return func(h *PrettyHandler) {
		h.colour = true
	}
}

// WithAutoColour enables automatic color output for the PrettyHandler.
func WithAutoColour() Option {
	return func(h *PrettyHandler) {
		h.colour = color.Enabled()
	}
}

// WithOutputEmptyAttrs enables output of empty attributes for the PrettyHandler.
func WithOutputEmptyAttrs() Option {
	return func(h *PrettyHandler) {
		h.outputEmptyAttrs = true
	}
}
