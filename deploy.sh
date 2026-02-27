#!/bin/bash

# config
REPO_URL="https://github.com/while-maybe/blogengine.git"
TARGET_DIR="$HOME/docker/blogengine"
TEMP_DIR="$HOME/docker/tmp/blogengine_deploy"
# ---------------------

set -e

echo "Starting Deployment to $TARGET_DIR"

# clean and create Temp Directory
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# clone the latest code
echo "Cloning repository..."
git clone --depth 1 "$REPO_URL" "$TEMP_DIR"

# see if target directory exists
mkdir -p "$TARGET_DIR"

# handle .env file
if [ ! -f "$TARGET_DIR/.env" ]; then
    echo "Creating default .env (Please edit this file at $TARGET_DIR/.env)"
    cp ".env" "$TARGET_DIR/.env"
else
    echo "Existing .env found, skipping overwrite."
fi

# ensure target directories are owned by current user
sudo chown -R "$USER:$USER" "$TARGET_DIR"

# sync configuration and content
echo "Syncing files..."

# Syncing while preserving ownership ($USER)
rsync -rtv --delete "$TEMP_DIR/observability/" "$TARGET_DIR/observability/"
rsync -rtv "$TEMP_DIR/sources/" "$TARGET_DIR/sources/"
rsync -rtv --delete "$TEMP_DIR/infra/" "$TARGET_DIR/infra/"
cp "$TEMP_DIR/docker-compose.production.yml" "$TARGET_DIR/docker-compose.yml"

# set permissions
echo "Setting permissions..."
# own everything
chown -R "$USER:$USER" "$TARGET_DIR"

# Ensure directories are searchable/readable (755) and files are readable (644) by the Docker containers
find "$TARGET_DIR" -type d -exec chmod 755 {} +
find "$TARGET_DIR" -type f -exec chmod 644 {} +

echo "Starting Docker containers..."
cd "$TARGET_DIR"

docker compose down --remove-orphans
docker compose up -d --pull always --force-recreate

echo "Deployment Complete!"
