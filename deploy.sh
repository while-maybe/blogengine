#!/bin/bash

# config
REPO_URL="https://github.com/while-maybe/blogengine.git"
TARGET_DIR="$HOME/docker/blogengine"
TEMP_DIR="$HOME/docker/tmp/blogengine_deploy"
# ---------------------

set -e

echo "🚀 Starting Deployment to $TARGET_DIR"

# clean and create Temp Directory
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# clone the latest code
echo "📥 Cloning repository..."
git clone --depth 1 "$REPO_URL" "$TEMP_DIR"

# see if target directory exists
mkdir -p "$TARGET_DIR"

# handle .env file
if [ ! -f "$TARGET_DIR/.env" ]; then
    echo "📄 Creating default .env (Please edit this file at $TARGET_DIR/.env)"
    cp ".env" "$TARGET_DIR/.env"
else
    echo "✅ Existing .env found, skipping overwrite."
fi

# 5. Sync Configuration and Content
echo "🔄 Syncing files..."
# Syncing while preserving your ownership ($USER)
rsync -rtv --delete "$TEMP_DIR/observability/" "$TARGET_DIR/observability/"
rsync -rtv "$TEMP_DIR/sources/" "$TARGET_DIR/sources/"
cp "$TEMP_DIR/docker-compose.production.yml" "$TARGET_DIR/docker-compose.yml"

# 6. Set Permissions (The "Right" Way)
echo "🔐 Setting permissions..."
# Ensure YOU own everything
chown -R "$USER:$USER" "$TARGET_DIR"

# Ensure directories are searchable/readable (755) 
# and files are readable (644) by the Docker containers
find "$TARGET_DIR" -type d -exec chmod 755 {} +
find "$TARGET_DIR" -type f -exec chmod 644 {} +

# 7. Cleanup Temp Folder
rm -rf "$TEMP_DIR"

# 8. Run Docker Compose
echo "🐳 Starting Docker containers..."
cd "$TARGET_DIR"

docker compose down --remove-orphans
docker volume rm blogengine_lgtm_data

# Pull latest images and recreate containers
docker compose up -d --pull always --force-recreate

echo "✨ Deployment Complete!"