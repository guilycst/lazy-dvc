package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	ErrMissingProviderToken = fmt.Errorf("missing GitHub token: set LDVC_GH_TOKEN or LDVC_GH_TOKEN_FILE")
)

type Config struct {
	GH *GithubConfig

	// Cache configuration
	CacheDir      string
	CacheTTL      time.Duration
	CacheDisabled bool

	// Logging configuration
	LogFile string
	Verbose bool
}

type GithubConfig struct {
	OrgName     string
	TeamName    string
	Token       string
	TokenFile   string
	MinUserRole string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		GH: &GithubConfig{
			MinUserRole: "member",
		},
		CacheDir:      "/var/cache/lazy-dvc",
		CacheTTL:      5 * time.Minute,
		CacheDisabled: false,
	}

	// Load from /etc/lazy-dvc/env file first
	if err := loadEnvFile("/etc/lazy-dvc/env", cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	// Load from environment variables (override file values)
	if v := os.Getenv("LDVC_GH_ORG_NAME"); v != "" {
		cfg.GH.OrgName = v
	}
	if v := os.Getenv("LDVC_GH_TEAM_NAME"); v != "" {
		cfg.GH.TeamName = v
	}
	if v := os.Getenv("LDVC_GH_TOKEN"); v != "" {
		cfg.GH.Token = v
	}
	if v := os.Getenv("LDVC_GH_TOKEN_FILE"); v != "" {
		cfg.GH.TokenFile = v
	}
	if v := os.Getenv("LDVC_GH_MIN_USER_ROLE"); v != "" {
		cfg.GH.MinUserRole = v
	}
	if v := os.Getenv("LDVC_CACHE_DIR"); v != "" {
		cfg.CacheDir = v
	}
	if v := os.Getenv("LDVC_CACHE_TTL"); v != "" {
		if duration, err := time.ParseDuration(v); err == nil {
			cfg.CacheTTL = duration
		}
	}
	if os.Getenv("LDVC_CACHE_DISABLED") == "true" {
		cfg.CacheDisabled = true
	}
	if v := os.Getenv("LDVC_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}
	if os.Getenv("LDVC_VERBOSE") == "true" {
		cfg.Verbose = true
	}

	// Read token from file if specified
	if cfg.GH.Token == "" && cfg.GH.TokenFile != "" {
		content, err := os.ReadFile(cfg.GH.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file: %w", err)
		}
		cfg.GH.Token = strings.TrimSpace(string(content))
	}

	return cfg, nil
}

func loadEnvFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "LDVC_GH_ORG_NAME":
			if cfg.GH.OrgName == "" {
				cfg.GH.OrgName = value
			}
		case "LDVC_GH_TEAM_NAME":
			if cfg.GH.TeamName == "" {
				cfg.GH.TeamName = value
			}
		case "LDVC_GH_TOKEN":
			if cfg.GH.Token == "" {
				cfg.GH.Token = value
			}
		case "LDVC_GH_TOKEN_FILE":
			if cfg.GH.TokenFile == "" {
				cfg.GH.TokenFile = value
			}
		case "LDVC_GH_MIN_USER_ROLE":
			cfg.GH.MinUserRole = value
		case "LDVC_CACHE_DIR":
			cfg.CacheDir = value
		case "LDVC_CACHE_TTL":
			if duration, err := time.ParseDuration(value); err == nil {
				cfg.CacheTTL = duration
			}
		case "LDVC_CACHE_DISABLED":
			if value == "true" {
				cfg.CacheDisabled = true
			}
		case "LDVC_LOG_FILE":
			cfg.LogFile = value
		case "LDVC_VERBOSE":
			if value == "true" {
				cfg.Verbose = true
			}
		}
	}

	return nil
}
