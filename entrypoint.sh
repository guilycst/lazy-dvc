#!/bin/bash
set -e

export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-minioadmin}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-minioadmin}
export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-us-east-1}

mkdir -p /etc/lazy-dvc
echo "LDVC_GH_ORG_NAME=$LDVC_GH_ORG_NAME" > /etc/lazy-dvc/env
echo "LDVC_GH_TEAM_NAME=$LDVC_GH_TEAM_NAME" >> /etc/lazy-dvc/env
echo "LDVC_GH_TOKEN_FILE=/run/secrets/gh_token" >> /etc/lazy-dvc/env

RCLONE_ENDPOINT=${RCLONE_S3_ENDPOINT:-http://localhost:8070}

mkdir -p /root/.config/rclone

cat > /root/.config/rclone/rclone.conf << EOF
[s3]
type = s3
provider = Other
access_key_id = $AWS_ACCESS_KEY_ID
secret_access_key = $AWS_SECRET_ACCESS_KEY
endpoint = $RCLONE_ENDPOINT
region = $AWS_DEFAULT_REGION
EOF

RCLONE_VFS_CACHE_MODE=${RCLONE_VFS_CACHE_MODE:-full}
RCLONE_VFS_CACHE_MAX_SIZE=${RCLONE_VFS_CACHE_MAX_SIZE:-50G}
RCLONE_ATTR_TIMEOUT=${RCLONE_ATTR_TIMEOUT:-1s}
RCLONE_DIR_CACHE_TIME=${RCLONE_DIR_CACHE_TIME:-1m}
RCLONE_VFS_READ_CHUNK_SIZE=${RCLONE_VFS_READ_CHUNK_SIZE:-128k}
RCLONE_VFS_READ_AHEAD=${RCLONE_VFS_READ_AHEAD:-256k}

RCLONE_ALLOW_OTHER_FLAG=""
if [ "${RCLONE_ALLOW_OTHER:-true}" = "true" ]; then
    RCLONE_ALLOW_OTHER_FLAG="--allow-other"
fi

rclone mount \
    --vfs-cache-mode "$RCLONE_VFS_CACHE_MODE" \
    --vfs-cache-max-size "$RCLONE_VFS_CACHE_MAX_SIZE" \
    $RCLONE_ALLOW_OTHER_FLAG \
    --attr-timeout "$RCLONE_ATTR_TIMEOUT" \
    --dir-cache-time "$RCLONE_DIR_CACHE_TIME" \
    --vfs-read-chunk-size "$RCLONE_VFS_READ_CHUNK_SIZE" \
    --vfs-read-ahead "$RCLONE_VFS_READ_AHEAD" \
    s3: /home/dvc-storage/data &

RCLONE_PID=$!

sleep 2

/usr/sbin/sshd -D

wait
