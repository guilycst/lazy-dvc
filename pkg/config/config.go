package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

var (
	ErrMissingProviderToken = fmt.Errorf("missing GitHub token: set LDVC_GH_TOKEN (or GH_TOKEN) or LDVC_GH_TOKEN_FILE (or GH_TOKEN_FILE)")
)

type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab    ProviderType = "gitlab"
	// ProviderBitbucket ProviderType = "bitbucket"
)

type Config struct {
	GH *GithubConfig
	// CacheDir specifies the directory where fetched public keys will be cached. Defaults to "./cache".
	CacheDir string `env:"CACHE_DIR"  default:"./cache" split_words:"true"`
	// CacheDuration specifies how long fetched public keys should be cached before being refreshed. Defaults to 1 minute.
	CacheDuration time.Duration `env:"CACHE_DURATION" default:"1s" split_words:"true"`
}

type GithubConfig struct {
	OrgName     string `env:"ORG_NAME" default:"" split_words:"true"`
	TeamName    string `env:"TEAM_NAME" default:"" split_words:"true"`
	Token       string `env:"TOKEN" default:"" split_words:"true"`
	TokenFile   string `env:"TOKEN_FILE" default:"" split_words:"true"`
	MinUserRole string `env:"MIN_USER_ROLE" default:"member" split_words:"true"`
}

func loadGithubConfig(cfg *Config) error {
	cfg.GH = &GithubConfig{}
	err := envconfig.Process("LDVC_GH", cfg.GH)
	if err != nil {
		return fmt.Errorf("failed to parse gh env vars: %w", err)
	}

	if cfg.GH.Token == "" && cfg.GH.TokenFile != "" {
		content, err := os.ReadFile(cfg.GH.TokenFile)
		if err != nil {
			return fmt.Errorf("failed to read provider token file: %w", err)
		}
		cfg.GH.Token = strings.TrimSpace(string(content))
	}

	if cfg.GH.Token == "" {
		return ErrMissingProviderToken
	}

	return nil
}

func LoadConfig(cfg *Config) error {
	err := envconfig.Process("LDVC", cfg)
	if err != nil {
		return fmt.Errorf("failed to parse env vars: %w", err)
	}

	err = loadGithubConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to load GitHub config: %w", err)
	}

	return nil
}
