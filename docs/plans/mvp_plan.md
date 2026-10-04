# Quadsmith MVP Plan

This document outlines the multi-stage implementation plan for the Quadsmith MVP. The focus is strictly on establishing a solid foundation for the Compatibility Engine and Build Evaluator, component/build CRUD operations, and providing a CLI to interact with the API. There will be **no GUI** and **no Authentication/User management** in the MVP phase. 

## 1. Core Architecture Decisions

*   **Language & Runtime**: Pure **Go (Golang)**. The entire backend (Engines, API) and CLI will be built in Go for maximum performance, minimal memory footprint, and native support for our core libraries.
*   **Pragmatic Database Agnosticism**: We use a clean Repository Pattern in Go (e.g., `ComponentRepository`) so business logic isn't tied to SQL strings. However, our Postgres implementation leverages native Relational features (Foreign Keys, JOINs) to handle data integrity and N+1 query performance rather than reinventing the wheel in application space.
*   **Monorepo Structure**: Standard Go workspace (`go.work`) separating modules like `/backend/api`, `/backend/engines`, and `/frontend/cli`. The database schemas live purely in `/db` (not a Go module).
*   **Data Modeling**: PostgreSQL, with component specifications strictly typed and encoded as Protobuf (`JSONB` in the DB). Raw JSON has been eliminated from the proto specs (e.g., `bec_outputs`, `uart_connections` are now strictly typed messages). 
*   **Draft Builds**: Allowed natively at the DB level by making component foreign keys on the `Build` table nullable.
*   **Sub-assemblies**: Modeled as distinct CRUD resources (`FlightStack`, `VtxConfiguration`, `ReceiverConfiguration`), allowing reuse across different builds.
*   **Filtering**: Pure CEL (Common Expression Language) for querying component attributes, using the native, highly-optimized Go runtime.
*   **Scraping**: Isolated from the core application. Scraping tools will be housed in an isolated package to keep production dependencies clean.

## 2. Technology Stack Evaluation

The technology list is strictly curated to provide immense value, high performance, and alignment with our Protobuf-first architecture.

*   **Go (Golang)**: The foundation. Provides blazing fast execution for the Build Evaluator physics and Compatibility loops, built-in concurrency (goroutines), and zero tooling fatigue (formatting, testing, and building are built-in).
*   **PostgreSQL**: The database. Chosen for its robust "Hybrid" capabilities—acting as a rock-solid relational DB for Users/Builds while offering best-in-class `JSONB` support for unstructured Protobuf component payloads.
*   **pgx & sqlc**: Replaces heavy ORMs like Prisma. `pgx` is the high-performance Postgres driver. `sqlc` generates type-safe Go functions from raw SQL queries (`queries.sql`), giving us ORM-like safety without sacrificing performance or control.
*   **golang-migrate / goose**: Handles schema migrations via plain SQL files (`001_init.up.sql`), eliminating opaque declarative magic and giving us full control over the DB structure.
*   **@bufbuild/protobuf & Buf**: Canonical Protobuf management. We use `buf` to automatically generate Go types and interfaces from our `quadsmith/v1` proto files.
*   **google/cel-go**: The native Go implementation of CEL, originally built by Google. Extremely fast and perfectly suited for our dynamic component filtering.
*   **connect-go**: The native Go implementation of ConnectRPC. Generates API handlers from our `.proto` files automatically. It supports HTTP/1.1 and JSON out of the box, making it perfectly testable via curl or Postman while preserving gRPC compatibility.
*   **spf13/cobra**: The industry-standard Go CLI framework (used by Kubernetes, Docker). We will use it to build the `quadsmith` CLI application, interacting seamlessly with the generated ConnectRPC client.

## 3. Implementation Stages

### Stage 1: Protobuf & Schema Migration (Completed / Adapting to Go)
*   Finalize `quadsmith/v1/hardware.proto` to strictly type all properties.
*   Add evaluation output protos (`evaluator.proto`, `compatibility.proto`).
*   Tear down Node.js/Prisma configurations and initialize the `go.work` workspace.
*   Translate the Prisma schema into raw Postgres SQL (`001_init.up.sql`) using standard Foreign Keys, and set up `sqlc` to generate the Go database access layer.

### Stage 2: Core Compatibility & Evaluation Engines (Current)
*   **Goal**: Implement the pure physics and logical checks in Go.
*   **Package**: `/backend/engines`
*   **Tasks**:
    1.  Create the Build Evaluator (TWR, flight times, top speed).
    2.  Create the Compatibility Checker (physical mount mismatches, voltage range checks, protocol checks).
    3.  Unit test everything rigorously with Go's built-in `testing` package. This stage depends on zero database code.

### Stage 3: Data Access & CEL Querying
*   **Goal**: Connect the DB to the application logic.
*   **Package**: `/backend/api` (internal database repositories)
*   **Tasks**:
    1.  Rewrite the seed script in Go to populate realistic, valid protobuf components.
    2.  Implement the CEL evaluator logic using `google/cel-go` to parse `query="data.kv_rating > 2000"` and filter Postgres `JSONB` result sets.
    3.  Implement Go Repositories (CRUD wrappers) for components, sub-assemblies, and builds using `sqlc`.

### Stage 4: RPC API Server (Go + ConnectRPC)
*   **Goal**: Expose the backend over a network protocol.
*   **Package**: `/backend/api`
*   **Tasks**:
    1.  Define API service endpoints in `quadsmith/v1/api.proto` (e.g., `GetComponent`, `ListComponents`, `EvaluateBuild`).
    2.  Set up the Go HTTP server utilizing `connect-go` handlers.
    3.  Wire up the DB CRUD operations and the Engine packages to the RPC handlers.

### Stage 5: The CLI Client
*   **Goal**: Provide a functional way to interact with the platform.
*   **Package**: `/frontend/cli`
*   **Tasks**:
    1.  Generate a ConnectRPC client in Go.
    2.  Build the command-line interface using `spf13/cobra`.
    3.  Implement commands: `quadsmith search "data.kv_rating > 2000"`, `quadsmith check-build <build-id>`, and `quadsmith eval <build-id>`.

### Stage 6: Scraping & Tooling (Optional MVP Phase)
*   **Goal**: Isolate extraction tooling from core platform runtime.
*   **Tasks**:
    1.  Build Go-based scraping utilities to auto-generate `.bin` files containing valid `quadsmith.v1.Component` payloads from manufacturer websites.
