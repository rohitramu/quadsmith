# <a id="user-created-and-managed-components" >User-Created and Managed Components</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [True Table-Per-Type (TPT) Inheritance Architecture](#true-table-per-type-tpt-inheritance-architecture)
  - [Creator-Owned with Administrator Override Model](#creator-owned-with-administrator-override-model)
  - [20-Collection Scope and Per-Resource ConnectRPC CRUD API](#20-collection-scope-and-per-resource-connectrpc-crud-api)
  - [Steam-Style Hybrid Deletion and Archive Visibility](#steam-style-hybrid-deletion-and-archive-visibility)
  - [Deterministic ID Slug Generation and Immutable URLs](#deterministic-id-slug-generation-and-immutable-urls)
  - [Strict Domain Validation and Author Attribution](#strict-domain-validation-and-author-attribution)
  - [Responsive Dynamic Modal Forms, Forge Quick-Add, Profile Tab, and Full CLI CRUD](#responsive-dynamic-modal-forms-forge-quick-add-profile-tab-and-full-cli-crud)
- [4. Alternatives considered](#4-alternatives-considered)
- [5. Assumptions](#5-assumptions)
- [6. Future work](#6-future-work)

---

## <a id="1-background" >1. Background</a>

In Quadsmith, physical drone hardware specifications, pilot gear, and companion
firmware profiles are cataloged across 20 distinct relational tables and served
via dedicated ConnectRPC services. Currently, component data is read-only for
regular users, populated exclusively via pre-compiled
[seed textprotos](../../test/seed_data) and maintenance commands.

Furthermore, investigation of the current architecture reveals an underlying
data desynchronization flaw between the domain tables (such as
[`builds`](../../src/backend/db/quadsmith_api.sql#L280-L310) and
[`motors`](../../src/backend/db/quadsmith_api.sql#L230-L255)) and the unified
[`entities`](../../src/backend/db/quadsmith_api.sql#L313-L330) registry. More
than seven columns (`name`, `description`, `primary_display_image`,
`reference_links`, `media`, and author info) are duplicated between tables. In
[`DeleteBuild`](../../src/backend/shared/cmd/protoc-gen-server/main.go#L418-L424),
archiving sets `entities.is_archived = true`, but because the `builds` table
lacks an `is_archived` column and `ListBuilds` queries `builds` directly,
archived builds continue to leak into public listings.

Enabling pilots to author components provides the ideal opportunity to eliminate
this duplication by establishing a clean, unified **Table-Per-Type (TPT)
Inheritance** architecture across the entire platform.

---

## <a id="2-goal" >2. Goal</a>

Enable authenticated Quadsmith users to create, view, update, and manage
components across all 20 catalog categories via both the Web application and the
CLI. Establish a clean Table-Per-Type (TPT) inheritance schema where the unified
`entities` table is the single canonical source of truth for identity,
ownership, and common metadata, completely eliminating data desynchronization
while maintaining sub-millisecond query performance and strict compatibility
rules.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Adopt Table-Per-Type (TPT) inheritance: `entities` canonically owns
>   identity, ownership, and metadata; subtype tables store only specs.
> - Eliminate redundant columns (`name`, `description`, `media`, `user_uuid`,
>   `links`) across `builds` and all 20 component tables.
> - Authorize operations via creator-ownership with admin overrides, logging
>   `user_uuid` on `entities`.
> - Implement Steam-style hybrid deletion: hard delete if unreferenced, or
>   delist/archive via `entities.is_archived` if referenced.
> - Provide schema-driven modal editors in the web UI, quick-add shortcuts in
>   the Forge wizard, and CLI CRUD commands.

```text
               +--------------------------------------------+
               |                  entities                  |
               |--------------------------------------------|
               | uuid (PK, UUIDv7)                          |
               | collection ('motors', 'frames', 'builds')  |
               | id (unique slug, e.g. 'tmotor-f80-pro')    |
               | name, description                          |
               | author_or_manufacturer                     |
               | user_uuid (FK -> users.uuid)               | <--- Single Source of Truth
               | is_archived, created_at, updated_at        |
               +---------------------+----------------------+
                                     | 1:1 (Shared UUID)
            +------------------------+------------------------+
            |                                                 |
            v                                                 v
+-----------------------+                         +-----------------------+
|        motors         |                         |        builds         |
|-----------------------|                         |-----------------------|
| uuid (PK, FK->entities|                         | uuid (PK, FK->entities|
| kv, stator_size       |                         | frame_uuid, motor_uuid|
| min/max_voltage       |                         | fc_uuid, esc_uuids    |
| max_current_a, mount  |                         | rx_uuids, vtx_uuid    |
+-----------------------+                         +-----------------------+
```

### <a id="true-table-per-type-tpt-inheritance-architecture" >True Table-Per-Type (TPT) Inheritance Architecture</a>

Quadsmith will adopt true Table-Per-Type (TPT) relational inheritance. The
canonical [`entities`](../../src/backend/db/quadsmith_api.sql#L313-L330) table
acts as the base entity for every catalog component and user build:

- **Canonical Base Metadata in `entities`**: `uuid` (PK), `collection`, `id`,
  `name`, `description`, `primary_display_image`, `reference_links`, `media`,
  `author_or_manufacturer`, `user_uuid` (FK to `users(uuid)`), `is_archived`,
  `created_at`, `updated_at`.
- **Pure Domain Subtype Tables**: The 20 component tables (`motors`, `frames`,
  `batteries`, etc.) and the `builds` table drop their redundant copies of
  `name`, `description`, `primary_display_image`, `reference_links`, `media`,
  and `user_uuid`. They retain **only** their domain-specific specifications and
  foreign keys, keyed by `uuid` referencing `entities(uuid) ON DELETE CASCADE`.
- **Sub-Millisecond 1:1 Primary Key Joins**: When querying a component or build
  (`Get` and `List`), the database performs an
  `INNER JOIN entities USING (uuid)` and
  `LEFT JOIN users ON users.uuid = entities.user_uuid` to dynamically project
  `users.id AS author_username`. Because both tables share the identical UUID
  primary key in memory, PostgreSQL executes an in-memory index scan adding less
  than **0.2 milliseconds** to query latency while eliminating thousands of
  lines of duplicated data.
- **Unified Cross-Collection Profile Queries**: Fetching all components created
  by a user executes in $O(1)$ time with an indexed query against `entities`:

```sql
SELECT uuid, collection, id, name, primary_display_image, created_at
FROM entities
WHERE user_uuid = $1 AND collection != 'builds' AND is_archived = false
ORDER BY created_at DESC;
```

#### <a id="alternatives-considered-tpt" >Alternatives considered</a>

##### <a id="denormalized-dual-write" >Denormalized Dual-Write Mirroring</a>

Store `name`, `description`, `user_uuid`, and `is_archived` redundantly on both
the specific component tables and `entities`.

**Pros:**

- Avoids an inner join on `List` queries.

**Cons:**

- Causes severe desynchronization bugs (as proven by the existing `DeleteBuild`
  bug where archived builds leak into listings).
- Duplicates large JSONB and text columns across 21 separate tables.

##### <a id="pure-concrete-tables-no-supertype" >Pure Concrete Tables Without Supertype</a>

Remove `entities` entirely and store all community reviews, questions, and
social collections by joining polymorphic table references.

**Pros:**

- Removes the `entities` table.

**Cons:**

- Destroys global search and forces social features to manage 20 separate
  foreign keys.
- Requires a 20-table `UNION ALL` across all catalog tables to render a user's
  profile.

### <a id="creator-owned-with-administrator-override-model" >Creator-Owned with Administrator Override Model</a>

Every component and build in Quadsmith is created by a user and records the
author's identity on the base `entities` record (`user_uuid` referencing
`users(uuid)`, with `author_username` projected dynamically on query). The
seeder acts as a user (e.g. `@alex`), as all catalog items in production
originate from user contributions.

- **Creation**: Any authenticated user can create new components. Anonymous
  requests are rejected with `CodeUnauthenticated`.
- **Modification**: Only the original creator
  (`userCtx.UserUUID == entity.user_uuid`) or an administrator
  (`userCtx.Role == "ADMIN"`) can edit an existing component. Unauthorized
  attempts are rejected with `CodePermissionDenied`.
- **Deletion**: Only the original creator or an administrator can delete or
  archive a component. Unauthorized delete attempts return
  `CodePermissionDenied`.

#### <a id="alternatives-considered-ownership" >Alternatives considered</a>

##### <a id="open-community-wiki" >Open Community Wiki</a>

Allow any authenticated user to edit any component in the catalog, similar to
Wikipedia.

**Pros:**

- Enables crowd-sourced spec corrections without admin intervention.
- Eliminates bottlenecks when original creators become inactive.

**Cons:**

- High risk of specification vandalism and accidental regressions that break
  compatibility checks for other pilots' existing builds.
- Requires complex revision rollback tooling and edit conflict resolution.

##### <a id="moderation-queue" >Moderation Queue</a>

Require all user-submitted components and edits to undergo mandatory review and
approval by an administrator before becoming visible.

**Pros:**

- Guarantees 100% human-verified data in the public catalog.
- Prevents spam and invalid components from entering the system.

**Cons:**

- Severe administrative bottleneck that discourages community contributions.
- Blocks users from completing custom aircraft configurations in the Forge
  wizard while waiting for approval.

##### <a id="denormalized-author-username-columns" >Denormalized Author Username Columns</a>

Store `author_username` redundantly alongside `user_uuid` in `entities`,
`reviews`, `questions`, and `collections`.

**Pros:**

- Avoids joining the `users` table to project the author's handle.

**Cons:**

- Violates the Single Source of Truth rule; usernames permanently go stale when
  a user changes their handle via `AuthService.ChangeUsername`.
- Violates the `AGENTS.md` architecture convention requiring all database user
  references to use foreign key UUIDs.
- Unnecessary write-side overhead because read queries already join `users` to
  project avatar URLs and active profiles.

### <a id="20-collection-scope-and-per-resource-connectrpc-crud-api" >20-Collection Scope and Per-Resource ConnectRPC CRUD API</a>

The create and manage capabilities will span all 20 component collections
defined in Quadsmith:

- **Hardware Components (11)**: `Frame`, `Motor`, `Battery`, `FlightController`,
  `ElectronicSpeedController`, `Receiver`, `Antenna`, `Propeller`, `Camera`,
  `VideoTransmitter`, `GpsReceiver`.
- **Gear Components (4)**: `Charger`, `FieldTool`, `Goggles`, `Transmitter`.
- **Software & Firmware (5)**: `FcFirmware`, `EscFirmware`, `VtxFirmware`,
  `RadioFirmware`, `SoftwareTool`.

Adhering strictly to Quadsmith's **Top-Down Structural Invariant** (Services
First, Callers Precede Callees) from [`AGENTS.md`](../../AGENTS.md#L192-L210),
each component's `.proto` file (e.g.
[`motor.proto`](../../proto/quadsmith/api/motor.proto)) will declare standard
CRUD RPCs organized top-down:

```protobuf
syntax = "proto3";

package quadsmith.api;

// 1. Services First
service MotorService {
  rpc GetMotor(GetMotorRequest) returns (Motor);
  rpc ListMotors(ListMotorsRequest) returns (ListMotorsResponse);
  rpc CreateMotor(CreateMotorRequest) returns (Motor);
  rpc UpdateMotor(UpdateMotorRequest) returns (Motor);
  rpc DeleteMotor(DeleteMotorRequest) returns (DeleteMotorResponse);
}

// 2. Request & Response Messages (Precede referenced domain types)
message CreateMotorRequest {
  Motor motor = 1;
}

message UpdateMotorRequest {
  Motor motor = 1;
}

message DeleteMotorRequest {
  string id = 1;
}

message DeleteMotorResponse {
  bool is_archived = 1;
}

message GetMotorRequest {
  string id = 1;
  repeated string columns = 2;
}

message ListMotorsRequest {
  string filter = 1;
  int32 page_size = 2;
  string page_token = 3;
  repeated string columns = 4;
  repeated string sort = 5;
}

message ListMotorsResponse {
  repeated Motor motors = 1;
  string next_page_token = 2;
}

// 3. Domain Model (Referenced by Services and Requests above)
message Motor {
  option (quadsmith.sql.table) = {name: "motors"};
  option (quadsmith.common.collection_path) = "components/hardware/motors";

  string uuid = 1 [(quadsmith.sql.column) = {is_primary_key: true, references: "entities(uuid)"}];
  uint32 kv = 2 [(quadsmith.sql.column) = {create_index: true}];
  float stator_diameter_mm = 3;
  float stator_height_mm = 4;
  float weight_g = 5;
  float min_voltage = 6;
  float max_voltage = 7;
  float max_current_a = 8;
  repeated string prop_mount_interfaces = 9;
  string mount_pattern = 10;
}
```

The Go ConnectRPC handlers will be generated automatically by enhancing
[`protoc-gen-server`](../../src/backend/shared/cmd/protoc-gen-server/main.go) to
generate standard `Create<Type>`, `Update<Type>`, and `Delete<Type>` methods for
all messages with table and collection options, performing the atomic
`entities` + `<subtype_table>` transaction.

#### <a id="alternatives-considered-scope-rpc" >Alternatives considered</a>

##### <a id="hardware-only-scope" >Hardware-Only Scope</a>

Limit creation and management exclusively to the 11 hardware parts assembled in
quadcopter builds.

**Pros:**

- Marginally smaller initial API surface to audit.
- Focuses specifically on parts that impact the build configurator.

**Cons:**

- Creates an inconsistent developer experience where some catalog collections
  cannot be contributed to.
- Misses user demand for custom radio transmitters, goggles, and firmware forks.

##### <a id="polymorphic-component-service" >Polymorphic ComponentService</a>

Create a single unified `ComponentService` with polymorphic `oneof` payloads
handling all component types.

**Pros:**

- Consolidates all component mutation endpoints into one RPC service.
- Simplifies client routing for generic component actions.

**Cons:**

- Violates Quadsmith's One-File-Per-Resource architectural standard and Google
  AIP guidelines.
- Forces clients and generator scripts to navigate fragile nested `oneof`s and
  weakens static compile-time type safety.

### <a id="steam-style-hybrid-deletion-and-archive-visibility" >Steam-Style Hybrid Deletion and Archive Visibility</a>

Components are subject to Steam-style hybrid deletion to prevent dangling
references and protect historical builds:

- **Unreferenced Components**: If a component is not referenced by any `builds`,
  has zero `reviews`, zero `questions`, and zero `collection_items`, it is
  permanently hard-deleted from its component table, `entities`, and
  `blob_references`.
- **Referenced Components**: If referenced anywhere, the component is
  soft-deleted by setting `entities.is_archived = true`.

```text
                  +----------------------------------+
                  |  User Requests Component Delete  |
                  +-----------------+----------------+
                                    |
                                    v
                  +-----------------+----------------+
                  | Check References:                |
                  | - builds (FK references)         |
                  | - reviews & questions            |
                  | - collection_items               |
                  +-----------------+----------------+
                                    |
                   +----------------+----------------+
                   |                                 |
           [Zero References]               [Referenced in System]
                   |                                 |
                   v                                 v
     +---------------------------+     +---------------------------+
     |       Hard Deletion       |     |       Delist / Archive    |
     | - DELETE FROM component   |     | - UPDATE entities         |
     | - DELETE FROM entities    |     |   SET is_archived = true  |
     | - DELETE FROM blob_refs   |     | - Component hidden in List|
     | Returns is_archived=false |     | - Retained in builds & Get|
     +---------------------------+     | Returns is_archived=true  |
                                       +---------------------------+
```

- **Catalog Listings (`List...`)**: Automatically filters
  `entities.is_archived = false` via the standard `entities` join, ensuring
  catalog browsing stays completely clean of delisted parts.
- **The Forge Build Wizard**: The component picker strictly excludes archived
  parts so pilots do not select obsolete or delisted hardware for new builds.
- **Direct Lookup (`Get...`)**: Direct lookup by ID/UUID resolves successfully
  so historical builds, review threads, and shared links render properly. An
  `[Archived]` or `[Delisted]` status badge is prominently displayed.

#### <a id="alternatives-considered-deletion" >Alternatives considered</a>

##### <a id="strict-deletion-blocking" >Strict Deletion Blocking</a>

Strictly reject deletion requests with an error if the component is referenced
by any build, review, or question.

**Pros:**

- Extremely simple implementation requiring no archiving state.

**Cons:**

- Terrible user experience: users who made a typo or no longer want their
  component listed are permanently trapped if someone else referenced it in a
  test build.

##### <a id="hard-cascade-deletion" >Hard Cascade Deletion</a>

Permit hard deletion and nullify or cascade delete foreign keys across all
referencing builds.

**Pros:**

- Completely removes unwanted data from the database.

**Cons:**

- Corrupts other pilots' saved builds, breaks bill-of-materials integrity, and
  crashes the compatibility evaluator engine.

### <a id="deterministic-id-slug-generation-and-immutable-urls" >Deterministic ID Slug Generation and Immutable URLs</a>

Component IDs serve as public URL slugs (e.g.
`/components/hardware/motors/tmotor-f80-pro-1900kv`) and CLI identifiers:

- **Auto-Generation**: If the client leaves `id` empty during creation, a slug
  is deterministically generated from `manufacturer` and `name` using hyphenated
  lowercase alphanumeric characters. If empty, it falls back to
  `<collection>-<uuid[:8]>`.
- **Custom Slugs**: Clients can provide a custom `id`, which is validated to
  ensure it does not match UUID v7 formatting (preventing collision with primary
  keys).
- **Collision Deduplication**: If an `id` already exists within the collection,
  the backend increments a numerical copy suffix (`-2`, `-3`) using the
  established `buildid.NextCopyID` algorithm.
- **URL Immutability**: Once created, the `id` slug is immutable during updates
  to guarantee that external links, browser bookmarks, and CLI scripts never
  break. Physical and electrical specifications, names, images, and links remain
  fully editable.

#### <a id="alternatives-considered-slugs" >Alternatives considered</a>

##### <a id="mutable-slug-updates" >Mutable Slug Updates</a>

Allow users to modify the component's `id` slug during updates.

**Pros:**

- Allows fixing typos in URL slugs.

**Cons:**

- Breaks incoming web links, bookmarks, and external references unless complex
  HTTP redirect alias tables are introduced.

##### <a id="mandatory-manual-slugs" >Mandatory Manual Slugs</a>

Require users to manually specify an ID slug for every component they create.

**Pros:**

- Forces authors to explicitly think about their URL structure.

**Cons:**

- Adds unnecessary friction to the creation process compared to auto-generating
  from manufacturer and model name.

### <a id="strict-domain-validation-and-author-attribution" >Strict Domain Validation and Author Attribution</a>

To protect the integrity of the compatibility checker, motor thrust curves, and
electrical safety checks:

- **Universal Validation**:
  - `name`: Required, 1 to 256 characters.
  - `manufacturer`: Required, 1 to 128 characters.
  - `reference_links`: Validated with
    [`validation.ValidateReferenceLinks`](../../src/backend/shared/pkg/validation/validation.go).
  - `media` and `primary_display_image`: Valid URL or `/blobs/<hash>` format.
- **Hardware-Specific Domain Bounds**:
  - `Motor`: `kv > 0`, `stator_diameter_mm > 0`, `stator_height_mm > 0`,
    `min_voltage > 0`, `max_voltage >= min_voltage`, `max_current_a > 0`.
  - `Battery`: `cell_count >= 1`, `capacity_mah > 0`, `min_voltage > 0`,
    `max_voltage >= min_voltage`.
  - `FlightController`: `min_voltage > 0`, `max_voltage >= min_voltage`.
  - `ElectronicSpeedController`: `continuous_current_a > 0`, `min_voltage > 0`,
    `max_voltage >= min_voltage`.
  - `Frame`: `wheelbase_mm > 0`, `motor_count >= 3`.
- **Author Attribution**:
  - All components and builds are created by users (with the seeder acting as a
    user in sandbox and test environments).
  - Rather than displaying artificial provenance columns or official badges,
    components display the author's username (`Added by @username`) with a
    clickable link to their user profile.

#### <a id="alternatives-considered-validation" >Alternatives considered</a>

##### <a id="schema-type-validation-only" >Schema Type Validation Only</a>

Rely solely on Protobuf primitive types without checking positive ranges or
voltage ordering.

**Pros:**

- Less backend validation logic.

**Cons:**

- Allows garbage data that causes NaN or runtime crashes in the compatibility
  evaluator.

##### <a id="provenance-columns-and-official-badges" >Provenance Columns and Official Badges</a>

Introduce special "Official" badges and a dedicated "Provenance" table column to
differentiate platform-seeded components from user-created items.

**Pros:**

- Visually separates pre-cataloged hardware from new community additions.

**Cons:**

- Violates the core domain model that all components and builds are added by
  users.
- Adds unnecessary visual noise to data tables.
- Imposes an artificial hierarchy on community-maintained hardware entries.

### <a id="responsive-dynamic-modal-forms-forge-quick-add-profile-tab-and-full-cli-crud" >Responsive Dynamic Modal Forms, Forge Quick-Add, Profile Tab, and Full CLI CRUD</a>

#### Web UI Implementation

1. **Collection Pages (`CollectionPage.tsx`)**:
   - Add a prominent **"+ Add Component"** button to the collection header
     toolbar next to the column selector and search filter.
   - Clicking the button opens a responsive modal dialog containing a dynamic
     form generated from the collection's field definitions
     ([`hardwareCollections.tsx`](../../src/frontend/web/app/src/lib/hardwareCollections.tsx)).
   - Supports drag-and-drop file upload to `/api/blobs` as well as direct
     manufacturer image URLs with live image preview.
2. **Product Detail Pages (`ProductPage.tsx`)**:
   - If the logged-in user is the component creator or an administrator, display
     **"Edit"** and **"Delete"** actions in the product toolbar.
   - "Edit" opens the modal pre-populated with existing data.
   - "Delete" triggers a confirmation dialog detailing whether the item will be
     permanently erased or delisted/archived.
   - If the item is archived, render a clear `[Archived / Delisted]`
     notification banner.
3. **The Forge Build Wizard (`BuildWizardPage.tsx`)**:
   - In each component selection step (Frame, Motor, FC, ESC, etc.), provide an
     inline **"Don't see your part? Add custom component"** button.
   - Submitting the modal immediately inserts the component and auto-selects it
     for the current build, allowing uninterrupted aircraft assembly.
4. **User Profile Pages (`UserProfilePage.tsx`)**:
   - Add a **"Components"** tab alongside "Builds" and "Collections".
   - Displays all components contributed by the user with category badges,
     creation dates, and direct management shortcuts for the profile owner.

#### CLI Implementation (`qs`)

Enhance [`generate_cli.go`](../../src/frontend/cli/generate_cli.go) to generate
uniform `create`, `update`, and `delete` commands across all component
collections:

```bash
# Create a new motor from interactive flags or JSON file
qs components hardware motors create --name "F80 Pro" --manufacturer "T-Motor" --kv 1900 ...
qs components hardware motors create -f motor.json

# Update existing motor specifications
qs components hardware motors update tmotor-f80-pro-1900kv --max-current 55.0

# Delete or archive a motor
qs components hardware motors delete tmotor-f80-pro-1900kv
```

#### <a id="alternatives-considered-ui" >Alternatives considered</a>

##### <a id="dedicated-full-page-forms" >Dedicated Full-Page Forms</a>

Create dedicated standalone routes (e.g. `/components/hardware/motors/new`).

**Pros:**

- More screen space for very large forms.

**Cons:**

- Navigates the user away from their current browse context or build wizard
  flow, causing loss of unsaved aircraft build state in the Forge.

##### <a id="web-only-crud" >Web-Only CRUD Without CLI Support</a>

Implement component management exclusively in the React web frontend, leaving
the Go CLI as read-only.

**Pros:**

- Saves CLI code generation updates.

**Cons:**

- Violates Quadsmith's core architectural principle of full parity between the
  Web UI, CLI, and ConnectRPC APIs.

---

## <a id="4-alternatives-considered" >4. Alternatives considered</a>

### <a id="monolithic-jsonb-component-store" >Monolithic JSONB Component Store</a>

Store all component specifications within a single `components` table utilizing
a PostgreSQL JSONB column for subtype specifications, queried via CEL JSON
operators.

**Pros:**

- Requires zero DDL schema migrations when introducing new component types or
  modifying hardware attributes.
- Consolidates all hardware into a single table.

**Cons:**

- Sacrifices strict relational schema constraints and compile-time type safety.
- Significantly slows complex compatibility queries and numerical filter
  evaluations compared to native B-tree indexed relational columns.

---

## <a id="5-assumptions" >5. Assumptions</a>

1. The current PostgreSQL database schema managed via `psqldef` can be cleanly
   refactored into Table-Per-Type (TPT) inheritance, moving shared columns to
   `entities` and stripping redundant columns from `builds` and the 20 component
   tables.
2. The existing blob storage service (`/api/blobs`) is accessible for component
   media uploads and supports standard image formats (JPEG, PNG, WebP).
3. The client authentication session provided by `AuthInterceptor` is available
   across all ConnectRPC requests and correctly identifies the user's role and
   UUID.

---

## <a id="6-future-work" >6. Future work</a>

### <a id="suggest-edit-workflow-for-community-components" >Suggest Edit Workflow for Community Components</a>

Allow community members to suggest corrections or spec additions for components
created by other users, queuing suggestions for creator or administrator
approval.

### <a id="community-flagging-and-moderation" >Community Flagging and Moderation</a>

Enable pilots to flag inaccurate specifications or spam components, surfacing
flagged entries in an administrative review dashboard.

### <a id="bulk-component-import" >Bulk Component Import</a>

Support importing multiple components from manufacturer CSV/JSON spec sheets
directly via the CLI and web interface.
