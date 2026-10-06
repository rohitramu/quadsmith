# Data Representation Cleanup Plan

## 1. Overview & Motivation

Quadsmith currently experiences architectural friction and data divergence across three layers:
1. **The Legacy Conceptual Model** (`backend/db/diagram.mermaid`, `docs/hardware_taxonomy_and_build_data_model.md`): A complex 20+ entity relational design carried over from an earlier TypeScript/Prisma architecture.
2. **The Active Protobuf API** (`proto/quadsmith/`): A domain model unifying 20 component types into a single `Component` message (`oneof`), alongside `Build`, ConnectRPC services, and a mixture of static enums and loose string IDs.
3. **The PostgreSQL Database** (`backend/db/schema.sql`): A hybrid database where all component attributes are serialized as JSON inside a `data JSONB` column, while legacy relational tables (`flight_stacks`, `vtx_configurations`, `tags`) and redundant columns (`release_date`, `weight_g`, `name`) sit unused or unpopulated.

This plan defines the steps to simplify the data architecture, reconcile the Protobuf definitions with the database schema, eliminate dead code and ghost abstractions, and establish Protobuf as the single source of truth for component schemas and build configurations.

---

## 2. Core Architectural Decisions

* **Protobuf as Domain Schema Authority**: Protobuf (`proto/quadsmith/*.proto`) defines all component types, builds, metrics, and API contracts. The database stores these messages as structured `JSONB` rather than maintaining redundant or incomplete parallel relational columns.
* **Lean Resource Identity Registry**: The PostgreSQL `resources` table serves as the universal index of IDs, UUID v7 primary keys, and resource types (`COMPONENT`, `BUILD`, `USER`). Top-level domain tables (`components`, `builds`) reference `resources(uuid)`.
* **Elimination of Ghost Sub-Assemblies**: Sub-assemblies (`FlightStack`, `VtxConfiguration`, `ReceiverConfiguration`) currently exist only as database tables and documentation, with no Protobuf messages, API endpoints, or seed data. These will be retired in favor of direct component references on the `Build` aggregate.
* **Complete `Build` Model**: `Build` must represent a fully functional drone, including direct references to flight controllers, ESCs, and batteries (which are currently missing, preventing end-to-end evaluation).
* **Schema-Driven CEL Environment**: Instead of manually registering a handful of hardcoded fields for CEL filtering, the CEL environment will dynamically derive its type registry from the Protobuf descriptor.

---

## 3. Workstreams

### Workstream 1: Database Schema Simplification & Sync
**Goal**: Streamline `backend/db/schema.sql` to eliminate unused tables and duplicate columns, and update `backend/db/diagram.mermaid` in tandem.

1. **Prune Redundant Columns in `components`**:
   - Current table contains `name`, `release_date`, `weight_g`, and `manufacturer_uuid`. During seeding and updates, `release_date` and `weight_g` are never written to SQL columns, and reads bypass them completely in favor of `data JSONB`.
   - Retain: `uuid` (PK, FK `resources.uuid`), `type` (indexed discriminator), and `data` (`JSONB NOT NULL`, indexed with GIN `jsonb_path_ops`).
2. **Prune Ghost Tables**:
   - Drop `flight_stacks`, `vtx_configurations`, and `receiver_configurations` (unimplemented sub-assemblies).
   - Drop `manufacturers` (unseeded, replaced by `company_ids` in proto).
   - Drop `tags` and `build_tags` (unreferenced by Protobuf or API).
3. **Simplify `builds` Table**:
   - Replace the 8 individual component FK columns (`frame_uuid`, `motor_uuid`, etc.) with `uuid` (PK, FK `resources.uuid`), optional `user_uuid`, and `data JSONB NOT NULL`.
   - Component relations remain queryable via GIN indexing on `builds.data`.
4. **Synchronize Diagram**:
   - Update `backend/db/diagram.mermaid` to match `backend/db/schema.sql`, reflecting the active entities (`Resource`, `User`, `Component`, `Build`) without dropping necessary foreign key relationships.
5. **Apply via Declarative Migration**:
   - Run `psqldef` against local Postgres to apply the clean schema.

---

### Workstream 2: Protobuf & Domain Model Realignment
**Goal**: Complete the `Build` model, resolve enum/string inconsistencies, and clean up dead types.

1. **Direct Component References in `Build` (`proto/quadsmith/build.proto`)**:
   - Add explicit references for:
     - `string flight_controller_id = 11;`
     - `string esc_id = 12;`
     - `string battery_id = 13;`
     - `string vtx_id = 14;`
     - `string receiver_id = 15;`
     - `repeated string antenna_ids = 16;`
   - Remove or deprecate unused sub-assembly fields (`flight_stack_id`, `vtx_config_id`, `receiver_config_id`).
2. **Taxonomy & Dictionary Standardization**:
   - Identify closed domains and model them consistently as Protobuf enums:
     - Mounting hole patterns (e.g., `MOUNT_20X20`, `MOUNT_30_5X30_5`, `MOUNT_25_5X25_5`, `MOUNT_16X16`, `MOUNT_9X9`, `MOUNT_12X12`).
     - RF protocols (e.g., `PROTOCOL_EXPRESSLRS`, `PROTOCOL_CROSSFIRE`, `PROTOCOL_TRACER`).
   - For open-ended or high-variance attributes (e.g., frequencies, firmwares), standardize on normalized string identifiers (e.g., `"2.4ghz"`, `"5.8ghz"`).
3. **Regenerate Bindings**:
   - Run `buf generate` to produce updated Go structs in `backend/api/gen/quadsmith`.

---

### Workstream 3: Data Access Layer & CEL Engine Overhaul
**Goal**: Make the backend repository and CEL query engine robust and schema-aware.

1. **Dynamic CEL Type Registration (`backend/api/internal/store/components.go`)**:
   - Replace the static 6-variable list in `cel.NewEnv(...)` with dynamic registration using the Protobuf descriptor (`pb.Component{}.ProtoReflect().Descriptor()`).
   - Enable full typed access for all component fields (`motor.kv_rating`, `weight_g`, `battery.cell_count_s`, `propeller.diameter_mm`, etc.).
2. **CEL-to-SQL Compiler Robustness (`backend/api/pkg/cel2sql`)**:
   - Update `FieldMapper` to support:
     - Root-level fields (e.g., `weight_g` $\rightarrow$ `(c.data->>'weight_g')::numeric`).
     - Relational fields (`id` $\rightarrow$ `r.id`, `type` $\rightarrow$ `c.type`).
     - Nested profile fields (`motor.kv_rating` $\rightarrow$ `(c.data->'motor'->>'kv_rating')::numeric`).
   - Replace suffix-guessing type casts with Protobuf-derived reflection types.
3. **Dead Code Removal**:
   - Remove duplicate/unreachable returns in `getComponentTypeString` across `backend/api/internal/store/components.go` and `backend/api/cmd/seed/main.go`.
4. **Build Store Simplification (`backend/api/internal/store/builds.go`)**:
   - Simplify `ListBuilds`, `GetBuild`, `CreateBuild`, and `UpdateBuild` to read and write `data JSONB` without managing 5 redundant SQL FK subqueries.

---

### Workstream 4: Engine Integration & API End-to-End Flow
**Goal**: Enable builds to be evaluated and checked directly as unified assemblies.

1. **Build-Level Evaluation**:
   - Update `EvaluateBuild` in `backend/api/internal/server/server.go` to support passing a `build_id` (in addition to explicit component IDs).
   - The server loads the build from `BuildStore`, resolves its referenced components (frame, motors, props, FC, ESC, and battery), and executes the physics calculation in `backend/engines/evaluator`.
2. **Build-Level Compatibility Check**:
   - Allow `CheckCompatibility` to accept a `build_id` and evaluate all associated parts.
3. **CLI Integration**:
   - Update `frontend/cli/main.go` so `cli builds evaluate <build-id>` passes the real build ID rather than splitting comma-separated strings as a temporary workaround.

---

### Workstream 5: Seed Pipeline & Test Suite Stabilization
**Goal**: Restore passing test suites and ensure seed data validates cleanly against all rules.

1. **Fix Broken Test Suite**:
   - Update `backend/engines/compatibility/checker_test.go` to construct components using `ResourceMetadata` (`Resource: (&pb.ResourceMetadata_builder{Id: proto.String(...)}).Build()`).
   - Verify all tests pass with `go test ./backend/engines/... ./backend/api/... ./frontend/cli/...`.
2. **Update Seed Data**:
   - Update `backend/db/seeds/components.textproto` to populate mount patterns and UART counts where applicable.
   - Update `backend/db/seeds/builds.textproto` to include flight controllers, ESCs, and batteries.
3. **Update Seed Loader (`backend/api/cmd/seed/main.go`)**:
   - Align upsert statements with the simplified schema.
