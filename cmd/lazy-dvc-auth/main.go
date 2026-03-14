package main

import (
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	targetUser := os.Args[1]

	if targetUser != "dvc-storage" {
		os.Exit(1)
	}

	org := os.Getenv("LDVC_GH_ORG_NAME")
	if org == "" {
		os.Exit(1)
	}

	cmd := exec.Command("lazypubk", "github",
		"--org", org,
	)

	team := os.Getenv("LDVC_GH_TEAM_NAME")
	if team != "" {
		cmd.Args = append(cmd.Args, "--team", team)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}
