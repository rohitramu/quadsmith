# Hardware Data Sourcing

Always check the manufacturer's official website and official spec sheets as the primary source of truth when scraping, extracting, or researching hardware component data. Do not rely solely on retailer websites, as they often contain missing, rounded, or erroneous data (such as missing EIRP or max transmit power). Manufacturer data should always serve as the tie-breaker in corroboration checks.

# Git Workflow & Commit Messages
- Do not run `git commit` yourself unless explicitly asked. Instead, leave changes in the working tree and suggest a commit message for the user to commit.
- Do not use conventional commit prefixes (like "feat:", "fix:", "chore:") at the start of commit messages. Write plain, descriptive commit messages.
- Never commit temporary or scratch scripts (e.g., `.cjs` helper scripts). Store temporary scripts in the artifact `scratch/` directory or delete them immediately before finishing a task.

# Project Planning
Do not make up names for implementation phases (e.g., "Phase 2") or dictate the roadmap structure without consulting the user first.

# Database & Schema Maintenance
- Whenever `db/schema.prisma` is modified, update `db/er_diagram.mermaid` to match, ensuring no existing entities or relationships (such as `Build` links) are dropped or abbreviated.
- The local Postgres database is managed by the Prisma VSCode extension. Never run a blocking foreground `npx prisma dev` task; if the database server must be started via CLI, always use `npx prisma dev -d` (`--detach`).
