#!/bin/bash
set -e

echo "🚀 Starting Deployment..."

sudo chown -R $USER:$USER .

git fetch origin main
git reset --hard origin/main
git clean -fd

docker compose -f docker-compose.production.yml up -d --build --remove-orphans

docker exec -u root blogengine chown -R appuser:appgroup /app/data

echo "✅ Deployment Successful!"
