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

var (
	verbose       bool
	logFile       string
	cacheTTL      string
	cacheDir      string
	cacheDisabled bool
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
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&logFile, "log-file", "", "Path to log file (default: stdout)")
	flag.StringVar(&cacheTTL, "cache-ttl", "5m", "Cache TTL duration (golang duration format)")
	flag.StringVar(&cacheDir, "cache-dir", pubkeyprovider.DefaultCacheDir, "Cache directory")
	flag.BoolVar(&cacheDisabled, "no-cache", false, "Disable caching")

	cfg := &config.Config{}
	err := config.LoadConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	flag.Func("org", "GitHub organization name", func(value string) error {
		cfg.GH.OrgName = value
		return nil
	})
	flag.Func("team", "GitHub team name", func(value string) error {
		cfg.GH.TeamName = value
		return nil
	})
	flag.Func("gh-token", "The token to authenticate with the provider", func(value string) error {
		cfg.GH.Token = value
		return nil
	})
	flag.Func("gh-token-file", "Path to a file containing the provider token", func(value string) error {
		cfg.GH.TokenFile = value
		return nil
	})

	flag.Parse()

	if envLogFile := os.Getenv("LDVC_LOG_FILE"); envLogFile != "" && logFile == "" {
		logFile = envLogFile
	}

	if envCacheTTL := os.Getenv("LDVC_CACHE_TTL"); envCacheTTL != "" && cacheTTL == "5m" {
		cacheTTL = envCacheTTL
	}

	if envCacheDisabled := os.Getenv("LDVC_CACHE_DISABLED"); envCacheDisabled == "true" {
		cacheDisabled = true
	}

	if err := logging.SetupLogger(logFile, "lazypubk", verbose); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	slog.Debug("Starting lazypubk...")

	if verbose {
		slog.Debug("Verbose logging enabled")
	}

	if cfg.GH.Token == "" && cfg.GH.TokenFile == "" {
		return fmt.Errorf("no provider token provided. Please set either LDVC_GH_TOKEN or LDVC_GH_TOKEN_FILE environment variable, or use the corresponding command line flags")
	}

	slog.Debug("Using GitHub provider")

	ghProvider := pubkeyprovider.NewGitHubProvider(cfg.GH.Token)

	var provider pubkeyprovider.Provider = ghProvider

	if !cacheDisabled {
		ttl, err := time.ParseDuration(cacheTTL)
		if err != nil {
			slog.Debug("Failed to parse cache TTL, using default", "error", err, "default", pubkeyprovider.DefaultCacheTTL)
			ttl = pubkeyprovider.DefaultCacheTTL
		}

		provider = pubkeyprovider.NewCachedProvider(ghProvider,
			pubkeyprovider.WithCacheDir(cacheDir),
			pubkeyprovider.WithCacheTTL(ttl),
		)
		slog.Debug("Caching enabled", "ttl", ttl, "dir", cacheDir)
	} else {
		slog.Debug("Caching disabled")
	}

	keys, err := provider.GetUsersPublicKeys(ctx, cfg.GH.OrgName, pubkeyprovider.WithTeamName(cfg.GH.TeamName), pubkeyprovider.WithMinUserRole(cfg.GH.MinUserRole))
	if err != nil {
		return fmt.Errorf("failed to fetch public keys from provider: %w", err)
	}

	slog.Debug("Fetched public keys from provider", "num_keys", len(keys))

	for _, key := range keys {
		fmt.Println(key)
	}

	return nil
}
