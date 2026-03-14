# lazy-dvc
[![CI](https://github.com/guilycst/lazy-dvc/actions/workflows/ci.yml/badge.svg)](https://github.com/guilycst/lazy-dvc/actions/workflows/ci.yml)
[![Docker](https://github.com/guilycst/lazy-dvc/actions/workflows/docker.yml/badge.svg)](https://github.com/guilycst/lazy-dvc/actions/workflows/docker.yml)

A serverless-style LFS alternative that uses GitHub Org membership as identity and S3 as storage, piped through a zero-config SSH tunnel.

## Pipeline

- `ci` workflow runs on push/PR to `main` with:
	- `gofmt` check
	- `go vet ./...`
	- `go test ./...`
	- `go build ./...`

- `docker` workflow runs on push/PR to `main` and on tags (`v*`):
	- PRs: build Docker image only (no publish)
	- Push to `main`: publish `latest` and `sha-*` tags
	- Push tag `vX.Y.Z`: publish version tag

Published image:

- `ghcr.io/guilycst/lazy-dvc`
