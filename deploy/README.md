# Quadsmith Deployment & Operations Guide

This directory contains the deployment manifests, declarative topology configs,
and automation scripts for running Quadsmith across local sandbox, staging, and
production environments.

---

## 1. Environments & Architecture Overview

Quadsmith uses a single, symmetric container topology across all three
environments (`sandbox`, `staging`, and `prod`):

- **`sandbox` (Local Ephemeral Test Harness)**: Started via `make sandbox` (or
  tested end-to-end via `make test`). Runs a standalone pre-seeded PostgreSQL 15
  container (`quadsmith-sandbox-db`) on `127.0.0.1:5432` alongside the stateless
  application stack in [`docker-compose.yaml`](docker-compose.yaml). Requires
  zero cloud credentials or internet access.
- **`staging` & `prod` (Live Cloud Environments)**: Configured declaratively via
  [`config/staging.textproto`](config/staging.textproto) and
  [`config/prod.textproto`](config/prod.textproto) and converged via a single
  command: `make deploy ENV=staging` or `make deploy ENV=prod`.
  - **Edge & Ingress**: Cloudflare DNS, CDN (with Smart Tiered Cache and RFC
    `Cache-Control` compliance), and outbound-only Cloudflare Tunnels
    (`cloudflared`).
  - **Compute**: Zero-public-IP Linux VMs on Google Cloud Compute Engine (starts
    with the Always Free `e2-micro` in `us-central1-a`), provisioned with a 2 GB
    `/swapfile`, `journald` 100 MB log cap, and Docker `json-file` log rotation.
  - **Database**: Managed **GCP Cloud SQL for PostgreSQL** (Enterprise Edition,
    starting on `db-f1-micro` with 10 GB SSD, `max_connections=100`, daily PITR
    backups, and `--no-assign-ip` Private VPC IP only) hosting both the
    `quadsmith_api` and `quadsmith_ops` logical databases.

---

## 2. One-Time Account & Secret Setup Checklist

Before running `make deploy` against a live environment for the first time,
complete the following one-time setup steps:

### Step 1: Cloudflare Zone & Scoped API Token

1. Add the `quadsmith.com` domain to Cloudflare (Free Plan).
2. In the Cloudflare dashboard overview page, record your **Account ID**
   (`CF_ACCOUNT_ID`) and **Zone ID** (`CF_ZONE_ID`).
3. Navigate to **My Profile &rarr; API Tokens &rarr; Create Custom Token** and
   create a token (`CF_API_TOKEN`) with the following permissions:
   - `Account : Cloudflare Tunnel : Edit`
   - `Zone : DNS : Edit`
   - `Zone : Cache Rules : Edit`
   - `Zone : Tiered Cache : Edit`

### Step 2: GCP Project & Deployer Service Account

Create a GCP project (matching `gcp_project_id` in
[`config/prod.textproto`](config/prod.textproto), e.g., `quadsmith-prod`) with
billing linked, and create the `quadsmith-cd` service account:

```bash
export PROJECT_ID="quadsmith-prod"

gcloud iam service-accounts create quadsmith-cd \
  --display-name="Quadsmith CI/CD Deployer" \
  --project="$PROJECT_ID"

for ROLE in \
  roles/serviceusage.serviceUsageAdmin \
  roles/compute.admin \
  roles/cloudsql.admin \
  roles/servicenetworking.networksAdmin \
  roles/artifactregistry.admin \
  roles/iam.serviceAccountUser \
  roles/iap.tunnelResourceAccessor; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:quadsmith-cd@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="$ROLE"
done

gcloud iam service-accounts keys create gcp-sa-key.json \
  --iam-account="quadsmith-cd@${PROJECT_ID}.iam.gserviceaccount.com"
```

Store the contents of `gcp-sa-key.json` as the `GCP_SA_KEY` secret and delete
the local key file.

### Step 3: Generate Cryptographic Application Secrets

Generate 256-bit random values for the database password, JWT signing secret,
and headless job runner token:

```bash
openssl rand -hex 32 # -> QS_DB_PASSWORD
openssl rand -hex 32 # -> QS_AUTH_SECRET
echo "qs_op_$(openssl rand -hex 24)" # -> QS_JOB_SERVER_AUTH_TOKEN
```

### Step 4: Configure GitHub Environment Secrets

In GitHub (**Settings &rarr; Environments**), create the `staging` and `prod`
environments and populate the following secrets:

| Secret Name                                         | Purpose                                                                                |
| :-------------------------------------------------- | :------------------------------------------------------------------------------------- |
| `GCP_SA_KEY`                                        | JSON key for the `quadsmith-cd` GCP service account                                    |
| `CF_ACCOUNT_ID`                                     | Cloudflare Account ID                                                                  |
| `CF_ZONE_ID`                                        | Cloudflare Zone ID for `quadsmith.com`                                                 |
| `CF_API_TOKEN`                                      | Scoped Cloudflare API Token                                                            |
| `QS_DB_PASSWORD`                                    | PostgreSQL `postgres` user password on Cloud SQL                                       |
| `QS_AUTH_SECRET`                                    | HMAC-SHA256 secret used by `server-api` to sign JWTs                                   |
| `QS_JOB_SERVER_AUTH_TOKEN`                          | Bearer token (`qs_op_...`) shared by `server-jobs` and `quadsmith_ops.internal_tokens` |
| `QS_SMTP_PASSWORD`                                  | SMTP password for transactional authentication emails                                  |
| `QS_GOOGLE_CLIENT_ID` / `QS_GOOGLE_CLIENT_SECRET`   | Google OAuth 2.0 Web Client credentials (`server-api`)                                 |
| `QS_GITHUB_CLIENT_ID` / `QS_GITHUB_CLIENT_SECRET`   | GitHub OAuth App credentials (`server-api`)                                            |
| `QS_DISCORD_CLIENT_ID` / `QS_DISCORD_CLIENT_SECRET` | Discord OAuth2 Application credentials (`server-api`)                                  |
| `QS_OAUTH_CLIENT_ID` / `QS_OAUTH_CLIENT_SECRET`     | Google OAuth 2.0 credentials for `qsops` operator login (`server-ops`)                 |

---

## 3. Deployment Configuration (`deploy/config/{staging,prod}.textproto`)

Infrastructure topology is defined declaratively in
[`config/staging.textproto`](config/staging.textproto) and
[`config/prod.textproto`](config/prod.textproto) and validated at build time by
[`src/proto/quadsmith/config/deploy.proto`](../src/proto/quadsmith/config/deploy.proto)
and `make build`:

- **`gcp_project_id`**: Target Google Cloud project ID.
- **`registry_region`**: GCP region hosting the `quadsmith-docker` Artifact
  Registry repository (`us-central1`).
- **`regions[]`**: Ordered list of regional deployments. The first entry
  (`regions[0]`) is the primary region hosting the writable Cloud SQL instance
  (`quadsmith-db-<env>`), while subsequent entries automatically provision
  in-region Cloud SQL cross-region read replicas
  (`quadsmith-db-<env>-<region>`).
  - **`db_tier`**: Cloud SQL machine tier for the region (`db-f1-micro` by
    default).
  - **`nodes[]`**: Ordered list of stateless Compute Engine VMs (`zone` +
    `machine_type`) running the Docker Compose application stack in that region.
- **`db_backup_retention_count`**: Number of pre-migration `pg_dump -Fc` backups
  retained on the primary VM before rotating oldest files (default: `10`).

Service hostnames (`quadsmith.com`, `api.quadsmith.com`, `ops.quadsmith.com`,
and their `staging-*` counterparts) are derived automatically from the existing
component configs in `src/go/*/config/{staging,prod}.textproto` so domain names
are never duplicated.

---

## 4. Running Deployments (`make deploy` & CI/CD)

### Automated GitHub Actions Pipeline

Pushing a version tag (`v*`) or triggering the **Deploy** workflow manually via
`workflow_dispatch` runs
[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) across three
sequential stages:

1. **`build-and-push`**: Compiles and pushes `server-web`, `server-api`,
   `server-ops`, and `server-jobs` container images in parallel to Google
   Artifact Registry.
2. **`migrate-db`**: Ensures all GCP, Cloud SQL, and Cloudflare resources exist,
   runs the automated breaking-change gate, captures a pre-migration database
   backup when DDL changes are present, applies `psqldef` schema migrations, and
   seeds the `server-jobs` internal token.
3. **`deploy-vm`**: Syncs `/opt/quadsmith/.env` and [`deploy/`](.) to each
   regional VM over IAP SSH and executes
   [`deploy_blue_green.sh`](deploy_blue_green.sh) for a zero-downtime blue-green
   cutover.

### Running From an Operator Workstation

With `gcloud`, `docker`, `psql`, `psqldef`, `jq`, and `socat` installed and the
required environment variables exported, run:

```bash
# Full convergent deployment to staging or prod
make deploy ENV=staging
make deploy ENV=prod

# Run only a specific stage (build | migrate | rollout)
make deploy ENV=prod STAGE=migrate
make deploy ENV=prod STAGE=rollout
```

---

## 5. Database Migrations, Pre-Migration Backups & Rollbacks

During the `migrate` stage, [`deploy.sh`](deploy.sh) opens an encrypted IAP TCP
tunnel through the primary VM to the Cloud SQL Private IP and runs
`psqldef --dry-run` for both `quadsmith_api` and `quadsmith_ops`:

1. **Zero-DDL Fast Path**: If `psqldef --dry-run` outputs no DDL statements,
   backups and migrations are skipped cleanly.
2. **Automated Breaking-Change Gate (`make check-breaking`)**: When DDL changes
   are detected, [`deploy.sh`](deploy.sh) exports the live schemas via
   `psqldef --export` and runs `make check-breaking` (`STRICT_BREAKING=1`). If a
   dropped table, dropped column, incompatible type change, or un-defaulted
   `NOT NULL` column is detected, the deployment aborts before touching Cloud
   SQL.
   - To intentionally apply a breaking schema change, trigger
     `.github/workflows/deploy.yml` via `workflow_dispatch` with
     `allow_breaking_changes: true` (or pass `--allow-breaking-changes=true` to
     `deploy/deploy.sh`).
3. **Automated Pre-Migration Snapshot (`pg_dump -Fc`)**: Before applying any
   DDL, [`deploy.sh`](deploy.sh) streams a compressed `pg_dump -Fc` snapshot of
   both `quadsmith_api` and `quadsmith_ops` to `/opt/quadsmith/backups/` on the
   primary VM (alongside Cloud SQL's managed daily backups and PITR).
4. **Restoring a Pre-Migration Snapshot**:

```bash
gcloud compute ssh quadsmith-prod-us-central1-a-1 \
  --tunnel-through-iap \
  --project=quadsmith-prod \
  --zone=us-central1-a \
  --command='
    source /opt/quadsmith/.env
    DB_HOST="$(echo "$QS_DATABASE_URL" | sed -E "s|.*@([^:/]+).*|\1|")"
    PGPASSWORD="$QS_DB_PASSWORD" pg_restore --clean --if-exists --no-owner \
      -h "$DB_HOST" -U postgres -d quadsmith_api \
      /opt/quadsmith/backups/quadsmith_api_<timestamp>.dump
  '
```

---

## 6. Adding New Geographic Regions

When Cloudflare Analytics shows sustained traffic (>15–20% of monthly sessions
or >5,000 MAU) in a continent not yet served by a local compute region (such as
Europe or Asia-Pacific) and `server-api` p95 latency in that geography exceeds
150 ms:

1. Append a new `regions` entry to
   [`config/prod.textproto`](config/prod.textproto):

   ```textproto
   regions {
     region: "europe-west1"
     db_tier: "db-f1-micro"
     nodes {
       zone: "europe-west1-b"
       machine_type: "e2-micro"
     }
   }
   ```

2. Run `make build` to validate the updated configuration with `check-config`.
3. Commit and run `make deploy ENV=prod`. [`deploy.sh`](deploy.sh) will
   automatically:
   - Provision the regional subnet and Cloud NAT in `europe-west1`.
   - Provision the in-region Cloud SQL read replica
     (`quadsmith-db-prod-europe-west1`) replicating asynchronously from
     `quadsmith-db-prod`.
   - Provision the `quadsmith-prod-europe-west1-b-1` VM, join it to the shared
     Cloudflare Tunnel (`cloudflared` automatically serves regional users from
     the nearest healthy connector), wire `QS_READ_DATABASE_URL` to the local
     read replica IP and `QS_DATABASE_URL` to the primary write IP, and roll out
     the application containers.

---

## 7. Scaling a Region Vertically or Horizontally

When a specific region experiences resource saturation (VM memory >80% or swap
thrashing, VM CPU >70% with depleted burst credits, or Cloud SQL CPU >70% / disk
I/O saturation), evaluate **vertical scaling** (stepping up `machine_type` or
`db_tier` in place) against **horizontal scaling** (adding VM nodes to `nodes[]`
in that region):

- **Memory Pressure (`e2-micro` 1 GB RAM >80% or Cloud SQL `db-f1-micro` 0.6 GB
  RAM buffer-cache misses)**: Prefer **vertical scaling** (`e2-micro` &rarr;
  `e2-small` 2 GB RAM or `e2-medium` 4 GB RAM; `db-f1-micro` &rarr;
  `db-g1-small` 1.7 GB RAM). Updating `machine_type` or `db_tier` in
  [`config/prod.textproto`](config/prod.textproto) and running
  `make deploy ENV=prod` resizes the target resource in place.
- **CPU Saturation & High Availability Within a Region**: Prefer **horizontal
  scaling** by appending a second `nodes` entry (e.g., in zone `us-central1-b`)
  to that region's `nodes[]` list in
  [`config/prod.textproto`](config/prod.textproto) and running
  `make deploy ENV=prod`. [`deploy.sh`](deploy.sh) provisions the additional VM
  (`quadsmith-prod-us-central1-b-2`), registers it on the shared Cloudflare
  Tunnel, and executes sequential blue-green rollouts across each node so the
  region maintains 100% uptime even during VM maintenance or zone events.

---

## 8. Day-2 Operations & Troubleshooting (`qsops` & IAP SSH)

### Connecting to a Regional VM via IAP SSH

Because VMs have no public IPv4 addresses, always connect using
`--tunnel-through-iap`:

```bash
gcloud compute ssh quadsmith-prod-us-central1-a-1 \
  --tunnel-through-iap \
  --project=quadsmith-prod \
  --zone=us-central1-a
```

### Inspecting Container Status & Logs on a VM

```bash
cd /opt/quadsmith/deploy
cat .active_slot
docker compose -f docker-compose.blue-green.yaml ps
docker compose -f docker-compose.blue-green.yaml logs --tail=100 -f
```

### Authenticating with the Operations CLI (`qsops`)

Build the production operations CLI (`make build` outputs `bin/qsops-prod`) and
authenticate via Google OAuth:

```bash
./bin/qsops-prod auth login
./bin/qsops-prod token list
```
