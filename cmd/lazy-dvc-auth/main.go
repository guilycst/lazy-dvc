package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/guilycst/lazy-dvc/pkg/logging"
)

var (
	verbose bool
	logFile string
)

func main() {
	if err := run(); err != nil {
		slog.Error("Failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&logFile, "log-file", "", "Path to log file (default: stdout)")
	flag.Parse()

	if len(flag.Args()) < 1 {
		slog.Error("Missing user argument")
		os.Exit(1)
	}

	targetUser := flag.Arg(0)

	if targetUser != "dvc-storage" {
		slog.Error("Invalid user", "user", targetUser)
		os.Exit(1)
	}

	if envLogFile := os.Getenv("LDVC_LOG_FILE"); envLogFile != "" && logFile == "" {
		logFile = envLogFile
	}

	if err := logging.SetupLogger(logFile, "lazy-dvc-auth", verbose); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	org := getEnv("LDVC_GH_ORG_NAME")
	team := getEnv("LDVC_GH_TEAM_NAME")
	tokenFile := getEnv("LDVC_GH_TOKEN_FILE")

	if tokenFile == "" {
		tokenFile = "/run/secrets/gh_token"
	}

	if org == "" {
		slog.Error("Missing LDVC_GH_ORG_NAME")
		os.Exit(1)
	}

	slog.Debug("Fetching keys", "org", org, "team", team, "user", targetUser)

	args := []string{"--org", org}
	if team != "" {
		args = append(args, "--team", team)
	}
	if verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command("lazypubk", args...)
	cmd.Env = append(os.Environ(),
		"LDVC_GH_TOKEN_FILE="+tokenFile,
		"LDVC_GH_ORG_NAME="+org,
	)
	if logFile != "" {
		cmd.Env = append(cmd.Env, "LDVC_LOG_FILE="+logFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		slog.Error("lazypubk failed", "error", err)
		os.Exit(1)
	}

	slog.Debug("Keys fetched successfully")
	return nil
}

func getEnv(key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	data, err := os.ReadFile("/etc/lazy-dvc/env")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}

	return ""
}
