package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/guilycst/lazy-dvc/pkg/config"
	"github.com/guilycst/lazy-dvc/pkg/logging"
)

const (
	DefaultLogFifo = "/tmp/authpubk_fifo"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed", "error", err)
		stop()
		<-ctx.Done()
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Load config from env vars and /etc/lazy-dvc/env
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Flags override env vars
	var (
		verbose bool
		logFile string
	)

	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&logFile, "log-file", "", "Path to log file (default: stderr)")
	flag.Parse()

	// Apply flag overrides
	if verbose {
		cfg.Verbose = true
	}
	if logFile != "" {
		cfg.LogFile = logFile
	}
	// Default to FIFO for prefixed logging when no log file specified
	if cfg.LogFile == "" {
		cfg.LogFile = DefaultLogFifo
	}

	// Validate user argument
	if len(flag.Args()) < 1 {
		slog.Error("Missing user argument")
		os.Exit(1)
	}

	targetUser := flag.Arg(0)

	if targetUser != "dvc-storage" {
		slog.Error("Invalid user", "user", targetUser)
		os.Exit(1)
	}

	// Setup logging
	close, err := logging.SetupLogger(ctx, cfg.LogFile, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer func() { _ = close() }()

	slog.DebugContext(ctx, "Starting authpubk")
	slog.DebugContext(ctx, "Configuration", "org", cfg.GH.OrgName, "team", cfg.GH.TeamName, "user", targetUser)

	// Validate required config
	if cfg.GH.OrgName == "" {
		slog.ErrorContext(ctx, "Missing LDVC_GH_ORG_NAME")
		os.Exit(1)
	}

	if cfg.GH.Token == "" && cfg.GH.TokenFile == "" {
		cfg.GH.TokenFile = "/run/secrets/gh_token"
	}

	slog.InfoContext(ctx, "Fetching keys", "org", cfg.GH.OrgName, "team", cfg.GH.TeamName, "user", targetUser)

	// Build lazypubk command
	args := []string{"--org", cfg.GH.OrgName}
	if cfg.GH.TeamName != "" {
		args = append(args, "--team", cfg.GH.TeamName)
	}
	if cfg.Verbose {
		args = append(args, "-v")
	}

	cmd := exec.CommandContext(ctx, "lazypubk", args...)
	cmd.Env = append(os.Environ(),
		"LDVC_GH_TOKEN_FILE="+cfg.GH.TokenFile,
		"LDVC_GH_ORG_NAME="+cfg.GH.OrgName,
	)
	if cfg.LogFile != "" {
		cmd.Env = append(cmd.Env, "LDVC_LOG_FILE="+cfg.LogFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "lazypubk failed", "error", err)
		return fmt.Errorf("failed to execute lazypubk: %w", err)
	}

	slog.InfoContext(ctx, "Keys fetched successfully")
	return nil
}
