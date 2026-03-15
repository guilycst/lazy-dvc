# Environment Variables

## Config

- `LDVC_CACHE_DIR` (fallback: `CACHE_DIR`) — local cache directory. Default: `./cache`.
- `LDVC_CACHE_DURATION` (fallback: `CACHE_DURATION`) — cache TTL as Go duration (e.g. `30s`, `5m`). Default: `1s`.

## GitHub Provider

- `LDVC_GH_ORG_NAME` (fallback: `GH_ORG_NAME`) — organization name.
- `LDVC_GH_TEAM_NAME` (fallback: `GH_TEAM_NAME`) — optional team slug.
- `LDVC_GH_MIN_USER_ROLE` (fallback: `GH_MIN_USER_ROLE`) — minimum role (`member` or `admin`). Default: `member`.
- `LDVC_GH_TOKEN` (fallback: `GH_TOKEN`) — GitHub token.
- `LDVC_GH_TOKEN_FILE` (fallback: `GH_TOKEN_FILE`) — path to a file containing the GitHub token.

Notes:

- Either token variable (`*_TOKEN`) or token file variable (`*_TOKEN_FILE`) must be set.
- If both prefixed and fallback variables are set, prefixed variables win.

