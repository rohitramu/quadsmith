#!/usr/bin/env bash
set -e

# Always run from the root of the repo
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Allow passing an optional port, default to 8080
export PORT="${1:-8080}"
export DB_PORT="${DB_PORT:-5432}"
export OPS_PORT="${OPS_PORT:-8081}"
export WEB_PORT="${WEB_PORT:-3001}"

echo "--- Stopping any existing sandbox instance ---"
docker compose -f test/sandbox/docker-compose.yaml down -v >/dev/null 2>&1 || true

echo "--- Building Sandbox Container Images ---"
docker compose -f test/sandbox/docker-compose.yaml build

echo "--- Starting Quadsmith Sandbox (API: $PORT, Ops: $OPS_PORT, Web: $WEB_PORT, DB: $DB_PORT) ---"
docker compose -f test/sandbox/docker-compose.yaml up
