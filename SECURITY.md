# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x     | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via:

1. GitHub Security Advisories (preferred): Use the ["Report a vulnerability" button](https://github.com/guilycst/lazy-dvc/security/advisories/new)
2. Email: Send details to the maintainers directly

### What to Include

Please provide:

- A description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Any suggested mitigations

### Response Timeline

- We will acknowledge receipt within 48 hours
- We will provide an initial assessment within 7 days
- We will work on a fix and coordinate disclosure

## Security Considerations

### Authentication

- lazy-dvc uses SSH public key authentication only (no passwords)
- Keys are fetched dynamically from GitHub organization membership
- No keys are stored on the server

### Network Security

- All communication is over SSH/SFTP (encrypted)
- Server runs on configurable port (default 2222)
- Consider running behind a firewall or VPN in production

### Data Security

- Data is stored on your own S3-compatible storage
- No data is stored on the lazy-dvc server itself
- Ensure your S3 backend has appropriate access controls

### Known Limitations

- GitHub API rate limits may affect authentication speed
- Organization membership changes may take time to propagate
- SSH key rotation requires GitHub sync