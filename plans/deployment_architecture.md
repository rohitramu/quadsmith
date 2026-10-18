# <a id="initial-deployment-architecture" >Initial Deployment Architecture, Provisioning, Migration Gate, and Blue-Green Rollout</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Top-Level `deploy/` Directory and Per-Region `DeployConfig` (`deploy/config/*.textproto`)](#top-level-deploy-directory-and-per-region-deployconfig-deploy-config-textproto)
  - [Cloud Infrastructure and Zero-IP Cloudflare Tunnel Ingress](#cloud-infrastructure-and-zero-ip-cloudflare-tunnel-ingress)
  - [GCP Cloud SQL for PostgreSQL over Private VPC Peering and SQL Provider Comparison](#gcp-cloud-sql-for-postgresql-over-private-vpc-peering-and-sql-provider-comparison)
  - [Ultra-Lean Nginx Reverse Proxy and Subdomain Routing](#ultra-lean-nginx-reverse-proxy-and-subdomain-routing)
  - [Portable RFC HTTP Caching for Static Assets, WASM, and Blobs](#portable-rfc-http-caching-for-static-assets-wasm-and-blobs)
  - [Stateless Container Topology and Host Memory Hygiene](#stateless-container-topology-and-host-memory-hygiene)
  - [Single Convergent `make deploy` Lifecycle and Provisioning Matrix](#single-convergent-make-deploy-lifecycle-and-provisioning-matrix)
  - [SQL Database Setup, Pre-Migration Backups, and Automated Migration Gate](#sql-database-setup-pre-migration-backups-and-automated-migration-gate)
  - [Zero-Downtime Blue-Green Container Cutover and Automatic Rollback](#zero-downtime-blue-green-container-cutover-and-automatic-rollback)
  - [Colocated Artifact Registry, GitHub Actions CD, and Operator Guide (`deploy/README.md`)](#colocated-artifact-registry-github-actions-cd-and-operator-guide-deploy-readme-md)
  - [Single-Region Capacity and Cost Analysis](#single-region-capacity-and-cost-analysis)
- [4. Alternatives considered](#4-alternatives-considered)
  - [Host Stateless Compute Layer on AWS or Azure Instead of GCP](#host-stateless-compute-layer-on-aws-or-azure-instead-of-gcp)
  - [Fly.io Independent Micro-VM Deployment](#fly-io-independent-micro-vm-deployment)
  - [Direct Cloud Run and Cloud Scheduler Deployment on Day One](#direct-cloud-run-and-cloud-scheduler-deployment-on-day-one)
  - [Separate One-Time Bootstrap Target (`make env-bootstrap`) Distinct from `make deploy`](#separate-one-time-bootstrap-target-make-env-bootstrap-distinct-from-make-deploy)
  - [Declarative Infrastructure-as-Code with Terraform or OpenTofu](#declarative-infrastructure-as-code-with-terraform-or-opentofu)
  - [Custom Go Infrastructure Provisioner in `qsops` (GCP and Cloudflare Go SDKs)](#custom-go-infrastructure-provisioner-in-qsops-gcp-and-cloudflare-go-sdks)
  - [Manual Web Console Setup (ClickOps)](#manual-web-console-setup-clickops)
  - [Sequential Numbered Migration Files (`golang-migrate` / `Goose`)](#sequential-numbered-migration-files-golang-migrate-goose)
  - [Execute Migrations Inside Service Container Entrypoints](#execute-migrations-inside-service-container-entrypoints)
  - [Docker Swarm Mode Rolling Updates (`docker stack deploy`)](#docker-swarm-mode-rolling-updates)
- [5. Assumptions](#5-assumptions)

---

## <a id="1-background" >1. Background</a>

Quadsmith's service architecture was aligned by language into
[`src/go`](../src/go) and [`src/react`](../src/react) in
[`infrastructure_alignment_and_container_architecture.md`](infrastructure_alignment_and_container_architecture.md).
Subsequently, background workers and operations maintenance were refactored in
[`tasks_and_jobs.md`](tasks_and_jobs.md) to establish clean boundaries:

1. **Tasks & Jobs** (5 scheduled maintenance procedures: `CleanupExpiredOtps`
   every 6h, `CleanupExpiredTokens` every 12h, `CleanupOrphanedBlobs` every 10m,
   `PurgeSoftDeletedAccounts` every 8h, and `SendUsernameReleaseWarnings` every
   24h—with SQL-engine maintenance delegated natively to PostgreSQL
   `autovacuum`—plus the manual `SetUserRole` task, alongside `ListTasks`,
   `GetTask`, `ListJobs`, and `GetJob`) are owned and served by
   [`server-ops`](../src/go/server-ops) under `OpsService`, protected by
   [`opsauth.NewOpsInterceptor`](../src/go/shared/pkg/opsauth/interceptor.go).
2. **Headless Scheduler** ([`server-jobs`](../src/go/server-jobs)) operates
   purely as a headless background cron daemon with zero HTTP APIs or database
   connections, running typed background schedulers via `robfig/cron/v3` that
   trigger the 5 maintenance RPCs on `server-ops`.
3. **Public APIs & Blobs** are served by [`server-api`](../src/go/server-api)
   via ConnectRPC alongside plain HTTP media endpoints (`/api/blobs`,
   `/blobs/{id}`) documented in
   [`blob_storage_and_streaming_architecture.md`](blob_storage_and_streaming_architecture.md).
4. **Web Frontend & Social Previews** are served by
   [`server-web`](../src/go/server-web), which serves the React 19 SPA, the 17.5
   MB WebAssembly physics engine
   ([`quadsmith-engine.wasm`](../src/react/app-web/public/quadsmith-engine.wasm)),
   and dynamically injects Open Graph / Twitter Card metadata for social media
   crawlers via [`static.go`](../src/go/shared/pkg/static/static.go).

Across all services and CLIs, configuration is standardized on **three
environments** (`sandbox`, `staging`, and `prod`, with the redundant `test`
environment consolidated into `sandbox`), validated at build time by
[`src/go/shared/cmd/check-config/main.go`](../src/go/shared/cmd/check-config/main.go).
Previously, `test/sandbox/docker-compose.yaml` and
`src/deploy/docker-compose.prod.yaml` maintained separate container topologies,
while [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml)
restarted containers in place without provisioning missing infrastructure,
validating or applying `psqldef` database migrations, or performing
zero-downtime blue-green cutovers.

This plan—together with its two companion operational plans,
[`adding_deployment_regions.md`](adding_deployment_regions.md) (for adding new
geographic regions when telemetry shows underserved regional traffic) and
[`scaling_deployment_regions.md`](scaling_deployment_regions.md) (for scaling
individual regions vertically or horizontally based on per-region
telemetry)—supersedes
[`e2_micro_deployment_and_evolution.md`](archive/e2_micro_deployment_and_evolution.md).
It establishes a top-level `deploy/` directory, typed per-region deployment
infrastructure configurations (`deploy/config/{staging,prod}.textproto`), a
single convergent `make deploy` command (handling idempotent GCP/CF/VM/Cloud SQL
bootstrap, pre-migration Cloud SQL `ON_DEMAND` snapshots, `make check-breaking`,
and `make db-migrate`), zero-downtime blue-green container cutover, and a
colocated operator runbook in `deploy/README.md`.

---

## <a id="2-goal" >2. Goal</a>

Deploy Quadsmith's four stateless Go services (`server-web`, `server-api`,
`server-ops`, `server-jobs`) to a Google Cloud Platform Always Free `e2-micro`
VM ($0.00/mo compute) running Docker Compose with zero public IP and zero open
inbound firewall ports via Cloudflare Tunnel, backed by **GCP Cloud SQL for
PostgreSQL** (`db-f1-micro`, ~$10.50/mo) over a Private Services Access internal
VPC IP (`--no-assign-ip`, <0.8 ms latency, $0.00/GB internal bandwidth),
orchestrated from a top-level `deploy/` directory via a single convergent
`make deploy` command (`deploy/config/{staging,prod}.textproto`) with automated
SQL backup/migration gating, zero-downtime blue-green container cutover, and
operator documentation in `deploy/README.md`.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Organize stack orchestration in top-level `deploy/` with per-region
>   `DeployConfig` textprotos (`deploy/config/{staging,prod}.textproto`) and
>   `deploy/README.md`.
> - Deploy 4 stateless Go services + Nginx (~4 MB RAM) on 1 GCP `e2-micro` VM
>   ($0/mo) behind Cloudflare Tunnel with zero public IP.
> - Pair with Private-IP GCP Cloud SQL (`db-f1-micro`, ~$10.50/mo) for <0.8 ms
>   SQL RTT and $0/GB internal VPC bandwidth.
> - Unify idempotent GCP/CF/VM/Cloud SQL bootstrap, pre-migration Cloud SQL
>   snapshots, `make check-breaking`, and `make db-migrate` under `make deploy`.
> - Perform zero-downtime blue-green container cutovers (`nginx -s reload`)
>   gated on `/healthz` with automatic rollback.

This proposal defines Quadsmith's baseline production architecture leveraging
GCP's Always Free `e2-micro` compute tier, **GCP Cloud SQL for PostgreSQL** over
Private VPC Peering, and Cloudflare's Anycast edge while preserving portability
across cloud and CDN providers. By delegating PostgreSQL persistence to GCP
Cloud SQL over an internal Private Services Access IP, all four services on the
VM remain 100% stateless ("cattle, not pets"), enjoy sub-millisecond database
latency (`0.3 – 0.8 ms` RTT), and incur **$0.00/GB** in database network
transfer (completely exempt from GCP's 1 GB/month internet egress limit).
Cloudflare Tunnel (`cloudflared`) connects outbound to Cloudflare, eliminating
public IPv4 addresses on both the VM and Cloud SQL. An ultra-lean Nginx gateway
routes subdomains to the active blue/green container slot, while standard HTTP
`Cache-Control` headers allow Cloudflare to absorb 99% of static asset, WASM,
and media blob traffic at the edge.

```mermaid
flowchart TD
    CF["Cloudflare Edge (RFC Cache-Control Respect Mode: quadsmith.com, api, ops)"]

    subgraph VPC["Global GCP VPC (us-central1 | Private Services Access 10.x.x.x — Zero Public IPs)"]
        subgraph VM["GCP e2-micro VM (us-central1-a, 2 vCPU burstable, 1 GB RAM, 30 GB pd-standard Disk)"]
            Tunnel["Standalone cloudflared Container (--network host)"]

            subgraph Compose["Top-Level deploy/ Stack (docker-compose.yaml / docker-compose.blue-green.yaml)"]
                Nginx["Nginx Gateway (127.0.0.1:8084 -> :80, upstreams.conf)"]
                Web["server-web (:8080, active slot)"]
                API["server-api (:8080, active slot)"]
                Ops["server-ops (:8080, active slot)"]
                Jobs["server-jobs (Headless Cron, active slot)"]

                Nginx -->|Host: quadsmith.com| Web
                Nginx -->|Host: api.quadsmith.com| API
                Nginx -->|Host: ops.quadsmith.com| Ops
                Web -->|OG RPC| API
                Jobs -->|Task RPC| Ops
            end

            Tunnel -->|http://127.0.0.1:8084| Nginx
        end

        DB[("GCP Cloud SQL for PostgreSQL (db-f1-micro, --no-assign-ip)<br>Private VPC IP 10.x.1.x | 10 GB SSD | PITR Enabled<br>Logical DBs: quadsmith_api & quadsmith_ops")]
    end

    CF -->|Outbound QUIC/HTTP2 Tunnel| Tunnel
    API -->|"Private VPC Queries (<0.8ms RTT, $0/GB)"| DB
    Ops -->|"Private VPC Queries (<0.8ms RTT, $0/GB)"| DB
```

### <a id="top-level-deploy-directory-and-per-region-deployconfig-deploy-config-textproto" >Top-Level `deploy/` Directory and Per-Region `DeployConfig` (`deploy/config/*.textproto`)</a>

In
[`infrastructure_alignment_and_container_architecture.md`](infrastructure_alignment_and_container_architecture.md),
[`src/`](../src) was organized strictly by language and schema ecosystem
(`src/db/`, `src/go/`, `src/proto/`, `src/react/`), with each Go service keeping
its own `Dockerfile` and runtime `config/*.textproto` colocated under
`src/go/<service>/`. Multi-container Compose orchestration and cloud deployment
scripts are repository-level operational concerns—peers to [`plans/`](../plans),
[`src/`](../src), [`test/`](../test), and [`tools/`](../tools)—and are therefore
housed in top-level `deploy/`:

```text
quadsmith/
├── deploy/                            # Cloud deployment & Compose stack orchestration
│   ├── config/
│   │   ├── staging.textproto          # Validated live staging topology
│   │   └── prod.textproto             # Validated live production topology
│   ├── nginx/
│   │   ├── nginx.conf                 # Subdomain reverse-proxy gateway config
│   │   └── upstreams.conf             # Active blue/green slot upstreams
│   ├── README.md                      # Operator deployment, region & scaling guide
│   ├── deploy.sh                      # Convergent bootstrap, DB backup/migrate & rollout
│   ├── deploy_blue_green.sh           # Zero-downtime blue-green slot cutover on VM
│   ├── docker-compose.yaml            # Unified profile-free Compose stack
│   └── docker-compose.blue-green.yaml # Dual-slot blue-green Compose stack
├── plans/                             # Architectural design plans & archive/
├── src/                               # Language/schema trees (db, go, proto, react)
├── test/                              # Sandbox harness, integration & Playwright tests
└── tools/                             # Linters & verification scripts
```

To support independent **per-region** vertical and horizontal scaling alongside
multi-region expansion while maintaining 100% consistency with
[`src/proto/quadsmith/config/server_api.proto`](../src/proto/quadsmith/config/server_api.proto)
and
[`src/go/shared/cmd/check-config/main.go`](../src/go/shared/cmd/check-config/main.go),
live deployment infrastructure is modeled in
`src/proto/quadsmith/config/deploy.proto` and validated by `check-config` during
`make build` and `make verify` (enforcing `ValidateNoUnsetFields` on
`deploy/config/{staging,prod}.textproto`, while omitting `sandbox` since
`sandbox` is an ephemeral local Docker harness without cloud infrastructure or
persisted data):

```protobuf
// src/proto/quadsmith/config/deploy.proto
syntax = "proto3";

package quadsmith.config;

option go_package = "quadsmith/shared/gen/quadsmith/config;config";

// ---------------------------------------------------------
// Domain model
// ---------------------------------------------------------

message DeployConfig {
  optional string environment = 1;
  optional string gcp_project_id = 2;
  optional string registry_region = 3;
  repeated RegionConfig regions = 4;
  optional int32 db_backup_retention_count = 5;
}

message RegionConfig {
  optional string region = 1;
  optional string db_tier = 2;
  repeated VmNodeConfig nodes = 3;
}

message VmNodeConfig {
  optional string zone = 1;
  optional string machine_type = 2;
}
```

Architectural invariants and values already declared in component configs are
not duplicated in `DeployConfig`:

- **Per-Service Hostname Mapping (Single Source of Truth)**: Rather than
  repeating hostnames or service mappings in `DeployConfig`, `make deploy` reads
  each server binary's public hostname directly from its existing environment
  `.textproto` config:
  - **`server-web` (`web:8080`)**: host of `auth.oauth.web_base_url` in
    [`src/go/server-api/config/{staging,prod}.textproto`](../src/go/server-api/config/prod.textproto#L56)
    (`staging.quadsmith.com` / `quadsmith.com` + `www.quadsmith.com`).
  - **`server-api` (`api:8080`)**: host of `api_url` in
    [`src/go/server-web/config/{staging,prod}.textproto`](../src/go/server-web/config/prod.textproto#L18)
    (`staging-api.quadsmith.com` / `api.quadsmith.com`).
  - **`server-ops` (`ops:8080`)**: host of `ops_server_url` in
    [`src/go/cli-ops/config/{staging,prod}.textproto`](../src/go/cli-ops/config/prod.textproto#L2)
    (`staging-ops.quadsmith.com` / `ops.quadsmith.com`).
  - **`server-jobs`**: headless background cron daemon (no inbound HTTP port or
    hostname). All hostnames for the environment route through the environment's
    Cloudflare Tunnel (`quadsmith-${ENV}`) to the local Nginx gateway
    (`127.0.0.1:8084`), where `deploy/nginx/nginx.conf` dispatches each `Host`
    header to `web_upstream`, `api_upstream`, or `ops_upstream`.
- **Host, Cloud SQL & Runtime Constants**: Debian 12 OS image, 30 GB
  `pd-standard` boot disk, 2 GB swapfile, Google's fixed IAP TCP forwarding
  range (`35.235.240.0/20`), deterministic VM naming
  (`quadsmith-${ENV}-${zone}-${idx}`), deterministic Cloud SQL naming
  (`quadsmith-db-${ENV}-${region}` for each region's writable Cloud SQL primary
  in `regions[]`), `/opt/quadsmith` remote directory, Cloudflare Smart Tiered
  Cache + RFC `Cache-Control` compliance, and the SQL schema paths
  (`src/db/quadsmith_api.sql` and `src/db/quadsmith_ops.sql`) live as constants
  in `deploy/deploy.sh`.

Each live environment (`staging` and `prod`) declares its regional compute and
database topology concisely:

```textproto
# deploy/config/prod.textproto
environment: "prod"
gcp_project_id: "quadsmith-prod"
registry_region: "us-central1"

regions {
  region: "us-central1"
  db_tier: "db-f1-micro"
  nodes {
    zone: "us-central1-a"
    machine_type: "e2-micro"
  }
}

db_backup_retention_count: 5
```

This nested `regions` &rarr; `nodes` structure directly supports all three
evolution paths without schema changes:

1. **Adding a New Region** (detailed in
   [`adding_deployment_regions.md`](adding_deployment_regions.md)): Append a new
   `regions { ... }` block (e.g., `region: "europe-west1"`).
2. **Scaling a Specific Region Vertically** (detailed in
   [`scaling_deployment_regions.md`](scaling_deployment_regions.md)): Update
   `machine_type` (e.g., `"e2-small"`) or `db_tier` (e.g., `"db-g1-small"`)
   inside that specific region's block while leaving other regions untouched.
3. **Scaling a Specific Region Horizontally** (detailed in
   [`scaling_deployment_regions.md`](scaling_deployment_regions.md)): Append an
   additional `nodes { zone: "us-central1-b" ... }` block inside that region's
   `regions` entry so multiple VMs in the region share the regional Cloud SQL
   instance and Cloudflare Tunnel.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-src-deploy-directory" >Keep `deploy/` Nested Under `src/deploy/`</a>

Leave `docker-compose.yaml`, `nginx/nginx.conf`, `deploy.sh`, and
`config/{staging,prod}.textproto` inside `src/deploy/`.

**Pros:**

- Keeps all `.textproto` files nested somewhere under `src/`.

**Cons:**

- Violates the language/schema taxonomy of `src/` (`src/db/`, `src/go/`,
  `src/proto/`, `src/react/`), since multi-container stack orchestration and
  cloud provisioning are top-level repository operations (peer to `test/`,
  `tools/`, and `plans/`) rather than application source code.
- Requires `context: ../..` in Docker Compose files instead of `context: ..`.

##### <a id="alt-flat-nodes-list-without-regionconfig" >Flat `nodes[]` List with a Global `db_tier` Field</a>

Use a flat `repeated VmNodeConfig nodes` list on `DeployConfig` with a single
top-level `db_tier` string rather than grouping nodes under `RegionConfig`.

**Pros:**

- One fewer message type in `deploy.proto`.

**Cons:**

- Cannot express independent per-region Cloud SQL tiers (e.g., scaling the US
  primary to `db-g1-small` while keeping a newer EU read replica on
  `db-f1-micro`), and obscures which VMs share a regional Cloud SQL instance
  when scaling horizontally within a region.

##### <a id="alt-untyped-env-files-for-deploy" >Untyped `.env.staging` and `.env.prod` Shell Variable Files</a>

Store deployment regions, zones, and VM sizes in flat `deploy/staging.env` and
`deploy/prod.env` shell files instead of Protobuf `.textproto` files.

**Pros:**

- Can be sourced directly by bash via `source deploy/prod.env`.

**Cons:**

- Breaks consistency with the rest of Quadsmith, where every component uses
  Protobuf-validated `.textproto` files.
- Cannot represent structured lists (`regions` and `nodes`) and bypasses
  `check-config` compile-time validation.

</details>

### <a id="cloud-infrastructure-and-zero-ip-cloudflare-tunnel-ingress" >Cloud Infrastructure and Zero-IP Cloudflare Tunnel Ingress</a>

The initial host VM is provisioned under the GCP Compute Engine Always Free
tier, which grants **744 instance-hours per month** (1 concurrent 24/7 VM) per
billing account perpetually:

- **Instance Type**: `e2-micro` (2 shared vCPUs burstable, 1 GB RAM).
- **Region**: `us-central1` (Iowa), `us-east1` (South Carolina), or `us-west1`
  (Oregon). Colocating in `us-central1` aligns with Artifact Registry and the
  primary Cloud SQL instance.
- **Boot Disk**: 30 GB `pd-standard` (Standard Persistent Disk), which is 100%
  covered by the 30 GB/month Always Free tier allowance.
- **External IP**: **None** (`--no-address`). The VM is provisioned with zero
  public IPv4 address (Cloud NAT provides outbound internet access for pulling
  `cloudflare/cloudflared` and reaching Cloudflare's edge).
- **Inbound Firewall**: Only IAP SSH forwarding (`35.235.240.0/20` on port 22)
  is permitted for deployment management; all public ingress firewall rules are
  disabled.

In live environments (`staging` and `prod`), a lightweight `cloudflared`
container is provisioned once during VM bootstrap as a standalone,
auto-restarting container outside the application Docker Compose stack:

```bash
docker run -d \
  --name cloudflared \
  --restart unless-stopped \
  --network host \
  cloudflare/cloudflared:latest \
  tunnel --no-autoupdate run --token "${QS_CLOUDFLARE_TUNNEL_TOKEN}"
```

It establishes redundant, outbound-only QUIC/HTTP2 tunnels to Cloudflare's
global Anycast edge. In Cloudflare DNS, `CNAME` records for `@`, `www`, `api`,
and `ops` (or `staging`, `staging-api`, and `staging-ops`) point to
`<tunnel-uuid>.cfargotunnel.com`, routing all traffic down the tunnel to
`http://127.0.0.1:8084` (where the `nginx` container binds on the VM's loopback
interface).

Running `cloudflared` as a standalone container outside `docker-compose.yaml`
delivers three critical advantages:

1. **$0.00 Public IP Cost Guarantee**: Eliminates Google's $0.005/hour
   ($3.65/month) in-use external IPv4 address fee on every VM.
2. **Zero Inbound Attack Surface**: The VM cannot be port-scanned,
   DDoS-attacked, or directly probed from the public internet.
3. **Zero-Profile Compose Parity & Persistent Tunnel Uptime**: Because
   `cloudflared` is decoupled from the application Compose file, `sandbox` and
   live environments share the exact same profile-free `docker-compose.yaml`,
   and application container updates never restart or drop active QUIC tunnel
   connections to Cloudflare's edge.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-public-static-ipv4" >Public Static IPv4 with GCP Firewall Rules</a>

Allocate a GCP external static IPv4 address, point Cloudflare DNS `A` records to
it, and restrict GCP ingress firewall rules to Cloudflare's published IP ranges.

**Pros:**

- Standard DNS architecture allowing direct IP diagnostics if needed.
- Does not require running the `cloudflared` daemon container.

**Cons:**

- Costs ~$3.65/month per VM due to Google's mandatory in-use IPv4 billing.
- Exposes an internet-reachable network interface susceptible to firewall rule
  misconfiguration.

##### <a id="alt-direct-cloudflared-ingress" >Direct cloudflared Ingress Without Nginx</a>

Configure `cloudflared`'s ingress rules to route each hostname directly to its
respective container (`api.quadsmith.com` &rarr; `http://127.0.0.1:8081`,
`quadsmith.com` &rarr; `http://127.0.0.1:8080`).

**Pros:**

- Eliminates the Nginx container entirely, saving ~4 MB of RAM.
- Simplifies Compose file by having one fewer service.

**Cons:**

- Couples internal routing strictly to Cloudflare's proprietary tunnel schema
  and prevents atomic single-node blue-green upstream reloads.
- Precludes local offline testing of subdomain routing or unified reverse proxy
  access logging on the VM without an active Cloudflare Tunnel.

</details>

### <a id="gcp-cloud-sql-for-postgresql-over-private-vpc-peering-and-sql-provider-comparison" >GCP Cloud SQL for PostgreSQL over Private VPC Peering and SQL Provider Comparison</a>

Quadsmith standardizes on **GCP Cloud SQL for PostgreSQL** (Enterprise Edition,
starting on `db-f1-micro` with 10 GB SSD in `us-central1`) connected to the
global GCP VPC via **Private Services Access**
(`servicenetworking.googleapis.com`) with **`--no-assign-ip`** (Private IP
only):

- **Sub-Millisecond Latency (`0.3 – 0.8 ms` RTT)**: Because the `e2-micro` VM
  and Cloud SQL instance reside in the same GCP region (`us-central1`) and
  communicate directly over their internal `10.x.x.x` VPC addresses, every SQL
  query executes in under a millisecond.
- **$0.00/GB Internal Bandwidth (Exempt from GCP's 1 GB Internet Egress Cap)**:
  Same-region VPC traffic between Compute Engine and Cloud SQL is completely
  free and unmetered. By contrast, connecting from a GCP VM to an external
  AWS-hosted database (such as Neon or Supabase) sends every SQL query,
  `pg_dump` backup, and user-uploaded `BYTEA` media blob across the public
  internet—consuming GCP's 1 GB/month free internet egress allowance ($0.12/GB
  thereafter).
- **Zero Public Database Attack Surface**: Configured with `--no-assign-ip`, the
  Cloud SQL instance has no public IP address and cannot be reached from the
  internet.
- **Single Instance Hosting Multiple Logical Databases**: A single
  `quadsmith-db-prod` Cloud SQL instance hosts both `quadsmith_api` and
  `quadsmith_ops` logical databases (and can optionally host `staging`'s logical
  databases or run a separate `quadsmith-db-staging` instance).
- **Automated Daily Backups & Point-in-Time Recovery (PITR)**: Enabled at
  creation (`--backup-start-time=04:00 --enable-point-in-time-recovery`), which
  both protects production data and serves as the prerequisite for spinning up
  cross-region Cloud SQL read replicas in
  [`adding_deployment_regions.md`](adding_deployment_regions.md).
- **Connection Limit Tuning (`max_connections=100`)**: Cloud SQL's default
  `max_connections` on `db-f1-micro` (25) is raised to `100` via
  `--database-flags=max_connections=100` so primary and read replica instances
  comfortably accommodate blue-green cutovers and multi-region write pools.

To document the engineering trade-offs across managed PostgreSQL providers and
cloud compute hosts evaluated during design:

| Provider & Tier                                              | Underlying Cloud & Region                                       | Compute, RAM & Max Conns                                             | Cold Start / Idle Behavior                                                  | Latency (RTT) to `server-api` VM on GCP                                                      | Bandwidth / Egress to `server-api`                                                  | Storage & Initial `BYTEA` Blob Fit                                       | Monthly Cost (`staging` + `prod`)                                                       |
| :----------------------------------------------------------- | :-------------------------------------------------------------- | :------------------------------------------------------------------- | :-------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------- | :---------------------------------------------------------------------------------- | :----------------------------------------------------------------------- | :-------------------------------------------------------------------------------------- |
| **GCP Cloud SQL**<br>_(Enterprise `db-f1-micro` — Selected)_ | **GCP Native**<br>(`us-central1`, `europe-west1`, `asia-east1`) | Shared vCPU, **0.6 GB RAM**<br>_(100 max conns via flag)_            | **0 ms** _(Always-on VM)_                                                   | **0.3 – 0.8 ms**<br>_(Same-VPC Private IP)_                                                  | **Unlimited & $0.00/GB**<br>_(Exempt from GCP's 1 GB internet egress limit)_        | **10 GB SSD** included in base<br>_(Fits ~80+ max-size 100 MB videos)_   | **~$10.50 / mo**<br>_(1 shared instance, or ~$21/mo for 2 separate instances)_          |
| **Neon**<br>_(Free Tier)_                                    | **AWS Only**<br>_(Azure regions deprecated Apr 2026)_           | Autoscales **0.25–2 CU**<br>(**1–8 GB RAM**)<br>_(10k pooled conns)_ | **300 – 800 ms**<br>_(Suspends after 5m idle; capped at **100 CU-hrs/mo**)_ | **15 – 22 ms** _(Iowa &harr; Ohio)_<br>**2 – 4 ms** _(Oregon `us-west1` &harr; `us-west-2`)_ | **5 GB/mo free**<br>_(Cross-cloud traffic counts against GCP 1 GB internet egress)_ | **1 GB total** across project<br>_(Fits only ~8 max-size 100 MB videos)_ | **$0.00 / mo**<br>_(10-min cron + `min_connections: 1` exhausts 100 CU-hrs in 16 days)_ |
| **Neon**<br>_(Launch Pay-As-You-Go)_                         | **AWS Only**                                                    | Autoscales **0.25–16 CU**<br>(**1–64 GB RAM**)                       | **0 ms** _(always-on)_ or **300–800 ms** _(scale-to-zero)_                  | **2 – 4 ms** _(Oregon)_<br>**15 – 22 ms** _(Iowa &harr; Ohio)_                               | Included in compute; $0.09/GB overage                                               | **$0.35/GB/mo** _(2x Cloud SQL SSD price)_                               | **~$19.35 / mo** _(0.25 CU 24/7)_ + $0.35/GB                                            |
| **Supabase**<br>_(Free / Pro Tier)_                          | **AWS Only**                                                    | Shared ARM vCPU, **0.5 GB RAM**<br>_(60 direct / 200 pooled)_        | **0 ms** _(Runs 24/7; pauses after 7d inactivity)_                          | **2 – 4 ms** _(Oregon)_<br>**15 – 22 ms** _(Iowa &harr; Ohio)_                               | **5 GB/mo free** raw DB egress _(Cross-cloud counts against GCP 1 GB egress)_       | **500 MB** _(Free)_<br>**8 GB** _($25/mo Pro)_                           | **$0.00 / mo** _(Free, no backups)_<br>**$25.00 / mo** _(Pro)_                          |
| **Aiven**<br>_(Free / $5 Developer)_                         | **AWS / DO** _(Free/Dev)_<br>**GCP `us-central1`** _(Startup+)_ | **1 vCPU, 1 GB RAM**<br>_(Hard cap: **20 max conns**)_               | **0 ms** _(Always-on on $5 Dev tier)_                                       | **10 – 25 ms** _(Free/Dev)_<br>**1 – 2 ms** _(Startup $19+ on GCP)_                          | Public IP over TLS _(Counts against GCP 1 GB egress on Free/Dev)_                   | **1 GB** _(Free)_<br>**8 GB** _($5/mo Dev)_                              | **$0.00 / mo** _(Free)_<br>**$5.00 / mo** _(Dev 8 GB)_                                  |
| **Xata / Crunchy Bridge**<br>_(Entry Paid Tiers)_            | **GCP Native**<br>(`us-central1`)                               | **1–2 vCPU, 1–2 GB RAM**                                             | **0 ms** _(Always-on VM)_                                                   | **1 – 2 ms**<br>_(Same GCP region)_                                                          | Public/Peered TLS                                                                   | **$0.10–$0.28/GB/mo**                                                    | **~$9 / mo** _(Xata Micro)_<br>**$35 / mo** _(Crunchy Hobby-2)_                         |

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-neon-serverless-postgres" >Neon Serverless Postgres (Free / Launch Tier on AWS)</a>

Use Neon's serverless PostgreSQL (hosted on AWS) over public TLS instead of GCP
Cloud SQL.

**Pros:**

- $0.00/month Free Tier with up to 2 CU (8 GB RAM) burst compute and built-in
  PgBouncer connection pooling.

**Cons:**

- **CU-Hour Quota Exhaustion**: Neon Free Tier caps active compute at 100
  CU-hours/month (400 hours/month at minimum 0.25 CU). Because `server-jobs`
  runs `cleanup_orphaned_blobs` every 10 minutes and Neon stays awake for 5
  minutes after each query (and `pgxpool` `min_connections: 1` sends 1-minute
  keepalives), a 0.25 CU instance stays awake 365–730 hours/month (91–182.5
  CU-hours), exhausting the free quota mid-month or costing ~$19.35/month on the
  Launch tier (nearly 2x Cloud SQL `db-f1-micro`).
- **Cold Starts & Cross-Cloud Egress**: Scaling to zero incurs 300–800 ms cold
  starts, and because Neon runs on AWS, every SQL query and `BYTEA` blob upload
  from the GCP `e2-micro` VM traverses the public internet and counts against
  GCP's 1 GB/month free internet egress limit.

##### <a id="alt-supabase-managed-postgres" >Supabase Managed Postgres (Free / Pro Tier on AWS)</a>

Use Supabase's AWS-hosted PostgreSQL service over public TLS instead of GCP
Cloud SQL.

**Pros:**

- Always-on nano instance at $0.00/month for up to 2 projects (`staging` and
  `prod`) with zero cold starts as long as `server-jobs` runs at least weekly.

**Cons:**

- **500 MB Storage Ceiling**: The 500 MB Free Tier disk limit can hold only ~4
  max-size (100 MB) flight video blobs in `quadsmith_api.blobs`, and stepping up
  to Supabase Pro jumps immediately to $25/month (+$20/month per read replica).
- **No Automated Free-Tier Backups & Cross-Cloud Egress**: Lacks automated PITR
  backups on the Free Tier and traverses the public internet between GCP and
  AWS, consuming GCP's 1 GB/month free internet egress quota.

##### <a id="alt-aiven-developer-postgres" >Aiven for PostgreSQL ($0 Free / $5 Developer Tier)</a>

Use Aiven's $5/month Developer tier (1 vCPU, 1 GB RAM, 8 GB storage) instead of
GCP Cloud SQL.

**Pros:**

- Low $5/month flat price for an always-on 1 GB RAM, 8 GB storage instance.

**Cons:**

- Hard cap of 20 maximum connections and no custom GCP region or Private VPC
  peering selection until stepping up to Aiven's $19–$75+/month tiers, meaning
  all database traffic traverses public internet IPs with 10–25 ms RTT.

</details>

### <a id="ultra-lean-nginx-reverse-proxy-and-subdomain-routing" >Ultra-Lean Nginx Reverse Proxy and Subdomain Routing</a>

A dedicated, Alpine-based Nginx container (`nginx:alpine`) acts as the internal
ingress gateway on port 80 within the Docker Compose network (bound to
`127.0.0.1:${NGINX_PORT:-8084}:80` on the host loopback interface).

To ensure all three HTTP-serving Go services (`server-web`, `server-api`, and
`server-ops`; `server-jobs` is a headless cron client) listen on a uniform
internal port (`8080`) behind Nginx and redirect OAuth flows to the canonical
domain,
[`src/go/server-web/config/prod.textproto`](../src/go/server-web/config/prod.textproto)
and
[`src/go/server-api/config/prod.textproto`](../src/go/server-api/config/prod.textproto)
(along with their `staging.textproto` counterparts) are updated from `port: 443`
to `port: 8080`, matching
[`server-ops`](../src/go/server-ops/config/prod.textproto), and `server-api`'s
`oauth.web_base_url`, `from_address`, and `reply_to` fields are aligned from
`quadsmith.net` to `quadsmith.com`.

Furthermore, three critical proxy and security behaviors are configured across
`nginx.conf` and `server-web`:

1. **Single-Hop `X-Forwarded-For` Preservation**:
   [`src/go/shared/pkg/auth/service.go`](../src/go/shared/pkg/auth/service.go)
   resolves the client IP using `trusted_proxy_hops: 1` in
   [`src/go/server-api/config/prod.textproto`](../src/go/server-api/config/prod.textproto).
   Because Nginx is only reachable internally from `cloudflared` on `127.0.0.1`,
   Nginx passes `proxy_set_header X-Forwarded-For $http_x_forwarded_for;` rather
   than `$proxy_add_x_forwarded_for`, avoiding appending `127.0.0.1` and keeping
   `trusted_proxy_hops: 1` accurate across both Docker Compose and Cloud Run.
2. **Unbuffered 100 MB Blob Uploads**:
   [`server-api`](../src/go/server-api/main.go) accepts up to 100 MB on
   `/api/blobs` (matching Cloudflare Free Tier's 100 MB upload limit). Nginx's
   default 1 MB limit is raised via `client_max_body_size 100m;` with
   `proxy_request_buffering off;` so large flight video uploads stream directly
   to `server-api` without buffering into VM memory.
3. **Media Blob Routing on Both Web & API Hosts + Content-Security-Policy**:
   Because `<img src="/blobs/...">` and `<video src="/blobs/...">` elements
   rendered by the React SPA resolve relative to `quadsmith.com` (or
   `http://127.0.0.1:8084` in local gateway tests), the `quadsmith.com`
   (`default_server`) block in `nginx.conf` proxies `location /blobs/` and
   `location /api/blobs` directly to `http://api_upstream`, while
   [`ComponentFormModal.tsx`](../src/react/app-web/src/components/ComponentFormModal.tsx)
   is updated to use `getApiBaseUrl()` and `getStoredAccessToken()` (matching
   [`BuildPhotoModal.tsx`](../src/react/app-web/src/components/BuildPhotoModal.tsx)
   and
   [`AccountSettingsModal.tsx`](../src/react/app-web/src/components/AccountSettingsModal.tsx)).
   Simultaneously, the `Content-Security-Policy` header in
   [`src/go/server-web/main.go`](../src/go/server-web/main.go) is updated to
   include
   `img-src 'self' data: blob: https: http://localhost:* http://127.0.0.1:*;`
   and `media-src 'self' blob: https: http://localhost:* http://127.0.0.1:*;` so
   browser `URL.createObjectURL(file)` previews and media streams are never
   blocked by CSP.

```nginx
# deploy/nginx/nginx.conf
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    sendfile on;
    keepalive_timeout 65;

    include /etc/nginx/conf.d/upstreams.conf;

    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $http_cf_connecting_ip;
    proxy_set_header X-Forwarded-For $http_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;

    # Web SPA & Dynamic Open Graph Server (Default Server)
    server {
        listen 80 default_server;
        server_name quadsmith.com www.quadsmith.com staging.quadsmith.com _;
        client_max_body_size 10m;

        location = /healthz {
            access_log off;
            return 200 "OK\n";
        }

        location /api/blobs {
            client_max_body_size 100m;
            proxy_request_buffering off;
            proxy_pass http://api_upstream;
        }

        location /blobs/ {
            proxy_pass http://api_upstream;
        }

        location / {
            proxy_pass http://web_upstream;
        }
    }

    # Public ConnectRPC API & Plain HTTP Blob Streaming
    server {
        listen 80;
        server_name api.quadsmith.com staging-api.quadsmith.com;
        client_max_body_size 100m;

        location /api/blobs {
            proxy_request_buffering off;
            proxy_pass http://api_upstream;
        }

        location / {
            proxy_pass http://api_upstream;
        }
    }

    # Internal Ops ConnectRPC API
    server {
        listen 80;
        server_name ops.quadsmith.com staging-ops.quadsmith.com;
        client_max_body_size 2m;

        location / {
            proxy_pass http://ops_upstream;
        }
    }
}
```

Nginx was selected over Caddy because its compiled C runtime consumes only **~3
MB – 5 MB RAM** at idle compared to Caddy's ~20 MB Go runtime.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-caddy-gateway" >Caddy Reverse Proxy Gateway</a>

Use Caddy in Docker Compose instead of Nginx.

**Pros:**

- Concise Caddyfile configuration syntax.
- Shares the same Go ecosystem as Quadsmith's backend services.

**Cons:**

- Consumes ~15 MB – 25 MB RAM (5x more memory than Nginx).
- Caddy's primary advantage—automatic Let's Encrypt TLS termination—is redundant
  because Cloudflare Tunnel terminates TLS at the edge.

</details>

### <a id="portable-rfc-http-caching-for-static-assets-wasm-and-blobs" >Portable RFC HTTP Caching for Static Assets, WASM, and Blobs</a>

To keep the architecture 100% portable across any CDN (Cloudflare, AWS
CloudFront, Fastly, Google Cloud CDN) while staying within GCP's **1 GB/month
free network egress limit**, all caching policies are driven strictly by
standard RFC 9111 `Cache-Control` headers emitted by our Go binaries:

1. **Hashed Vite Bundles (`/assets/*` in
   [`static.go`](../src/go/shared/pkg/static/static.go))**:
   - Vite compiles JS and CSS bundles with content hashes in their filenames
     (e.g., `/assets/index-D8f3a1b2.js`).
   - [`ServeSPA()`](../src/go/shared/pkg/static/static.go) sets
     `Cache-Control: public, max-age=31536000, immutable` on all requests under
     `/assets/`.
2. **Versioned 17.5 MB WASM Engine (`/quadsmith-engine.wasm?v=<hash>`,
   `/wasm_exec.js?v=<hash>`) & Public Assets (`/og-default.png`)**:
   - [`src/react/app-web/public/quadsmith-engine.wasm`](../src/react/app-web/public/quadsmith-engine.wasm)
     is **17.5 MB**—by far the largest asset in the SPA (30x larger than all JS
     and CSS bundles combined). Without edge caching, just 58 page loads would
     exhaust GCP's 1 GB/month free egress.
   - **Out-of-the-Box Cloudflare Trap**: Cloudflare's default cached extension
     allowlist excludes `.wasm` files, marking them `CF-Cache-Status: DYNAMIC`
     and pulling all 17.5 MB from the origin VM on every visit unless a Cache
     Rule is configured. With the single blanket Cache Rule below, Cloudflare
     caches `.wasm` across all 300+ global PoPs (well within Cloudflare Free
     Tier's **512 MB per-file cache limit**) and automatically applies **Brotli
     compression** to `application/wasm` responses (~3.5–4.0 MB over the wire).
   - **Size & Atomic Cache-Busting Optimization (`?v=<sha256>`)**:
     - Update [`Makefile`](../Makefile) and
       [`src/go/server-web/Dockerfile`](../src/go/server-web/Dockerfile) to
       compile `quadsmith-engine.wasm` with `-ldflags="-w -s"`, stripping DWARF
       debug symbols and reducing the raw binary size by ~30% (~17.5 MB &rarr;
       ~12 MB) across both local and CI Docker builds.
     - Reorder [`src/go/server-web/Dockerfile`](../src/go/server-web/Dockerfile)
       so the Go `builder` stage compiles `quadsmith-engine.wasm` and copies
       `wasm_exec.js` before the `web-builder` stage runs `npm run build`
       (matching [`Makefile`](../Makefile)).
     - Update
       [`src/react/app-web/vite.config.ts`](../src/react/app-web/vite.config.ts)
       to compute the 16-character SHA-256 content hash of
       `public/quadsmith-engine.wasm` + `public/wasm_exec.js` at build time and
       inject `__WASM_HASH__`, which
       [`src/react/app-web/src/lib/wasmEngine.ts`](../src/react/app-web/src/lib/wasmEngine.ts)
       appends as `/wasm_exec.js?v=<hash>` and
       `/quadsmith-engine.wasm?v=<hash>`. This guarantees **zero version skew**
       between the React JS bundle and WASM binary immediately after a
       deployment.
     - Update [`ServeSPA()`](../src/go/shared/pkg/static/static.go) so static
       files requested with a `?v=` content-hash query parameter receive
       `Cache-Control: public, max-age=31536000, immutable` (allowing 1 full
       year of immutable edge caching for WASM), while unversioned static files
       (e.g., `/og-default.png`) emit
       `Cache-Control: public, max-age=3600, stale-while-revalidate=86400`
       alongside precomputed SHA-256 `ETag` headers.
3. **SPA HTML & Dynamic Open Graph Cards (`index.html`)**:
   - [`ServeSPA()`](../src/go/shared/pkg/static/static.go) emits
     `Cache-Control: no-cache` on rendered `index.html` responses so CDNs and
     browsers never cache crawler-injected Open Graph metadata across routes or
     retain stale asset references after deployments.
4. **Content-Addressed Media Blobs (`/blobs/{id}` in
   [`blob/handler.go`](../src/go/shared/pkg/blob/handler.go))**:
   - Already emits `Cache-Control: public, max-age=31536000, immutable` and
     `ETag: "<blob_id>"`.
5. **Single Standard-Compliance Rule & Smart Tiered Cache on Cloudflare**:
   - Unlike CloudFront or Fastly, Cloudflare's default proxy mode only caches a
     hardcoded list of file extensions and skips `.wasm` and extensionless paths
     like `/blobs/blob_<id>`.
   - `make deploy` configures **one blanket Cloudflare Cache Rule** for the zone
     (`All incoming requests` &rarr; `Eligible for cache` &rarr;
     `Edge TTL: Use cache-control header if present, bypass cache if not`) and
     enables **Cloudflare Smart Tiered Cache** (free 1-click API setting).
     Tiered Cache designates Upper-Tier PoPs near `us-central1` as the cache
     concentrator for all 300+ global edge PoPs, so the 12 MB WASM binary and
     media blobs are pulled from the `e2-micro` VM only 1–3 times per deployment
     worldwide.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-cloudflare-pages-htmlrewriter" >Cloudflare Pages with Edge Functions (HTMLRewriter)</a>

Host the React SPA on Cloudflare Pages and rewrite `static.go`'s social crawler
Open Graph injection in TypeScript using Cloudflare Workers' `HTMLRewriter` API.

**Pros:**

- Retires `server-web` completely, saving ~25 MB of RAM on the VM.
- Serves both HTML and static assets from Cloudflare's edge.

**Cons:**

- High vendor lock-in: `HTMLRewriter` is proprietary to Cloudflare's V8 Workers
  runtime and cannot run on AWS, GCP, Fastly, or standard containers.
- Splits application logic across Go and TypeScript with two separate deployment
  pipelines.

##### <a id="alt-split-hosting-user-agent-routing" >Split Static Hosting with CDN User-Agent Routing</a>

Host static assets on Cloudflare Pages or S3 and configure CDN conditional
routing rules to proxy requests with social bot `User-Agent` headers to
`server-web` on the VM.

**Pros:**

- Avoids porting Go Open Graph logic to TypeScript.
- Offloads browser HTML requests to static storage.

**Cons:**

- Introduces "split-brain" deployment race conditions between the static host
  and backend container.
- Couples routing architecture to CDN-specific `User-Agent` regex rules that are
  difficult to test locally.

</details>

### <a id="stateless-container-topology-and-host-memory-hygiene" >Stateless Container Topology and Host Memory Hygiene</a>

Rather than maintaining separate Docker Compose manifests for local sandbox and
live environments, both `test/sandbox/docker-compose.yaml` and
`src/deploy/docker-compose.prod.yaml` are replaced by
`deploy/docker-compose.yaml` (with a companion dual-slot
`deploy/docker-compose.blue-green.yaml` for zero-downtime cutovers on live VMs):

- **100% Symmetric External-DB & Edge Boundary**:
  - In `sandbox`, [`test/start_sandbox.sh`](../test/start_sandbox.sh) and
    [`test/run_sandbox_tests.sh`](../test/run_sandbox_tests.sh) start the seeded
    PostgreSQL database as a standalone container (`quadsmith-sandbox-db`) on
    `127.0.0.1:${DB_PORT:-5432}` before starting the Compose stack.
  - In `staging` and `prod`, `cloudflared` runs as a standalone host container
    forwarding to `127.0.0.1:${NGINX_PORT:-8084}`, while `api` and `ops` connect
    to the Private VPC IP of the GCP Cloud SQL instance (`jobs` is a pure
    stateless RPC client calling `http://ops:8080`).
- **Zero Profiles Needed**: Because all five maintenance tasks in
  [`src/go/shared/pkg/auth/job.go`](../src/go/shared/pkg/auth/job.go) and
  [`src/go/shared/pkg/blob/cleanup.go`](../src/go/shared/pkg/blob/cleanup.go)
  are idempotent (with `SendUsernameReleaseWarnings` using transactional
  `FOR UPDATE OF hh SKIP LOCKED`), `jobs` runs unconditionally on every node,
  and all five Compose services (`nginx`, `api`, `ops`, `jobs`, `web`) start in
  every environment without any `profiles:` configuration.
- **Clean `QS_CONFIG` Environment Selection Across the 3 Environments**:
  - Each service's Dockerfile (`src/go/server-{api,ops,jobs,web}/Dockerfile`)
    sets `ENV QS_CONFIG=/app/src/go/server-<name>/config/prod.textproto` and
    `ENTRYPOINT ["/app/server-<name>"]` (removing the hardcoded `--config` CLI
    flag so `QS_CONFIG` is respected natively).
  - Locally, `docker compose -f deploy/docker-compose.yaml up --build` builds
    images from source and defaults `QS_CONFIG` via `${QS_ENV:-sandbox}` to
    `sandbox.textproto` and `host.docker.internal:${DB_PORT:-5432}`.
  - In `staging` and `prod`, setting `QS_ENV=staging` or `QS_ENV=prod` selects
    `staging.textproto` or `prod.textproto` and pulls pre-built images from
    Google Artifact Registry.

```yaml
# deploy/docker-compose.yaml
services:
  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "127.0.0.1:${NGINX_PORT:-8084}:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/upstreams.conf:/etc/nginx/conf.d/upstreams.conf:ro
    depends_on:
      web:
        condition: service_healthy
      api:
        condition: service_healthy
      ops:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O - http://127.0.0.1:80/healthz || exit 1"]
      interval: 100ms
      timeout: 1s
      retries: 50
      start_period: 200ms

  api:
    image: ${QS_IMAGE_REGISTRY:-quadsmith}/server-api:${QS_IMAGE_TAG:-latest}
    build:
      context: ..
      dockerfile: src/go/server-api/Dockerfile
    restart: unless-stopped
    ports:
      - "127.0.0.1:${API_PORT:-8081}:8080"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    environment:
      - QS_CONFIG=/app/src/go/server-api/config/${QS_ENV:-sandbox}.textproto
      - QS_DATABASE_URL=${QS_DATABASE_URL:-postgres://postgres:postgres@host.docker.internal:${DB_PORT:-5432}/quadsmith_api?sslmode=disable}
      - QS_READ_DATABASE_URL=${QS_READ_DATABASE_URL:-}
      - QS_WEB_BASE_URL=${QS_WEB_BASE_URL:-http://localhost:${PORT:-8080}}
      - QS_AUTH_SECRET=${QS_AUTH_SECRET:-}
      - QS_SMTP_PASSWORD=${QS_SMTP_PASSWORD:-}
      - QS_GOOGLE_CLIENT_ID=${QS_GOOGLE_CLIENT_ID:-}
      - QS_GOOGLE_CLIENT_SECRET=${QS_GOOGLE_CLIENT_SECRET:-}
      - QS_GITHUB_CLIENT_ID=${QS_GITHUB_CLIENT_ID:-}
      - QS_GITHUB_CLIENT_SECRET=${QS_GITHUB_CLIENT_SECRET:-}
      - QS_DISCORD_CLIENT_ID=${QS_DISCORD_CLIENT_ID:-}
      - QS_DISCORD_CLIENT_SECRET=${QS_DISCORD_CLIENT_SECRET:-}
    healthcheck:
      test:
        ["CMD-SHELL", "wget -q -O - http://127.0.0.1:8080/healthz || exit 1"]
      interval: 100ms
      timeout: 1s
      retries: 50
      start_period: 200ms

  ops:
    image: ${QS_IMAGE_REGISTRY:-quadsmith}/server-ops:${QS_IMAGE_TAG:-latest}
    build:
      context: ..
      dockerfile: src/go/server-ops/Dockerfile
    restart: unless-stopped
    ports:
      - "127.0.0.1:${OPS_PORT:-8082}:8080"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    environment:
      - QS_CONFIG=/app/src/go/server-ops/config/${QS_ENV:-sandbox}.textproto
      - QS_DATABASE_URL=${QS_DATABASE_URL:-postgres://postgres:postgres@host.docker.internal:${DB_PORT:-5432}/quadsmith_api?sslmode=disable}
      - QS_INTERNAL_DATABASE_URL=${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@host.docker.internal:${DB_PORT:-5432}/quadsmith_ops?sslmode=disable}
      - QS_OAUTH_CLIENT_ID=${QS_OAUTH_CLIENT_ID:-}
      - QS_OAUTH_CLIENT_SECRET=${QS_OAUTH_CLIENT_SECRET:-}
    healthcheck:
      test:
        ["CMD-SHELL", "wget -q -O - http://127.0.0.1:8080/healthz || exit 1"]
      interval: 100ms
      timeout: 1s
      retries: 50
      start_period: 200ms

  jobs:
    image: ${QS_IMAGE_REGISTRY:-quadsmith}/server-jobs:${QS_IMAGE_TAG:-latest}
    build:
      context: ..
      dockerfile: src/go/server-jobs/Dockerfile
    restart: unless-stopped
    environment:
      - QS_CONFIG=/app/src/go/server-jobs/config/${QS_ENV:-sandbox}.textproto
      - QS_OPS_SERVER_URL=http://ops:8080
      - QS_JOB_SERVER_AUTH_TOKEN=${QS_JOB_SERVER_AUTH_TOKEN:-}
    depends_on:
      ops:
        condition: service_healthy

  web:
    image: ${QS_IMAGE_REGISTRY:-quadsmith}/server-web:${QS_IMAGE_TAG:-latest}
    build:
      context: ..
      dockerfile: src/go/server-web/Dockerfile
    restart: unless-stopped
    ports:
      - "127.0.0.1:${PORT:-8080}:8080"
    environment:
      - QS_CONFIG=/app/src/go/server-web/config/${QS_ENV:-sandbox}.textproto
      - QS_API_URL=http://api:8080
    depends_on:
      api:
        condition: service_healthy
    healthcheck:
      test:
        ["CMD-SHELL", "wget -q -O - http://127.0.0.1:8080/healthz || exit 1"]
      interval: 100ms
      timeout: 1s
      retries: 50
      start_period: 200ms
```

#### <a id="sandbox-helper-scripts-and-makefile-integration" >Sandbox Helper Scripts and Makefile Integration</a>

Because `db` now runs as a standalone container (`quadsmith-sandbox-db`) outside
`deploy/docker-compose.yaml`,
[`test/start_sandbox.sh`](../test/start_sandbox.sh) (invoked by `make sandbox`)
and [`test/run_sandbox_tests.sh`](../test/run_sandbox_tests.sh) (invoked by
`make test` and `make ci-test`) are updated to manage both the standalone
database container and the unified Compose stack:

1. **Cleanup Trap (`EXIT INT TERM`)**: Both scripts stop and remove
   `quadsmith-sandbox-db` (`docker rm -f quadsmith-sandbox-db`) alongside
   `docker compose -f deploy/docker-compose.yaml down -v`.
2. **Standalone Sandbox DB Startup & Readiness Check**: Both scripts build
   `quadsmith-sandbox-db` from
   [`test/sandbox/db/Dockerfile`](../test/sandbox/db/Dockerfile), launch it
   bound to `127.0.0.1:${DB_PORT:-5432}:5432`, and poll
   `docker exec quadsmith-sandbox-db pg_isready -U postgres -d quadsmith_api`
   until healthy before starting `deploy/docker-compose.yaml`.
3. **Gateway Readiness Verification**: `is_sandbox_ready` in
   `test/run_sandbox_tests.sh` verifies
   `http://127.0.0.1:${NGINX_PORT:-8084}/healthz` in addition to `web`, `api`,
   and `ops` (while `jobs` runs as a headless cron client).

#### <a id="memory-budget-and-host-hygiene" >Memory Budget & Host Hygiene</a>

The memory budget for the 1 GB VM is strictly disciplined:

| Container / Process          | Runtime / Language             | Expected Idle RAM | Peak RAM Under Load |
| :--------------------------- | :----------------------------- | :---------------- | :------------------ |
| `cloudflared` _(standalone)_ | Go (Cloudflare binary)         | ~15 MB            | ~25 MB              |
| `nginx`                      | C (Alpine)                     | ~4 MB             | ~10 MB              |
| `server-web`                 | Go (static asset & OG handler) | ~25 MB            | ~40 MB              |
| `server-api`                 | Go (ConnectRPC + pgxpool)      | ~25 MB            | ~45 MB              |
| `server-ops`                 | Go (ConnectRPC + pgxpool)      | ~20 MB            | ~35 MB              |
| `server-jobs`                | Go (headless cron client)      | ~10 MB            | ~20 MB              |
| Linux OS / systemd / Docker  | Linux Kernel / Dockerd         | ~150 MB           | ~200 MB             |
| **Total**                    |                                | **~249 MB**       | **~375 MB**         |

To guarantee host stability, prevent out-of-memory kernel panics, and stay
within Cloud SQL connection limits:

1. **Database Connection Pool Enforcement (`pgxpool`)**: Update
   [`src/go/server-api/main.go`](../src/go/server-api/main.go) to apply
   `max_connections` and `min_connections` via `pgxpool.ParseConfig` /
   `pgxpool.NewWithConfig`, matching
   [`server-ops`](../src/go/server-ops/main.go). Right-size both
   `prod.textproto` and `staging.textproto` connection pools for single-node
   deployment (`server-api`: `max_connections: 15, min_connections: 1`;
   `server-ops`: `max_connections: 5, min_connections: 1` on both `database` and
   `internal_database`), capping total VM connections at 20 on `quadsmith_api`
   and 5 on `quadsmith_ops` (with only 3 total idle connections, since
   `server-jobs` has zero database pools).
2. **Database-Backed Operations Token Seeding**: Rather than embedding static
   `bootstrap_tokens` in
   [`server-ops/config/prod.textproto`](../src/go/server-ops/config/prod.textproto)
   or
   [`server-ops/config/staging.textproto`](../src/go/server-ops/config/staging.textproto),
   initial `ROBOT` (`QS_JOB_SERVER_AUTH_TOKEN`, `group_name = 'robots'`) and
   operator `HUMAN` (`group_name = 'operators'`) token SHA-256 hashes are seeded
   directly into `quadsmith_ops.internal_tokens` via SQL during database setup,
   allowing every live token to be audited and revoked dynamically via
   `RevokeInternalToken`.
3. **Low-Overhead Background Task Schedules**: With `RunDatabaseMaintenance` and
   `SeedDatabase` removed in favor of PostgreSQL `autovacuum` and
   `cleanup_orphaned_blobs` scheduled every 10 minutes (`600s`), `server-jobs`
   executes only ~153 background task RPCs per day (~4,600/month across all 5
   tasks)—a 90% reduction in background database wakeups and
   `quadsmith_ops.internal_api_calls` audit log growth.
4. **2 GB Linux Swap File**: Provision a 2 GB swapfile on the boot disk
   (`fallocate -l 2G /swapfile && mkswap /swapfile && swapon /swapfile`) with
   `vm.swappiness=10`, providing a safety buffer for transient spikes.
5. **Docker JSON Log Rotation**: Configure Docker daemon-wide
   (`/etc/docker/daemon.json`) with `"log-driver": "json-file"`,
   `"log-opts": {"max-size": "10m", "max-file": "3"}` to prevent unpruned logs
   from consuming disk space.
6. **Daily Docker System Prune**: A daily cron job executes
   `docker image prune -af --filter "until=72h"` to automatically clean old
   images after deployments.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-separate-sandbox-and-prod-compose-files" >Separate Docker Compose Files for Sandbox and Production</a>

Maintain two distinct files (`test/sandbox/docker-compose.yaml` and
`deploy/docker-compose.prod.yaml`) instead of a single unified
`deploy/docker-compose.yaml`.

**Pros:**

- Keeps `db` inside `test/sandbox/docker-compose.yaml` without changing
  `test/start_sandbox.sh` or `test/run_sandbox_tests.sh`.

**Cons:**

- Duplicates `api`, `ops`, `jobs`, and `web` service definitions across two
  files that can drift out of sync.
- Does not exercise `nginx.conf` routing during local `make sandbox` or
  `make test` runs.

##### <a id="alt-compose-profiles-for-db-and-cloudflared" >Include `db` and `cloudflared` Inside `docker-compose.yaml` with Profiles</a>

Keep `db` (`profiles: ["sandbox"]`) and `cloudflared` (`profiles: ["live"]`)
inside `deploy/docker-compose.yaml` and toggle them via `COMPOSE_PROFILES`.

**Pros:**

- Consolidates every container definition into a single YAML file.

**Cons:**

- Requires passing `COMPOSE_PROFILES` environment flags in every command.
- Couples the `cloudflared` tunnel daemon lifecycle to application Compose
  restarts.

##### <a id="alt-host-database-on-vm" >Host PostgreSQL in Docker on the VM</a>

Include a `db` service running PostgreSQL in the production Compose stack on the
`e2-micro` VM, backed by a host volume mount.

**Pros:**

- Zero external database subscription or setup required.
- Sub-millisecond database queries over the local bridge network.

**Cons:**

- Consumes 150 MB – 250 MB of RAM, leaving negligible memory headroom on the
  `e2-micro`.
- Transforms the VM into a stateful pet requiring custom automated `pg_dump`
  cron scripts and disaster recovery backups.
- Host corruption or hypervisor reboot risks database corruption.

##### <a id="alt-config-bootstrap-tokens" >Static Bootstrap Tokens in Production Configs</a>

Define `bootstrap_tokens` entries templated with environment variables in
`server-ops/config/prod.textproto` and `server-ops/config/staging.textproto`.

**Pros:**

- Allows initial authentication before any rows are inserted into
  `quadsmith_ops.internal_tokens`.

**Cons:**

- Config-level bootstrap tokens bypass `internal_tokens` expiration and cannot
  be revoked dynamically via `RevokeInternalToken` without redeploying
  containers.

</details>

### <a id="single-convergent-make-deploy-lifecycle-and-provisioning-matrix" >Single Convergent `make deploy` Lifecycle and Provisioning Matrix</a>

To compare the candidate provisioning mechanisms across our engineering
criteria:

| Provisioning Approach                                                   | Config Source of Truth                                                       | External State Backend Needed                               | Third-Party SDK / Plugin Bloat                                  | Suitability for Quadsmith                                                     |
| :---------------------------------------------------------------------- | :--------------------------------------------------------------------------- | :---------------------------------------------------------- | :-------------------------------------------------------------- | :---------------------------------------------------------------------------- |
| **Convergent `make deploy` + `deploy/config/*.textproto`** _(Proposed)_ | **`deploy/config/{staging,prod}.textproto`** _(validated by `check-config`)_ | **None** _(queries live GCP/CF/PG APIs directly)_           | **Zero** _(uses `gcloud`, `curl`, `psql`, `psqldef`)_           | **Optimal** _(single command for 1st and 100th deploy)_                       |
| **Terraform / OpenTofu**                                                | Separate `.tf` / `.tfvars` HCL files                                         | **Yes** _(requires bootstrapping a GCS state bucket first)_ | **3 Providers** (`google`, `cloudflare`, `postgresql`)          | **Overkill** _(plaintext secrets in `tfstate`; chicken-and-egg bucket)_       |
| **Custom Go SDK CLI (`qsops infra`)**                                   | `.textproto`                                                                 | **None**                                                    | **Heavy** (`cloud.google.com/go`, `cloudflare-go` in `go.work`) | **Poor** _(pollutes application `go.work.sum` with cloud control-plane SDKs)_ |
| **Manual Web Console (ClickOps)**                                       | None _(ad-hoc console clicks)_                                               | **None**                                                    | **Zero**                                                        | **Poor** _(error-prone, undocumented drift across environments)_              |

```mermaid
flowchart TD
    Config["deploy/config/{staging,prod}.textproto (Validated by check-config)"]

    subgraph MakeDeploy["Single Convergent Command: make deploy ENV=staging|prod"]
        subgraph PhaseA["Phase A: Idempotent Infrastructure & Cloud SQL Bootstrap (Creates on 1st Run | ~3s No-Op Thereafter)"]
            GCPCheck["1. Ensure GAR Repo, IAP Firewall Rule & Global VPC Private Services Access Exist"]
            SQLCheck["2. Ensure Primary Cloud SQL Instance (regions[0]), Logical DBs (quadsmith_api, quadsmith_ops) & Regional Read Replicas (regions[1..N]) Exist"]
            CFCheck["3. Ensure Cloudflare Tunnel, Proxied CNAMEs, Tiered Cache & RFC Cache Rule Exist"]
            VMCheck["4. Ensure Regional Compute VMs (regions[].nodes[]) & Standalone cloudflared Exist"]
            GCPCheck --> SQLCheck --> CFCheck --> VMCheck
        end

        subgraph PhaseB["Phase B: SQL Backup, Migration Gate & Blue-Green Rollout (Runs on Every Deploy)"]
            Build["5. Build & Push 4 Service Images to GAR (:latest & :sha-<commit>)"]
            Diff{"6. psqldef --dry-run Detects Pending Schema Changes?"}
            Backup["7a. Run make check-breaking, Create Cloud SQL ON_DEMAND Pre-Migration Snapshot & Apply make db-migrate"]
            Seed["7b. Ensure Robot & Initial Operator Token SHA-256 Hashes in quadsmith_ops.internal_tokens"]
            BlueGreen["8. Sync .env (with Local Cloud SQL Private IPs) & deploy/ to Each VM Node -> Run deploy_blue_green.sh"]

            Build --> Diff
            Diff -->|Yes (DDL Diff Non-Empty)| Backup --> Seed
            Diff -->|No (Schema Up-to-Date)| Seed
            Seed --> BlueGreen
        end

        VMCheck --> Build
    end

    Config ==>|Loaded & Validated| GCPCheck
```

Folding the idempotent bootstrap checks—including **GCP Cloud SQL** instance and
cross-region replica creation—directly into `make deploy` (backed by
`deploy/deploy.sh` reading `deploy/config/${ENV}.textproto`) delivers three
major operational advantages:

1. **Zero Separate Bootstrap Command**: Bringing up `staging` or `prod` for the
   very first time uses the exact same `make deploy ENV=staging` or
   `make deploy ENV=prod` command that CI runs on every release.
2. **Automatic Self-Healing & Multi-Region / Per-Region Convergence**: If a VM
   is ever terminated, a region's `machine_type` or `db_tier` is scaled in
   `prod.textproto`, or a new `regions` block is appended, the next
   `make deploy` automatically converges the Cloud SQL instances, VMs, and
   containers in one pass.
3. **Zero State-File Secret Leakage**: `make deploy` queries live provider APIs
   and pipes runtime secrets directly into `/opt/quadsmith/.env` on each VM and
   `quadsmith_ops.internal_tokens` in Cloud SQL without persisting a state file.

We distinguish between **(A) Human Account Prerequisites** (done once manually
because cloud billing accounts, domain registrars, and OAuth provider portals
require interactive human verification) and **(B) Automated Idempotent
Infrastructure Checks** (driven by `deploy/config/${ENV}.textproto` inside
`make deploy`):

- **Part A: One-Time Human Account & Secret Prerequisites**:
  1. **Cloudflare Zone & Scoped API Token**: Add `quadsmith.com` to Cloudflare
     and create a scoped Cloudflare API Token (`CF_API_TOKEN`, `CF_ACCOUNT_ID`,
     `CF_ZONE_ID`) with permissions: `Account:Cloudflare Tunnel:Edit`,
     `Zone:DNS:Edit`, `Zone:Cache Rules:Edit`, and
     `Account:Access: Apps and Policies:Edit`.
  2. **GCP Project & Deployer Service Account**: Create the GCP project matching
     `gcp_project_id` in `deploy/config/${ENV}.textproto` linked to a billing
     account, enable Compute, Cloud SQL Admin (`sqladmin.googleapis.com`),
     Service Networking (`servicenetworking.googleapis.com`), Artifact Registry,
     and IAP APIs, and create the `quadsmith-cd` service account (`GCP_SA_KEY`).
  3. **OAuth, SMTP, DB & Auth Secrets**: Create OAuth 2.0 client IDs/secrets in
     Google Cloud Console, GitHub Developer Settings, and Discord Developer
     Portal, plus `QS_DB_PASSWORD` (`openssl rand -hex 24`), `QS_SMTP_PASSWORD`,
     `QS_AUTH_SECRET` (`openssl rand -hex 32`), and `QS_JOB_SERVER_AUTH_TOKEN`
     (`openssl rand -hex 32`).
- **Part B: Idempotent Bootstrap Functions in `deploy/deploy.sh`**:
  1. **`ensure_gcp_network_and_registry`**: Ensures the Artifact Registry
     repository (`quadsmith-docker` in `registry_region` with a 3-version
     cleanup policy), IAP firewall rule (`allow-ssh-ingress-from-iap` for
     `35.235.240.0/20`), Cloud NAT router (for outbound internet access from
     `--no-address` VMs), and global VPC Private Services Access IP range
     (`servicenetworking.googleapis.com` `/16` peering) exist in
     `gcp_project_id`.
  2. **`ensure_cloud_sql_topology`**:
     - For each configured region in `regions[]`, ensures a writable Cloud SQL
       for PostgreSQL primary instance (`quadsmith-db-${ENV}-${region}`, tier
       `${db_tier}`, `--no-assign-ip`, `--network=default`,
       `--backup-start-time=04:00`, `--enable-point-in-time-recovery`,
       `--database-flags=cloudsql.logical_replication=on,max_connections=100`)
       exists and matches the region's configured `db_tier`.
     - Ensures the `quadsmith_api` and `quadsmith_ops` logical databases exist
       on each regional Cloud SQL instance via `gcloud sql databases create`.
     - When multiple regions are configured in `regions[]`, ensures the
       `quadsmith_global_pub` PostgreSQL Logical Replication publication and
       subscriptions (replicating the PII-free `users` stub, hardware catalog,
       and public community tables while keeping user PII strictly local to each
       region, as detailed in
       [`adding_deployment_regions.md`](adding_deployment_regions.md)) are
       active across the regional primaries.
     - Queries the Private VPC IP (`ipAddresses[0].ipAddress`) of each regional
       Cloud SQL instance to construct `QS_DATABASE_URL`,
       `QS_INTERNAL_DATABASE_URL`, and `QS_PEER_DATABASE_URLS`.
  3. **`ensure_cloudflare_tunnel_and_edge`**: Ensures the named Cloudflare
     Tunnel (`quadsmith-${ENV}` -> `http://127.0.0.1:8084`), proxied `CNAME`
     records (read from `server-api`, `server-web`, and `cli-ops` `.textproto`
     configs), Smart Tiered Cache, and RFC `Cache-Control` Cache Rule exist.
  4. **`ensure_compute_nodes`**: Iterates over `regions[].nodes[]` in
     `deploy/config/${ENV}.textproto`. For each VM node (`zone`,
     `machine_type`), ensures the `--no-address` Debian 12 VM exists, matches
     the configured `machine_type` (resizing in place if changed), and has its 2
     GB swapfile, Docker daemon log rotation, daily image prune cron, Artifact
     Registry auth, and standalone `cloudflared` daemon container running—with
     zero persistent state or backup files on the VM disk.
  5. **`run_migration_gate_and_seed`**: Executes the 4-step database migration
     gate (check `psqldef --dry-run` &rarr; if DDL diff is non-empty, run
     `make check-breaking` and create a Cloud SQL `ON_DEMAND` pre-migration
     snapshot retaining `db_backup_retention_count` snapshots &rarr; apply
     `make db-migrate` and seed `internal_tokens`).

| Subsystem / Layer                      | Tier 1: One-Time Human Account Setup _(Browser / CLI Once)_                                                                                    | Tier 2: Idempotent Bootstrap in `make deploy` _(Created on 1st Run; ~3s No-Op Thereafter)_                                                                                                                                              | Tier 3: Recurring Per-Deployment Rollout in `make deploy` _(Runs on Every Release)_                                                                                                                                                     |
| :------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Deployment Config**                  | Author `deploy/config/{staging,prod}.textproto` (project, regions, `db_tier`, `nodes`).                                                        | Load & validate `deploy/config/${ENV}.textproto` via `check-config` / `LoadDeployConfig`.                                                                                                                                               | Read target regions, Cloud SQL tiers, and VM specs from `deploy/config/${ENV}.textproto`.                                                                                                                                               |
| **Accounts, IAM & VPC**                | Create Cloudflare Zone & API token; create GCP Project, Billing link & `quadsmith-cd` SA; create OAuth apps; store GitHub Environment Secrets. | Ensure IAP firewall rule (`35.235.240.0/20` -> `:22`), Cloud NAT, and global VPC Private Services Access (`/16`) exist.                                                                                                                 | Authenticate GitHub Actions runner via `GCP_SA_KEY`; sync runtime `/opt/quadsmith/.env` to each VM node over IAP.                                                                                                                       |
| **Artifact Registry**                  | Enable `artifactregistry.googleapis.com`.                                                                                                      | Ensure `quadsmith-docker` repository & 3-version retention cleanup policy exist in `registry_region`.                                                                                                                                   | Build & push `server-web`, `server-api`, `server-ops`, and `server-jobs` (`:latest` & `:sha-<commit>`).                                                                                                                                 |
| **Cloudflare Edge & DNS**              | Add `quadsmith.com` domain to Cloudflare account.                                                                                              | Ensure Cloudflare Tunnel (`quadsmith-${ENV}`), proxied `CNAME` records, Smart Tiered Cache, and RFC `Cache-Control` Cache Rule exist.                                                                                                   | **None** _(Tunnel and edge cache rules stay active; new `?v=<hash>` WASM & Vite bundle URLs bust cache automatically)_.                                                                                                                 |
| **GCP Cloud SQL (Primary & Replicas)** | Enable `sqladmin.googleapis.com` & `servicenetworking.googleapis.com`.                                                                         | Ensure primary `quadsmith-db-${ENV}-${region}` (`--no-assign-ip`, daily backups at `04:00 UTC`, 7-day PITR), logical DBs (`quadsmith_api`, `quadsmith_ops`), and any regional read replicas exist at `db_tier`; seed `internal_tokens`. | Run `psqldef --dry-run`; **when DDL changes exist**, run `make check-breaking`, create Cloud SQL `ON_DEMAND` pre-migration snapshot (retaining `db_backup_retention_count` snapshots), and apply `make db-migrate` over Private VPC IP. |
| **Compute Engine Host VMs**            | Enable `compute.googleapis.com` & `iap.googleapis.com`.                                                                                        | For each node in `regions[].nodes[]`: ensure VM (`--no-address`, `machine_type`, 30 GB disk), 2 GB `/swapfile`, `/etc/docker/daemon.json` log rotation, daily image prune cron, and standalone `cloudflared` container are running.     | **None** _(Host OS, swap, Docker daemon, and standalone `cloudflared` container remain uninterrupted and 100% stateless)_.                                                                                                              |
| **Application Containers & Nginx**     | **None.**                                                                                                                                      | Ensure persistent `nginx` gateway container is running on `127.0.0.1:8084`.                                                                                                                                                             | SCP `deploy/` manifests over IAP to each VM node; pull new images into inactive `blue`/`green` slot; gate on `/healthz`; run `nginx -s reload`; drain & remove old slot.                                                                |

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-ansible-playbook" >Ansible Playbooks for VM and Cloud Provisioning</a>

Author Ansible playbooks (`ansible-playbook deploy.yml -i inventory.gcp.yml`) to
provision the GCP VM and configure Docker, swap, and Nginx over SSH.

**Pros:**

- Built-in YAML modules for Linux file, sysctl, and systemd management.

**Cons:**

- Requires installing Python, Ansible, and `google.cloud` Galaxy collections on
  the runner, and configuring a custom `gcloud compute ssh --tunnel-through-iap`
  proxy wrapper since the `e2-micro` VM has no public IP address.

</details>

### <a id="sql-database-setup-pre-migration-backups-and-automated-migration-gate" >SQL Database Setup, Pre-Migration Backups, and Automated Migration Gate</a>

Quadsmith manages its two PostgreSQL database schemas declaratively using
`psqldef` (from the [`sqldef`](https://github.com/sqldef/sqldef) project) with
single desired-state DDL files maintained in
[`src/db/quadsmith_api.sql`](../src/db/quadsmith_api.sql) and
[`src/db/quadsmith_ops.sql`](../src/db/quadsmith_ops.sql), and detects
backwards-incompatible changes via
[`schemacheck.Compare()`](../src/go/shared/pkg/schemacheck/schemacheck.go#L311-L439)
and
[`schema_breaking_test.go`](../src/go/shared/internal/regression/schema_breaking_test.go).

Because the Cloud SQL instance has only a Private VPC IP (`--no-assign-ip`),
`make deploy` establishes an ephemeral local IAP TCP forwarding tunnel through
the primary VM (`gcloud compute ssh ... --tunnel-through-iap` with
`-L 15432:<cloud-sql-private-ip>:5432 -N -f`) during the migration gate so
`psqldef` and `go test` on the runner can inspect and migrate Cloud SQL securely
over its Private IP without ever exposing Cloud SQL to the public internet:

```mermaid
flowchart TD
    subgraph Stage1["Step 1: Build & Push Images"]
        Build["Compile & Push 4 Docker Images (server-web, server-api, server-ops, server-jobs)"]
    end

    subgraph Stage2["Step 2: Cloud SQL Bootstrap, Backup & Migration Gate (Over IAP Tunnel to Private VPC IP)"]
        Create["Ensure Cloud SQL Instance & quadsmith_api / quadsmith_ops Databases Exist"]
        DryRun{"psqldef --dry-run Detects Pending DDL Changes?"}
        Check["Export Live Schema via psqldef --export -> Run make check-breaking (STRICT_BREAKING=1)"]
        Guard{"Breaking Schema Change Detected?"}
        Dump["Create Cloud SQL ON_DEMAND Pre-Migration Snapshot & Prune Old Snapshots > db_backup_retention_count"]
        Apply["Run make db-migrate (Transactional psqldef Apply) & Seed internal_tokens"]
        Create --> DryRun
        DryRun -->|Yes (Existing DB)| Check --> Guard
        Guard -->|0 Breaking Changes or allow_breaking_changes=true| Dump --> Apply
    end

    subgraph Stage3["Step 3: Zero-Downtime Blue-Green Cutover"]
        Deploy["IAP SSH to Each VM Node & Run deploy_blue_green.sh"]
    end

    Abort["Abort Pipeline Immediately (Running Containers & DB Untouched)"]

    Build --> Create
    DryRun -->|No DDL Changes (or Initial Empty DB)| Apply
    Guard -->|Unapproved Breaking Change| Abort
    Apply -->|Migration Succeeds| Deploy
    Apply -->|DDL Error Rolled Back by Postgres| Abort
```

1. **Idempotent Cloud SQL Database Creation (One-Time Setup)**:
   - `make deploy` verifies that `quadsmith_api` and `quadsmith_ops` exist on
     `quadsmith-db-${ENV}-${PRIMARY_REGION}` via `gcloud sql databases list` and
     creates them if missing.
2. **Extend `schema_breaking_test.go` and Add `make check-breaking`**:
   - Update
     [`src/go/shared/internal/regression/schema_breaking_test.go`](../src/go/shared/internal/regression/schema_breaking_test.go)
     to check both `src/db/quadsmith_api.sql` and `src/db/quadsmith_ops.sql`
     across all eight breaking-change classes (`TABLE_DROPPED`,
     `COLUMN_DROPPED`, `COLUMN_TYPE_CHANGED`, `COLUMN_MADE_NOT_NULL`,
     `NEW_COLUMN_NOT_NULL`, `UNIQUE_CONSTRAINT_ADDED`, `PRIMARY_KEY_MODIFIED`,
     `FOREIGN_KEY_DROPPED`).
   - Support `QS_LIVE_SCHEMA_DIR` (populated via `psqldef --export` from the
     live databases) and `BREAKING_BASE_REF` so CD compares target
     `src/db/*.sql` files directly against the live database's current schema
     rather than only `HEAD~1`.
   - Add `make check-breaking` in [`Makefile`](../Makefile):

     ```makefile
     check-breaking:
     	@echo "--- Running Strict Backwards-Compatibility Gate ---"
     	@cd src/go/shared && STRICT_BREAKING=1 go test -mod=vendor -count=1 -v ./internal/regression/...
     ```

     To intentionally deploy a breaking schema change, an operator can trigger
     `deploy.yml` via `workflow_dispatch` with `allow_breaking_changes: true`,
     which runs `make warn-breaking` instead of failing the gate.
3. **Three-Tier Stateless Cloud SQL Backup Schedule (`ON_DEMAND` Pre-Migration
   Snapshots + Daily Automated Backups + 7-Day PITR)**:
   - To keep all Compute Engine VMs 100% stateless (zero `.dump` files on VM
     boot disks), all backups are managed natively by **GCP Cloud SQL** across
     three complementary tiers:
     1. **Daily Automated Backups (`04:00 UTC`)**: 7 daily incremental block
        snapshots included free ($0.00/mo) with every Cloud SQL instance.
     2. **Continuous 7-Day Point-in-Time Recovery (PITR)**: Continuous
        PostgreSQL Write-Ahead Log (WAL) archiving
        (`--enable-point-in-time-recovery`) allowing recovery to any exact
        second within a rolling 7-day window.
     3. **Conditional Pre-Migration `ON_DEMAND` Snapshots
        (`db_backup_retention_count`)**: Whenever `psqldef --dry-run` detects
        pending schema migrations on an existing database, `make deploy` logs
        the exact UTC PITR timestamp (`Pre-migration PITR restore timestamp`),
        creates a native block-level Cloud SQL snapshot via
        `gcloud sql backups create` on `quadsmith-db-${ENV}-${PRIMARY_REGION}`,
        and prunes older `ON_DEMAND` snapshots beyond
        `db_backup_retention_count` (`5` in `prod`, `3` in `staging`, costing
        only ~$0.02–$0.40/mo for incremental changed blocks). Unlike PITR, these
        explicit pre-migration snapshots never age out after 7 days.
   - If `psqldef --dry-run` reports zero schema changes (a code-only or
     frontend-only deployment), the pre-migration snapshot and migration steps
     are skipped in <1 second.
4. **Parameterized `make db-migrate` & Idempotent Token Seeding**:
   - Decouple `db-migrate` from the `generate` prerequisite in
     [`Makefile`](../Makefile) and parse `QS_DATABASE_URL` and
     `QS_INTERNAL_DATABASE_URL` (defaulting to
     `postgres://postgres:postgres@127.0.0.1:5432/quadsmith_{api,ops}?sslmode=disable`
     for local sandbox runs) into `psqldef` flags (`-h`, `-p`, `-U`, `-W`,
     `<dbname>`, and `PGSSLMODE`).
   - After `make db-migrate` applies the declarative schema inside PostgreSQL's
     transactional DDL wrapper, execute an idempotent
     `INSERT INTO internal_tokens (...) ON CONFLICT (token_hash) DO NOTHING` on
     `quadsmith_ops` so the `ROBOT` token (`QS_JOB_SERVER_AUTH_TOKEN`) and any
     configured initial operator `HUMAN` token hash are guaranteed to exist.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-vm-local-pg-dump-backups" >Store `pg_dump -Fc` Pre-Migration Snapshots on the Primary VM Disk (`/opt/quadsmith/backups`) or in GCS</a>

Run `pg_dump -Fc` over SSH and store `.dump` files on the primary VM's 30 GB
boot disk (or upload them to a Google Cloud Storage bucket) instead of using
native Cloud SQL `ON_DEMAND` block snapshots.

**Pros:**

- Produces standalone `.dump` files that can be passed directly to `pg_restore`.

**Cons:**

- Storing backups on the primary VM violates VM statelessness ("cattle, not
  pets")—if the VM is recreated, resized, or replaced, local backups are lost.
- Because `pg_dump` exports a full copy of all `BYTEA` media blobs in
  `quadsmith_api.blobs` on every schema migration rather than incremental
  changed blocks, 5 retained dumps of a 6 GB database would consume 30 GB and
  exhaust the `e2-micro`'s 30 GB disk (or cost ~$0.90/mo in GCS vs.
  ~$0.08–$0.40/mo for incremental Cloud SQL snapshots).

##### <a id="alt-daily-backups-and-pitr-only" >Rely Solely on Cloud SQL's 7 Daily Automated Backups and 7-Day PITR Without Pre-Migration `ON_DEMAND` Snapshots</a>

Remove `db_backup_retention_count` and pre-migration snapshot creation, relying
exclusively on Cloud SQL's 7 free automated daily backups and 7-day continuous
WAL Point-in-Time Recovery (PITR).

**Pros:**

- Incurs $0.00/month in additional snapshot storage beyond the 7 included daily
  backups.

**Cons:**

- Both automated daily backups and PITR expire on a strict 7-day rolling window;
  if a subtle schema migration regression is discovered more than 7 days after a
  release, the pre-migration state is no longer recoverable, whereas `ON_DEMAND`
  snapshots persist until rotated by newer migrations.

##### <a id="alt-migrate-before-build" >Run Database Migrations Before or in Parallel with Image Builds</a>

Execute database migrations immediately when the workflow starts, in parallel
with Docker image builds, to shave ~15 seconds off total pipeline duration.

**Pros:**

- Slightly faster end-to-end GitHub Actions workflow execution time.

**Cons:**

- If a Docker image build fails compilation or asset bundling, the production
  database schema has already been mutated while the old application containers
  remain deployed.

##### <a id="alt-custom-migrate-db-regex-script" >Custom `migrate_db.sh` Script Regex-Matching `psqldef --dry-run` Output</a>

Write a standalone bash script (`deploy/migrate_db.sh`) that runs
`psqldef --dry-run` and uses `grep -E` regexes to block `DROP TABLE` and
`DROP COLUMN` statements.

**Pros:**

- Does not require running `go test` during migration checks.

**Cons:**

- Duplicates breaking-change detection logic that already exists in
  [`schemacheck.go`](../src/go/shared/pkg/schemacheck/schemacheck.go) and misses
  non-`DROP` breaking changes such as adding a `NOT NULL` column without a
  `DEFAULT`, changing column data types, or adding a `UNIQUE` constraint to an
  existing column.

##### <a id="alt-skip-drop-flag" >Pass `psqldef --skip-drop` Unconditionally in CI</a>

Run `psqldef --skip-drop` on every deployment so `psqldef` silently ignores any
removed tables, columns, or indexes instead of failing the gate.

**Pros:**

- Never blocks a deployment when a column or table is removed from
  `src/db/*.sql`.

**Cons:**

- Causes silent schema drift between `src/db/quadsmith_api.sql` and the live
  production database, leaving orphaned columns whose `NOT NULL` constraints can
  break `INSERT` queries at runtime.

</details>

### <a id="zero-downtime-blue-green-container-cutover-and-automatic-rollback" >Zero-Downtime Blue-Green Container Cutover and Automatic Rollback</a>

Because `server-web` (~25 MB), `server-api` (~25 MB), `server-ops` (~20 MB), and
the headless `server-jobs` scheduler (~10 MB) consume only **~80 MB of combined
idle RAM**—and the 1 GB `e2-micro` VM has ~751 MB of available RAM plus a 2 GB
swapfile—the host can comfortably run both the active (`blue`) and candidate
(`green`) container sets side-by-side for the few seconds required to verify
`/healthz` readiness and reload Nginx, eliminating the ~1–2 second
`502 Bad Gateway` window of in-place container recreation.

```mermaid
flowchart TD
    Start["deploy/deploy.sh Invokes deploy/deploy_blue_green.sh on VM"]
    Detect["Detect Active Slot (blue) & Select Candidate Slot (green)"]
    Boot["Pull Images & Start Candidate Slot (web-green, api-green, ops-green, jobs-green)"]
    Health{"Poll HTTP /healthz on web-green, api-green, ops-green (Up to 15s)"}

    subgraph Cutover["Atomic Cutover & Teardown"]
        Switch["Write nginx/upstreams.conf -> green & Run nginx -t && nginx -s reload"]
        Drain["Gracefully Drain & Stop Old Slot (blue)"]
        Switch --> Drain
    end

    subgraph FailureRollback["Automatic Rollback"]
        Logs["Dump Candidate Container Logs"]
        Cleanup["Stop & Remove Candidate Slot (green)"]
        KeepActive["Keep blue Active (Zero Customer Impact) & Exit 1"]
        Logs --> Cleanup --> KeepActive
    end

    Start --> Detect --> Boot --> Health
    Health -->|All 3 HTTP Services Return 200 OK| Switch
    Health -->|Any Timeout or Non-200| Logs
```

1. **Dual-Slot Compose Service Topology
   (`deploy/docker-compose.blue-green.yaml`)**:
   - To keep `deploy/docker-compose.yaml` fast and conflict-free for local
     `make sandbox` and `make test` runs, the dual-slot `blue` and `green`
     service definitions live in `deploy/docker-compose.blue-green.yaml`.
   - `nginx` (and the standalone `cloudflared` container) run continuously so
     the Cloudflare edge tunnel and `127.0.0.1:8084` gateway never restart
     during application deployments.
   - `blue` slot (`web-blue`, `api-blue`, `ops-blue`, `jobs-blue`) and `green`
     slot (`web-green`, `api-green`, `ops-green`, `jobs-green`) isolate internal
     service communication (`web-<slot>` &rarr; `http://api-<slot>:8080`,
     `jobs-<slot>` &rarr; `http://ops-<slot>:8080`).
   - During the ~5-second overlap window, running both slots adds only **~80 MB
     of RAM** (~329 MB total on the 1 GB VM) and opens only **3 initial database
     connections** (`min_connections: 1` on `api-<next>` and `ops-<next>`).
2. **Atomic Nginx Upstream Switching (`deploy/nginx/upstreams.conf`)**:
   - `deploy/nginx/upstreams.conf` is mounted into the `nginx` container at
     `/etc/nginx/conf.d/upstreams.conf` and included by
     `deploy/nginx/nginx.conf` (defaulting to `web:8080`, `api:8080`, `ops:8080`
     in local sandbox and rewritten by `deploy/deploy_blue_green.sh` to
     `web-<slot>:8080`, `api-<slot>:8080`, `ops-<slot>:8080` on the VM).
   - Once the candidate slot passes `/healthz`, `deploy/deploy_blue_green.sh`
     atomically updates `upstreams.conf` and executes:

     ```bash
     docker compose -f deploy/docker-compose.blue-green.yaml exec -T nginx sh -c "nginx -t && nginx -s reload"
     ```

   - [Nginx's `reload` signal (`SIGHUP`)](https://nginx.org/en/docs/control.html#reconfiguration)
     spawns new worker processes routing to the new slot while signaling old
     worker processes to finish serving all in-flight requests (including 100 MB
     `/api/blobs` uploads) before exiting—guaranteeing **zero dropped packets**.
3. **Health-Check Gating and Automatic Rollback**:
   - `deploy/deploy_blue_green.sh` polls
     `wget -q -O- http://<service>:8080/healthz` from inside the `nginx`
     container for `web-<next>`, `api-<next>`, and `ops-<next>` (and verifies
     `jobs-<next>` is `running`) every 1 second for up to 15 seconds.
   - If any candidate container fails to return `200 OK` within 15 seconds,
     `upstreams.conf` is untouched, the failing candidate container logs
     (`--tail=50`) are dumped, the candidate slot is stopped and removed, and
     the script exits `1` with zero customer impact.
   - After a successful reload onto `<next>`, the script waits 2 seconds and
     stops the previous slot with `--timeout 30`, freeing the ~80 MB of RAM.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-host-port-mapping-per-slot" >Bind Blue and Green Slots to Distinct Host Ports</a>

Expose `blue` on host ports `8080–8082` and `green` on `8090–8092` instead of
using Docker bridge network service names.

**Pros:**

- Allows probing `/healthz` directly from the host OS via `localhost:<port>`.

**Cons:**

- Unnecessarily binds extra host network ports and complicates port bookkeeping
  when Docker's internal DNS (`web-blue:8080`, `web-green:8080`) already
  isolates slots cleanly.

##### <a id="alt-nginx-backup-directive-only" >Static Nginx `upstream` with `backup` Directive</a>

List both `server api-blue:8080;` and `server api-green:8080 backup;` in a
static `nginx.conf` without reloading Nginx.

**Pros:**

- Avoids rewriting `upstreams.conf` or calling `nginx -s reload`.

**Cons:**

- Nginx only routes to `backup` servers after primary servers fail connection
  attempts, which still incurs failed TCP handshakes during container shutdown
  and prevents deterministic active-slot selection.

##### <a id="alt-traefik-dynamic-docker-labels" >Replace Nginx with Traefik Dynamic Docker Label Discovery</a>

Use Traefik watching `/var/run/docker.sock` to automatically discover and switch
container backends via Docker labels.

**Pros:**

- Eliminates the need for an `upstreams.conf` reload script.

**Cons:**

- Traefik's Go runtime consumes ~35 MB – 50 MB of RAM (~10x more than Nginx's ~4
  MB C footprint) and requires mounting the privileged Docker socket into the
  ingress proxy container.

##### <a id="alt-post-cutover-rollback" >Switch Traffic Immediately and Roll Back on 5xx Error Rate</a>

Reload Nginx immediately after `docker compose up -d` and monitor Nginx access
logs for `502`/`500` responses to trigger a rollback.

**Pros:**

- Slightly faster cutover when containers are healthy.

**Cons:**

- Exposes real production users to failed requests if a container fails startup
  validation or crashes on initialization.

</details>

### <a id="colocated-artifact-registry-github-actions-cd-and-operator-guide-deploy-readme-md" >Colocated Artifact Registry, GitHub Actions CD, and Operator Guide (`deploy/README.md`)</a>

Docker images are hosted in a private **Google Artifact Registry** repository
(`quadsmith-docker`) located in the **same GCP region as the primary VM**
(`us-central1`):

- **$0.00 Egress**: Network egress between Google Artifact Registry and a
  Compute Engine VM in the same region is completely free (and only
  $0.01/GB–$0.02/GB when pulled by secondary regional VMs).
- **Native IAM Authentication**: The VM's attached Compute Engine default
  service account is granted the `roles/artifactregistry.reader` role. Running
  `gcloud auth configure-docker us-central1-docker.pkg.dev` allows the VM to
  pull private images without generating, storing, or rotating external personal
  access tokens.
- **Automated Retention Cleanup**: Configure an Artifact Registry cleanup policy
  retaining only the **3 most recent tagged versions** of each image. Because
  the four Alpine Go images total ~120 MB compressed per build, 3 versions
  consume ~360 MB, remaining comfortably within Artifact Registry's **500 MB
  free storage tier**.

#### <a id="github-actions-deployment-pipeline" >GitHub Actions Deployment Pipeline (`.github/workflows/deploy.yml`)</a>

The CI/CD workflow in
[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) is guarded so
that while iterating without live environments (when `QS_GCP_PROJECT_ID` is not
configured in GitHub Secrets), the workflow cleanly skips without failing CI.
Once secrets are configured, `.github/workflows/deploy.yml` executes
`make deploy ENV=prod` (or in three chained stages: `build-and-push` &rarr;
`migrate-db` &rarr; `deploy-vm`):

1. **`build-and-push`**: Builds the 4 images in parallel using BuildKit (with
   `cache-dependency-path: "src/go/go.work.sum"`) and pushes to
   `us-central1-docker.pkg.dev/<PROJECT_ID>/quadsmith-docker/<image>:latest` and
   `:sha-<COMMIT_SHA>`.
2. **`migrate-db` (`needs: build-and-push`)**: Ensures the Cloud SQL topology
   and target SQL databases exist, opens the IAP tunnel to the primary Cloud SQL
   Private IP, checks `psqldef --dry-run`, and whenever schema changes are
   present, runs `make check-breaking` (or `make warn-breaking` if triggered via
   `workflow_dispatch` with `allow_breaking_changes: true`), creates a
   pre-migration Cloud SQL `ON_DEMAND` snapshot, applies `make db-migrate`, and
   seeds `internal_tokens`.
3. **`deploy-vm` (`needs: migrate-db`)**: Reads
   `deploy/config/${ENV}.textproto`, ensures all GCP/Cloudflare/VM resources
   exist across `regions[].nodes[]`, syncs `/opt/quadsmith/.env` (with each
   region's local Cloud SQL read/write Private IPs) and `deploy/` over IAP SSH
   (`gcloud compute ssh --tunnel-through-iap`), and executes
   `deploy/deploy_blue_green.sh` sequentially across each node.

#### <a id="operator-deployment-guide-deploy-readme-md" >Operator Deployment Guide (`deploy/README.md`)</a>

To colocate operational instructions directly beside the files they describe
(`deploy/config/`, `deploy/deploy.sh`, `deploy/docker-compose.yaml`) while
keeping top-level [`plans/`](../plans) dedicated to architectural design plans,
we author the operator deployment runbook at `deploy/README.md` (linked from the
root [`README.md`](../README.md)) structured into eight actionable sections:

1. **Environments & Architecture Overview**: Explains the distinction between
   `sandbox` (local ephemeral Docker test harness started via `make sandbox`
   with zero cloud dependencies) and live environments (`staging` and `prod`,
   configured via `deploy/config/{staging,prod}.textproto` and deployed via
   `make deploy`).
2. **One-Time Account & Secret Setup Checklist**: Step-by-step instructions and
   exact CLI/console commands to create the Cloudflare zone & scoped API token
   (`CF_ACCOUNT_ID`, `CF_ZONE_ID`, `CF_API_TOKEN`), GCP project & `quadsmith-cd`
   deployer service account (`GCP_SA_KEY`), OAuth provider credentials, and
   secrets (`QS_DB_PASSWORD`, `QS_AUTH_SECRET`, `QS_JOB_SERVER_AUTH_TOKEN`),
   plus a complete table of required GitHub Environment Secrets.
3. **Deployment Configuration (`deploy/config/{staging,prod}.textproto`)**:
   Documents the `DeployConfig` fields (`gcp_project_id`, `registry_region`,
   `regions`, `db_tier`, `nodes`, `db_backup_retention_count`) and how
   `make deploy` derives per-service hostnames from
   `src/go/*/config/{staging,prod}.textproto`.
4. **Running Deployments (`make deploy ENV=staging|prod` & CI/CD)**: Shows how
   the single convergent `make deploy` command works both from GitHub Actions
   (`.github/workflows/deploy.yml`) and from an operator workstation.
5. **Backup Schedule, Database Migrations & Rollback Runbook**: Documents the
   three-tier stateless Cloud SQL backup schedule (7 free daily automated
   backups at `04:00 UTC`, 7-day 1-second continuous PITR, and pre-migration
   `ON_DEMAND` snapshots rotated to `db_backup_retention_count`), the automated
   `make check-breaking` migration gate, and step-by-step rollback instructions
   for container rollbacks, additive schema reverts, in-place snapshot restores
   (`gcloud sql backups restore`), and 1-second PITR clone recovery
   (`gcloud sql instances clone --point-in-time=...`).
6. **Adding New Geographic Regions**: Step-by-step instructions for monitoring
   geographic traffic telemetry, appending a new `regions` block to
   `deploy/config/prod.textproto`, provisioning a local Cloud SQL cross-region
   read replica, and verifying regional routing (per
   [`adding_deployment_regions.md`](adding_deployment_regions.md)).
7. **Scaling a Region Vertically or Horizontally**: Step-by-step decision matrix
   and instructions for scaling an individual region's VM (`machine_type`) or
   Cloud SQL instance (`db_tier`) vertically vs. adding intra-region horizontal
   VM nodes (`nodes[]`) or graduating a region to Cloud Run + Cloud Scheduler,
   plus offloading `/blobs/*` to Cloudflare R2 (per
   [`scaling_deployment_regions.md`](scaling_deployment_regions.md)).
8. **Day-2 Operations & Troubleshooting (`qsops` & IAP SSH)**: Provides
   copy-pasteable commands for connecting to zero-public-IP VMs over IAP
   (`gcloud compute ssh --tunnel-through-iap`), inspecting container health and
   logs, and managing operator tokens (`qsops auth`).

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-ghcr" >GitHub Container Registry (`ghcr.io`)</a>

Publish images to GitHub Container Registry instead of GCP Artifact Registry.

**Pros:**

- Keeps all code and container packaging consolidated under GitHub.
- 500 MB free storage included with GitHub Free.

**Cons:**

- Requires creating and managing a GitHub Personal Access Token (PAT) on the VM
  for `docker login`.
- Images are pulled over the public internet, consuming GitHub bandwidth (1 GB
  free/mo, then $0.50/GB).

##### <a id="alt-on-vm-build" >Compile and Build Docker Images on the VM</a>

Git pull the repository on the VM and execute `docker compose build`.

**Pros:**

- Eliminates the need for any container registry.

**Cons:**

- `npm run build` and `go build` will immediately trigger an out-of-memory
  kernel panic on a 1 GB RAM machine.
- Saturates the burstable CPU credits of the `e2-micro`, throttling performance.

##### <a id="alt-docs-deployment-md" >Separate `docs/deployment.md` File in `docs/`</a>

Place the operator deployment runbook in `docs/deployment.md` rather than in
`deploy/README.md`.

**Pros:**

- Places the deployment guide in a `docs/` directory.

**Cons:**

- Separates the deployment instructions from the `deploy/` directory containing
  `deploy.sh`, `docker-compose.yaml`, and `config/*.textproto`.
- Loses automatic GitHub directory README rendering when browsing `deploy/` and
  reintroduces a single-file `docs/` directory.

##### <a id="alt-keep-instructions-only-in-plans" >Keep Deployment Instructions Only Inside `plans/`</a>

Rely on the architectural design documents in `plans/` as the operator
documentation instead of creating a dedicated `deploy/README.md` guide.

**Pros:**

- Avoids maintaining a separate operator README in `deploy/`.

**Cons:**

- Plan documents in `plans/` are structured around design rationales, trade-off
  tables, and alternatives considered rather than a linear, step-by-step
  operator runbook.

</details>

### <a id="single-region-capacity-and-cost-analysis" >Single-Region Capacity and Cost Analysis</a>

Because Cloudflare caches 100% of `/assets/*`, `/quadsmith-engine.wasm` (12 MB
stripped), and `/blobs/{id}`, and because all SQL traffic between the `e2-micro`
VM and Cloud SQL stays on the free internal GCP VPC (`$0.00/GB`), each user
**app load** only consumes origin internet egress for the initial `index.html`
(~2 KB) and 2–3 ConnectRPC JSON responses (~18 KB total)—just **~20 KB of origin
internet egress** and **~3.5 ms of Go CPU time** per app load:

- **Static Asset & WASM Egress Offload**: Static JS/CSS (`/assets/*`), the 12 MB
  stripped WASM engine (`/quadsmith-engine.wasm`), and media blobs
  (`/blobs/{id}`) consume **0 bytes of per-visit origin egress** once cached by
  Cloudflare Tiered Cache (~50–100 MB/month total to seed Upper-Tier PoPs across
  releases).
- **Zero Database Egress Overhead**: Because Cloud SQL communicates with
  `server-api` and `server-ops` over same-region Private VPC Peering, neither
  SQL queries nor 100 MB `BYTEA` blob uploads/reads consume any of GCP's 1
  GB/month free internet egress allowance.
- **Within 1 GB Free GCP Internet Egress (~$10.50/mo Total for Cloud SQL)**:
  Supports **~45,000 – 60,000 app loads/month** ($\frac{920,000\text{
  KB}}{20\text{ KB}} \approx 46,000\text{ – }60,000$ after accounting for
  Upper-Tier cache seeding) with **$0.00 in compute and network egress charges**
  (paying only ~$10.50/month for the `db-f1-micro` Cloud SQL instance + 10 GB
  SSD).
- **With ~$1.00 – $5.00 of Pay-As-You-Go GCP Internet Egress (~$11.50 –
  $15.50/mo Total)**: Because Cloud SQL has no 5 GB data transfer ceiling over
  Private IP, the single `e2-micro` + `db-f1-micro` pair scales smoothly past
  350,000+ app loads/month without hitting a free-tier database transfer wall.
- **Pure `e2-micro` CPU Headroom**: At 0.25 baseline vCPU (648,000
  vCPU-seconds/month) and 3.5 ms/load, the VM CPU itself can handle over
  **1,500,000 – 5,000,000 app loads/month** (~50–70 sustained RPS and ~500 burst
  RPS) before requiring per-region vertical or horizontal scaling.

| Traffic & Quota Regime (Single-Region `us-central1` Topology) | Limiting Resource / Quota               | Supported App Loads / Month                              | Monthly Active Users (MAU) | Peak Origin RPS                                          | Estimated Monthly Cost                                                        |
| :------------------------------------------------------------ | :-------------------------------------- | :------------------------------------------------------- | :------------------------- | :------------------------------------------------------- | :---------------------------------------------------------------------------- |
| **Within 1 GB Free GCP Egress**                               | 1 GB/mo GCP Always Free Internet Egress | **~45,000 – 60,000**<br>_(~1,500–2,000 loads/day)_       | **~10,000 – 20,000**       | **~50 – 70 RPS** _(sustained)_<br>**~500 RPS** _(burst)_ | **~$10.50 / mo**<br>_($0 VM + ~$10.50 Cloud SQL `db-f1-micro`)_               |
| **With Pay-As-You-Go GCP Egress**                             | `db-f1-micro` RAM (0.6 GB) & 10 GB SSD  | **~300,000 – 1,000,000**<br>_(~10,000–33,000 loads/day)_ | **~50,000 – 150,000**      | **~50 – 70 RPS** _(sustained)_<br>**~500 RPS** _(burst)_ | **~$11.50 – $13.00 / mo**<br>_($0 VM + ~$10.50 Cloud SQL + ~$1–$2.50 egress)_ |

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-unstripped-wasm-no-tiered-cache" >Unstripped WASM Binary Without Cloudflare Tiered Cache</a>

Serve the unstripped 17.5 MB `quadsmith-engine.wasm` with standard per-PoP edge
caching rather than stripping DWARF symbols and enabling Cloudflare Smart Tiered
Cache.

**Pros:**

- Retains DWARF debug symbols in the production `.wasm` binary.
- Saves toggling Smart Tiered Cache in Cloudflare.

**Cons:**

- Every cache miss across Cloudflare's 300+ global PoPs pulls 17.5 MB directly
  from the `e2-micro` VM, exhausting the 1 GB free GCP egress limit after just
  ~58 regional cache-warming requests.

</details>

---

## <a id="4-alternatives-considered" >4. Alternatives considered</a>

<details>
<summary><strong>Alternatives considered</strong></summary>

### <a id="host-stateless-compute-layer-on-aws-or-azure-instead-of-gcp" >Host Stateless Compute Layer on AWS or Azure Instead of GCP</a>

Host the stateless Docker Compose stack on **AWS** (`t4g.micro` EC2 / Lightsail
and AWS RDS `db.t4g.micro`) or **Azure** (`B2pts v2` / `B1s` VM and Azure
PostgreSQL Flexible Server `B1ms`) instead of GCP (`e2-micro` + Cloud SQL
`db-f1-micro`).

**Pros:**

- **Year-1 Free Native Managed Postgres**: Both AWS (RDS `db.t4g.micro`: 1 GB
  RAM, 20 GB SSD) and Azure (Flexible Server `B1ms`: 2 GB RAM, 32 GB SSD) offer
  750 hours/month of free managed PostgreSQL + a free 1 GB RAM VM for the
  **first 12 months** ($0.00/mo in Year 1).
- **100 GB/month Perpetual Free Internet Egress**: AWS and Azure include 100
  GB/month of free outbound internet egress (100x GCP's 1 GB/month).

**Cons:**

- **12-Month Expiration Cliff (2x Higher Year 2+ Cost)**: Neither AWS nor Azure
  offers a perpetually free 1 GB RAM VM. Starting in Month 13, AWS EC2 + RDS
  costs **~$22.50/month** ($8.50 VM + $14.00 RDS) and Azure VM + Flexible Server
  costs **~$24.50/month** ($8.50 VM + $16.00 DB)—more than **double** the
  perpetual **~$10.50/month** cost of GCP's Always Free `e2-micro` ($0.00/mo
  forever) + GCP Cloud SQL `db-f1-micro` (~$10.50/mo).
- **Single-Region VPC Silos**: AWS VPCs and Azure VNets are regional, requiring
  explicit cross-region VPC peering or Transit Gateways when adding regions,
  whereas GCP VPCs are natively global.

### <a id="fly-io-independent-micro-vm-deployment" >Fly.io Independent Micro-VM Deployment</a>

Deploy Quadsmith's four services across four dedicated micro-VMs on Fly.io from
day one.

**Pros:**

- Fully managed Firecracker micro-VMs with zero Linux host administration.
- Built-in zero-config internal WireGuard mesh (`.internal`).

**Cons:**

- Costs ~$8.00 – $12.00/month for compute alone compared to GCP's $0.00/month
  Always Free `e2-micro` tier, plus Fly Postgres is unmanaged.
- Requires maintaining four separate `fly.toml` manifests and deployment
  pipelines.

### <a id="direct-cloud-run-and-cloud-scheduler-deployment-on-day-one" >Direct Cloud Run and Cloud Scheduler Deployment on Day One</a>

Deploy `server-web`, `server-api`, and `server-ops` as serverless containers on
GCP Cloud Run from day one, using Cloud Scheduler for maintenance triggers.

**Pros:**

- Near-zero idle compute cost ($0.20/month for Cloud Scheduler).
- Zero host maintenance and automated scaling to zero.

**Cons:**

- Suffers from 1.5s – 3.0s cold starts on sporadic initial traffic unless paying
  ~$35+/month per service for `--min-instances=1`, whereas the Always Free
  `e2-micro` VM stays warm 24/7 at $0.00/month (leaving Cloud Run as an optional
  per-region horizontal scaling graduation path in
  [`scaling_deployment_regions.md`](scaling_deployment_regions.md)).

### <a id="separate-one-time-bootstrap-target-make-env-bootstrap-distinct-from-make-deploy" >Separate One-Time Bootstrap Target (`make env-bootstrap`) Distinct from `make deploy`</a>

Maintain two separate `Makefile` targets and scripts: `make env-bootstrap` (run
manually once from a developer workstation to create the VM, tunnel, DNS
records, and Cloud SQL databases) and `make deploy` (run by CI to push images
and restart containers).

**Pros:**

- Avoids running ~3 seconds of `gcloud`/`curl` existence checks during recurring
  CI deployments.

**Cons:**

- Introduces a second `Makefile` command and script that is only executed once
  per environment and can easily drift out of sync with `make deploy`.
- Cannot automatically self-heal or provision a newly added region or scaled VM
  in `deploy/config/prod.textproto` directly from the CI/CD deployment workflow.

### <a id="declarative-infrastructure-as-code-with-terraform-or-opentofu" >Declarative Infrastructure-as-Code with Terraform or OpenTofu</a>

Manage all GCP resources, Cloudflare Tunnels/DNS/Cache Rules, and Cloud SQL
databases using Terraform or OpenTofu HCL modules (`deploy/terraform/`).

**Pros:**

- Industry-standard declarative state graph with `terraform plan` previews.
- Built-in provider ecosystems for Google Cloud and Cloudflare.

**Cons:**

- **Chicken-and-Egg State Bootstrapping**: Requires manually creating and
  securing a remote GCS state bucket before Terraform can run.
- **Plaintext Secrets in `terraform.tfstate`**: Terraform stores generated
  `cloudflare_zero_trust_tunnel_cloudflared` tokens, GCP service account keys,
  and database passwords in plaintext inside `terraform.tfstate`.
- **Splits Configuration Format**: Introduces HCL alongside Quadsmith's
  standardized Protobuf `.textproto` configuration architecture.

### <a id="custom-go-infrastructure-provisioner-in-qsops-gcp-and-cloudflare-go-sdks" >Custom Go Infrastructure Provisioner in `qsops` (GCP and Cloudflare Go SDKs)</a>

Build a `qsops infra deploy` subcommand in Go using
`cloud.google.com/go/compute`, `cloud.google.com/go/artifactregistry`, and
`github.com/cloudflare/cloudflare-go`.

**Pros:**

- Keeps infrastructure execution written in Go alongside the rest of the backend
  codebase.

**Cons:**

- **Massive Dependency Bloat**: Pulling the GCP Compute/SQL/IAM/Artifact
  Registry Go SDKs and Cloudflare Go SDK into `src/go/go.work` adds dozens of
  transitive dependencies and bloats `vendor/` for cloud control-plane calls
  that are already handled concisely by `gcloud` and `curl`.

### <a id="manual-web-console-setup-clickops" >Manual Web Console Setup (ClickOps)</a>

Document a step-by-step markdown runbook instructing the operator to click
through the Google Cloud Console and Cloudflare Dashboard to create `staging`
and `prod`.

**Pros:**

- Zero infrastructure scripting required upfront.

**Cons:**

- Highly error-prone: missing a single setting (such as forgetting
  `--no-address` on the VM, `--no-assign-ip` on Cloud SQL, omitting the
  Cloudflare `.wasm` Cache Rule, or misconfiguring the IAP firewall range)
  silently causes unexpected monthly billing or broken production behavior.

### <a id="sequential-numbered-migration-files-golang-migrate-goose" >Sequential Numbered Migration Files (`golang-migrate` / `Goose`)</a>

Replace declarative `psqldef` desired-state schema files (`src/db/*.sql`) with
sequentially numbered `0001_init.up.sql` / `0001_init.down.sql` migration files
executed by `golang-migrate` or `goose`.

**Pros:**

- Explicit per-migration scripts allow interleaving custom data backfills with
  DDL changes.

**Cons:**

- Directly violates the repository's declarative schema rule in
  [`AGENTS.md`](../AGENTS.md) requiring a single desired state per database in
  `src/db/<database>.sql` managed via `psqldef`.
- Scatters table definitions across dozens of historical migration files, making
  it difficult to inspect the current schema state at a glance.

### <a id="execute-migrations-inside-service-container-entrypoints" >Execute Migrations Inside Service Container Entrypoints</a>

Bundle `psqldef` and `src/db/*.sql` inside the `server-api` and `server-ops`
Docker images and run `psqldef` automatically inside `main.go` on container
startup.

**Pros:**

- Requires no separate `migrate-db` stage in `.github/workflows/deploy.yml`.

**Cons:**

- Inflates every production container image with the `psqldef` binary and DDL
  files.
- Creates dangerous race conditions on `pg_catalog` locks when multiple
  container replicas start concurrently (such as during blue-green switchover or
  multi-region scaling).
- Slows down container startup and `/healthz` readiness.

### <a id="docker-swarm-mode-rolling-updates" >Docker Swarm Mode Rolling Updates (`docker stack deploy`)</a>

Initialize single-node Docker Swarm (`docker swarm init`) on the `e2-micro` VM
and use `deploy.update_config.order: start-first` in Compose.

**Pros:**

- Built-in `start-first` rolling update orchestration inside the Docker daemon.

**Cons:**

- Docker Swarm's internal Raft consensus store and ingress routing mesh add ~80
  MB – 120 MB of continuous daemon memory overhead on the 1 GB `e2-micro` VM.

</details>

---

## <a id="5-assumptions" >5. Assumptions</a>

1. The GCP project has a valid billing account attached to unlock the Compute
   Engine Always Free tier (`1x e2-micro` in `us-central1`, `us-east1`, or
   `us-west1` + 30 GB `pd-standard` disk) and provision the `db-f1-micro` Cloud
   SQL for PostgreSQL instance.
2. The domain (`quadsmith.com`) is registered and its nameservers are delegated
   to a Cloudflare account (Free Tier is sufficient).
3. Administrative access to `ops.quadsmith.com` (`server-ops`) is doubly
   protected at the edge by a **Cloudflare Zero Trust Access policy** and inside
   the binary by `opsauth.NewOpsInterceptor` (`quadsmith_ops.internal_tokens`).
4. All schema migrations deployed via automated CI/CD are backwards-compatible
   (expand-and-contract pattern) so active `blue` containers continue
   functioning during the ~5–10 second window between `make db-migrate`
   completion and the `nginx -s reload` cutover to `green`.
