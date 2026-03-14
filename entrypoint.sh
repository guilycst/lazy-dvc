#!/bin/bash
set -e

export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_DEFAULT_REGION=us-east-1

versitygw create bucket --bucket dvc-storage-bucket 2>/dev/null || true

versitygw serve :8070 &
GW_PID=$!

sleep 2

rclone mount \
    --vfs-cache-mode full \
    --vfs-cache-max-size 50G \
    --allow-other \
    --attr-timeout 1s \
    --dir-cache-time 1m \
    --vfs-read-chunk-size 128k \
    --vfs-read-ahead 256k \
    --log-level DEBUG \
    --syslog \
    --verbose \
    s3:dvc-storage-bucket /home/dvc-storage/data &

RCLONE_PID=$!

sleep 2

/usr/sbin/sshd

wait
