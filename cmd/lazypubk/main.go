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
		stop()
		<-ctx.Done()
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	parseFlags(cfg)

	close, err := logging.SetupLogger(ctx, cfg.LogFile, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer close()

	slog.DebugContext(ctx, "Starting lazypubk")
	slog.DebugContext(ctx, "Configuration", "org", cfg.GH.OrgName, "team", cfg.GH.TeamName, "cache_ttl", cfg.CacheTTL, "cache_disabled", cfg.CacheDisabled)

	if cfg.GH.Token == "" && cfg.GH.TokenFile == "" {
		return fmt.Errorf("no GitHub token provided: set LDVC_GH_TOKEN or LDVC_GH_TOKEN_FILE")
	}

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

func parseFlags(cfg *config.Config) {
	flag.Func("log-file", "Path to log file", func(value string) error {
		cfg.LogFile = value
		return nil
	})
	flag.Func("cache-ttl", "Cache TTL duration", func(value string) error {
		if duration, err := time.ParseDuration(value); err == nil {
			cfg.CacheTTL = duration
		}
		return nil
	})
	flag.Func("cache-dir", "Cache directory", func(value string) error {
		cfg.CacheDir = value
		return nil
	})
	flag.BoolFunc("v", "Enable verbose logging", func(value string) error {
		cfg.Verbose = true
		return nil
	})
	flag.BoolFunc("verbose", "Enable verbose logging", func(value string) error {
		cfg.Verbose = true
		return nil
	})
	flag.BoolFunc("no-cache", "Disable caching", func(value string) error {
		cfg.CacheDisabled = true
		return nil
	})
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
}
