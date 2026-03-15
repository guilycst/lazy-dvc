package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

func NewHandler(w io.Writer, verbose bool) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if verbose {
		opts.Level = slog.LevelDebug
	}

	textHandler := slog.NewTextHandler(w, opts)
	return textHandler
}

func SetupLogger(ctx context.Context, logFile string, verbose bool) (func() error, error) {
	var w io.Writer = os.Stderr
	cf := func() error { return nil }

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return cf, fmt.Errorf("failed to open log file: %w", err)
		}
		cf = f.Close
		w = f
	}

	logger := slog.New(NewHandler(w, verbose))
	slog.SetDefault(logger)

	return cf, nil
}

func SetupLoggerWithWriter(w io.Writer, verbose bool) {
	logger := slog.New(NewHandler(w, verbose))
	slog.SetDefault(logger)
}
