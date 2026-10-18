#!/usr/bin/env bash
set -e

# Always run from the root of the repo
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Allow passing an optional port, default to 8080
export PORT="${1:-8080}"
export DB_PORT="${DB_PORT:-5432}"
export OPS_PORT="${OPS_PORT:-8082}"
export OPS_WEB_PORT="${OPS_WEB_PORT:-8083}"
export CRON_PORT="${CRON_PORT:-8085}"
export API_PORT="${API_PORT:-8081}"
export NGINX_PORT="${NGINX_PORT:-8084}"

cleanup() {
    docker compose -f deploy/docker-compose.yaml down -v --remove-orphans >/dev/null 2>&1 || true
    docker rm -f quadsmith-sandbox-db >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "--- Stopping any existing sandbox instance ---"
cleanup

echo "--- Building and Starting Sandbox Database (quadsmith-sandbox-db on port $DB_PORT) ---"
docker build -t quadsmith-sandbox-db -f test/sandbox/db/Dockerfile .
docker run -d --name quadsmith-sandbox-db -p "${DB_PORT}:5432" quadsmith-sandbox-db >/dev/null
until docker exec quadsmith-sandbox-db pg_isready -h 127.0.0.1 -U postgres -d quadsmith_ops >/dev/null 2>&1; do
    sleep 0.1
done

echo "--- Building Sandbox Container Images ---"
docker compose -f deploy/docker-compose.yaml build

echo "--- Starting Quadsmith Sandbox (Web on $PORT, Gateway on $NGINX_PORT, Ops API on $OPS_PORT, Ops Web on $OPS_WEB_PORT, Cron on $CRON_PORT, API on $API_PORT, DB on $DB_PORT) ---"
if [ -f "test/sandbox/db/seed_data/users.textproto" ]; then
    echo "Pre-seeded users:"
    awk '
        /^[[:space:]]*id:[[:space:]]*"/ { gsub(/^[[:space:]]*id:[[:space:]]*"|"[[:space:]]*$/, ""); id=$0 }
        /^[[:space:]]*email:[[:space:]]*"/ { gsub(/^[[:space:]]*email:[[:space:]]*"|"[[:space:]]*$/, ""); email=$0 }
        /^[[:space:]]*}/ { if (email != "" && id != "") { print "  - " email " (@" id ")"; email=""; id="" } }
    ' test/sandbox/db/seed_data/users.textproto | sort
fi
QS_DATABASE_URL="postgres://postgres:postgres@host.docker.internal:${DB_PORT}/quadsmith_api?sslmode=disable" \
QS_INTERNAL_DATABASE_URL="postgres://postgres:postgres@host.docker.internal:${DB_PORT}/quadsmith_ops?sslmode=disable" \
docker compose -f deploy/docker-compose.yaml up --remove-orphans
