package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type PrefixHandler struct {
	inner  slog.Handler
	prefix string
}

func NewPrefixHandler(inner slog.Handler, prefix string) *PrefixHandler {
	return &PrefixHandler{
		inner:  inner,
		prefix: prefix,
	}
}

func (h *PrefixHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *PrefixHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("prefix", h.prefix))
	return h.inner.Handle(ctx, r)
}

func (h *PrefixHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrefixHandler{
		inner:  h.inner.WithAttrs(attrs),
		prefix: h.prefix,
	}
}

func (h *PrefixHandler) WithGroup(name string) slog.Handler {
	return &PrefixHandler{
		inner:  h.inner.WithGroup(name),
		prefix: h.prefix,
	}
}

func NewHandler(w io.Writer, prefix string, verbose bool) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if verbose {
		opts.Level = slog.LevelDebug
	}

	textHandler := slog.NewTextHandler(w, opts)
	return NewPrefixHandler(textHandler, prefix)
}

func SetupLogger(logFile string, prefix string, verbose bool) error {
	var w io.Writer = os.Stdout

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		w = f
	}

	logger := slog.New(NewHandler(w, prefix, verbose))
	slog.SetDefault(logger)

	return nil
}
