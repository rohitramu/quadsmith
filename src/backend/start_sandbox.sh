#!/usr/bin/env bash
set -e

# Always run from the root of the repo
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

# Allow passing an optional port, default to 8080
PORT="${1:-8080}"

echo "--- Stopping any existing sandbox instance ---"
docker rm -f quadsmith-sandbox-run >/dev/null 2>&1 || true

echo "--- Building Sandbox Docker Image ---"
DOCKER_BUILDKIT=0 docker build -t quadsmith-sandbox -f src/backend/Dockerfile .

echo "--- Starting Quadsmith Sandbox (API on port $PORT, DB on 5432) ---"
# Passing the PORT env var down into the container so the Go server listens on it
docker run --rm --name quadsmith-sandbox-run \
  -e PORT="$PORT" \
  -p "$PORT":"$PORT" \
  -p 5432:5432 \
  quadsmith-sandbox
