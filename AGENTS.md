# Agent instructions

## Hardware Data Sourcing

Always check the manufacturer's official website and official spec sheets as the primary source of truth when scraping, extracting, or researching hardware component data. Do not rely solely on retailer websites, as they often contain missing, rounded, or erroneous data (such as missing EIRP or max transmit power). Manufacturer data should always serve as the tie-breaker in corroboration checks.

## Git Workflow & Commit Messages

- Do not run `git commit` yourself unless explicitly asked. Instead, leave changes in the working tree and suggest a commit message for the user to commit.
- Do not use conventional commit prefixes (like "feat:", "fix:", "chore:") at the start of commit messages. Write plain, descriptive commit messages.
- Never commit temporary or scratch scripts (e.g., `.cjs` helper scripts). Always store temporary or scratch scripts in the `.tmp/` directory at the root of the repository, and delete them immediately before finishing a task.
- When suggesting a commit message, do not mention changes to AGENTS.md in the subject line (top line) unless it is the only changed file.

## Project Planning

Do not make up names for implementation phases (e.g., "Phase 2") or dictate the roadmap structure without consulting the user first.

## Database & Schema Maintenance

- Whenever `backend/db/schema.sql` is modified, update `backend/db/diagram.mermaid` to match, ensuring no existing entities or relationships (such as `Build` links) are dropped or abbreviated.
- The local Postgres database is managed declaratively via `psqldef` (from the `sqldef` project). We maintain a single desired state in `backend/db/schema.sql` rather than sequential migration files. Never use Prisma commands.
- Apply schema changes by running `psqldef` against the local database. Ensure the `go.work` structure reflects any module changes.

## Database IDs & Resource Names

- The primary key for all database tables is a hidden `uuid` (UUID v7).
- A human-readable `id` (slug or resource path) is stored as a UNIQUE TEXT column.
- All Foreign Keys should reference the `uuid` (e.g., `frame_uuid`), never the `id`.
- The CLI, API, and Protobufs should expose and prefer the human-readable `id` for UX, performing the UUID translation internally in the backend SQL queries (falling back to UUID matching if the `id` isn't matched).

## Path Referencing

- Whenever providing example commands, file paths, or directory references in chat, always use relative paths from the root of the repository (e.g., `frontend/cli/qs` instead of `./qs`).
