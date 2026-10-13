#!/usr/bin/env bash
set -e

# Always run from the root of the repo
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Allow passing an optional port, default to 8080
PORT="${1:-8080}"
DB_PORT="${DB_PORT:-5432}"
OPS_PORT="${OPS_PORT:-8081}"
WEB_PORT="${WEB_PORT:-3001}"

echo "--- Stopping any existing sandbox instance ---"
docker rm -f quadsmith-sandbox-run test-pg >/dev/null 2>&1 || true

echo "--- Building Sandbox Docker Image ---"
docker build -t quadsmith-sandbox -f test/Dockerfile .

echo "--- Starting Quadsmith Sandbox (API: $PORT, Ops: $OPS_PORT, Web: $WEB_PORT, DB: $DB_PORT) ---"
# Passing the PORT env var down into the container so the Go server listens on it
docker run --rm --name quadsmith-sandbox-run \
  -e QS_SERVER_PORT="$PORT" \
  -e QS_OPS_SERVER_PORT="$OPS_PORT" \
  -p "$PORT":"$PORT" \
  -p "$OPS_PORT":"$OPS_PORT" \
  -p "$WEB_PORT":3000 \
  -p "$DB_PORT":5432 \
  quadsmith-sandbox
