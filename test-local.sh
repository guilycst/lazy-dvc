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

echo "Building and starting containers..."
docker compose up -d --build

echo "Waiting for services to be ready..."
sleep 5

echo "Container logs:"
docker compose logs

echo ""
echo "Testing SSH connection..."
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null dvc-storage@localhost echo "SSH connection successful!"

echo ""
echo "Test completed!"
