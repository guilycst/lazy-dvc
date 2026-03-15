package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilycst/lazy-dvc/internal/pubkeyprovider"
	"github.com/guilycst/lazy-dvc/pkg/config"
	"github.com/guilycst/lazy-dvc/pkg/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("Failed", "error", err)
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
		verbose       bool
		logFile       string
		cacheTTL      string
		cacheDir      string
		cacheDisabled bool
	)

	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&logFile, "log-file", "", "Path to log file (default: stderr)")
	flag.StringVar(&cacheTTL, "cache-ttl", "", "Cache TTL duration (golang format)")
	flag.StringVar(&cacheDir, "cache-dir", "", "Cache directory")
	flag.BoolVar(&cacheDisabled, "no-cache", false, "Disable caching")

	flag.Func("org", "GitHub organization name", func(value string) error {
		cfg.GH.OrgName = value
		return nil
	})
	flag.Func("team", "GitHub team name", func(value string) error {
		cfg.GH.TeamName = value
		return nil
	})
	flag.Func("gh-token", "GitHub API token", func(value string) error {
		cfg.GH.Token = value
		return nil
	})
	flag.Func("gh-token-file", "Path to file containing GitHub API token", func(value string) error {
		cfg.GH.TokenFile = value
		return nil
	})

	flag.Parse()

	// Apply flag overrides
	if verbose {
		cfg.Verbose = true
	}
	if logFile != "" {
		cfg.LogFile = logFile
	}
	if cacheTTL != "" {
		if duration, err := time.ParseDuration(cacheTTL); err == nil {
			cfg.CacheTTL = duration
		}
	}
	if cacheDir != "" {
		cfg.CacheDir = cacheDir
	}
	if cacheDisabled {
		cfg.CacheDisabled = true
	}

	// Setup logging
	if err := logging.SetupLogger(cfg.LogFile, "lazypubk", cfg.Verbose); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	slog.DebugContext(ctx, "Starting lazypubk")
	slog.DebugContext(ctx, "Configuration", "org", cfg.GH.OrgName, "team", cfg.GH.TeamName, "cache_ttl", cfg.CacheTTL, "cache_disabled", cfg.CacheDisabled)

	// Validate required config
	if cfg.GH.Token == "" && cfg.GH.TokenFile == "" {
		return fmt.Errorf("no GitHub token provided: set LDVC_GH_TOKEN or LDVC_GH_TOKEN_FILE")
	}

	// Create provider
	ghProvider := pubkeyprovider.NewGitHubProvider(cfg.GH.Token)

	var provider pubkeyprovider.Provider = ghProvider

	if !cfg.CacheDisabled {
		provider = pubkeyprovider.NewCachedProvider(ghProvider,
			pubkeyprovider.WithCacheDir(cfg.CacheDir),
			pubkeyprovider.WithCacheTTL(cfg.CacheTTL),
		)
		slog.DebugContext(ctx, "Caching enabled", "ttl", cfg.CacheTTL, "dir", cfg.CacheDir)
	} else {
		slog.DebugContext(ctx, "Caching disabled")
	}

	// Fetch keys
	keys, err := provider.GetUsersPublicKeys(ctx, cfg.GH.OrgName,
		pubkeyprovider.WithTeamName(cfg.GH.TeamName),
		pubkeyprovider.WithMinUserRole(cfg.GH.MinUserRole),
	)
	if err != nil {
		return fmt.Errorf("failed to fetch public keys: %w", err)
	}

	slog.DebugContext(ctx, "Fetched public keys", "count", len(keys))

	for _, key := range keys {
		fmt.Println(key)
	}

	return nil
}
