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
  - **100% Stateless Compute VMs**: Zero-public-IP Linux VMs on Google Cloud
    Compute Engine (starting with the Always Free `e2-micro` in
    `us-central1-a`), provisioned with a 2 GB `/swapfile`, `journald` 100 MB log
    cap, and Docker `json-file` log rotation. VMs hold zero persistent data or
    backups and can be replaced or scaled horizontally at any time.
  - **Managed Database & Native Backups**: **GCP Cloud SQL for PostgreSQL**
    (Enterprise Edition, starting on `db-f1-micro` with 10 GB SSD,
    `max_connections=100`, daily automated backups, 7-day 1-second Point-in-Time
    Recovery, pre-migration on-demand snapshots, and `--no-assign-ip` Private
    VPC IP only) hosting both the `quadsmith_api` and `quadsmith_ops` logical
    databases.

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
  (`quadsmith-db-<env>-<region>`), while subsequent entries automatically
  provision in-region Cloud SQL cross-region read replicas.
  - **`db_tier`**: Cloud SQL machine tier for the region (`db-f1-micro` by
    default).
  - **`nodes[]`**: Ordered list of stateless Compute Engine VMs (`zone` +
    `machine_type`) running the Docker Compose application stack in that region.
- **`db_backup_retention_count`**: Number of Cloud SQL `ON_DEMAND` pre-migration
  snapshots retained (`5` in `prod`, `3` in `staging`) before automatically
  pruning older `ON_DEMAND` snapshots (in addition to Cloud SQL's 7 free daily
  automated backups and 7-day continuous PITR).

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
   Artifact Registry (`:latest` and `:sha-<commit>`).
2. **`migrate-db`**: Ensures all GCP, Cloud SQL, and Cloudflare resources exist,
   runs the automated breaking-change gate, creates a Cloud SQL `ON_DEMAND`
   pre-migration snapshot when DDL changes are present, applies `psqldef` schema
   migrations, and seeds the `server-jobs` internal token.
3. **`deploy-vm`**: Syncs `/opt/quadsmith/.env` and [`deploy/`](.) to each
   regional VM over IAP SSH and executes
   [`deploy_blue_green.sh`](deploy_blue_green.sh) for a zero-downtime blue-green
   cutover.

### Running From an Operator Workstation

With `gcloud`, `docker`, `psql`, `psqldef`, `jq`, and `socat` installed and the
required environment variables exported, run:

```bash
# Full convergent deployment to staging or prod (build + migrate + rollout)
make deploy ENV=staging
make deploy ENV=prod

# Run only a specific stage (build | migrate | rollout)
make deploy ENV=prod STAGE=build
make deploy ENV=prod STAGE=migrate
make deploy ENV=prod STAGE=rollout
```

---

## 5. Backup Schedule, Database Migrations & Rollback Runbook

### Three-Tier Cloud SQL Backup Schedule

All database backups are stored natively in **GCP Cloud SQL** as incremental
block-level snapshots and Write-Ahead Log (WAL) streams. Zero backup files are
stored on any Compute Engine VM, keeping all VMs 100% stateless:

| Backup Tier                                     | Schedule / Trigger                                                                                                                  | Retention Window                                                                         | Recovery Granularity                          | Purpose                                                                         |
| :---------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------- | :-------------------------------------------- | :------------------------------------------------------------------------------ |
| **1. Daily Automated Backups**                  | Automatically every day at **`04:00 UTC`**                                                                                          | **7 daily snapshots** _(included free with Cloud SQL)_                                   | 24-hour snapshots                             | Baseline disaster recovery                                                      |
| **2. Continuous Point-in-Time Recovery (PITR)** | Continuous streaming of PostgreSQL WAL logs (`--enable-point-in-time-recovery`)                                                     | Rolling **7 days**                                                                       | **Any exact second** (`YYYY-MM-DDTHH:MM:SSZ`) | Recovering to the exact second before an accidental write, delete, or migration |
| **3. Pre-Migration On-Demand Snapshots**        | Triggered automatically by [`deploy.sh`](deploy.sh) right before `make db-migrate` whenever `psqldef --dry-run` detects DDL changes | Most recent **`db_backup_retention_count`** migrations (`5` in `prod`, `3` in `staging`) | Exact pre-migration state                     | Named pre-migration restore points that **never expire after 7 days**           |

### Automated Database Migration Gate

During the `migrate` stage, [`deploy.sh`](deploy.sh) opens an encrypted IAP TCP
tunnel through the primary VM to the Cloud SQL Private IP and runs
`psqldef --dry-run` for both `quadsmith_api` and `quadsmith_ops`:

1. **Zero-DDL Fast Path**: If `psqldef --dry-run` outputs no DDL statements,
   pre-migration snapshot creation and schema migration are skipped cleanly.
2. **Automated Breaking-Change Gate (`make check-breaking`)**: When DDL changes
   are detected, [`deploy.sh`](deploy.sh) exports the live schemas via
   `psqldef --export` and runs `make check-breaking` (`STRICT_BREAKING=1`). If a
   dropped table, dropped column, incompatible type change, or un-defaulted
   `NOT NULL` column is detected, the deployment aborts before touching Cloud
   SQL.
   - **Approving an Intentional Breaking Change**: Trigger
     [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) via
     `workflow_dispatch` with `allow_breaking_changes: true` (or run
     `make deploy ENV=prod ALLOW_BREAKING_CHANGES=true`).
3. **Pre-Migration Cloud SQL Snapshot & PITR Timestamp Log**: Before applying
   DDL, [`deploy.sh`](deploy.sh) prints the exact UTC timestamp
   (`Pre-migration PITR restore timestamp: YYYY-MM-DDTHH:MM:SSZ`), creates a
   Cloud SQL `ON_DEMAND` snapshot (`gcloud sql backups create`), and prunes
   older `ON_DEMAND` snapshots beyond `db_backup_retention_count`.

### Complete Rollback Runbook

#### Scenario A: Rolling Back Application Code / Containers (No DB Restore Needed)

Because normal schema changes are additive and backwards-compatible, the
previous container release continues to run cleanly against the current database
schema:

- **Automatic Rollback During Cutover**: If any candidate container in the new
  slot (`blue` or `green`) fails `/healthz` within 15 seconds,
  [`deploy_blue_green.sh`](deploy_blue_green.sh) automatically aborts, dumps the
  failing container logs, tears down the candidate slot, and leaves Nginx
  routing 100% of traffic to the previous slot.
- **Manual Rollback After Cutover**: Re-run the `rollout` stage pinned to the
  previous commit's image tag:

  ```bash
  make deploy ENV=prod STAGE=rollout QS_IMAGE_TAG=sha-<previous_commit_sha>
  ```

#### Scenario B: Rolling Back an Additive Schema Change (No Data Lost)

If a migration added a table, column, or index that you want to remove without
losing existing data:

1. Revert the schema commit in git (`git revert <commit_sha>`).
2. Run `make deploy` with `ALLOW_BREAKING_CHANGES=true` so `psqldef` applies the
   reverse `DROP` statement inside a single PostgreSQL transaction:

   ```bash
   make deploy ENV=prod ALLOW_BREAKING_CHANGES=true
   ```

#### Scenario C: In-Place Restore from a Pre-Migration On-Demand or Daily Snapshot

To roll the entire Cloud SQL instance back in place to a pre-migration
`ON_DEMAND` snapshot (or one of the 7 daily `AUTOMATED` backups):

```bash
# 1. List available backups (shows ID, WINDOW_START_TIME, TYPE, and DESCRIPTION)
gcloud sql backups list \
  --instance=quadsmith-db-prod-us-central1 \
  --project=quadsmith-prod

# 2. Restore the selected BACKUP_ID in place
gcloud sql backups restore <BACKUP_ID> \
  --restore-instance=quadsmith-db-prod-us-central1 \
  --project=quadsmith-prod
```

#### Scenario D: Point-in-Time Recovery (PITR) to an Exact Second Within the Last 7 Days

If an accidental data modification or bad migration occurred and you want to
rewind to the exact second before it happened (using the
`Pre-migration PITR restore timestamp` logged by [`deploy.sh`](deploy.sh) or any
UTC timestamp within the last 7 days):

```bash
# 1. Clone the primary instance to the exact target second
gcloud sql instances clone quadsmith-db-prod-us-central1 \
  quadsmith-db-prod-recovery \
  --point-in-time="2026-10-18T00:12:34Z" \
  --project=quadsmith-prod
```

Once `quadsmith-db-prod-recovery` is ready, you can either:

1. **Surgically restore a single table or database** (preserving all unaffected
   data written after the incident) by piping `pg_dump` from the recovery
   clone's Private IP into `quadsmith-db-prod-us-central1`, and then deleting
   the temporary clone:

   ```bash
   gcloud sql instances delete quadsmith-db-prod-recovery --project=quadsmith-prod
   ```

2. **Take a snapshot of the recovery clone and restore it onto the primary**:

   ```bash
   gcloud sql backups create --instance=quadsmith-db-prod-recovery --project=quadsmith-prod
   gcloud sql backups restore <RECOVERY_BACKUP_ID> \
     --restore-instance=quadsmith-db-prod-us-central1 \
     --project=quadsmith-prod
   gcloud sql instances delete quadsmith-db-prod-recovery --project=quadsmith-prod
   ```

---

## 6. Adding New Geographic Regions

See
[`../plans/adding_deployment_regions.md`](../plans/adding_deployment_regions.md)
for the complete multi-region expansion architecture, telemetry thresholds, and
per-user regional data residency design (`users.home_region` + PostgreSQL 16+
Logical Replication).

When Cloudflare Analytics shows sustained traffic ($\ge 15\%$ of dynamic
requests or $\ge 10,000$ MAU) in a continent not yet served by a local compute
region (such as Europe or Asia-Pacific) and edge-to-origin latency in that
geography exceeds 120 ms P50:

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

2. Run `make verify` to validate the updated configuration with `check-config`.
3. Commit and run `make deploy ENV=prod`. [`deploy.sh`](deploy.sh) will
   automatically:
   - Provision the regional Cloud SQL instance
     (`quadsmith-db-prod-europe-west1`) over the global GCP VPC
     (`--no-assign-ip`).
   - Provision the `quadsmith-prod-europe-west1-b-1` VM, join it to the shared
     Cloudflare Tunnel (`cloudflared` automatically routes regional users to the
     nearest healthy connector), configure the regional database URLs, and roll
     out the application containers.

---

## 7. Scaling a Region Vertically or Horizontally

See
[`../plans/scaling_deployment_regions.md`](../plans/scaling_deployment_regions.md)
for the full per-region vertical vs. horizontal compute, Cloud SQL, and
Cloudflare R2 blob storage scaling comparison and workflows.

When a specific region experiences resource saturation (VM memory >75% or swap
activity, VM CPU >70% with depleted burst credits, or Cloud SQL CPU >70% /
memory >80%), evaluate **vertical scaling** (stepping up `machine_type` or
`db_tier` in place) against **horizontal scaling** (adding VM nodes to `nodes[]`
in that region):

- **Memory Pressure (`e2-micro` 1 GB RAM >75% or Cloud SQL `db-f1-micro` 0.6 GB
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
