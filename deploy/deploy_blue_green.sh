#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_FILE="deploy/docker-compose.blue-green.yaml"
UPSTREAMS_FILE="deploy/nginx/upstreams.conf"
SCRAPE_FILE="deploy/victoriametrics/scrape.yml"
ACTIVE_SLOT_FILE="deploy/.active_slot"

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
    QS_ROLLOUT_HEALTH_TIMEOUT_SEC
    QS_ROLLOUT_DRAIN_TIMEOUT_SEC
    QS_ROLLOUT_RAMP_STAGES
)

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var:-}" ]; then
        echo "ERROR: Required runtime environment variable '${var}' is not set or is empty." >&2
        exit 1
    fi
done

write_scrape_targets() {
    local active_slot="$1"
    local candidate_slot="$2"
    local tmp_file="${SCRAPE_FILE}.tmp"

    if [ -n "$active_slot" ] && [ -n "$candidate_slot" ] && [ "$active_slot" != "$candidate_slot" ]; then
        cat > "$tmp_file" <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "quadsmith-services"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "web-${active_slot}:8080"
          - "api-${active_slot}:8081"
          - "ops-api-${active_slot}:8082"
          - "ops-web-${active_slot}:8083"
          - "cron-${active_slot}:8084"
          - "web-${candidate_slot}:8080"
          - "api-${candidate_slot}:8081"
          - "ops-api-${candidate_slot}:8082"
          - "ops-web-${candidate_slot}:8083"
          - "cron-${candidate_slot}:8084"
EOF
    else
        local slot="${candidate_slot:-$active_slot}"
        cat > "$tmp_file" <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "quadsmith-services"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "web-${slot}:8080"
          - "api-${slot}:8081"
          - "ops-api-${slot}:8082"
          - "ops-web-${slot}:8083"
          - "cron-${slot}:8084"
EOF
    fi
    cat "$tmp_file" > "$SCRAPE_FILE"
    rm -f "$tmp_file"
}

write_weighted_upstreams() {
    local active_slot="$1"
    local active_weight="$2"
    local candidate_slot="$3"
    local candidate_weight="$4"
    local tmp_file="${UPSTREAMS_FILE}.tmp"

    if [ -n "$active_slot" ] && [ "$active_weight" -gt 0 ] && [ "$candidate_weight" -lt 100 ]; then
        cat > "$tmp_file" <<EOF
upstream web_upstream {
    server web-${active_slot}:8080 weight=${active_weight};
    server web-${candidate_slot}:8080 weight=${candidate_weight};
    keepalive 16;
}

upstream api_upstream {
    server api-${active_slot}:8081 weight=${active_weight};
    server api-${candidate_slot}:8081 weight=${candidate_weight};
    keepalive 32;
}

upstream ops_api_upstream {
    server ops-api-${active_slot}:8082 weight=${active_weight};
    server ops-api-${candidate_slot}:8082 weight=${candidate_weight};
    keepalive 8;
}

upstream ops_web_upstream {
    server ops-web-${active_slot}:8083 weight=${active_weight};
    server ops-web-${candidate_slot}:8083 weight=${candidate_weight};
    keepalive 8;
}
EOF
    else
        cat > "$tmp_file" <<EOF
upstream web_upstream {
    server web-${candidate_slot}:8080 weight=100;
    keepalive 16;
}

upstream api_upstream {
    server api-${candidate_slot}:8081 weight=100;
    keepalive 32;
}

upstream ops_api_upstream {
    server ops-api-${candidate_slot}:8082 weight=100;
    keepalive 8;
}

upstream ops_web_upstream {
    server ops-web-${candidate_slot}:8083 weight=100;
    keepalive 8;
}
EOF
    fi
    cat "$tmp_file" > "$UPSTREAMS_FILE"
    rm -f "$tmp_file"
}

check_candidate_health() {
    docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "
        wget -q -O - http://api-${NEXT_SLOT}:8081/healthz >/dev/null 2>&1 &&
        wget -q -O - http://ops-api-${NEXT_SLOT}:8082/healthz >/dev/null 2>&1 &&
        wget -q -O - http://ops-web-${NEXT_SLOT}:8083/healthz >/dev/null 2>&1 &&
        wget -q -O - http://cron-${NEXT_SLOT}:8084/healthz >/dev/null 2>&1 &&
        wget -q -O - http://web-${NEXT_SLOT}:8080/healthz >/dev/null 2>&1
    " && [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q --status running "cron-${NEXT_SLOT}" 2>/dev/null || true)" ]
}

ACTIVE_SLOT=""
if [ -f "$ACTIVE_SLOT_FILE" ]; then
    RECORDED_SLOT="$(tr -d '[:space:]' < "$ACTIVE_SLOT_FILE")"
    if [ "$RECORDED_SLOT" = "blue" ] && [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-blue 2>/dev/null || true)" ]; then
        ACTIVE_SLOT="blue"
        NEXT_SLOT="green"
    elif [ "$RECORDED_SLOT" = "green" ] && [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-green 2>/dev/null || true)" ]; then
        ACTIVE_SLOT="green"
        NEXT_SLOT="blue"
    fi
fi

if [ -z "$ACTIVE_SLOT" ]; then
    if grep -q "api-blue:8081" "$UPSTREAMS_FILE" 2>/dev/null && \
       [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-blue 2>/dev/null || true)" ]; then
        ACTIVE_SLOT="blue"
        NEXT_SLOT="green"
    elif grep -q "api-green:8081" "$UPSTREAMS_FILE" 2>/dev/null && \
         [ -n "$(docker compose -f "$COMPOSE_FILE" ps -q api-green 2>/dev/null || true)" ]; then
        ACTIVE_SLOT="green"
        NEXT_SLOT="blue"
    else
        ACTIVE_SLOT=""
        NEXT_SLOT="blue"
    fi
fi

CANDIDATE_SERVICES=("api-${NEXT_SLOT}" "ops-api-${NEXT_SLOT}" "ops-web-${NEXT_SLOT}" "cron-${NEXT_SLOT}" "web-${NEXT_SLOT}")

echo "--- Blue-Green Cutover: active=${ACTIVE_SLOT:-none} -> candidate=${NEXT_SLOT} (ramp=${QS_ROLLOUT_RAMP_STAGES}) ---"

if [ -n "$ACTIVE_SLOT" ]; then
    write_scrape_targets "$ACTIVE_SLOT" "$ACTIVE_SLOT"
else
    write_scrape_targets "$NEXT_SLOT" "$NEXT_SLOT"
fi

echo "--- Ensuring standing telemetry stack (victoriametrics, victorialogs, grafana) ---"
docker compose -f "$COMPOSE_FILE" up -d victoriametrics victorialogs grafana

echo "--- Pulling candidate slot (${NEXT_SLOT}) images ---"
PULL_ATTEMPT=1
while true; do
    if timeout --foreground 120s docker compose -f "$COMPOSE_FILE" pull "${CANDIDATE_SERVICES[@]}"; then
        break
    fi
    if [ "$PULL_ATTEMPT" -ge 3 ]; then
        echo "ERROR: Failed to pull candidate slot (${NEXT_SLOT}) images after ${PULL_ATTEMPT} attempts." >&2
        exit 1
    fi
    echo "Warning: Image pull attempt ${PULL_ATTEMPT}/3 failed; retrying in 5s..." >&2
    sleep 5
    PULL_ATTEMPT=$((PULL_ATTEMPT + 1))
done

echo "--- Starting candidate slot (${NEXT_SLOT}) containers ---"
docker compose -f "$COMPOSE_FILE" up -d "${CANDIDATE_SERVICES[@]}"
write_scrape_targets "$ACTIVE_SLOT" "$NEXT_SLOT"

if [ -z "$ACTIVE_SLOT" ]; then
    write_weighted_upstreams "" 0 "$NEXT_SLOT" 100
fi

docker compose -f "$COMPOSE_FILE" up -d nginx

echo "--- Gating cutover on /healthz readiness for slot ${NEXT_SLOT} (timeout: ${QS_ROLLOUT_HEALTH_TIMEOUT_SEC}s) ---"
HEALTHY=false
for _ in $(seq 1 "$QS_ROLLOUT_HEALTH_TIMEOUT_SEC"); do
    if check_candidate_health; then
        HEALTHY=true
        break
    fi
    sleep 1
done

if [ "$HEALTHY" != "true" ]; then
    echo "ERROR: Candidate slot (${NEXT_SLOT}) failed initial health checks within ${QS_ROLLOUT_HEALTH_TIMEOUT_SEC}s. Aborting..." >&2
    if [ -n "$ACTIVE_SLOT" ]; then
        write_scrape_targets "$ACTIVE_SLOT" "$ACTIVE_SLOT"
    fi
    docker compose -f "$COMPOSE_FILE" logs --tail=50 "${CANDIDATE_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" stop -t 10 "${CANDIDATE_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" rm -f "${CANDIDATE_SERVICES[@]}" || true
    exit 1
fi

if [ -n "$ACTIVE_SLOT" ] && [ "$ACTIVE_SLOT" != "$NEXT_SLOT" ]; then
    IFS=',' read -r -a STAGE_ENTRIES <<< "$QS_ROLLOUT_RAMP_STAGES"
    for stage_entry in "${STAGE_ENTRIES[@]}"; do
        candidate_weight="${stage_entry%%:*}"
        hold_sec="${stage_entry##*:}"
        active_weight=$((100 - candidate_weight))

        echo "--- Traffic Ramp Stage: ${ACTIVE_SLOT}=${active_weight}% -> ${NEXT_SLOT}=${candidate_weight}% (hold: ${hold_sec}s) ---"
        write_weighted_upstreams "$ACTIVE_SLOT" "$active_weight" "$NEXT_SLOT" "$candidate_weight"
        docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "nginx -t && nginx -s reload"

        if [ "$hold_sec" -gt 0 ]; then
            elapsed=0
            step_sec=5
            if [ "$hold_sec" -lt 5 ]; then
                step_sec=1
            fi
            while [ "$elapsed" -lt "$hold_sec" ]; do
                remaining=$((hold_sec - elapsed))
                if [ "$remaining" -lt "$step_sec" ]; then
                    sleep "$remaining"
                    elapsed=$((elapsed + remaining))
                else
                    sleep "$step_sec"
                    elapsed=$((elapsed + step_sec))
                fi
                if ! check_candidate_health; then
                    echo "ERROR: Candidate slot (${NEXT_SLOT}) failed health check during ${candidate_weight}% traffic stage (${elapsed}s/${hold_sec}s). Reverting 100% traffic to ${ACTIVE_SLOT}..." >&2
                    write_weighted_upstreams "" 0 "$ACTIVE_SLOT" 100
                    write_scrape_targets "$ACTIVE_SLOT" "$ACTIVE_SLOT"
                    docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "nginx -t && nginx -s reload"
                    docker compose -f "$COMPOSE_FILE" logs --tail=50 "${CANDIDATE_SERVICES[@]}" || true
                    docker compose -f "$COMPOSE_FILE" stop -t 10 "${CANDIDATE_SERVICES[@]}" || true
                    docker compose -f "$COMPOSE_FILE" rm -f "${CANDIDATE_SERVICES[@]}" || true
                    exit 1
                fi
                if [ $((elapsed % 60)) -eq 0 ] && [ "$elapsed" -lt "$hold_sec" ]; then
                    echo "--- [Stage ${candidate_weight}%] Candidate slot (${NEXT_SLOT}) healthy (${elapsed}s / ${hold_sec}s elapsed) ---"
                fi
            done
        fi
    done
else
    echo "--- Initial slot (${NEXT_SLOT}) healthy. Reloading Nginx with 100% traffic on ${NEXT_SLOT} ---"
    write_weighted_upstreams "" 0 "$NEXT_SLOT" 100
    docker compose -f "$COMPOSE_FILE" exec -T nginx sh -c "nginx -t && nginx -s reload"
fi

write_scrape_targets "$NEXT_SLOT" "$NEXT_SLOT"
printf '%s\n' "$NEXT_SLOT" > "$ACTIVE_SLOT_FILE"

if [ -n "$ACTIVE_SLOT" ] && [ "$ACTIVE_SLOT" != "$NEXT_SLOT" ]; then
    OLD_SERVICES=("web-${ACTIVE_SLOT}" "cron-${ACTIVE_SLOT}" "ops-web-${ACTIVE_SLOT}" "ops-api-${ACTIVE_SLOT}" "api-${ACTIVE_SLOT}")
    echo "--- Draining and stopping previous slot (${ACTIVE_SLOT}, timeout: ${QS_ROLLOUT_DRAIN_TIMEOUT_SEC}s) ---"
    sleep 2
    docker compose -f "$COMPOSE_FILE" stop -t "$QS_ROLLOUT_DRAIN_TIMEOUT_SEC" "${OLD_SERVICES[@]}" || true
    docker compose -f "$COMPOSE_FILE" rm -f "${OLD_SERVICES[@]}" || true
fi

echo "--- Blue-Green deployment complete (active slot: ${NEXT_SLOT}) ---"
