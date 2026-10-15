#!/usr/bin/env bash
set -e

# Always run from the root of the repo
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Allow passing an optional port, default to 8080
export PORT="${1:-8080}"
export DB_PORT="${DB_PORT:-5432}"
export OPS_PORT="${OPS_PORT:-8081}"
export API_PORT="${API_PORT:-8082}"

echo "--- Stopping any existing sandbox instance ---"
docker compose -f test/sandbox/docker-compose.yaml down -v >/dev/null 2>&1 || true

echo "--- Building Sandbox Container Images ---"
docker compose -f test/sandbox/docker-compose.yaml build

echo "--- Starting Quadsmith Sandbox (Web on $PORT, Ops on $OPS_PORT, API on $API_PORT, DB on $DB_PORT) ---"
docker compose -f test/sandbox/docker-compose.yaml up
