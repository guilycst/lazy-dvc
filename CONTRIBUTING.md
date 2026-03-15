# Contributing to lazy-dvc

Thank you for your interest in contributing to lazy-dvc!

## Development

### Prerequisites

- Go 1.25+
- Docker& Docker Compose
- GitHub PAT with `read:org` scope (for testing)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/guilycst/lazy-dvc.git
cd lazy-dvc

# Install dependencies
go mod download

# Build
go build ./...

# Run tests
go test ./...
```

### Running Locally

```bash
# Set environment variables
export LDVC_GH_TOKEN=your_token
export LDVC_GH_ORG_NAME=your_org

# Build and run
docker compose up -d --build

# Test SSH connection
ssh -p 2222 dvc-storage@localhost
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linters:
   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   go build ./...
   ```
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to your fork (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Coding Standards

- Run `gofmt` on all code
- Run `go vet` to catch common errors
- Add tests for new functionality
- Update documentation for API changes

## Project Structure

```
cmd/
├── lazy-dvc-auth/    # SSH AuthorizedKeysCommand wrapper
├── lazypubk/         # Public key fetcher CLI
└── restricted-shell/ # Restricted shell for SSH users

internal/
└── pubkeyprovider/   # GitHub key provider implementation

pkg/
└── config/          # Configuration handling
```

## Questions?

Open an issue with the "question" label or start a discussion.