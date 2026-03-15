package main

import (
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	targetUser := os.Args[1]

	if targetUser != "dvc-storage" {
		os.Exit(1)
	}

	org := getEnv("LDVC_GH_ORG_NAME")
	team := getEnv("LDVC_GH_TEAM_NAME")
	_ = getEnv("LDVC_GH_TOKEN_FILE")

	if org == "" {
		os.Exit(1)
	}

	cmd := exec.Command("lazypubk", "github",
		"--org", org,
		"-v",
	)

	if team != "" {
		cmd.Args = append(cmd.Args, "--team", team)
	}

	cmd.Env = append(os.Environ(),
		"LDVC_GH_TOKEN_FILE=/run/secrets/gh_token",
		"LDVC_GH_ORG_NAME="+org,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
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
