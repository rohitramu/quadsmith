# <a id="infrastructure-alignment-and-container-architecture" >Infrastructure Alignment and Container Architecture</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Language and Framework Project Organization (src/go and src/react)](#language-and-framework-project-organization-src-go-and-src-react)
  - [Per-App Production Container Packaging](#per-app-production-container-packaging)
  - [Pre-Seeded Docker Compose Sandbox and Test Harness](#pre-seeded-docker-compose-sandbox-and-test-harness)
  - [Scheduled Account Purge Daemon Registration and Configurable Cadence](#scheduled-account-purge-daemon-registration-and-configurable-cadence)
  - [Continuous Integration Workflow and Build Verification](#continuous-integration-workflow-and-build-verification)
- [4. Alternatives considered](#4-alternatives-considered)
  - [Role-Based Backend and Frontend Directory Split](#role-based-backend-and-frontend-directory-split)
  - [Platform-Based Web Directory Naming (src/web)](#platform-based-web-directory-naming-src-web)
  - [Centralized Deploy Directory](#centralized-deploy-directory)
  - [Layered Base and Overlay Configuration Architecture](#layered-base-and-overlay-configuration-architecture)
- [5. Assumptions](#5-assumptions)
- [6. Future work](#6-future-work)
  - [Automated Production Release Pipeline](#automated-production-release-pipeline)

---

## <a id="1-background" >1. Background</a>

Recent architectural refactors decoupled static asset delivery from the backend
API servers, introducing a standalone HTTP/2 reverse-proxy binary in
[`server-web`](../../src/go/server-web/main.go) and implementing an account
soft-deletion and hard-purge lifecycle in
[`src/go/shared/pkg/auth/job.go`](../../src/go/shared/pkg/auth/job.go).

However, several structural, operational, and container configurations drifted:

- The legacy directory taxonomy divided code by conceptual role into
  `src/backend` and `src/frontend`. This created an architectural boundary
  violation: the CLI applications ([`src/go/cli`](../../src/go/cli) and
  [`src/go/cli-ops`](../../src/go/cli-ops)) are compiled Go binaries that depend
  directly on internal Go packages inside `src/backend/shared`, forcing Go code
  and `go.work` declarations to bridge across the role-based hierarchy.
- The legacy production container specification in `src/Dockerfile` only
  compiled `server-api` and retained deprecated static asset environment
  variables (`QS_SERVER_STATIC_DIR`).
- Configuration textprotos for the web server remained stranded under
  `src/frontend/web/config`, while the server implementation resided under
  `src/backend/server-web`.
- The automated account purge procedure remained unregistered in the job daemon
  schedule configurations and ops catalog.
- The CI configuration in
  [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) referenced legacy
  paths and lacked Markdown formatting validations.

---

## <a id="2-goal" >2. Goal</a>

Establish a cohesive, language-aligned architecture under `src/go` and
`src/react` that makes cross-app code sharing clean and natural; package each
service with a dedicated production container; accelerate local and CI testing
with a pre-seeded multi-container Docker Compose environment; and activate the
automated account purge daemon.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Reorganize `src/` by language into `src/go/` and `src/react/`.
> - Package each service with its own dedicated production Dockerfile.
> - Relocate web server configs to `src/go/server-web/config/`.
> - Replace monolithic sandbox with pre-seeded Docker Compose in `test/`.
> - Schedule account purge daemon to execute every 8 hours.

This proposal realigns the repository structure by organizing application code
and libraries according to their implementation language and ecosystem rather
than arbitrary role boundaries. Grouping all Go modules under `src/go/`
eliminates awkward cross-tier dependencies—such as Go CLIs importing packages
from a `backend/` folder—and allows all Go tooling (`go.work`, `gopls`,
`golangci-lint`) to operate within a single, dedicated directory tree.

Similarly, placing web frontend applications under `src/react/` isolates
Node/TypeScript dependencies, establishing a clean workspace structure for
future frontend expansion. Production packaging transitions to dedicated
per-service Dockerfiles, and local sandbox testing adopts a build-time
pre-seeded Docker Compose topology that delivers sub-two-second startup times.

### <a id="language-and-framework-project-organization-src-go-and-src-react" >Language and Framework Project Organization (src/go and src/react)</a>

The repository structure under [`src/`](../../src) will be reorganized cleanly
by programming language and framework:

```text
src/
├── db/                                # Canonical PostgreSQL DDL schemas & Mermaid diagrams
│   ├── quadsmith_api.sql, quadsmith_ops.sql
│   └── quadsmith_api.mermaid, quadsmith_ops.mermaid
│
├── go/                                # All Go modules, shared libraries, and tooling
│   ├── go.work                        # Go workspace defining all local modules
│   ├── shared/                        # Shared Go domain, CEL engine, auth, db, protos
│   │   ├── pkg/, gen/, cmd/
│   │   └── go.mod (quadsmith/shared)
│   ├── server-api/                    # ConnectRPC API server (Go) + Dockerfile
│   │   ├── main.go, config/
│   ├── server-ops/                    # Operations & admin server (Go) + Dockerfile
│   │   ├── main.go, config/
│   ├── server-jobs/                   # Scheduled jobs daemon (Go) + Dockerfile
│   │   ├── main.go, config/
│   ├── server-web/                    # Standalone Web HTTP/2 reverse proxy (Go) + Dockerfile
│   │   ├── main.go, config/
│   ├── cli/                           # qs Go CLI client
│   │   ├── main.go, config/
│   ├── cli-ops/                       # qsops Go CLI client
│   │   ├── main.go, config/
│   └── build_deps/                    # Toolchain Go dependencies
│       └── tools.go, go.mod
│
├── proto/                             # Canonical Protobuf schema definitions & buf config
│   ├── buf.gen.yaml, buf.yaml
│   └── quadsmith/                     # API, config, common, sql, data protobuf packages
│
└── react/                             # All React / TypeScript web applications
    ├── app-web/                       # Primary React 19 SPA (Vite, Tailwind CSS)
    │   ├── src/, public/, package.json, vite.config.ts
    ├── shared/                        # Shared UI components, hooks, lib, & generated protos
    │   ├── src/, package.json, tsconfig.json
    └── (future) app-ops/              # Future administrative & review web portal
```

Key architectural benefits of this structure:

1. **Zero Cross-Tier Boundary Violations**:
   - `cli` and `cli-ops` are Go applications that import
     `quadsmith/shared/pkg/...`. Under `src/go/`, `shared` is a peer module in
     the exact same language workspace. There is no confusing dependency from a
     "frontend" folder into a "backend" folder.
2. **Unified Go Workspace Context**:
   - [`src/go/go.work`](../../src/go/go.work) coordinates all Go modules. Go
     language servers (`gopls`), linters, and `go work vendor` operate
     exclusively within `src/go/` without scanning `node_modules` or TypeScript
     files.
3. **Dedicated Web Server Configuration**:
   - The web server textproto configuration files are relocated from
     `src/frontend/web/config/` to
     [`src/go/server-web/config/`](../../src/go/server-web/config), making
     `server-web` completely self-contained like the other Go servers.
4. **Clean Multi-App Frontend Expansion**:
   - React applications reside under `src/react/` with standard peer naming
     (`app-web/`). When adding an operations portal (`app-ops/`) or shared
     components (`shared/`), they configure standard npm/pnpm workspaces without
     affecting the Go workspace.

#### <a id="alternatives-considered-directory-reorg" >Alternatives considered</a>

##### <a id="role-based-backend-and-frontend-directory-split" >Role-Based Backend and Frontend Directory Split</a>

Retain the top-level categorization dividing code into `src/backend/` and
`src/frontend/`.

**Pros:**

- Matches conventional web service taxonomy grouping servers and clients.
- Requires fewer directory moves during migration.

**Cons:**

- Misplaces Go CLIs in `frontend/` while forcing them to import
  `backend/shared`.
- Forces `go.work` to span across disparate top-level architectural tiers.

##### <a id="platform-based-web-directory-naming-src-web" >Platform-Based Web Directory Naming (src/web)</a>

Name the frontend tree `src/web/` instead of `src/react/`.

**Pros:**

- Identifies the delivery platform rather than a specific UI framework.
- Neutral if non-React frontend technologies are introduced.

**Cons:**

- Asymmetric with `src/go/`, which names the implementation language.
- Obscures the specific React 19 toolchain powering the frontend applications.

### <a id="per-app-production-container-packaging" >Per-App Production Container Packaging</a>

Each server application under [`src/go`](../../src/go) will own its dedicated,
minimal, single-process `Dockerfile`. The obsolete, monolithic `src/Dockerfile`
will be removed.

1. **API Server (`src/go/server-api/Dockerfile`)**:
   - Multi-stage build compiling `server-api` with `CGO_ENABLED=0`.
   - Minimal Alpine runtime copying only `/app/server-api` and
     `src/go/server-api/config`.
   - Exposes port 8080.
2. **Ops Server (`src/go/server-ops/Dockerfile`)**:
   - Minimal Alpine runtime copying only `/app/server-ops` and
     `src/go/server-ops/config`.
   - Exposes port 8081.
3. **Job Server (`src/go/server-jobs/Dockerfile`)**:
   - Background daemon container copying only `/app/server-jobs` and
     `src/go/server-jobs/config`.
4. **Web Server (`src/go/server-web/Dockerfile`)**:
   - Multi-stage compilation: Node.js builder executing `npm run build` in
     [`src/react/app-web`](../../src/react/app-web), and Go builder compiling
     `server-web` from `src/go`.
   - Minimal Alpine runtime copying the compiled binary, bundled static assets,
     and `src/go/server-web/config`.
   - Exposes port 3000 / 443.

#### <a id="alternatives-considered-per-app-dockerfiles" >Alternatives considered</a>

##### <a id="single-multi-stage-dockerfile" >Single Root Multi-Stage Dockerfile</a>

Maintain a single `src/Dockerfile` utilizing BuildKit `--target` arguments to
select output images.

**Pros:**

- Avoids repeating Go and Alpine setup boilerplate across 4 Dockerfiles.
- Shares common compilation cache layers in a single file.

**Cons:**

- Violates self-containment by decoupling deployment specifications from app
  folders.
- Forces all service packaging modifications to touch a single shared file.

### <a id="pre-seeded-docker-compose-sandbox-and-test-harness" >Pre-Seeded Docker Compose Sandbox and Test Harness</a>

Local sandbox development (`make sandbox`) and integration testing
(`make test-e2e`) will transition to Docker Compose, housed cleanly in
[`test/sandbox/docker-compose.yaml`](../../test/sandbox/docker-compose.yaml).

To overcome the standard 9-second cold database initialization penalty, a
dedicated test database image
([`test/sandbox/db/Dockerfile`](../../test/sandbox/db/Dockerfile)) will
pre-apply SQL DDL schemas and pre-populate catalog data at image build time
using the database seeder binary.

Empirical testing validates that this reduces Compose startup from 9,162 ms to
1,666 ms while delivering full 1:1 multi-container network isolation:

| Dimension              | Legacy Monolithic Sandbox | Standard Compose (Cold Boot) | Pre-Seeded Compose (Proposed) |
| :--------------------- | :------------------------ | :--------------------------- | :---------------------------- |
| **Startup to Healthy** | 1,555 ms                  | 9,162 ms                     | **1,666 ms**                  |
| **Teardown Time**      | 257 ms                    | 973 ms                       | **781 ms**                    |
| **Containers**         | 1 (monolithic)            | 5 (with seeder)              | **5 (discrete services)**     |
| **Production Parity**  | Low (localhost sharing)   | High                         | **High (DNS discovery)**      |

The test harness scripts [`test/start_sandbox.sh`](../../test/start_sandbox.sh)
and [`test/run_sandbox_tests.sh`](../../test/run_sandbox_tests.sh) will be
refactored to orchestrate `docker compose -f test/sandbox/docker-compose.yaml`.

#### <a id="alternatives-considered-compose-sandbox" >Alternatives considered</a>

##### <a id="retain-monolithic-test-dockerfile" >Retain Monolithic test/Dockerfile</a>

Keep running Postgres, seeder, and all 4 Go servers within a single container.

**Pros:**

- Slightly faster teardown (257 ms vs. 781 ms).
- Requires only the core `docker` command without Docker Compose.

**Cons:**

- Fails to simulate multi-container network topologies and DNS resolution.
- Masked port binding collisions and shared localhost assumptions.

##### <a id="cold-seeding-docker-compose" >Runtime Cold Seeding in Docker Compose</a>

Boot a blank `postgres:15-alpine` container and run a separate seeder service.

**Pros:**

- Uses the vanilla upstream Postgres image without a custom test build step.

**Cons:**

- Imposes an 82% performance penalty (~9.2s startup) on every test run.
- Introduces startup race conditions between Postgres init and client services.

### <a id="scheduled-account-purge-daemon-registration-and-configurable-cadence" >Scheduled Account Purge Daemon Registration and Configurable Cadence</a>

The soft-deleted account scrubber in
[`HandlePurgeSoftDeletedAccounts`](../../src/go/shared/pkg/auth/job.go) will be
registered into the operational catalog and scheduled daemon:

1. **Ops Service Catalog
   ([`service.go`](../../src/go/server-ops/pkg/ops/service.go))**:
   - Register `purge_soft_deleted_accounts` (procedure
     `PurgeSoftDeletedAccounts`) in `ListScheduledJobs` with an initial interval
     of `8h`.
2. **Environment Configuration**:
   - Add the schedule entry to all 4 job server textproto files:
     - [`sandbox.textproto`](../../src/go/server-jobs/config/sandbox.textproto)
     - [`test.textproto`](../../src/go/server-jobs/config/test.textproto)
     - [`staging.textproto`](../../src/go/server-jobs/config/staging.textproto)
     - [`prod.textproto`](../../src/go/server-jobs/config/prod.textproto)
   - Configured with `interval: "8h"` by default, allowing operator adjustment
     per environment without code modification.

```textproto
schedules {
  name: "purge_soft_deleted_accounts"
  procedure: "PurgeSoftDeletedAccounts"
  interval: "8h"
}
```

#### <a id="alternatives-considered-purge-cadence" >Alternatives considered</a>

##### <a id="daily-twenty-four-hour-purge" >Daily 24-Hour Purge Cadence</a>

Run the purge scrubber once every 24 hours.

**Pros:**

- Minimizes background database queries.

**Cons:**

- Accounts that clear the 30-day grace period remain held in an inactive state
  for up to an extra day before final handle release.

### <a id="continuous-integration-workflow-and-build-verification" >Continuous Integration Workflow and Build Verification</a>

The GitHub Actions workflow in
[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) and the root
[`Makefile`](../../Makefile) will be aligned with current repository standards:

- Update Go installation step to Go `1.27`.
- Update web working directory from `src/frontend/web/app` to
  `src/react/app-web`.
- Update all Go module paths from `src/backend/...` and `src/frontend/cli...` to
  `src/go/...`.
- Update `src/proto/buf.gen.yaml` output targets to `../go/shared/gen`, `../db`,
  and `../react/shared/src/gen`.
- Add Prettier and Markdownlint verification steps matching the root
  [`Makefile`](../../Makefile).
- Ensure `make generate`, `make build`, and `make test` pass hermetically.

#### <a id="alternatives-considered-ci-sync" >Alternatives considered</a>

##### <a id="ci-docker-in-docker-compose" >Running Full Compose in CI Build Step</a>

Execute `make test-e2e` on every pull request push.

**Pros:**

- Verifies container networking on every commit.

**Cons:**

- Increases CI cycle duration compared to in-memory unit tests and hermetic Go
  package testing.

---

## <a id="4-alternatives-considered" >4. Alternatives considered</a>

### <a id="centralized-deploy-directory" >Centralized Deploy Directory</a>

Move all container specifications and environment textprotos to a root `deploy/`
folder (`deploy/docker/`, `deploy/config/`), stripping packaging from `src/`.

**Pros:**

- Strictly separates developer application code from DevOps infrastructure.
- Provides a natural repository-root context for all Docker builds.

**Cons:**

- Breaks locality: developers must navigate between `src/` and `deploy/` when
  tuning RPC job procedures or server timeouts.
- Breaks CLI compilation, which relies on `go:embed` to compile textprotos into
  immutable binaries.

### <a id="layered-base-and-overlay-configuration-architecture" >Layered Base and Overlay Configuration Architecture</a>

Define application domain invariants in a single `base.textproto` per app and
merge minimal environment overlays using `proto.Merge` at runtime.

**Pros:**

- Eliminates repeating identical token TTLs and limits across 4 environment
  files.
- Makes environment files exceptionally small (5–10 lines).

**Cons:**

- Violates Quadsmith's core principle of zero runtime magic and fully explicit
  configuration declarations.
- Complicates build-time configuration verification tools (`check-config`).

---

## <a id="5-assumptions" >5. Assumptions</a>

1. Docker and the Docker Compose plugin (`docker compose`) are installed on
   development and CI host machines.
2. The Table-Per-Type (TPT) PostgreSQL schema and seed dataset fit comfortably
   within the pre-seeded test database container layers.
3. Network access to `smtp.resend.com` is only required in staging and
   production; sandbox and test environments continue using console email stubs.

---

## <a id="6-future-work" >6. Future work</a>

### <a id="automated-production-release-pipeline" >Automated Production Release Pipeline</a>

Implement automated image publishing in GitHub Actions to push tagged per-app
production containers to Google Artifact Registry upon merging to `main`.
