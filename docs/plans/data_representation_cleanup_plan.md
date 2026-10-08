# Quadsmith Core Architecture Plan

## 1. Overview & Motivation

This plan establishes the core **Protobuf-Driven Relational Database** architecture for Quadsmith from a clean slate. 

To guarantee perfect synchronization across the stack (API, CLI, and Database) and eliminate boilerplate, Protobuf will act as the absolute single source of truth. From these `.proto` definitions, we will statically generate a strict, type-safe, multi-table relational SQL schema. This ensures zero data divergence and allows us to leverage fast, standard relational database features without manual schema management.

---

## 2. Core Architectural Decisions

### 2.1 Protobuf as the Single Source of Truth
All domain entities (`Motor`, `Frame`, `Build`, `User`) will be defined directly as top-level Protobuf messages without any generic envelopes. 
Custom Protobuf options (`sql.proto`) will be used to define relational database constraints (Table Names, Primary Keys, Foreign Keys) directly within the `.proto` files.

### 2.2 One File Per Resource (AIP Alignment)
Following Google API Design Guidelines, each distinct resource type will be fully encapsulated within its own `.proto` file (e.g., `motor.proto`, `build.proto`). Each file will contain the domain message, the request/response messages, and the gRPC service definition for that specific resource.

### 2.3 Automated SQL Schema Generation
Instead of manually maintaining `schema.sql`, we will implement a custom Go script (using `google.golang.org/protobuf/reflect/protoreflect`) that runs during `make generate`. 
This script will read the compiled Protobuf descriptors and output standard `CREATE TABLE` and `ALTER TABLE` statements, directly overwriting `backend/db/schema.sql`.

### 2.4 Declarative Migrations (`psqldef`)
The generated `schema.sql` will be applied to the PostgreSQL database using `psqldef`. This completely eliminates the need to write manual up/down migration scripts. `psqldef` will compute the diff between the generated schema and the live database and apply the necessary alterations.

### 2.5 Strongly-Typed CEL Filtering
Client API requests will continue to use CEL strings for dynamic filtering (e.g., `weight_g > 30`). 
However, instead of compiling these CEL expressions into slow JSONB path operators (`data->>'weight_g'`), the backend `cel2sql` compiler will map them directly to the generated SQL columns (`WHERE weight_g > 30`), allowing the database to utilize standard B-Tree indexes for maximum performance.

### 2.6 Strict CI Enforcement
To guarantee the database schema and Protobuf definitions never drift, `schema.sql` will be committed to source control. 
A `pre-commit` hook will ensure files are generated locally, and the CI pipeline will strictly enforce synchronization by running `make generate && git diff --exit-code`.

---

## 3. Implementation Workstreams

### Workstream 1: Custom Protobuf SQL Options
**Goal**: Define the extensions required to map Protobuf messages to relational tables.

1. **Create `proto/quadsmith/_sql.proto`**:
   - Define custom extensions for `google.protobuf.MessageOptions` (`TableOptions table = 51000;`).
   - Define custom extensions for `google.protobuf.FieldOptions` (`ColumnOptions column = 51000;`).
2. **Define Domain Models**:
   - Create distinct `.proto` files for each resource (e.g., `motor.proto`, `build.proto`).
   - Apply the custom SQL options to the domain models (e.g., `option (quadsmith.sql.table) = {name: "motors"};`).

### Workstream 2: Schema Generator Script
**Goal**: Build the Go tool that translates Protobuf definitions into SQL.

1. **Implement `cmd/protoc-gen-sql/main.go`**:
   - Use `protoreflect` to iterate over all messages in the `quadsmith` package.
   - Map Protobuf types to PostgreSQL types (`string` -> `TEXT`, `int32` -> `INTEGER`, `float` -> `REAL`).
   - Parse custom options to generate Primary Keys and Foreign Keys.
   - Output the formatted SQL directly to `backend/db/schema.sql`.
2. **Update Build Tooling**:
   - Add the generator script to the `Makefile` `generate` target.

### Workstream 3: Repository & Query Engine Implementation
**Goal**: Build a standard Go backend using relational SQL columns.

1. **Implement `cel2sql` Compiler**:
   - Write a CEL AST visitor to map Protobuf fields directly to SQL columns.
2. **Implement Store Interfaces**:
   - Implement repository layers (`backend/api/internal/store/*`) that query the specific generated tables (`motors`, `builds`).
   - Use standard `JOIN`s for resolving relationships (e.g., fetching a `Build` and its `Motors`).

### Workstream 4: Synchronization & CI Enforcement
**Goal**: Ensure the generated schema and diagrams are strictly maintained.

1. **Update `backend/db/diagram.mermaid`**:
   - Sync the diagram to reflect the new strict relational schema.
2. **Add Git Hooks**:
   - Add a `pre-commit` hook that automatically runs `make generate` and stages `schema.sql`.
3. **Configure CI Pipeline**:
   - Add a step to the GitHub Actions / CI pipeline that runs `make generate` followed by `git diff --exit-code` to fail the build if the checked-in schema is outdated.

### Workstream 5: Tests & Seed Data
**Goal**: Establish comprehensive test coverage and clean seeding capabilities.

1. **Create Seed Data**:
   - Write `motors.textproto`, `builds.textproto`, etc. to populate the database with realistic sample data.
2. **Implement `seed/main.go`**:
   - Write a seed script that inserts directly into the generated tables, maintaining foreign key insertion order (e.g., insert hardware before builds).
3. **Write Unit Tests**:
   - Write `backend/engines/compatibility/checker_test.go` and other tests using the newly defined domain models.
