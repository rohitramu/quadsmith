#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_FILE="deploy/docker-compose.blue-green.yaml"
UPSTREAMS_FILE="deploy/nginx/upstreams.conf"

REQUIRED_VARS=(
    QS_ENV
    QS_IMAGE_REGISTRY
    QS_IMAGE_TAG
    QS_DATABASE_URL
    QS_INTERNAL_DATABASE_URL
    QS_AUTH_SECRET
    QS_SMTP_PASSWORD
    QS_GOOGLE_CLIENT_ID
    QS_GOOGLE_CLIENT_SECRET
    QS_GITHUB_CLIENT_ID
    QS_GITHUB_CLIENT_SECRET
    QS_DISCORD_CLIENT_ID
    QS_DISCORD_CLIENT_SECRET
    QS_OAUTH_CLIENT_ID
    QS_OAUTH_CLIENT_SECRET
    QS_JOB_SERVER_AUTH_TOKEN
)

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var:-}" ]; then
        echo "ERROR: Required runtime environment variable '${var}' is not set or is empty." >&2
        exit 1
    fi
done

write_upstreams() {
    local slot="$1"
    local tmp_file="${UPSTREAMS_FILE}.tmp"
    cat > "$tmp_file" <<EOF
upstream web_upstream {
    server web-${slot}:8080;
    keepalive 16;
}

upstream api_upstream {
    server api-${slot}:8080;
    keepalive 32;
}

upstream ops_upstream {
    server ops-${slot}:8080;
    keepalive 8;
}
EOF
    mv -f "$tmp_file" "$UPSTREAMS_FILE"
}

ACTIVE_SLOT=""
if grep -q "api-blue:8080" "$UPSTREAMS_FILE" 2>/dev/null && \
   [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-blue 2>/dev/null || true)" ]; then
    ACTIVE_SLOT="blue"
    NEXT_SLOT="green"
elif grep -q "api-green:8080" "$UPSTREAMS_FILE" 2>/dev/null && \
     [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-green 2>/dev/null || true)" ]; then
    ACTIVE_SLOT="green"
    NEXT_SLOT="blue"
else
    ACTIVE_SLOT=""
    NEXT_SLOT="blue"
fi

CANDIDATE_SERVICES=("api-${NEXT_SLOT}" "ops-${NEXT_SLOT}" "jobs-${NEXT_SLOT}" "web-${NEXT_SLOT}")

echo "--- Blue-Green Cutover: active=${ACTIVE_SLOT:-none} -> candidate=${NEXT_SLOT} ---"

echo "--- Pulling candidate slot (${NEXT_SLOT}) images ---"
docker compose -f "$COMPOSE_FILE" pull "${CANDIDATE_SERVICES[@]}"

echo "--- Starting candidate slot (${NEXT_SLOT}) containers ---"
docker compose -f "$COMPOSE_FILE" up -d "${CANDIDATE_SERVICES[@]}"

if [ -z "$ACTIVE_SLOT" ]; then
    write_upstreams "$NEXT_SLOT"
fi

docker compose -f "$COMPOSE_FILE" up -d nginx

echo "--- Gating cutover on /healthz readiness for slot ${NEXT_SLOT} (timeout: 15s) ---"
HEALTHY=false
for _ in $(seq 1 15); do
    if docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "
        wget -q -O - http://api-${NEXT_SLOT}:8080/healthz >/dev/null 2>&1 &&
        wget -q -O - http://ops-${NEXT_SLOT}:8080/healthz >/dev/null 2>&1 &&
        wget -q -O - http://web-${NEXT_SLOT}:8080/healthz >/dev/null 2>&1
    " && [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q --status running "jobs-${NEXT_SLOT}" 2>/dev/null || true)" ]; then
        HEALTHY=true
        break
    fi
    sleep 1
done

if [ "$HEALTHY" != "true" ]; then
    echo "ERROR: Candidate slot (${NEXT_SLOT}) failed health checks within 15s. Rolling back..."
    docker compose -f "$COMPOSE_FILE" logs --tail=50 "${CANDIDATE_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" stop -t 10 "${CANDIDATE_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" rm -f "${CANDIDATE_SERVICES[@]}" || true
    exit 1
fi

echo "--- Candidate slot (${NEXT_SLOT}) is healthy. Switching Nginx upstreams and reloading ---"
write_upstreams "$NEXT_SLOT"
docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "nginx -t && nginx -s reload"

if [ -n "$ACTIVE_SLOT" ] && [ "$ACTIVE_SLOT" != "$NEXT_SLOT" ]; then
    OLD_SERVICES=("web-${ACTIVE_SLOT}" "jobs-${ACTIVE_SLOT}" "api-${ACTIVE_SLOT}" "ops-${ACTIVE_SLOT}")
    echo "--- Draining and stopping previous slot (${ACTIVE_SLOT}) ---"
    sleep 2
    docker compose -f "$COMPOSE_FILE" stop -t 30 "${OLD_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" rm -f "${OLD_SERVICES[@]}" || true
fi

echo "--- Blue-Green deployment complete (active slot: ${NEXT_SLOT}) ---"
