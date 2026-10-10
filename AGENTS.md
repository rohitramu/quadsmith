# Agent instructions

## Hardware Data Sourcing

Always check the manufacturer's official website and official spec sheets as the primary source of truth when scraping, extracting, or researching hardware component data. Do not rely solely on retailer websites, as they often contain missing, rounded, or erroneous data (such as missing EIRP or max transmit power). Manufacturer data should always serve as the tie-breaker in corroboration checks.

## Git Workflow & Commit Messages

- Never commit temporary or scratch scripts (e.g., `.cjs` helper scripts). Always store temporary or scratch scripts in the `.tmp/` directory at the root of the repository, and delete them immediately before finishing a task.
- After creating a commit, always push to the remote. If you need to do a force push, ask the user first.
- When writing a commit message, do not mention changes to AGENTS.md in the subject line (top line) unless it is the only changed file.

## Project Planning

- **Direct Requests vs. Plans & Questions**:
  - When the user directly asks for something (e.g., a specific change, feature, or bugfix), implement it directly in the codebase without pausing to ask for approval.
  - Only ask for approval before modifying codebase files if:
    1. The user asked a question that needs to be answered/clarified before proceeding, or
    2. You are actively discussing a **plan** — defined specifically as either a **visual design** (iterating via preview artifacts) or a **planning document** (e.g., via the `/plan` action).
  - All iterations during plan discussion or visual design review must remain strictly confined to the plan or preview artifacts until approved.
- **Prominent Approval Callouts**: Whenever waiting on user approval (when answering questions or discussing a plan), prominently display a dedicated callout at the end of the message. Title the callout using the format `Awaiting approval to <action>` (e.g., "Awaiting approval to Implement", "Awaiting approval to Execute"), making it immediately clear that progress is paused pending explicit confirmation.

## Database & Schema Maintenance

- Whenever `backend/db/schema.sql` is modified, update `backend/db/diagram.mermaid` to match, ensuring no existing entities or relationships (such as `Build` links) are dropped or abbreviated.
- The local Postgres database is managed declaratively via `psqldef` (from the `sqldef` project). We maintain a single desired state in `backend/db/schema.sql` rather than sequential migration files. Never use Prisma commands.
- Apply schema changes by running `psqldef` against the local database. Ensure the `go.work` structure reflects any module changes.

## Database IDs & Resource Names

- The primary key for all database tables is a hidden `uuid` (UUID v7).
- A human-readable `id` (unique identifier or resource path) is stored as a UNIQUE TEXT column.
- All Foreign Keys should reference the `uuid` (e.g., `frame_uuid`), never the `id`.
- The CLI, API, and Protobufs should expose and prefer the human-readable `id` for UX, performing the UUID translation internally in the backend SQL queries (falling back to UUID matching if the `id` isn't matched).

## Path Referencing

- Whenever providing example commands, file paths, or directory references in chat, always use relative paths from the root of the repository (e.g., `frontend/cli/qs` instead of `./qs`).

## Development & Sandbox Execution

- `make sandbox` starts the local server and Docker sandbox environment. It is a long-running daemon process that does not terminate unless an error occurs.
- Never wait for `make sandbox` to complete execution. If executing `make sandbox`, run it as a background daemon process and monitor its startup output/logs instead of blocking on termination.
- If only verifying build integrity, code generation, or breaking change warnings without needing a running server, use `make build` or `make warn-breaking` instead.

## Testing

- Always run `make test` and ensure all tests pass whenever you make changes to functional code, schemas, or tests. Do not consider a task complete or push code if there are failing tests.
- You do not need to run the test suite if system behavior has not changed (e.g., when only updating `AGENTS.md` or documentation).

## UI Changes & Visual Previews

Whenever committing changes that modify the UI (such as frontend web components, pages, layouts, or styling):
- **Dual Previews (Desktop & Mobile)**:
  - Generate **two separate self-contained HTML preview artifacts** for every UI change:
    1. **Desktop Preview** (`<name>_preview_desktop.html`): Built for the Antigravity Desktop studio/side-pane. Features the full layout, side-by-side builder controls, continuous interactive sliders (60 FPS), complete data tables/metrics dashboards, and interactive simulations.
    2. **Mobile Preview** (`<name>_preview_mobile.html`): Specifically optimized for the Android Antigravity app.
  - **Artifact Links Only (No Inline Embeds & No Repetitive Notes)**: Share direct clickable markdown links to both preview artifacts (e.g., `[Open Desktop Preview](file:///...)` and `[Open Mobile Preview](file:///...)`) for full-page viewing. Never embed preview iframes or `<agent-embed>` directly in the conversation. Do not include repetitive preview notes, technical constraints, or mobile limitation breakdowns in the message unless explicitly requested.
- **Desktop Preview as Implementation Guide**:
  - When planning changes or translating designs into code, **always use the desktop preview** as the primary visual plan and implementation specification. Do not simplify the final codebase to match mobile preview compromises.
- **Visual Parity for Mobile**:
  - Align the mobile preview to mirror the desktop preview's visual aesthetic as closely as possible (colors, borders, typography, card elevation, status indicators).
- **Mobile Android Embed Optimization Rules**:
  - **Zero-Collapse Auto Height (≤ 140px–160px)**: Never use `100vh`, `min-h-screen`, `h-screen`, or viewport-relative heights. Sizing relative to the viewport causes recursive measurement loops that collapse the iframe to a ~150px sliver. Keep the entire mobile preview under ~140px–160px tall so it displays completely within the Android chat embed with **zero vertical scrolling**.
  - **100% Pure-CSS Interactivity**: Do not rely on JavaScript for state changes, tabs, or calculations in mobile embeds (Android WebView inline embeds frequently sandbox or disable script execution). Implement all interactive states (tabs, progress levels, payload weights, theme toggles) using hidden radio/checkbox inputs (`<input type="radio" class="sr-only">`) and CSS `:checked` sibling selectors (`#radio:checked ~ .container ...`).
  - **Discrete Tap Presets Over Draggable Sliders**: Continuous `<input type="range">` drag gestures are intercepted and canceled by Android's parent scroll view (`touchcancel`). Use discrete, tap-friendly pill buttons (`[ 0% ]`, `[ 25% ]`, `[ 50% ]`, `[ 75% ]`, `[ 100% ]` or `[ 0g ]`, `[ 60g ]`, `[ 120g ]`, `[ 250g ]`, `[ 400g ]`) with `touch-action: manipulation;`.
- **Theme Toggle in Previews**: Always include an interactive light/dark mode theme toggle (using CSS `:checked` fallback on mobile and JavaScript on desktop) in generated HTML preview artifacts, enabling visual verification in both light and dark themes.
- **Pre-Share Rendering Verification**: Always verify that HTML previews render correctly before sharing them with the user. Run headless Chrome (`google-chrome-stable --headless --disable-gpu --screenshot=<path> ...`) across desktop and mobile viewports, inspect screenshots with `view_file`, and verify that layout, typography, borders, and colors display properly without visual defects.
- **Self-Contained Styling & CSP Safety**: HTML previews must be completely self-contained. Do not rely on external CDNs (such as `cdn.tailwindcss.com`) that may be blocked by iframe Content Security Policies. Use the allowlisted gstatic Tailwind script (`https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js`) alongside embedded `<style>` fallback rules so previews render reliably in any sandboxed or offline environment.
- **Fidelity & Implementation Alignment**: Previews must match the real implementation as closely as possible. Do not include mock UI elements, decorative sections, or controls in the preview that will not be built into the final codebase.
- **Visual Plan & Specification**: Treat HTML previews as a visual plan and implementation guide alongside any planning document. The final frontend implementation must faithfully mirror the layout, components, data fields, and styling shown in the approved desktop preview.
- **Design Iteration vs. Direct Execution**: When discussing a visual design plan, keep iterations strictly confined to the preview artifacts until approved. When the user directly requests a specific UI change (outside of discussing a visual design or planning document), implement it directly in the codebase alongside the updated preview artifacts, tests, and commit/push.

