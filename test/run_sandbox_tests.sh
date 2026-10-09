#!/usr/bin/env bash
set -e

# Always run from the repo root
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-8080}"
DB_PORT="${DB_PORT:-5432}"

API_URL="${QS_API_URL:-http://127.0.0.1:${PORT}}"
DB_URL="${DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith}"

is_sandbox_ready() {
    curl -s -f -m 1 "${API_URL}" >/dev/null 2>&1
}

CONTAINER_NAME="quadsmith-sandbox-test-run"

cleanup() {
    echo "--- Stopping test sandbox container ---"
    docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "--- Stopping any existing sandbox instance ---"
docker rm -f "$CONTAINER_NAME" quadsmith-sandbox-run test-pg >/dev/null 2>&1 || true

RUNNING_SANDBOX_CONTAINERS=$(docker ps -q --filter ancestor=quadsmith-sandbox 2>/dev/null || true)
if [ -n "$RUNNING_SANDBOX_CONTAINERS" ]; then
    docker rm -f $RUNNING_SANDBOX_CONTAINERS >/dev/null 2>&1 || true
fi

echo "--- Building Sandbox Docker Image ---"
docker build -t quadsmith-sandbox -f test/Dockerfile .

echo "--- Starting fresh Sandbox container on port ${PORT} (DB: ${DB_PORT}) ---"
docker run -d --name "$CONTAINER_NAME" \
    -e PORT="$PORT" \
    -p "${PORT}":"${PORT}" \
    -p "${DB_PORT}":5432 \
    quadsmith-sandbox

echo "--- Waiting for Sandbox to become healthy ---"
MAX_RETRIES=40
COUNT=0
until is_sandbox_ready; do
    sleep 1
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -ge "$MAX_RETRIES" ]; then
        echo "Error: Sandbox failed to become ready after ${MAX_RETRIES} seconds."
        docker logs "$CONTAINER_NAME" | tail -n 50
        exit 1
    fi
done
echo "--- Sandbox is ready! ---"

echo "--- Running Sandbox Tests ---"
export QS_API_URL="${API_URL}"
export DATABASE_URL="${DB_URL}"
export GOWORK="${ROOT_DIR}/src/go.work"

go test -count=1 -v ./test/...
