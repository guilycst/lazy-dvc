# DVC SSH Storage Setup

A Dockerized SSH/SFTP storage backend for DVC, authenticated via GitHub organization keys.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                 lazy-dvc Docker Setup               │
│                                                     │
│  ┌──────────────┐    ┌─────────────────────────┐   │
│  │   Client     │───►│  SSH/SFTP (:22/2222)    │   │
│  │   (DVC)      │    │  ├── lazy-dvc-auth      │   │
│  │              │    │  │   (GitHub key fetch)  │   │
│  │              │    │  └── restricted-shell   │   │
│  │              │    │                         │   │
│  │              │    │  /home/dvc-storage/data │   │
│  │              │    │         │               │   │
│  │              │    │         ▼               │   │
│  │              │    │    rclone mount         │   │
│  │              │    │         │               │   │
│  └──────────────┘    └─────────┼───────────────┘   │
│                               │                    │
│                       ┌───────▼────────┐            │
│                       │   versitygw    │            │
│                       │   (:8070 S3)   │            │
│                       └────────────────┘            │
└─────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Start the Server

```bash
# Build and start
docker compose up -d --build

# Or use the test script
./test-local.sh
```

### 2. Configure DVC

```bash
# Add SSH remote (note: /data because of chroot)
dvc remote add -d storage ssh://dvc-storage@localhost:2222/data

# Test connection
dvc pull
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LDVC_GH_ORG_NAME` | GitHub organization to fetch keys from | (required) |
| `LDVC_GH_TEAM_NAME` | GitHub team to filter keys (optional) | - |
| `SSH_PORT` | SSH port | `2222` |
| `RCLONE_S3_ENDPOINT` | S3 backend URL | `http://versitygw:8070` |

### Docker Secrets

| Secret | Description |
|--------|-------------|
| `gh_token` | GitHub PAT (Personal Access Token) for API access |

### SSH/SFTP Configuration

The server is pre-configured with:

- **SFTP**: Enabled via `internal-sftp` subsystem
- **Public Key Auth**: Keys fetched from GitHub org
- **Restricted Shell**: Interactive SSH disabled, SFTP allowed
- **Chroot**: User confined to `/home/dvc-storage` (appears as `/` in SFTP)

## Authentication

Users authenticate using their **GitHub public keys**:

1. User's public key must be in a GitHub organization member's account
2. The organization name is configured via `LDVC_GH_ORG_NAME`
3. Optionally filter by team with `LDVC_GH_TEAM_NAME`

### Adding Your Key

1. Go to [GitHub SSH Keys](https://github.com/settings/keys)
2. Add your public key
3. Ensure your GitHub account is a member of the configured organization

## DVC Usage Examples

### Basic Operations

```bash
# Pull data from remote
dvc pull

# Push data to remote
dvc push

# Check status
dvc status

# Sync all
dvc sync
```

### Advanced Configuration

```bash
# Set custom port
dvc remote modify storage port 2222

# Limit concurrent sessions (helps with server load)
dvc remote modify storage max_sessions 5

# Check remote config
dvc remote list
dvc remote storage -v
```

## Troubleshooting

### SSH Connection Fails

```bash
# Test SSH directly
ssh -p 2222 dvc-storage@localhost
```

### SFTP Not Working

```bash
# Test SFTP
sftp -P 2222 dvc-storage@localhost
sftp> ls
```

### Keys Not Accepted

1. Check your key is on GitHub: https://github.com/settings/keys
2. Verify org membership: https://github.com/orgs/<org>/people
3. Check server logs: `docker compose logs lazy-dvc`

### Performance Issues

```bash
# Increase SFTP sessions (default: 10)
dvc remote modify storage max_sessions 10
```

## Production Deployment

### Docker Compose Example

```yaml
services:
  lazy-dvc:
    ports:
      - "2222:22"
    environment:
      - LDVC_GH_ORG_NAME=your-org
      - LDVC_GH_TEAM_NAME=your-team
    secrets:
      - gh_token
```

Then configure DVC:
```bash
dvc remote add -d storage ssh://dvc-storage@<host>:2222/data
```

### Security Considerations

- Keys are fetched dynamically from GitHub on each login
- User is chrooted to their home directory
- Interactive shell is disabled
- TCP forwarding is disabled
- All authentication is public-key based (no passwords)

## Development

### Building

```bash
docker build -t lazy-dvc .
```

### Testing

```bash
# Full test with DVC
./test-dvc.sh

# Just SSH test
./test-local.sh
```
