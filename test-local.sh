#!/bin/bash
set -e

if [ -z "$LDVC_GH_TOKEN" ]; then
    echo "Error: LDVC_GH_TOKEN environment variable not set"
    exit 1
fi

if [ -z "$LDVC_GH_ORG_NAME" ]; then
    echo "Error: LDVC_GH_ORG_NAME environment variable not set"
    exit 1
fi

echo "Building lazy-dvc image..."
docker build -t lazy-dvc .

echo "Stopping and removing existing container if any..."
docker rm -f lazy-dvc-test 2>/dev/null || true

echo "Starting container..."
docker run -d \
    --privileged \
    -p 2222:22 \
    -p 8070:8070 \
    -e LDVC_GH_TOKEN="$LDVC_GH_TOKEN" \
    -e LDVC_GH_ORG_NAME="$LDVC_GH_ORG_NAME" \
    -e LDVC_GH_TEAM_NAME="$LDVC_GH_TEAM_NAME" \
    --name lazy-dvc-test \
    lazy-dvc

echo "Waiting for services to start..."
sleep 5

echo "Container logs:"
docker logs lazy-dvc-test

echo ""
echo "Testing SSH connection..."
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null dvc-storage@localhost echo "SSH connection successful!"

echo ""
echo "Test completed!"
