#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ENV="${ENV:-${QS_ENV:-prod}}"
STAGE="${STAGE:-all}"
ALLOW_BREAKING_CHANGES="${ALLOW_BREAKING_CHANGES:-false}"

for arg in "$@"; do
    case "$arg" in
        --env=*) ENV="${arg#*=}" ;;
        --stage=*) STAGE="${arg#*=}" ;;
        --allow-breaking-changes=*) ALLOW_BREAKING_CHANGES="${arg#*=}" ;;
    esac
done

CONFIG_FILE="deploy/config/${ENV}.textproto"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Deployment config file '$CONFIG_FILE' not found (supported: staging, prod)."
    exit 1
fi

ENV_PREFIX="$(echo "$ENV" | tr '[:lower:]' '[:upper:]')"
resolve_env_secret() {
    local base_name="$1"
    local prefixed_name="${ENV_PREFIX}_${base_name}"
    if [ -n "${!prefixed_name:-}" ]; then
        printf '%s' "${!prefixed_name}"
    else
        printf '%s' "${!base_name:-}"
    fi
}

CF_ACCOUNT_ID="$(resolve_env_secret CF_ACCOUNT_ID)"
CF_ZONE_ID="$(resolve_env_secret CF_ZONE_ID)"
CF_API_TOKEN="$(resolve_env_secret CF_API_TOKEN)"
QS_DB_PASSWORD="$(resolve_env_secret QS_DB_PASSWORD)"
QS_AUTH_SECRET="$(resolve_env_secret QS_AUTH_SECRET)"
QS_JOB_SERVER_AUTH_TOKEN="$(resolve_env_secret QS_JOB_SERVER_AUTH_TOKEN)"
QS_SMTP_PASSWORD="$(resolve_env_secret QS_SMTP_PASSWORD)"
QS_GOOGLE_CLIENT_ID="$(resolve_env_secret QS_GOOGLE_CLIENT_ID)"
QS_GOOGLE_CLIENT_SECRET="$(resolve_env_secret QS_GOOGLE_CLIENT_SECRET)"
QS_GITHUB_CLIENT_ID="$(resolve_env_secret QS_GITHUB_CLIENT_ID)"
QS_GITHUB_CLIENT_SECRET="$(resolve_env_secret QS_GITHUB_CLIENT_SECRET)"
QS_DISCORD_CLIENT_ID="$(resolve_env_secret QS_DISCORD_CLIENT_ID)"
QS_DISCORD_CLIENT_SECRET="$(resolve_env_secret QS_DISCORD_CLIENT_SECRET)"
QS_OAUTH_CLIENT_ID="$(resolve_env_secret QS_OAUTH_CLIENT_ID)"
QS_OAUTH_CLIENT_SECRET="$(resolve_env_secret QS_OAUTH_CLIENT_SECRET)"

echo "--- Validating configuration files with check-config ---"
go run -mod=vendor ./src/go/shared/cmd/check-config

GCP_PROJECT_ID="$(awk -F'"' '/^[[:space:]]*gcp_project_id:/ {print $2}' "$CONFIG_FILE")"
REGISTRY_REGION="$(awk -F'"' '/^[[:space:]]*registry_region:/ {print $2}' "$CONFIG_FILE")"
BACKUP_RETENTION="$(awk '/^[[:space:]]*db_backup_retention_count:/ {print $2}' "$CONFIG_FILE")"
GAR_REPOSITORY="quadsmith-docker"
IMAGE_REGISTRY="${REGISTRY_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}"
GIT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo "local")"
IMAGE_TAG="${QS_IMAGE_TAG:-sha-${GIT_SHA}}"

# Extract service hostnames from single-source-of-truth component textprotos
WEB_HOST="$(awk -F'"' '/web_base_url:/ {print $2}' "src/go/server-api/config/${ENV}.textproto" | sed -E 's|^https?://||; s|/.*$||')"
API_HOST="$(awk -F'"' '/api_url:/ {print $2}' "src/go/server-web/config/${ENV}.textproto" | sed -E 's|^https?://||; s|/.*$||')"
OPS_HOST="$(awk -F'"' '/ops_server_url:/ {print $2}' "src/go/cli-ops/config/${ENV}.textproto" | sed -E 's|^https?://||; s|/.*$||')"

# Parse regions and nodes from deploy/config/${ENV}.textproto:
# Emits lines: REGION|<region>|<db_tier> and NODE|<region>|<zone>|<machine_type>|<idx>
mapfile -t TOPOLOGY_LINES < <(awk '
    /^[[:space:]]*regions[[:space:]]*\{/ { in_region=1; region=""; db_tier=""; node_idx=0; next }
    in_region && /^[[:space:]]*nodes[[:space:]]*\{/ { in_node=1; zone=""; mtype=""; next }
    in_node && /^[[:space:]]*zone:/ { gsub(/.*:[[:space:]]*"|"[[:space:]]*$/, ""); zone=$0; next }
    in_node && /^[[:space:]]*machine_type:/ { gsub(/.*:[[:space:]]*"|"[[:space:]]*$/, ""); mtype=$0; next }
    in_node && /^[[:space:]]*\}/ {
        node_idx++;
        print "NODE|" region "|" zone "|" mtype "|" node_idx;
        in_node=0;
        next
    }
    in_region && !in_node && /^[[:space:]]*region:/ { gsub(/.*:[[:space:]]*"|"[[:space:]]*$/, ""); region=$0; next }
    in_region && !in_node && /^[[:space:]]*db_tier:/ { gsub(/.*:[[:space:]]*"|"[[:space:]]*$/, ""); db_tier=$0; next }
    in_region && !in_node && /^[[:space:]]*\}/ {
        print "REGION|" region "|" db_tier;
        in_region=0;
        next
    }
' "$CONFIG_FILE")

PRIMARY_REGION=""
PRIMARY_ZONE=""
PRIMARY_VM=""
for line in "${TOPOLOGY_LINES[@]}"; do
    IFS='|' read -r kind f1 f2 f3 f4 <<< "$line"
    if [ "$kind" = "NODE" ] && [ -z "$PRIMARY_VM" ]; then
        PRIMARY_REGION="$f1"
        PRIMARY_ZONE="$f2"
        PRIMARY_VM="quadsmith-${ENV}-${f2}-${f4}"
    fi
done

ensure_gcp_network_and_registry() {
    echo "--- [1/4] Ensuring GCP Artifact Registry, IAP Firewall, Cloud NAT & Private Services Access (${GCP_PROJECT_ID}) ---"
    if ! gcloud artifacts repositories describe "$GAR_REPOSITORY" \
        --project="$GCP_PROJECT_ID" --location="$REGISTRY_REGION" >/dev/null 2>&1; then
        gcloud artifacts repositories create "$GAR_REPOSITORY" \
            --project="$GCP_PROJECT_ID" \
            --location="$REGISTRY_REGION" \
            --repository-format=docker \
            --description="Quadsmith container images (${ENV})"
    fi

    if ! gcloud compute firewall-rules describe allow-ssh-ingress-from-iap \
        --project="$GCP_PROJECT_ID" >/dev/null 2>&1; then
        gcloud compute firewall-rules create allow-ssh-ingress-from-iap \
            --project="$GCP_PROJECT_ID" \
            --direction=INGRESS \
            --action=ALLOW \
            --rules=tcp:22 \
            --source-ranges=35.235.240.0/20
    fi

    if ! gcloud compute addresses describe google-managed-services-default \
        --global --project="$GCP_PROJECT_ID" >/dev/null 2>&1; then
        gcloud compute addresses create google-managed-services-default \
            --global \
            --purpose=VPC_PEERING \
            --prefix-length=16 \
            --network=default \
            --project="$GCP_PROJECT_ID"
        gcloud services vpc-peerings connect \
            --service=servicenetworking.googleapis.com \
            --ranges=google-managed-services-default \
            --network=default \
            --project="$GCP_PROJECT_ID"
    fi

    for line in "${TOPOLOGY_LINES[@]}"; do
        IFS='|' read -r kind reg _ <<< "$line"
        if [ "$kind" = "REGION" ]; then
            local router_name="quadsmith-nat-router-${reg}"
            if ! gcloud compute routers describe "$router_name" \
                --project="$GCP_PROJECT_ID" --region="$reg" >/dev/null 2>&1; then
                gcloud compute routers create "$router_name" \
                    --project="$GCP_PROJECT_ID" --network=default --region="$reg"
                gcloud compute routers nats create "quadsmith-nat-${reg}" \
                    --project="$GCP_PROJECT_ID" \
                    --router="$router_name" \
                    --region="$reg" \
                    --auto-allocate-nat-external-ips \
                    --nat-all-subnet-ip-ranges
            fi
        fi
    done
}

declare -A REGION_DB_IPS
ensure_cloud_sql_topology() {
    echo "--- [2/4] Ensuring Regional GCP Cloud SQL Primaries & Logical Databases (${ENV}) ---"
    for line in "${TOPOLOGY_LINES[@]}"; do
        IFS='|' read -r kind reg db_tier <<< "$line"
        if [ "$kind" != "REGION" ]; then
            continue
        fi
        local instance_name="quadsmith-db-${ENV}-${reg}"
        if ! gcloud sql instances describe "$instance_name" --project="$GCP_PROJECT_ID" >/dev/null 2>&1; then
            echo "Creating Cloud SQL primary instance ${instance_name} (${db_tier}) in ${reg}..."
            gcloud sql instances create "$instance_name" \
                --project="$GCP_PROJECT_ID" \
                --database-version=POSTGRES_16 \
                --tier="$db_tier" \
                --region="$reg" \
                --network="projects/${GCP_PROJECT_ID}/global/networks/default" \
                --no-assign-ip \
                --storage-type=SSD \
                --storage-size=10GB \
                --storage-auto-increase \
                --backup-start-time=04:00 \
                --enable-point-in-time-recovery \
                --database-flags=cloudsql.logical_replication=on,max_connections=100 \
                --root-password="${QS_DB_PASSWORD}"
        else
            local current_tier
            current_tier="$(gcloud sql instances describe "$instance_name" --project="$GCP_PROJECT_ID" --format="value(settings.tier)")"
            if [ "$current_tier" != "$db_tier" ]; then
                echo "Scaling Cloud SQL instance ${instance_name} tier: ${current_tier} -> ${db_tier}..."
                gcloud sql instances patch "$instance_name" \
                    --project="$GCP_PROJECT_ID" \
                    --tier="$db_tier" \
                    --quiet
            fi
        fi

        for db_name in quadsmith_api quadsmith_ops; do
            if ! gcloud sql databases describe "$db_name" \
                --instance="$instance_name" --project="$GCP_PROJECT_ID" >/dev/null 2>&1; then
                gcloud sql databases create "$db_name" \
                    --instance="$instance_name" --project="$GCP_PROJECT_ID"
            fi
        done

        local db_ip
        db_ip="$(gcloud sql instances describe "$instance_name" --project="$GCP_PROJECT_ID" --format="value(ipAddresses[0].ipAddress)")"
        REGION_DB_IPS["$reg"]="$db_ip"
    done
}

CF_TUNNEL_TOKEN="${QS_CLOUDFLARE_TUNNEL_TOKEN:-}"
ensure_cloudflare_tunnel_and_edge() {
    if [ -z "${CF_API_TOKEN:-}" ] || [ -z "${CF_ACCOUNT_ID:-}" ] || [ -z "${CF_ZONE_ID:-}" ]; then
        echo "--- [3/4] Skipping Cloudflare API checks (CF_API_TOKEN / CF_ACCOUNT_ID / CF_ZONE_ID not set) ---"
        return 0
    fi
    echo "--- [3/4] Ensuring Cloudflare Tunnel, Proxied DNS CNAMEs & Edge Cache Rules (quadsmith-${ENV}) ---"
    local tunnel_name="quadsmith-${ENV}"
    local tunnel_resp
    tunnel_resp="$(curl -fsSL -H "Authorization: Bearer ${CF_API_TOKEN}" \
        "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/cfd_tunnel?name=${tunnel_name}&is_deleted=false")"
    local tunnel_id
    tunnel_id="$(printf '%s' "$tunnel_resp" | grep -oE '"id":"[^"]+"' | head -n1 | cut -d'"' -f4 || true)"

    if [ -z "$tunnel_id" ]; then
        local create_resp
        create_resp="$(curl -fsSL -X POST -H "Authorization: Bearer ${CF_API_TOKEN}" \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"${tunnel_name}\",\"config_src\":\"cloudflare\"}" \
            "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/cfd_tunnel")"
        tunnel_id="$(printf '%s' "$create_resp" | grep -oE '"id":"[^"]+"' | head -n1 | cut -d'"' -f4)"
    fi

    # Configure tunnel ingress to forward all traffic to local Nginx gateway on 127.0.0.1:8084
    curl -fsSL -X PUT -H "Authorization: Bearer ${CF_API_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"config\":{\"ingress\":[{\"service\":\"http://127.0.0.1:8084\"}]}}" \
        "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/cfd_tunnel/${tunnel_id}/configurations" >/dev/null

    if [ -z "$CF_TUNNEL_TOKEN" ]; then
        local token_resp
        token_resp="$(curl -fsSL -H "Authorization: Bearer ${CF_API_TOKEN}" \
            "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/cfd_tunnel/${tunnel_id}/token")"
        CF_TUNNEL_TOKEN="$(printf '%s' "$token_resp" | grep -oE '"result":"[^"]+"' | cut -d'"' -f4)"
    fi

    # Enable Cloudflare Smart Tiered Cache
    curl -fsSL -X PATCH -H "Authorization: Bearer ${CF_API_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"value":"on"}' \
        "https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/argo/tiered_caching" >/dev/null || true

    local hostnames=("$WEB_HOST" "$API_HOST" "$OPS_HOST")
    if [ "$ENV" = "prod" ]; then
        hostnames+=("www.${WEB_HOST}")
    fi
    for fqdn in "${hostnames[@]}"; do
        local existing_dns
        existing_dns="$(curl -fsSL -H "Authorization: Bearer ${CF_API_TOKEN}" \
            "https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/dns_records?type=CNAME&name=${fqdn}")"
        if ! printf '%s' "$existing_dns" | grep -q "\"name\":\"${fqdn}\""; then
            curl -fsSL -X POST -H "Authorization: Bearer ${CF_API_TOKEN}" \
                -H "Content-Type: application/json" \
                -d "{\"type\":\"CNAME\",\"name\":\"${fqdn}\",\"content\":\"${tunnel_id}.cfargotunnel.com\",\"proxied\":true}" \
                "https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/dns_records" >/dev/null
        fi
    done
}

ensure_compute_nodes() {
    echo "--- [4/4] Ensuring Regional Compute Engine VMs & Host Hygiene ---"
    for line in "${TOPOLOGY_LINES[@]}"; do
        IFS='|' read -r kind reg zone mtype idx <<< "$line"
        if [ "$kind" != "NODE" ]; then
            continue
        fi
        local vm_name="quadsmith-${ENV}-${zone}-${idx}"
        if ! gcloud compute instances describe "$vm_name" \
            --project="$GCP_PROJECT_ID" --zone="$zone" >/dev/null 2>&1; then
            echo "Creating Compute Engine VM ${vm_name} (${mtype}) in ${zone}..."
            gcloud compute instances create "$vm_name" \
                --project="$GCP_PROJECT_ID" \
                --zone="$zone" \
                --machine-type="$mtype" \
                --no-address \
                --boot-disk-size=30GB \
                --boot-disk-type=pd-standard \
                --image-family=debian-12 \
                --image-project=debian-cloud \
                --scopes=cloud-platform
        else
            local current_mtype
            current_mtype="$(gcloud compute instances describe "$vm_name" \
                --project="$GCP_PROJECT_ID" --zone="$zone" --format="value(machineType.basename())")"
            if [ "$current_mtype" != "$mtype" ]; then
                echo "Resizing VM ${vm_name} machine type: ${current_mtype} -> ${mtype}..."
                gcloud compute instances stop "$vm_name" --project="$GCP_PROJECT_ID" --zone="$zone" --quiet
                gcloud compute instances set-machine-type "$vm_name" \
                    --project="$GCP_PROJECT_ID" --zone="$zone" --machine-type="$mtype" --quiet
                gcloud compute instances start "$vm_name" --project="$GCP_PROJECT_ID" --zone="$zone" --quiet
            fi
        fi

        gcloud compute ssh "$vm_name" \
            --tunnel-through-iap \
            --project="$GCP_PROJECT_ID" \
            --zone="$zone" \
            --command="
                set -e
                if [ ! -f /swapfile ]; then
                    sudo fallocate -l 2G /swapfile
                    sudo chmod 600 /swapfile
                    sudo mkswap /swapfile
                    sudo swapon /swapfile
                    echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab >/dev/null
                    sudo sysctl vm.swappiness=10
                fi
                if ! command -v docker >/dev/null 2>&1; then
                    sudo apt-get update -qq
                    sudo apt-get install -y -qq docker.io docker-compose-plugin wget curl
                fi
                sudo mkdir -p /etc/docker /opt/quadsmith/deploy/nginx
                sudo chown -R \"\$(whoami)\" /opt/quadsmith
                if [ ! -f /etc/docker/daemon.json ]; then
                    echo '{\"log-driver\":\"json-file\",\"log-opts\":{\"max-size\":\"10m\",\"max-file\":\"3\"}}' | sudo tee /etc/docker/daemon.json >/dev/null
                    sudo systemctl restart docker
                fi
                gcloud auth configure-docker ${REGISTRY_REGION}-docker.pkg.dev --quiet
                if [ -n '${CF_TUNNEL_TOKEN}' ] && [ -z \"\$(docker ps -q -f name=^cloudflared\$)\" ]; then
                    docker rm -f cloudflared >/dev/null 2>&1 || true
                    docker run -d --name cloudflared --restart unless-stopped --network host \
                        cloudflare/cloudflared:latest tunnel --no-autoupdate run --token '${CF_TUNNEL_TOKEN}'
                fi
            "
    done
}

build_and_push_images() {
    echo "--- Building and pushing 4 service images to ${IMAGE_REGISTRY} (tag: ${IMAGE_TAG}) ---"
    gcloud auth configure-docker "${REGISTRY_REGION}-docker.pkg.dev" --quiet
    for svc in server-web server-api server-ops server-jobs; do
        docker build -f "src/go/${svc}/Dockerfile" \
            -t "${IMAGE_REGISTRY}/${svc}:latest" \
            -t "${IMAGE_REGISTRY}/${svc}:${IMAGE_TAG}" \
            .
        docker push "${IMAGE_REGISTRY}/${svc}:latest"
        docker push "${IMAGE_REGISTRY}/${svc}:${IMAGE_TAG}"
    done
}

run_migration_gate_and_seed() {
    local instance_name="quadsmith-db-${ENV}-${PRIMARY_REGION}"
    local primary_db_ip="${REGION_DB_IPS[$PRIMARY_REGION]:-}"
    if [ -z "$primary_db_ip" ]; then
        primary_db_ip="$(gcloud sql instances describe "$instance_name" --project="$GCP_PROJECT_ID" --format="value(ipAddresses[0].ipAddress)")"
        REGION_DB_IPS["$PRIMARY_REGION"]="$primary_db_ip"
    fi

    echo "--- Opening ephemeral IAP TCP tunnel to Primary Cloud SQL (${primary_db_ip}:5432) via ${PRIMARY_VM} ---"
    local local_port=15432
    gcloud compute ssh "$PRIMARY_VM" \
        --tunnel-through-iap \
        --project="$GCP_PROJECT_ID" \
        --zone="$PRIMARY_ZONE" \
        -- -L "${local_port}:${primary_db_ip}:5432" -N -f

    trap 'pkill -f "${local_port}:${primary_db_ip}:5432" >/dev/null 2>&1 || true' EXIT

    local api_diff ops_diff
    api_diff="$(PGSSLMODE=require psqldef -U postgres -W "${QS_DB_PASSWORD}" -h 127.0.0.1 -p "$local_port" --dry-run quadsmith_api < src/db/quadsmith_api.sql | grep -v '^--' || true)"
    ops_diff="$(PGSSLMODE=require psqldef -U postgres -W "${QS_DB_PASSWORD}" -h 127.0.0.1 -p "$local_port" --dry-run quadsmith_ops < src/db/quadsmith_ops.sql | grep -v '^--' || true)"

    if [ -n "${api_diff}${ops_diff}" ]; then
        echo "--- Pending DDL changes detected by psqldef --dry-run ---"
        mkdir -p .tmp/live_schema
        PGSSLMODE=require psqldef -U postgres -W "${QS_DB_PASSWORD}" -h 127.0.0.1 -p "$local_port" --export quadsmith_api > .tmp/live_schema/quadsmith_api.sql
        PGSSLMODE=require psqldef -U postgres -W "${QS_DB_PASSWORD}" -h 127.0.0.1 -p "$local_port" --export quadsmith_ops > .tmp/live_schema/quadsmith_ops.sql

        if grep -q "CREATE TABLE" .tmp/live_schema/quadsmith_api.sql; then
            if [ "$ALLOW_BREAKING_CHANGES" = "true" ]; then
                echo "--- allow_breaking_changes=true: Running make warn-breaking ---"
                QS_LIVE_SCHEMA_DIR="$(pwd)/.tmp/live_schema" make warn-breaking
            else
                echo "--- Running strict migration gate (make check-breaking) against live Cloud SQL schema ---"
                QS_LIVE_SCHEMA_DIR="$(pwd)/.tmp/live_schema" make check-breaking
            fi

            local ts pitr_ts
            ts="$(date -u +%Y%m%dT%H%M%SZ)"
            pitr_ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
            echo "--- Creating Cloud SQL ON_DEMAND pre-migration backup on ${instance_name} (retaining ${BACKUP_RETENTION}) ---"
            echo "--- Pre-migration PITR restore timestamp: ${pitr_ts} ---"
            gcloud sql backups create \
                --instance="$instance_name" \
                --description="Pre-migration ${GIT_SHA} (${ts})" \
                --project="$GCP_PROJECT_ID"

            mapfile -t OLD_BACKUP_IDS < <(
                gcloud sql backups list \
                    --instance="$instance_name" \
                    --project="$GCP_PROJECT_ID" \
                    --filter="type=ON_DEMAND AND status=SUCCESSFUL" \
                    --sort-by="~windowStartTime" \
                    --format="value(id)" | tail -n +$((BACKUP_RETENTION + 1))
            )
            for old_id in "${OLD_BACKUP_IDS[@]}"; do
                [ -z "$old_id" ] && continue
                echo "--- Pruning expired ON_DEMAND backup ID ${old_id} on ${instance_name} ---"
                gcloud sql backups delete "$old_id" \
                    --instance="$instance_name" \
                    --project="$GCP_PROJECT_ID" \
                    --quiet
            done
        fi

        echo "--- Applying declarative database migrations via make db-migrate ---"
        QS_DATABASE_URL="postgres://postgres:${QS_DB_PASSWORD}@127.0.0.1:${local_port}/quadsmith_api?sslmode=require" \
        QS_INTERNAL_DATABASE_URL="postgres://postgres:${QS_DB_PASSWORD}@127.0.0.1:${local_port}/quadsmith_ops?sslmode=require" \
        make db-migrate
    else
        echo "--- Zero pending DDL changes detected; skipping pre-migration Cloud SQL backup ---"
    fi

    echo "--- Ensuring robot token hash in quadsmith_ops.internal_tokens ---"
    local robot_hash
    robot_hash="$(printf '%s' "${QS_JOB_SERVER_AUTH_TOKEN}" | sha256sum | awk '{print $1}')"
    PGPASSWORD="${QS_DB_PASSWORD}" PGSSLMODE=require psql -h 127.0.0.1 -p "$local_port" -U postgres -d quadsmith_ops -v ON_ERROR_STOP=1 -c "
        INSERT INTO internal_tokens (uuid, id, name, token_hash, token_type, group_name)
        VALUES ('018f0000-0000-7000-8000-000000000001', 'tok_robot_jobs', 'server-jobs scheduler', '${robot_hash}', 'ROBOT', 'robots')
        ON CONFLICT (token_hash) DO NOTHING;
    "

    pkill -f "${local_port}:${primary_db_ip}:5432" >/dev/null 2>&1 || true
    trap - EXIT
}

rollout_blue_green_nodes() {
    echo "--- Rolling out zero-downtime Blue-Green containers across all VM nodes ---"
    for line in "${TOPOLOGY_LINES[@]}"; do
        IFS='|' read -r kind reg zone _ idx <<< "$line"
        if [ "$kind" != "NODE" ]; then
            continue
        fi
        local vm_name="quadsmith-${ENV}-${zone}-${idx}"
        local local_db_ip="${REGION_DB_IPS[$reg]:-}"
        if [ -z "$local_db_ip" ]; then
            local_db_ip="$(gcloud sql instances describe "quadsmith-db-${ENV}-${reg}" --project="$GCP_PROJECT_ID" --format="value(ipAddresses[0].ipAddress)")"
            REGION_DB_IPS["$reg"]="$local_db_ip"
        fi

        local env_tmp=".tmp/${vm_name}.env"
        mkdir -p .tmp
        cat > "$env_tmp" <<EOF
QS_ENV=${ENV}
QS_IMAGE_REGISTRY=${IMAGE_REGISTRY}
QS_IMAGE_TAG=${IMAGE_TAG}
QS_DATABASE_URL=postgres://postgres:${QS_DB_PASSWORD}@${local_db_ip}:5432/quadsmith_api?sslmode=require
QS_INTERNAL_DATABASE_URL=postgres://postgres:${QS_DB_PASSWORD}@${local_db_ip}:5432/quadsmith_ops?sslmode=require
QS_AUTH_SECRET=${QS_AUTH_SECRET}
QS_SMTP_PASSWORD=${QS_SMTP_PASSWORD:-}
QS_GOOGLE_CLIENT_ID=${QS_GOOGLE_CLIENT_ID}
QS_GOOGLE_CLIENT_SECRET=${QS_GOOGLE_CLIENT_SECRET}
QS_GITHUB_CLIENT_ID=${QS_GITHUB_CLIENT_ID}
QS_GITHUB_CLIENT_SECRET=${QS_GITHUB_CLIENT_SECRET}
QS_DISCORD_CLIENT_ID=${QS_DISCORD_CLIENT_ID}
QS_DISCORD_CLIENT_SECRET=${QS_DISCORD_CLIENT_SECRET}
QS_OAUTH_CLIENT_ID=${QS_OAUTH_CLIENT_ID}
QS_OAUTH_CLIENT_SECRET=${QS_OAUTH_CLIENT_SECRET}
QS_JOB_SERVER_AUTH_TOKEN=${QS_JOB_SERVER_AUTH_TOKEN}
EOF

        gcloud compute scp --tunnel-through-iap \
            "$env_tmp" "${vm_name}:/opt/quadsmith/.env" \
            --project="$GCP_PROJECT_ID" --zone="$zone"
        rm -f "$env_tmp"

        gcloud compute scp --tunnel-through-iap --recurse \
            deploy/docker-compose.yaml \
            deploy/docker-compose.blue-green.yaml \
            deploy/deploy_blue_green.sh \
            deploy/nginx \
            "${vm_name}:/opt/quadsmith/deploy/" \
            --project="$GCP_PROJECT_ID" --zone="$zone"

        gcloud compute ssh "$vm_name" \
            --tunnel-through-iap \
            --project="$GCP_PROJECT_ID" \
            --zone="$zone" \
            --command="cd /opt/quadsmith && chmod +x deploy/deploy_blue_green.sh && set -a && source .env && set +a && ./deploy/deploy_blue_green.sh"
    done
}

case "$STAGE" in
    all)
        ensure_gcp_network_and_registry
        ensure_cloud_sql_topology
        ensure_cloudflare_tunnel_and_edge
        ensure_compute_nodes
        build_and_push_images
        run_migration_gate_and_seed
        rollout_blue_green_nodes
        ;;
    bootstrap)
        ensure_gcp_network_and_registry
        ensure_cloud_sql_topology
        ensure_cloudflare_tunnel_and_edge
        ensure_compute_nodes
        ;;
    build)
        build_and_push_images
        ;;
    migrate)
        ensure_gcp_network_and_registry
        ensure_cloud_sql_topology
        ensure_compute_nodes
        run_migration_gate_and_seed
        ;;
    rollout)
        ensure_gcp_network_and_registry
        ensure_cloud_sql_topology
        ensure_cloudflare_tunnel_and_edge
        ensure_compute_nodes
        rollout_blue_green_nodes
        ;;
    *)
        echo "Error: Unknown stage '${STAGE}' (expected: all, bootstrap, build, migrate, rollout)"
        exit 1
        ;;
esac
