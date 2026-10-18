# Agent instructions

## Hardware Data Sourcing

Always check the manufacturer's official website and official spec sheets as the
primary source of truth when scraping, extracting, or researching hardware
component data. Do not rely solely on retailer websites, as they often contain
missing, rounded, or erroneous data (such as missing EIRP or max transmit
power). Manufacturer data should always serve as the tie-breaker in
corroboration checks.

## Git Workflow & Commit Messages

- Never commit temporary or scratch scripts (e.g., `.cjs` helper scripts).
  Always store temporary or scratch scripts in the `.tmp/` directory at the root
  of the repository, and delete them immediately before finishing a task.
- After creating a commit, always push to the remote. If you need to do a force
  push, ask the user first.
- When writing a commit message, do not mention changes to AGENTS.md in the
  subject line (top line) unless it is the only changed file.
- **Plan Implementation & Commits**:
  - **Mention Active Plans in Related Commits**: Every commit that relates to or
    impacts one or more active plans (directly in `plans/`) must explicitly cite
    those plans (by relative path, e.g., `plans/<plan_name>.md`) in the commit
    message body. Archived plans in `plans/archive/` must be ignored and not
    cited.
  - Implement plans in a single commit that saves the finalized plan to
    `plans/<plan_name>.md`.
  - Include the **Goal** section, a **TL;DR of the Proposal**, and the plan path
    reference in the commit message body following the subject line.
  - For follow-up commits:
    - Explicitly mention the affected active plan(s) in the commit message body.
    - If the plan changes: update the plan in `plans/<plan_name>.md` in that
      commit.
    - If fixing an incorrect implementation: treat it as a standard bugfix
      commit (no plan document update needed, but still cite the active plan).
    - If following up immediately on the branch's latest commit: amend the
      commit and update the plan/message as needed. If already pushed, ask
      before force-pushing.

## Code Cleanliness & Breaking Changes

- This codebase is not deployed anywhere and is iterating toward its first
  version. Prioritize code cleanliness, simplicity, and architectural elegance
  above all else.
- Do not check for breaking changes, do not run breaking change checks, and
  ignore any warnings about breaking changes.
- Never add backwards-compatibility shims, legacy adapters, reserved protobuf
  tags, or deprecated database columns. Design all schemas, APIs, Protobuf
  messages, and data models cleanly and directly as if building from scratch.

## Go Import Conventions

- **Aliasing Rules**:
  - Avoid using import aliases in Go files unless:
    - The import is for a generated Protobuf package, in which case use a `*pb`
      suffix or alias (e.g., `pb`, `apipb`, `configpb`, `timestamppb`).
    - There is a package naming conflict or collision between multiple imports
      in the same file (e.g., disambiguating
      `quadsmith/shared/gen/quadsmith/config` and
      `quadsmith/shared/pkg/config`).
    - Standard blank imports (`_ "embed"`) or side-effect registrations are
      strictly required.
  - Never use redundant import aliases where the alias matches the imported
    package's default identifier (e.g., do not write
    `connect "connectrpc.com/connect"`).
- **Import Grouping**:
  - Separate import groups with a single blank line and order them as follows:
    1. Standard library imports (e.g., `"context"`, `"fmt"`).
    2. Third-party external dependencies (e.g., `"connectrpc.com/connect"`).
    3. Other internal `quadsmith` packages (e.g., `"quadsmith/shared/pkg/..."`)
       as the second-to-last group.
    4. Protobuf (`pb`) packages (e.g.,
       `pb "quadsmith/shared/gen/quadsmith/api"`,
       `"quadsmith/shared/gen/quadsmith/..."`, or `*pb` types) as the last
       group.

## Project Planning

- **Direct Requests vs. Plans & Questions**:
  - When the user directly asks for something (e.g., a specific change, feature,
    or bugfix), implement it directly in the codebase without pausing to ask for
    approval.
  - Only ask for approval before modifying codebase files if:

    1. The user asked a question that needs to be answered/clarified before
       proceeding, or
    2. You are actively discussing a **plan** — defined specifically as either a
       **visual design** (iterating via preview artifacts) or a **planning
       document** (e.g., via the `/plan` action).

  - All iterations during plan discussion or visual design review must remain
    strictly confined to the plan or preview artifacts until approved.
- **Prominent Approval Callouts**: Whenever waiting on user approval (when
  answering questions or discussing a plan), prominently display a dedicated
  callout at the end of the message. Title the callout using the format
  `Awaiting approval to <action>` (e.g., "Awaiting approval to Implement",
  "Awaiting approval to Execute"), making it immediately clear that progress is
  paused pending explicit confirmation.
- **Plan Document Structure & Lifecycle**:
  - Author and iterate on plans within the agent artifact directory
    (`<appDataDir>/brain/<conversation-id>/<plan_name>.md`) during review.
  - In artifact files, use `&nbsp;` on blank lines strictly between adjacent
    prose text paragraphs for preview spacing, and `file:///` scheme links for
    editor clickability. Never use `&nbsp;` directly above or below headings,
    horizontal rules (`---`), tables, lists, or code blocks; use regular empty
    blank lines around them.
  - Upon approval, convert internal file links to repo-relative paths, remove
    `&nbsp;` spacing, and commit the finalized plan to `plans/<plan_name>.md`
    using pure standard Markdown.
  - Every plan document must adhere to the standardized structure:
    - **Table of Contents**: Linked list of all Level 1 headings present in the
      doc, plus nested design decision headings.
    - **Horizontal Separators**: Use `---` only between Level 1 headings.
    - **Blank Lines for Headings, Lists & Code Blocks**: Bulleted lists,
      headings (except on the very first line of the file), and code blocks must
      have a blank line before and after them.
    - **Table Boundaries**: Markdown tables must always have a clean, regular
      empty blank line before and after them. Never place `&nbsp;` immediately
      after a table, which breaks table termination and merges text into cells.
    - **Collapsible Details Blocks**: Always leave a clean blank line
      immediately after `</summary>` and before `</details>` so contained
      Markdown elements (subheadings, lists, code blocks) parse cleanly.
    - **Mermaid Format for All Diagrams**: All diagrams in plan documents
      (architecture, flowcharts, sequence, state, class, and entity-relationship
      diagrams) must be authored in Mermaid format (` ```mermaid `) rather than
      plain-text or ASCII art (` ```text `). The sole exception is
      folder/directory structures, which may use ` ```text ` blocks formatted
      like `tree` output (`├──`, `└──`, `│`, trailing `/` on directories, and
      aligned `#` comments).
    - **Line Wrapping at 80 Characters**: Lines should wrap at 80 characters
      whenever possible.
    - **Custom Anchors for All Headings**: Every heading must have a unique
      anchor formatted as `<hN><a id="anchor-text" >Header text</a></hN>` (e.g.,
      `## <a id="anchor-text" >Header text</a>`), picking an appropriate, unique
      heading ID.
    - **Omission of Empty Optional Sections**: Omit Background, Alternatives
      considered, Assumptions, Future work, or Open questions entirely if empty
      (do not include placeholders like "None", and exclude omitted sections
      from the Table of Contents).
    - **Document All Discussed Alternatives**: Document all alternatives
      explored during discussion, regardless of quality or viability.
    - **Concise Bulleted Summaries**: Use concise bulleted short items whenever
      helpful to explain complex concepts cleanly across plans, alternatives,
      and chat explanations.
    - **Clickable Reference Links**: Include as many clickable reference links
      as appropriate using descriptive anchor text as evidence for statements.
      For committed repository files, use relative links, never absolute paths.
      External links should prioritize official documentation, Wikipedia, and
      StackExchange.
    - **1. Background (optional)**: Context, and explicit references to any
      superseded plans.
    - **2. Goal**
    - **3. Proposal**: Starts with `> **TL;DR:** <explanation>` blockquote (at
      most 5 statements, with each statement 20 words or less, crisp and punchy,
      formatted as either a short paragraph or concise bulleted list), followed
      by a high-level concept explanation paragraph, then 1 sub-heading per
      design decision (`### <a id="custom-anchor-text" >Design Decision</a>`)
      with the proposal explanation directly under it (using Mermaid diagrams,
      tables, images, UI screenshots, UI concept images, artifacts, or code
      snippets with path references as appropriate), and nested collapsible
      `<details><summary><strong>Alternatives considered</strong></summary>`
      `... </details>` containing 1 sub-heading per alternative
      (`##### <a id="custom-anchor-text" >Alternative Name</a>`), each with an
      unbulleted ~2 sentence summary explaining the alternative, an unbulleted
      `**Pros:**` label, a blank line, a bulleted list of pros compared to the
      proposal, a blank line, an unbulleted `**Cons:**` label, a blank line, and
      a bulleted list of cons compared to the proposal.
    - **4. Alternatives considered (optional)**: High-level conceptual
      alternatives inside a collapsible
      `<details><summary><strong>Alternatives considered</strong></summary>`
      `... </details>` block, containing 1 sub-heading per alternative
      (`### <a id="custom-anchor-text" >Alternative Name</a>`), each with an
      unbulleted ~2 sentence summary explaining the alternative, an unbulleted
      `**Pros:**` label, a blank line, a bulleted list of pros, a blank line, an
      unbulleted `**Cons:**` label, a blank line, and a bulleted list of cons.
    - **5. Assumptions (optional)**: Numbered list.
    - **6. Future work (optional)**: 1 sub-heading per idea.
    - **7. Open questions (optional)**: Numbered list.
- **Plan Maintenance & Archival**:
  - Keep all non-archived plans in `plans/` up-to-date with every new commit.
  - If a new plan contradicts an existing plan:
    - Do not modify the existing plan document.
    - Reference the superseded plan(s) in the `Background` section of the new
      plan.
    - Move superseded plans into `plans/archive/`.
    - Notify the user of this situation during the plan discussion.
  - If unsure whether an existing plan should be updated or archived, ask the
    user.

## Database & Schema Maintenance

- Whenever `src/db/<database>.sql` (or any `<database>.sql`) is modified, update
  `src/db/<database>.mermaid` (or `<database>.mermaid`) to match, ensuring no
  existing entities or relationships (such as `Build` links) are dropped or
  abbreviated.
- The local Postgres database is managed declaratively via `psqldef` (from the
  `sqldef` project). We maintain a single desired state per database in
  `src/db/<database>.sql` (e.g., `src/db/quadsmith_api.sql`) rather than
  sequential migration files. Never use Prisma commands.
- Apply schema changes by running `psqldef` against the local database. Ensure
  the `go.work` structure reflects any module changes.

## Database IDs & Resource Names

- The primary key for all database tables is a hidden `uuid` (UUID v7).
- A human-readable `id` (unique identifier or resource path) is stored as a
  UNIQUE TEXT column.
- All Foreign Keys should reference the `uuid` (e.g., `frame_uuid`), never the
  `id`.
- The CLI, API, and Protobufs should expose and prefer the human-readable `id`
  for UX, performing the UUID translation internally in the backend SQL queries
  (falling back to UUID matching if the `id` isn't matched).

## Protobuf Structure & Organization

- **Proto File Section Structure**: All Protobuf files (`.proto`) must be
  structured with the following section order (omitting any section that has no
  definitions in that file):
  1. `// Services`
  2. `// Domain model`
  3. `// Request/response messages`
  4. `// Everything else`
  - Never number section headers (e.g., do not write `// 1. Services`), and
    never include parenthetical ordering explanations (e.g., do not write
    `(Services First)`).
- **Message Reference Ordering**: Sort messages based on what references them —
  a message should never be above what references them.
  - When a message references another definition, the referenced definition must
    appear below the referencing definition.
  - When multiple definitions exist at the same tier without references between
    them, sort them in alphabetical order as a lowest-precedence tie-breaker.

## Path Referencing

- Whenever providing example commands, file paths, or directory references in
  chat, always use relative paths from the root of the repository (e.g.,
  `frontend/cli/qs` instead of `./qs`).

## Development & Sandbox Execution

- **Long-Running Commands & Non-Blocking Execution**:
  - Never execute potentially long-running commands (e.g., full test suites
    `make test`, `make update-snapshots`, Docker container builds, heavy code
    compilation, or benchmark scripts) with large synchronous wait windows that
    risk hanging indefinitely.
  - Always dispatch potentially long-running commands to the background using a
    short initial wait (`WaitMsBeforeAsync: 500ms–2000ms`) so execution
    immediately transitions to a background task, or configure `IsDaemon: true`
    for standing background services.
  - Monitor background tasks asynchronously with reasonable timeouts (e.g. using
    `schedule` timers, status checks, or reactive task completion notifications)
    so operations never hang undetected or block turns indefinitely.
- **Server & Sandbox Long-Running Daemon Processes**:
  - Whenever starting the server directly or indirectly (e.g., via
    `make sandbox`, `test/start_sandbox.sh`, `docker run ... quadsmith-sandbox`,
    or `./server`), the command runs a persistent daemon that **will not exit on
    its own**.
  - Upon successful startup, it will hang on
    `"Starting Quadsmith API Server on port 8080..."` (or the configured port)
    indefinitely while serving requests.
  - **Never wait synchronously** for server startup commands to complete. Always
    launch them as background daemon tasks (`IsDaemon: true` or background
    tasks), monitor startup logs asynchronously, and send a **SIGTERM** (or
    kill/cancel the task) when finished.
- If only verifying build integrity or code generation without needing a running
  server, use `make build` instead.

## Testing

- Only run `make test` when making changes to files under `src/`, `test/`,
  `deploy/docker-compose.yaml`, `deploy/nginx/`, `.dockerignore`, or build/test
  rules in `Makefile`. Always ensure all tests pass before considering a task
  complete or pushing code.
- `make test` automatically runs `make verify`, so if `make test` is run,
  `make verify` does not need to be run separately.
- When only modifying files outside those paths (such as `AGENTS.md`,
  `README.md`, `plans/`, `deploy/README.md`, `.github/workflows/`, or cloud-only
  deployment scripts and configs like `deploy/deploy.sh`,
  `deploy/deploy_blue_green.sh`, `deploy/docker-compose.blue-green.yaml`, and
  `deploy/config/*.textproto`), do not run `make test`; run `make verify`
  instead to ensure formatting, Markdown linting, config validation, and shell
  syntax checks pass before committing.
- **Playwright Visual Regression Snapshots**:
  - Only update Playwright screenshots (`make update-snapshots` or
    `UPDATE_SNAPSHOTS=true`) when a change is explicitly expected to alter the
    visual rendering of those specific screens. Never update snapshots
    preemptively or blindly to silence test failures.
  - When screenshots are expected to change, always run the Playwright tests in
    standard comparison mode first (`make test`) to verify that **only** the
    expected screenshot comparisons fail across all 5 browser profiles and no
    unrelated pages have regressed.
  - Once you have confirmed that only the expected screenshot comparisons fail,
    run `make update-snapshots` to update baselines across all 5 browser
    profiles, verify the modified files in `test/web/snapshots/`, and re-run
    `make test` to confirm they pass.

## UI Changes & Visual Previews

Whenever committing changes that modify the UI (such as frontend web components,
pages, layouts, or styling):

- **Dual Previews (Desktop & Mobile)**:
  - Generate **two separate self-contained HTML preview artifacts** for every UI
    change:

    1. **Desktop Preview** (`<name>_preview_desktop.html`): Built for the
       Antigravity Desktop studio/side-pane. Features the full layout,
       side-by-side builder controls, continuous interactive sliders (60 FPS),
       complete data tables/metrics dashboards, and interactive simulations.
    2. **Mobile Preview** (`<name>_preview_mobile.html`): Specifically optimized
       for the Android Antigravity app.

  - **Artifact Links Only (No Inline Embeds & No Repetitive Notes)**: Share
    direct clickable markdown links to both preview artifacts (e.g.,
    `[Open Desktop Preview](file:///...)` and
    `[Open Mobile Preview](file:///...)`) for full-page viewing. Never embed
    preview iframes or `<agent-embed>` directly in the conversation. Do not
    include repetitive preview notes, technical constraints, or mobile
    limitation breakdowns in the message unless explicitly requested.
- **Desktop Preview as Implementation Guide**:
  - When planning changes or translating designs into code, **always use the
    desktop preview** as the primary visual plan and implementation
    specification. Do not simplify the final codebase to match mobile preview
    compromises.
- **Visual Parity for Mobile**:
  - Align the mobile preview to mirror the desktop preview's visual aesthetic as
    closely as possible (colors, borders, typography, card elevation, status
    indicators).
- **Mobile Android Embed Optimization Rules**:
  - **Zero-Collapse Auto Height (≤ 140px–160px)**: Never use `100vh`,
    `min-h-screen`, `h-screen`, or viewport-relative heights. Sizing relative to
    the viewport causes recursive measurement loops that collapse the iframe to
    a ~150px sliver. Keep the entire mobile preview under ~140px–160px tall so
    it displays completely within the Android chat embed with **zero vertical
    scrolling**.
  - **100% Pure-CSS Interactivity**: Do not rely on JavaScript for state
    changes, tabs, or calculations in mobile embeds (Android WebView inline
    embeds frequently sandbox or disable script execution). Implement all
    interactive states (tabs, progress levels, payload weights, theme toggles)
    using hidden radio/checkbox inputs (`<input type="radio" class="sr-only">`)
    and CSS `:checked` sibling selectors (`#radio:checked ~ .container ...`).
  - **Discrete Tap Presets Over Draggable Sliders**: Continuous
    `<input type="range">` drag gestures are intercepted and canceled by
    Android's parent scroll view (`touchcancel`). Use discrete, tap-friendly
    pill buttons (`[ 0% ]`, `[ 25% ]`, `[ 50% ]`, `[ 75% ]`, `[ 100% ]` or
    `[ 0g ]`, `[ 60g ]`, `[ 120g ]`, `[ 250g ]`, `[ 400g ]`) with
    `touch-action: manipulation;`.
- **Theme Toggle in Previews**: Always include an interactive light/dark mode
  theme toggle (using CSS `:checked` fallback on mobile and JavaScript on
  desktop) in generated HTML preview artifacts, enabling visual verification in
  both light and dark themes.
- **Pre-Share Rendering Verification**: Always verify that HTML previews render
  correctly before sharing them with the user. Run headless Chrome
  (`google-chrome-stable --headless --disable-gpu --screenshot=<path> ...`)
  across desktop and mobile viewports, inspect screenshots with `view_file`, and
  verify that layout, typography, borders, and colors display properly without
  visual defects.
- **Self-Contained Styling & CSP Safety**: HTML previews must be completely
  self-contained. Do not rely on external CDNs (such as `cdn.tailwindcss.com`)
  that may be blocked by iframe Content Security Policies. Use the allowlisted
  gstatic Tailwind script
  (`https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js`) alongside
  embedded `<style>` fallback rules so previews render reliably in any sandboxed
  or offline environment.
- **Fidelity & Implementation Alignment**: Previews must match the real
  implementation as closely as possible. Do not include mock UI elements,
  decorative sections, or controls in the preview that will not be built into
  the final codebase.
- **Visual Plan & Specification**: Treat HTML previews as a visual plan and
  implementation guide alongside any planning document. The final frontend
  implementation must faithfully mirror the layout, components, data fields, and
  styling shown in the approved desktop preview.
- **Design Iteration vs. Direct Execution**: When discussing a visual design
  plan, keep iterations strictly confined to the preview artifacts until
  approved. When the user directly requests a specific UI change (outside of
  discussing a visual design or planning document), implement it directly in the
  codebase alongside the updated preview artifacts, tests, and commit/push.
