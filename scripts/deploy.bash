#!/usr/bin/env bash
echo "=== Running deployment script ==="
CODE_ROOT_DIR="$(pwd)/.."

cd $CODE_ROOT_DIR || error_with_message "Failed to change directory to $CODE_ROOT_DIR"
echo "Building garden center website"
cd www/sramek-garden-center
npm run buildstatic

cd $CODE_ROOT_DIR || error_with_message "Failed to change directory to $CODE_ROOT_DIR"
echo "Building transportation website"
cd www/sramek-transportation
npm run buildstatic

cd $CODE_ROOT_DIR || error_with_message "Failed to change directory to $CODE_ROOT_DIR"
echo "Running pulumi up"
cd infra
pulumi up