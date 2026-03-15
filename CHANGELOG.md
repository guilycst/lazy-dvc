# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2025-03-15

### Added
- GitHub SSH key authentication via `lazypubk` binary
- SSHAuthorizedKeysCommand bridge via `authpubk` binary
- Minimal restricted shell via `noshell` binary
- Caching with TTL and file locking to avoid GitHub API rate limits
- Unified container logging with process prefixes
- Alpine-based Docker image (Go 1.26 + Alpine 3.23)
- Process supervision in entrypoint.sh (no zombie processes)
- Health checks for SSH and rclone mount
- Multi-arch Docker images (linux/amd64, linux/arm64)
- Multi-platform binaries (linux/darwin/windows, amd64/arm64)
- CI/CD pipeline with automated releases

### Security
- Public key auth only (no passwords)
- Keys fetched dynamically from GitHub org/team
- User chrooted to data directory
- Interactive shell disabled
- TCP forwarding disabled

### Binaries
- `lazypubk` - Fetches SSH public keys from GitHub org/team members
- `authpubk` - SSH AuthorizedKeysCommand wrapper
- `noshell` - Minimal shell for SSH/SFTP sessions