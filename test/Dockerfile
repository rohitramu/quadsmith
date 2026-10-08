# ==============================================================================
# Quadsmith Sandbox / Test Container Image
# ==============================================================================
# Purpose:
#   Self-contained sandbox container for local development, automated testing,
#   and E2E verification. Started via `make sandbox` or `src/backend/start_sandbox.sh`.
#
# Included Components:
#   - PostgreSQL 15 Database (runs locally inside the container)
#   - psqldef (declarative schema management tool)
#   - Database schema (src/backend/db/schema.sql) and initial seed data
#   - Database Seeder binary (populates hardware catalog on startup)
#   - Go Backend API Server (ConnectRPC on port 8080)
#   - React 19 SPA Web Frontend (served directly from the API server)
#
# Usage:
#   - Automatically initializes Postgres, applies schema with psqldef, runs
#     the seeder, and starts the API server and web UI on container boot.
# ==============================================================================

# Stage 1: Build the Go binaries
FROM golang:1.27 AS builder
WORKDIR /app

# Copy go.work, go.mod files and vendor directory
COPY src/go.work src/go.work.sum ./src/
COPY src/build_deps/go.mod src/build_deps/go.sum ./src/build_deps/
COPY src/backend/api/go.mod src/backend/api/go.sum ./src/backend/api/
COPY src/frontend/cli/go.mod src/frontend/cli/go.sum* ./src/frontend/cli/
COPY test/go.mod test/go.sum* ./test/
COPY src/vendor/ ./src/vendor/

# Copy all source code (including pre-generated proto files)
COPY . .

# Build Server and Seeder
RUN cd src/backend/api && CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server ./cmd/server/main.go && CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/seeder ./cmd/seeder


# Stage 2: Build the Web UI
FROM node:22 AS web-builder
WORKDIR /app

# Copy package files
COPY src/frontend/web/package.json src/frontend/web/package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source and build
COPY src/frontend/web/ .
RUN npm run build


# Stage 3: The Sandbox (Postgres + Go API + Web UI)
FROM postgres:15
WORKDIR /app

# Install psqldef for schema management
RUN apt-get update && apt-get install -y wget && \
    wget -qO- https://github.com/sqldef/sqldef/releases/download/v0.17.15/psqldef_linux_amd64.tar.gz | tar xvz -C /usr/local/bin && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Environment Setup
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=quadsmith
ENV DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
ENV PORT=8080
ENV STATIC_DIR="/app/web/dist"

EXPOSE 5432 8080

# Inline the sandbox entrypoint script securely using printf (legacy builder compatible)
RUN printf '#!/usr/bin/env bash\n\
set -e\n\
echo "Starting PostgreSQL..."\n\
docker-entrypoint.sh postgres > /var/log/postgres.log 2>&1 &\n\
echo "Waiting for PostgreSQL to be ready..."\n\
until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 1; done\n\
echo "Applying Schema..."\n\
psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith < /app/src/backend/db/schema.sql\n\
echo "Seeding Database..."\n\
./seeder\n\
exec ./server\n' > /app/sandbox-entrypoint.sh

RUN chmod +x /app/sandbox-entrypoint.sh

# Copy binaries from builder
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/seeder /app/seeder

# Copy Web UI build from web-builder
COPY --from=web-builder /app/dist /app/web/dist

# Copy database schema and seeds at the very end
COPY src/backend/db/schema.sql /app/src/backend/db/schema.sql
COPY src/backend/db/seeds /app/src/backend/db/seeds

ENTRYPOINT ["/app/sandbox-entrypoint.sh"]
