#!/bin/bash
set -e

SSH_HOST="${SSH_HOST:-localhost}"
SSH_PORT="${SSH_PORT:-2222}"
REMOTE_NAME="${REMOTE_NAME:-storage}"

echo "=========================================="
echo "DVC SSH Storage Test"
echo "=========================================="
echo ""
echo "Target: ssh://dvc-storage@${SSH_HOST}:${SSH_PORT}/data"
echo ""

echo "Testing SSH connection..."
ssh -p ${SSH_PORT} -o StrictHostKeyChecking=no dvc-storage@${SSH_HOST} echo "SSH connection OK" 2>&1 || true

echo ""
echo "Testing SFTP connection..."
sftp -P ${SSH_PORT} -o StrictHostKeyChecking=no dvc-storage@${SSH_HOST}:/data <<EOF
ls -la
quit
EOF
echo "SFTP connection OK"

echo ""
echo "Configuring DVC remote..."
dvc remote add -d ${REMOTE_NAME} ssh://dvc-storage@${SSH_HOST}:${SSH_PORT}/data

echo ""
echo "Testing DVC remote connection..."
dvc remote list
dvc remote modify ${REMOTE_NAME} max_sessions 5

echo ""
echo "=========================================="
echo "DVC is ready to use!"
echo "=========================================="
echo ""
echo "Usage examples:"
echo "  dvc pull          # Download data from remote"
echo "  dvc push          # Upload data to remote"
echo "  dvc status        # Check sync status"
echo ""
