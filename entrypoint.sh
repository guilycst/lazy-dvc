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

rclone mount \
    --vfs-cache-mode full \
    --vfs-cache-max-size 50G \
    --allow-other \
    --attr-timeout 1s \
    --dir-cache-time 1m \
    --vfs-read-chunk-size 128k \
    --vfs-read-ahead 256k \
    s3: /home/dvc-storage/data &

RCLONE_PID=$!

sleep 2

/usr/sbin/sshd -D

wait
