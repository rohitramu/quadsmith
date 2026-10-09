# Agent instructions

## Hardware Data Sourcing

Always check the manufacturer's official website and official spec sheets as the primary source of truth when scraping, extracting, or researching hardware component data. Do not rely solely on retailer websites, as they often contain missing, rounded, or erroneous data (such as missing EIRP or max transmit power). Manufacturer data should always serve as the tie-breaker in corroboration checks.

## Git Workflow & Commit Messages

- Never commit temporary or scratch scripts (e.g., `.cjs` helper scripts). Always store temporary or scratch scripts in the `.tmp/` directory at the root of the repository, and delete them immediately before finishing a task.
- After creating a commit, always push to the remote. If you need to do a force push, ask the user first.
- When writing a commit message, do not mention changes to AGENTS.md in the subject line (top line) unless it is the only changed file.

## Project Planning

- Do not make up names for implementation phases (e.g., "Phase 2") or dictate the roadmap structure without consulting the user first.
- **No Implementation Without Explicit Instruction**: Never modify codebase files, schemas, or tests to implement anything discussed in a plan or demonstrated in a preview unless the user explicitly instructs you to proceed with implementation. All iterations during planning and preview review must remain strictly confined to the plan or preview artifacts.

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

- Always run `make test` and ensure all tests pass whenever you make changes to the codebase. Do not consider a task complete or push code if there are failing tests.

## UI Changes & Visual Previews

Whenever committing changes that modify the UI (such as frontend web components, pages, layouts, or styling):
- Create a self-contained HTML artifact (using embedded CSS and/or Tailwind) demonstrating the changed component, layout, or visual difference.
- Save the artifact into the conversation artifact directory with `UserFacing: true`.
- Embed and display the preview directly in the conversation using `<agent-embed src="file:///<artifact_path>"></agent-embed>` (or link to it for full-page views) so the user can visually verify the update.
- **Theme Toggle in Previews**: Always include an interactive light/dark mode theme toggle (e.g., a floating or header switch that toggles the `dark` class on `document.documentElement` or the preview root) in generated HTML preview artifacts, enabling visual verification in both light and dark themes.
- **Pre-Share Rendering Verification**: Always verify that HTML previews render correctly before sharing them with the user. Run headless Chrome (`google-chrome-stable --headless --disable-gpu --screenshot=<path> ...`) to capture a screenshot of the preview artifact, inspect it with `view_file`, and verify that layout, typography, borders, and colors display properly without visual defects or missing styles.
- **Self-Contained Styling & CSP Safety**: HTML previews must be completely self-contained. Do not rely on external CDNs (such as `cdn.tailwindcss.com`) that may be blocked by iframe Content Security Policies. Use the allowlisted gstatic Tailwind script (`https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js`) alongside embedded `<style>` fallback rules so previews render reliably in any sandboxed or offline environment.
- **Fidelity & Implementation Alignment**: Previews must match the real implementation as closely as possible. Do not include mock UI elements, decorative sections, or controls in the preview that will not be built into the final codebase.
- **Visual Plan & Specification**: Treat HTML previews as a visual plan and implementation guide alongside any planning document. The final frontend implementation must faithfully mirror the layout, components, data fields, and styling shown in the approved preview.
- **Design Iteration vs. Implementation**: Iterating on previews with the user is strictly design-phase work. Do not modify or push any application source code while iterating on previews. Wait for explicit user approval before translating approved previews into codebase changes.

