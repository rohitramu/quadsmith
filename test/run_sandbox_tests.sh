#!/usr/bin/env bash
set -e

# Always run from the repo root
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export PORT="${PORT:-8080}"
export DB_PORT="${DB_PORT:-5432}"
export OPS_PORT="${OPS_PORT:-8081}"
export API_PORT="${API_PORT:-8082}"

API_URL="${QS_CLI_API_URL:-${QS_API_URL:-http://127.0.0.1:${PORT}}}"
OPS_URL="${QS_OPS_SERVER_URL:-${QS_OPS_URL:-http://127.0.0.1:${OPS_PORT}}}"
WEB_URL="${QS_WEB_URL:-http://127.0.0.1:${PORT}}"
DB_URL="${QS_DATABASE_URL:-${DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_api}}"
INTERNAL_DB_URL="${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_ops}"

is_sandbox_ready() {
    curl -s -f -m 1 "${API_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${OPS_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "http://127.0.0.1:${API_PORT}/healthz" >/dev/null 2>&1
}

cleanup() {
    echo "--- Stopping test sandbox containers ---"
    docker compose -f test/sandbox/docker-compose.yaml down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "--- Stopping any existing sandbox instance ---"
cleanup

echo "--- Building Sandbox Container Images ---"
docker compose -f test/sandbox/docker-compose.yaml build

echo "--- Starting fresh Sandbox containers on port ${PORT} (Ops: ${OPS_PORT}, Direct API: ${API_PORT}, DB: ${DB_PORT}) ---"
docker compose -f test/sandbox/docker-compose.yaml up -d

echo "--- Waiting for Sandbox to become healthy ---"
MAX_RETRIES=40
COUNT=0
until is_sandbox_ready; do
    sleep 0.2
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -ge "$MAX_RETRIES" ]; then
        echo "Error: Sandbox failed to become ready after $((MAX_RETRIES / 5)) seconds."
        echo "=== Container logs ==="
        docker compose -f test/sandbox/docker-compose.yaml logs | tail -n 50
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
export GOWORK="${ROOT_DIR}/src/go/go.work"

go test -count=1 -v ./test/...
