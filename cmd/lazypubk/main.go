package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/guilycst/lazy-dvc/internal/pubkeyprovider"
	"github.com/guilycst/lazy-dvc/pkg/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Debug("Starting lazydvc...")

	start(ctx)
	stop()

	<-ctx.Done()
	slog.Debug("Received termination signal, shutting down...")
}

func start(ctx context.Context) {

	cfg := &config.Config{}
	err := config.LoadConfig(cfg)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	flag.Func("gh-token", "The token to authenticate with the provider", func(value string) error {
		cfg.GH.Token = value
		return nil
	})
	flag.Func("gh-token-file", "Path to a file containing the provider token", func(value string) error {
		cfg.GH.TokenFile = value
		return nil
	})

	flag.Parse()

	// This is fine for now since we only have one provider
	if cfg.GH.Token == "" && cfg.GH.TokenFile == "" {
		slog.Error("No provider token provided. Please set either LDVC_GH_TOKEN or LDVC_GH_TOKEN_FILE environment variable, or use the corresponding command line flags.")
		os.Exit(1)
	}

	slog.Debug("Using GitHub provider")
	provider := pubkeyprovider.NewGitHubProvider(cfg.GH.Token)

	keys, err := provider.GetUsersPublicKeys(ctx, cfg.GH.OrgName, pubkeyprovider.WithTeamName(cfg.GH.TeamName), pubkeyprovider.WithMinUserRole(cfg.GH.MinUserRole))
	if err != nil {
		slog.Error("Failed to fetch public keys from provider", "error", err)
		os.Exit(1)
	}

	slog.Debug("Fetched public keys from provider", "num_keys", len(keys))

	for _, key := range keys {
		fmt.Println(key)
	}

}
