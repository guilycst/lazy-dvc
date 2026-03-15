#!/bin/sh
# =============================================================================
# lazy-dvc Entrypoint Script
# =============================================================================
#
# This script manages the lifecycle of the lazy-dvc container, which requires
# two long-running processes:
#   1. rclone mount - Provides S3-backed storage mounted at /home/dvc-storage/data
#   2. sshd          - Accepts SSH/SFTP connections from DVC clients
#
# DESIGN DECISIONS:
#
# 1. Process Supervision (no init system)
#    We don't use dumb-init or tini because we need custom logic to:
#    - Log which process died
#    - Gracefully shutdown the other process before exiting
#    - Ensure the container exits with proper exit codes
#
# 2. Process Tree (who supervises whom?)
#    This script (PID 1) starts both processes in background, then waits for
#    either to exit. When one dies, we:
#    - Log which process died
#    - Send SIGTERM to the other for graceful shutdown
#    - Wait for graceful shutdown (with timeout)
#    - Exit with the dead process's exit code
#
#    This ensures Docker knows the container is unhealthy when either process
#    crashes, rather than having a zombie container running half-broken.
#
# 3. Log Prefixing
#    Log prefixes are handled as follows:
#    - [lazypubk] / [lazy-dvc-auth]: Handled internally via slog (Go binaries)
#    - [rclone]: rclone --log-format includes timestamp, we prefix in post
#    - [sshd]: sshd -e logs to stderr, we prefix in post
#
#    Note: We DON'T use pipe-to-sed for background processes because it breaks
#    PID tracking ($! would capture sed, not the actual process).
#
# 4. Graceful Shutdown
#    On SIGTERM, we stop both processes cleanly:
#    - rclone: unmounts gracefully (flushes pending writes)
#    - sshd:   accepts no new connections, waits for existing ones
#
# =============================================================================

set -e

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------

RCLONE_ENDPOINT=${RCLONE_S3_ENDPOINT:-http://localhost:8070}
RCLONE_VFS_CACHE_MODE=${RCLONE_VFS_CACHE_MODE:-full}
RCLONE_VFS_CACHE_MAX_SIZE=${RCLONE_VFS_CACHE_MAX_SIZE:-50G}
RCLONE_ATTR_TIMEOUT=${RCLONE_ATTR_TIMEOUT:-1s}
RCLONE_DIR_CACHE_TIME=${RCLONE_DIR_CACHE_TIME:-1m}
RCLONE_VFS_READ_CHUNK_SIZE=${RCLONE_VFS_READ_CHUNK_SIZE:-128k}
RCLONE_VFS_READ_AHEAD=${RCLONE_VFS_READ_AHEAD:-256k}

# Graceful shutdown timeout (seconds)
SHUTDOWN_TIMEOUT=10

# -----------------------------------------------------------------------------
# Environment Setup
# -----------------------------------------------------------------------------

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [entrypoint] $1"
}

log "Starting lazy-dvc container"

# Write environment for subprocesses to read
mkdir -p /etc/lazy-dvc
cat > /etc/lazy-dvc/env << EOF
LDVC_GH_ORG_NAME=${LDVC_GH_ORG_NAME}
LDVC_GH_TEAM_NAME=${LDVC_GH_TEAM_NAME}
LDVC_GH_TOKEN_FILE=/run/secrets/gh_token
LDVC_CACHE_TTL=${LDVC_CACHE_TTL:-5m}
LDVC_CACHE_DISABLED=${LDVC_CACHE_DISABLED:-false}
EOF

# Create cache directory with proper permissions
mkdir -p /var/cache/lazy-dvc
chown -R dvc-storage:dvc-storage /var/cache/lazy-dvc
chmod 755 /var/cache/lazy-dvc

# -----------------------------------------------------------------------------
# Named Pipes for Log Prefixing
# -----------------------------------------------------------------------------
# Using FIFOs allows us to prefix logs without breaking PID tracking.
# When you pipe (command | sed ...), $! captures sed's PID, not command's.
# With FIFOs, command writes to a pipe, sed reads from it, and $! stays correct.

RCLONE_FIFO="/tmp/rclone_fifo"
SSHD_FIFO="/tmp/sshd_fifo"
LAZYPUBK_FIFO="/tmp/lazypubk_fifo"
AUTH_FIFO="/tmp/lazy-dvc-auth_fifo"

mkfifo "$RCLONE_FIFO" "$SSHD_FIFO" "$LAZYPUBK_FIFO" "$AUTH_FIFO"

# Background log prefixers - these run for the container's lifetime
(sed 's/^/[rclone] /' < "$RCLONE_FIFO" >&2) &
(sed 's/^/[sshd] /' < "$SSHD_FIFO" >&2) &
(sed 's/^/[lazypubk] /' < "$LAZYPUBK_FIFO" >&2) &
(sed 's/^/[lazy-dvc-auth] /' < "$AUTH_FIFO" >&2) &

# Keep FIFOs open even if process crashes (prevents sed from exiting)
exec 3> "$RCLONE_FIFO"
exec 4> "$SSHD_FIFO"
exec 5> "$LAZYPUBK_FIFO"
exec 6> "$AUTH_FIFO"

# Export FIFO paths for child processes
export LDVC_LOG_FIFO="$LAZYPUBK_FIFO"
export LDVC_AUTH_LOG_FIFO="$AUTH_FIFO"

# -----------------------------------------------------------------------------
# Rclone Configuration
# -----------------------------------------------------------------------------

log "Configuring rclone"

mkdir -p /root/.config/rclone
cat > /root/.config/rclone/rclone.conf << EOF
[s3]
type = s3
provider = Other
access_key_id = ${AWS_ACCESS_KEY_ID:-minioadmin}
secret_access_key = ${AWS_SECRET_ACCESS_KEY:-minioadmin}
endpoint = ${RCLONE_ENDPOINT}
region = ${AWS_DEFAULT_REGION:-us-east-1}
EOF

# -----------------------------------------------------------------------------
# Process Management
# -----------------------------------------------------------------------------

RCLONE_PID=""
SSHD_PID=""

cleanup() {
    log "Received termination signal, shutting down..."
    
    if [ -n "$RCLONE_PID" ] && kill -0 "$RCLONE_PID" 2>/dev/null; then
        log "Stopping rclone (PID: $RCLONE_PID)"
        kill -TERM "$RCLONE_PID" 2>/dev/null || true
    fi
    
    if [ -n "$SSHD_PID" ] && kill -0 "$SSHD_PID" 2>/dev/null; then
        log "Stopping sshd (PID: $SSHD_PID)"
        kill -TERM "$SSHD_PID" 2>/dev/null || true
    fi
    
    # Wait for graceful shutdown
    local count=0
    while [ $count -lt $SHUTDOWN_TIMEOUT ]; do
        local rclone_dead=0
        local sshd_dead=0
        
        [ -z "$RCLONE_PID" ] || ! kill -0 "$RCLONE_PID" 2>/dev/null && rclone_dead=1
        [ -z "$SSHD_PID" ] || ! kill -0 "$SSHD_PID" 2>/dev/null && sshd_dead=1
        
        if [ $rclone_dead -eq 1 ] && [ $sshd_dead -eq 1 ]; then
            break
        fi
        
        sleep 1
        count=$((count + 1))
    done
    
    # Force kill if still running
    [ -n "$RCLONE_PID" ] && kill -9 "$RCLONE_PID" 2>/dev/null || true
    [ -n "$SSHD_PID" ] && kill -9 "$SSHD_PID" 2>/dev/null || true
    
    log "Shutdown complete"
}

trap cleanup EXIT TERM INT

# -----------------------------------------------------------------------------
# Start Rclone
# -----------------------------------------------------------------------------

log "Starting rclone mount"

RCLONE_ALLOW_OTHER_FLAG=""
if [ "${RCLONE_ALLOW_OTHER:-true}" = "true" ]; then
    RCLONE_ALLOW_OTHER_FLAG="--allow-other"
fi

# Start rclone - output goes to FIFO for prefixing
# This keeps rclone as the direct child, so $! captures rclone's PID
rclone mount \
    --vfs-cache-mode "$RCLONE_VFS_CACHE_MODE" \
    --vfs-cache-max-size "$RCLONE_VFS_CACHE_MAX_SIZE" \
    $RCLONE_ALLOW_OTHER_FLAG \
    --attr-timeout "$RCLONE_ATTR_TIMEOUT" \
    --dir-cache-time "$RCLONE_DIR_CACHE_TIME" \
    --vfs-read-chunk-size "$RCLONE_VFS_READ_CHUNK_SIZE" \
    --vfs-read-ahead "$RCLONE_VFS_READ_AHEAD" \
    --log-level INFO \
    s3: /home/dvc-storage/data >"$RCLONE_FIFO" 2>&1 &

RCLONE_PID=$!
log "rclone started (PID: $RCLONE_PID)"

# Wait for rclone to mount
sleep 2
if ! kill -0 "$RCLONE_PID" 2>/dev/null; then
    log "ERROR: rclone failed to start"
    exit 1
fi

if ! mountpoint -q /home/dvc-storage/data 2>/dev/null; then
    log "WARNING: rclone mount not ready yet, continuing anyway"
fi

# -----------------------------------------------------------------------------
# Start SSHD
# -----------------------------------------------------------------------------

log "Starting sshd"

# Start sshd in foreground (-D) with stderr logging (-e)
# Output goes to FIFO for prefixing, keeping sshd as direct child
/usr/sbin/sshd -D -e >"$SSHD_FIFO" 2>&1 &

SSHD_PID=$!
log "sshd started (PID: $SSHD_PID)"

# Wait for sshd to start
sleep 1
if ! kill -0 "$SSHD_PID" 2>/dev/null; then
    log "ERROR: sshd failed to start"
    exit 1
fi

# -----------------------------------------------------------------------------
# Supervisor Loop
# -----------------------------------------------------------------------------

log "Container startup complete, entering supervisor loop"
log "Logs from lazypubk and lazy-dvc-auth will be prefixed with their names"

# Wait for either process to exit (polling approach for POSIX sh compatibility)
while kill -0 "$RCLONE_PID" 2>/dev/null && kill -0 "$SSHD_PID" 2>/dev/null; do
    sleep 1
done

# Check which process died
if ! kill -0 "$RCLONE_PID" 2>/dev/null; then
    log "ERROR: rclone exited unexpectedly"
    EXIT_CODE=1
elif ! kill -0 "$SSHD_PID" 2>/dev/null; then
    log "ERROR: sshd exited unexpectedly"
    EXIT_CODE=1
fi

# Cleanup will run via trap
exit ${EXIT_CODE:-1}