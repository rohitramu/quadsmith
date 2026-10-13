#!/usr/bin/env bash
set -e

# Always run from the repo root
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-8080}"
DB_PORT="${DB_PORT:-5432}"
OPS_PORT="${OPS_PORT:-8081}"
WEB_PORT="${WEB_PORT:-3001}"

API_URL="${QS_CLI_API_URL:-${QS_API_URL:-http://127.0.0.1:${PORT}}}"
OPS_URL="${QS_OPS_SERVER_URL:-${QS_OPS_URL:-http://127.0.0.1:${OPS_PORT}}}"
WEB_URL="${QS_WEB_URL:-http://127.0.0.1:${WEB_PORT}}"
DB_URL="${QS_DATABASE_URL:-${DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_api}}"
INTERNAL_DB_URL="${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_ops}"

is_sandbox_ready() {
    curl -s -f -m 1 "${API_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${OPS_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${WEB_URL}/healthz" >/dev/null 2>&1
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

echo "--- Starting fresh Sandbox container on port ${PORT} (Ops: ${OPS_PORT}, Web: ${WEB_PORT}, DB: ${DB_PORT}) ---"
docker run -d --name "$CONTAINER_NAME" \
    -e QS_SERVER_PORT="$PORT" \
    -e QS_OPS_SERVER_PORT="$OPS_PORT" \
    -e QS_INTERNAL_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops" \
    -p "${PORT}":"${PORT}" \
    -p "${OPS_PORT}":"${OPS_PORT}" \
    -p "${WEB_PORT}":3000 \
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
        echo "=== Container logs ==="
        docker logs "$CONTAINER_NAME" | tail -n 50
        echo "=== Server Ops logs ==="
        docker exec "$CONTAINER_NAME" cat /var/log/server-ops.log 2>/dev/null || true
        echo "=== Server Web logs ==="
        docker exec "$CONTAINER_NAME" cat /var/log/server-web.log 2>/dev/null || true
        exit 1
    fi
done
echo "--- Sandbox is ready! ---"

echo "--- Running Sandbox Tests ---"
export QS_CLI_API_URL="${API_URL}"
export QS_API_URL="${API_URL}"
export QS_OPS_SERVER_URL="${OPS_URL}"
export QS_OPS_URL="${OPS_URL}"
export QS_WEB_URL="${WEB_URL}"
export QS_OPS_TOKEN="qs_op_sandbox_token_12345"
export QS_DATABASE_URL="${DB_URL}"
export DATABASE_URL="${DB_URL}"
export QS_INTERNAL_DATABASE_URL="${INTERNAL_DB_URL}"
export GOWORK="${ROOT_DIR}/src/go.work"

go test -count=1 -v ./test/...
