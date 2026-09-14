#!/usr/bin/env bash
# Keeps internal/repo/migrations in sync with the repo-root migrations/.
# Run after adding a migration: `./scripts/sync-migrations.sh`
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p internal/repo/migrations
cp migrations/*.up.sql migrations/*.down.sql internal/repo/migrations/
echo "synced $(ls migrations/*.up.sql | wc -l) migrations into internal/repo/migrations"
