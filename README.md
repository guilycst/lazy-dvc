# lazy-dvc

[![CI](https://github.com/guilycst/lazy-dvc/actions/workflows/ci.yml/badge.svg)](https://github.com/guilycst/lazy-dvc/actions/workflows/ci.yml)
[![Docker](https://github.com/guilycst/lazy-dvc/actions/workflows/docker.yml/badge.svg)](https://github.com/guilycst/lazy-dvc/actions/workflows/docker.yml)
[![Release](https://github.com/guilycst/lazy-dvc/actions/workflows/release.yml/badge.svg)](https://github.com/guilycst/lazy-dvc/actions/workflows/release.yml)

### Large file management via the identity you already have.

**lazy-dvc** is a specialized auth-bridge designed to make sharing large assets across a team as seamless as pushing code. It allows you to use your existing GitHub SSH keys to authenticate against a DVC remote, filtered by your GitHub Organization and Team membership.

---

## The Core Philosophy

> **If you are part of the GitHub Organization where the repository lives, you should already have access to the assets.**

By using your GitHub SSH keys as the source of truth, `lazy-dvc` ensures that:

- **No Secondary Auth** — If your public key is on GitHub, you're halfway there
- **Org/Team Filtering** — Access is automatically scoped to the teams you already manage on GitHub
- **Reduced Friction** — New team members don't need a "storage onboarding" session—they just clone and pull

---

## Why Not Git LFS?

Git LFS solves large file storage, but comes with significant tradeoffs:

**GitHub LFS quotas (free tier):**

| Plan | Storage | Bandwidth/month |
|------|---------|-----------------|
| GitHub Free | 10 GB | 10 GB |
| GitHub Pro | 10 GB | 10 GB |
| GitHub Team | 250 GB | 250 GB |
| Enterprise | 250 GB | 250 GB |

**How usage is measured:**

- **Uploads** → Counts against repository owner's storage (bandwidth not measured)
- **Downloads** → Counts against repository owner's bandwidth
- **Every push** → Entire file size charged again (not delta)
- **CI/CD pulls** → Each `dvc pull` in Actions counts against bandwidth

**Example:**
```
Push 500 MB file        → 500 MB storage used
Push 1 byte change      → Another 500 MB storage (total: 1 GB)
Pull twice              → 1 GB bandwidth used
CI runs 10 times/month   → 5 GB bandwidth used
```

**Other issues:**

| Issue | Git LFS |
|-------|---------|
| **Auth** | Requires separate credentials (HTTPS + PAT) for storage |
| **History** | Rebase/filter-branch corrupts LFS pointers |
| **CI/CD** | Every job needs `git lfs install` + credentials |
| **Partial clone** | Doesn't work well with `--filter` |
| **Locking** | Optional, easy to forget, causes conflicts |
| **Vendor lock** | GitHub LFS, GitLab LFS, etc. |
| **Quota exceeded** | Can't push new files, only retrieve pointers |

## Why Not Standard DVC?

DVC is excellent, but the default setup requires managing authentication separately:

```
Standard DVC requires TWO auth methods:

  Git access:     SSH keys → GitHub
  DVC storage:    AWS keys/SSH keys/HTTP creds → Storage (separate!)
```

**The friction adds up:**

| Task | Standard DVC | lazy-dvc |
|------|--------------|----------|
| New team member | Generate SSH key, distribute to storage server | Add to GitHub team |
| Offboarding | Manually revoke SSH key from storage | Remove from GitHub team |
| Access control | Per-user key management on storage server | GitHub team membership |
| CI/CD | Configure storage credentials in every job | Use existing SSH deploy keys |
| Audit trail | Separate logs per storage server | GitHub audit logs |

---

## How lazy-dvc Solves This

lazy-dvc unifies authentication through GitHub SSH keys—**one auth method for everything**:

```
┌─────────────┐                        ┌─────────────────┐
│   Developer │     SSH keys           │    GitHub       │
│             │ ─────────────────────► │   (org/team)    │
└─────────────┘                        └─────────────────┘
       │                                      │
       │                                      │ same keys
       │                                      │
       ▼                                      ▼
┌─────────────┐                        ┌─────────────────┐
│   dvc push  │ ──── SSH/SFTP ───────► │   lazy-dvc      │
│   dvc pull  │                        │   → S3 Backend  │
└─────────────┘                        └─────────────────┘
```

**The flow:**

1. Developer pushes to Git repository (SSH key #1)
2. Developer runs `dvc push` (same SSH key #1)
3. lazy-dvc fetches public keys from GitHub org/team
4. If the key matches → access granted to storage

**No separate credentials. No key distribution. No storage onboarding.**

---

## How it Works

```
┌─────────────┐     SSH/SFTP      ┌────────────────────┐
│   Developer │ ───────────────►  │    lazy-dvc        │
│   (DVC)     │                   │  ┌──────────────┐  │
│             │                   │  │  authpubk    │──┼──► GitHub API
└─────────────┘                   │  │ (fetches keys)│ │
                                  │  └──────────────┘  │
                                  │  ┌──────────────┐  │
                                  │  │ rclone mount │──┼──► S3 Backend
                                  │  │  /data       │  │
                                  │  └──────────────┘  │
                                  └────────────────────┘
```

1. **Identity** — Your GitHub Organization remains the source of truth
2. **Automation** — `lazy-dvc` (powered by `lazypubk`) fetches public keys for authorized team members in real-time
3. **Storage** — Your assets sit on your own infrastructure (S3/FUSE/Local), accessible via a standard DVC remote over SSH
4. **Convenience** — The user experience is a simple `dvc pull`, with no extra logins required

### Binaries

lazy-dvc ships with three binaries:

| Binary | Purpose |
|--------|---------|
| `lazypubk` | Core CLI that fetches SSH public keys from GitHub org/team members |
| `authpubk` | SSH AuthorizedKeysCommand wrapper — validates user and calls lazypubk |
| `noshell` | Minimal shell for SSH/SFTP sessions |

`authpubk` exists because SSH's `AuthorizedKeysCommand` expects a specific contract: it takes a username as argument and outputs authorized_keys format to stdout. This wrapper handles that integration while keeping `lazypubk` as a reusable standalone tool.

### Storage Backend

`lazy-dvc` uses **rclone** to mount any S3-compatible storage as the DVC remote. This gives you flexibility to use:

- **AWS S3** — Amazon's managed object storage
- **MinIO** — Self-hosted S3-compatible storage
- **Ceph RADOS** — Distributed storage with S3 gateway
- **VersityGW** — Lightweight S3-compatible gateway-by [Versity](https://versity.com/)
- **Any S3-compatible backend**

For the quick start example, we use [versitygw](https://github.com/versity/versitygw) because it's easy to set up locally. VersityGW is [battle-tested and production-ready](https://github.com/versity/versitygw) with comprehensive test coverage, security testing, and industry-standard S3 client validation. Use whatever S3 backend fits your needs.

---

## Quick Start

```bash
# 1. Clone and start
git clone https://github.com/guilycst/lazy-dvc.git
cd lazy-dvc

# 2. Set your GitHub token (needs read:org scope)
export LDVC_GH_TOKEN=ghp_xxxxx

# 2. Set your GitHub org name
export LDVC_GH_ORG_NAME=myorg

# 3. Build and run
docker compose up -d --build

# 4. Configure DVC
dvc remote add -d storage ssh://dvc-storage@localhost:2222/data

# 5. Test it works
dvc push
```

---

## Requirements

- Docker & Docker Compose
- GitHub account with SSH key added
- Membership in configured GitHub organization
- GitHub PAT with `read:org` scope

---

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LDVC_GH_TOKEN` | Yes | GitHub PAT (use Docker secret) |
| `LDVC_GH_ORG_NAME` | Yes | GitHub organization name |
| `LDVC_GH_TEAM_NAME` | No | Filter to specific team |
| `LDVC_CACHE_TTL` | No | Cache duration (default: `5m`, golang duration format) |
| `LDVC_CACHE_DISABLED` | No | Set to `true` to disable caching |
| `LDVC_LOG_FILE` | No | Path to log file (default: stdout) |

### Docker Secrets

Create `gh_token.txt` with your GitHub PAT:

```bash
echo "ghp_your_token_here" > gh_token.txt
```

### Rclone Configuration

For production S3 backends, configure these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RCLONE_S3_ENDPOINT` | — | S3 endpoint URL (required for S3) |
| `RCLONE_VFS_CACHE_MODE` | `full` | VFS cache mode |
| `RCLONE_VFS_CACHE_MAX_SIZE` | `50G` | Maximum cache size |
| `RCLONE_ALLOW_OTHER` | `true` | Allow other users to access mount |
| `RCLONE_ATTR_TIMEOUT` | `1s` | Attribute cache timeout |
| `RCLONE_DIR_CACHE_TIME` | `1m` | Directory cache timeout |
| `RCLONE_VFS_READ_CHUNK_SIZE` | `128k` | Read chunk size |
| `RCLONE_VFS_READ_AHEAD` | `256k` | Read-ahead buffer size |

See [rclone VFS documentation](https://rclone.org/commands/rclone_mount/#vfs-virtual-file-system) for more options.

### Caching

To avoid hitting GitHub API rate limits, `lazy-dvc` caches SSH public keys locally:

| Variable | Default | Description |
|----------|---------|-------------|
| `LDVC_CACHE_TTL` | `5m` | Cache duration (golang format: `5m`, `1h`, etc.) |
| `LDVC_CACHE_DISABLED` | `false` | Set to `true` to disable caching |

Cache location: `/var/cache/lazy-dvc/keys.json`

The cache uses a file-based lock mechanism to handle concurrent SSH connections safely. If a process crashes while holding the lock, the lock expires after 3 seconds, allowing other processes to take over.

### Logging

All container logs are written to stdout with process prefixes for easy filtering:

| Prefix | Process |
|--------|---------|
| `[lazypubk]` | Key fetching |
| `[authpubk]` | SSH auth wrapper |
| `[rclone]` | S3 mount operations |
| `[sshd]` | SSH connections |
| `[entrypoint]` | Container startup/shutdown |

```bash
# View all logs
docker compose logs -f lazy-dvc

# Filter by process
docker compose logs -f lazy-dvc | grep '\[sshd\]'
```

To write logs to a file instead of stdout:

```yaml
environment:
  - LDVC_LOG_FILE=/var/log/lazy-dvc.log
volumes:
  - ./logs:/var/log
```

### SSH/SFTP Access

| Property | Value |
|----------|-------|
| Host | `localhost` (or server IP) |
| Port | `2222` |
| User | `dvc-storage` |
| Auth | SSH public key (GitHub) |
| Root | `/data` (chrooted) |

---

## DVC Usage

### Basic Workflow

```bash
# Initialize DVC in your project
dvc init

# Add data
dvc add data/dataset.csv

# Push to remote
dvc push

# Pull from remote
dvc pull

# Check status
dvc status
```

### Configure Remote

```bash
# Add remote (one-time setup)
dvc remote add -d storage ssh://dvc-storage@your-server:2222/data

# Optional: tune performance
dvc remote modify storage max_sessions 5

# Verify
dvc remote list
```

---

## Troubleshooting

### "Permission denied (publickey)"

1. Check your SSH key is on GitHub: https://github.com/settings/keys
2. Verify org membership: https://github.com/orgs/\<org\>/people
3. Test manually: `ssh -p 2222 dvc-storage@localhost`

### "Connection closed by remote host"

- SFTP should work, SSH shell is intentionally restricted
- Test SFTP: `sftp -P 2222 dvc-storage@localhost`

### "No such file or directory"

- Use `/data` path (chrooted), not full path
- Correct: `ssh://dvc-storage@host:2222/data`
- Wrong: `ssh://dvc-storage@host:2222/home/dvc-storage/data`

### Debug Mode

```bash
# Check server logs
docker compose logs -f lazy-dvc

# Test auth manually
docker compose exec lazy-dvc /usr/local/bin/authpubk dvc-storage
```

---

## Production Deployment

### Single Server

```yaml
# docker-compose.prod.yml
services:
  lazy-dvc:
    ports:
      - "2222:22"
    environment:
      - LDVC_GH_ORG_NAME=your-org
      - LDVC_GH_TEAM_NAME=your-team
    secrets:
      - gh_token
    volumes:
      - s3-data:/data
```

### Production Tips

1. **Configure your S3 backend** — Set yourS3 endpoint and credentials:

   ```yaml
   environment:
     - RCLONE_S3_ENDPOINT=https://s3.amazonaws.com
     - AWS_ACCESS_KEY_ID=xxx
     - AWS_SECRET_ACCESS_KEY=xxx
   ```

2. **SSH Host Keys** — Accept the fingerprint on first connection:

   ```bash
   # First connection will show the fingerprint
   ssh -p 2222 dvc-storage@localhost
   # Or add to known_hosts manually:
   ssh-keyscan -p 2222 your-server >> ~/.ssh/known_hosts
   ```

3. **Monitor usage**:

   ```bash
   docker compose logs -f --tail=100
   ```

---

## Security

- Public key auth only (no passwords)
- Keys fetched dynamically from GitHub
- User chrooted to data directory
- Interactive shell disabled
- TCP forwarding disabled
- No data stored on server (S3 backend)

---

## CI/CD

- `ci` workflow: `gofmt`, `go vet`, `go test`, `go build`
- `docker` workflow: Build on PR, publish on push to main and tags

Published image: `ghcr.io/guilycst/lazy-dvc`

---

## License

MIT
