#!/usr/bin/env bash
set -e

# Always run from the repo root
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export PORT="${PORT:-8080}"
export DB_PORT="${DB_PORT:-5432}"
export OPS_PORT="${OPS_PORT:-8082}"
export OPS_WEB_PORT="${OPS_WEB_PORT:-8083}"
export CRON_PORT="${CRON_PORT:-8085}"
export API_PORT="${API_PORT:-8081}"
export NGINX_PORT="${NGINX_PORT:-8084}"

API_URL="${QS_CLI_API_URL:-${QS_API_URL:-http://127.0.0.1:${API_PORT}}}"
OPS_URL="${QS_OPS_SERVER_URL:-${QS_OPS_URL:-http://127.0.0.1:${OPS_PORT}}}"
OPS_WEB_URL="${QS_OPS_WEB_URL:-http://127.0.0.1:${OPS_WEB_PORT}}"
CRON_URL="${QS_CRON_URL:-http://127.0.0.1:${CRON_PORT}}"
WEB_URL="${QS_WEB_URL:-http://127.0.0.1:${PORT}}"
NGINX_URL="${QS_NGINX_URL:-http://127.0.0.1:${NGINX_PORT}}"
DB_URL="${QS_DATABASE_URL:-${DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_api}}"
INTERNAL_DB_URL="${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:${DB_PORT}/quadsmith_ops}"

is_sandbox_ready() {
    curl -s -f -m 1 "${API_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${OPS_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${OPS_WEB_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${CRON_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${WEB_URL}/healthz" >/dev/null 2>&1 && \
    curl -s -f -m 1 "${NGINX_URL}/healthz" >/dev/null 2>&1
}

cleanup() {
    echo "--- Stopping test sandbox containers ---"
    docker compose -f deploy/docker-compose.yaml down -v --remove-orphans >/dev/null 2>&1 || true
    docker rm -f quadsmith-sandbox-db >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "--- Stopping any existing sandbox instance ---"
cleanup

echo "--- Building and Starting Sandbox Database (quadsmith-sandbox-db on port ${DB_PORT}) ---"
docker build -t quadsmith-sandbox-db -f test/sandbox/db/Dockerfile .
docker run -d --name quadsmith-sandbox-db -p "${DB_PORT}:5432" quadsmith-sandbox-db >/dev/null
until docker exec quadsmith-sandbox-db pg_isready -h 127.0.0.1 -U postgres -d quadsmith_ops >/dev/null 2>&1; do
    sleep 0.1
done

echo "--- Building Sandbox Container Images ---"
docker compose -f deploy/docker-compose.yaml build

echo "--- Starting fresh Sandbox containers on port ${PORT} (Gateway: ${NGINX_PORT}, Ops API: ${OPS_PORT}, Ops Web: ${OPS_WEB_PORT}, Cron: ${CRON_PORT}, API: ${API_PORT}, DB: ${DB_PORT}) ---"
QS_DATABASE_URL="postgres://postgres:postgres@host.docker.internal:${DB_PORT}/quadsmith_api?sslmode=disable" \
QS_INTERNAL_DATABASE_URL="postgres://postgres:postgres@host.docker.internal:${DB_PORT}/quadsmith_ops?sslmode=disable" \
docker compose -f deploy/docker-compose.yaml up -d --remove-orphans

echo "--- Waiting for Sandbox to become healthy ---"
MAX_RETRIES=50
COUNT=0
until is_sandbox_ready; do
    sleep 0.2
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -ge "$MAX_RETRIES" ]; then
        echo "Error: Sandbox failed to become ready after $((MAX_RETRIES / 5)) seconds."
        echo "=== Container logs ==="
        docker compose -f deploy/docker-compose.yaml logs | tail -n 50
        exit 1
    fi
done
echo "--- Sandbox is ready! ---"

echo "--- Running Sandbox Tests ---"
export QS_CLI_API_URL="${API_URL}"
export QS_API_URL="${API_URL}"
export QS_OPS_SERVER_URL="${OPS_URL}"
export QS_OPS_URL="${OPS_URL}"
export QS_OPS_WEB_URL="${OPS_WEB_URL}"
export QS_CRON_URL="${CRON_URL}"
export QS_WEB_URL="${WEB_URL}"
export QS_NGINX_URL="${NGINX_URL}"
export QS_OPS_TOKEN="qs_op_sandbox_token_12345"
export QS_DATABASE_URL="${DB_URL}"
export DATABASE_URL="${DB_URL}"
export QS_INTERNAL_DATABASE_URL="${INTERNAL_DB_URL}"
export GOWORK="${ROOT_DIR}/src/go/go.work"

go test -count=1 -v ./test/...

echo "--- Building Web E2E Test Container ---"
docker build -t quadsmith-web-e2e -f test/web/Dockerfile .

echo "--- Running Web E2E and Visual Regression Tests ---"
mkdir -p "${ROOT_DIR}/.tmp/playwright-report"
mkdir -p "${ROOT_DIR}/.tmp/test-results"
mkdir -p "${ROOT_DIR}/test/web/snapshots"
docker run --rm --ipc=host --network host \
    -v "${ROOT_DIR}/test/web/snapshots:/app/snapshots" \
    -v "${ROOT_DIR}/.tmp/playwright-report:/app/playwright-report" \
    -v "${ROOT_DIR}/.tmp/test-results:/app/test-results" \
    -e BASE_URL="${WEB_URL}" \
    -e OPS_WEB_URL="${OPS_WEB_URL}" \
    -e HOST_UID="$(id -u)" \
    -e HOST_GID="$(id -g)" \
    ${PLAYWRIGHT_WORKERS:+-e PLAYWRIGHT_WORKERS=${PLAYWRIGHT_WORKERS}} \
    ${UPDATE_SNAPSHOTS:+-e UPDATE_SNAPSHOTS=true} \
    quadsmith-web-e2e "$@"
