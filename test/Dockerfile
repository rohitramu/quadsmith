# ==============================================================================
# Quadsmith Sandbox / Test Container Image
# ==============================================================================
# Purpose:
#   Self-contained sandbox container for local development, automated testing,
#   and E2E verification. Started via `make sandbox` or `test/start_sandbox.sh`.
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
ENV GOWORK=/app/src/go.work

# Copy go.work, go.mod files and vendor directory
COPY src/go.work src/go.work.sum* ./src/
COPY src/build_deps/go.mod src/build_deps/go.sum* ./src/build_deps/
COPY src/backend/api/go.mod src/backend/api/go.sum* ./src/backend/api/
COPY src/backend/api-server/go.mod src/backend/api-server/go.sum* ./src/backend/api-server/
COPY src/backend/worker/go.mod src/backend/worker/go.sum* ./src/backend/worker/
COPY src/backend/pkg/config/go.mod src/backend/pkg/config/go.sum* ./src/backend/pkg/config/
COPY src/frontend/cli/go.mod src/frontend/cli/go.sum* ./src/frontend/cli/
COPY src/frontend/config/go/go.mod src/frontend/config/go/go.sum* ./src/frontend/config/go/
COPY test/go.mod test/go.sum* ./test/
COPY src/vendor/ ./src/vendor/

# Copy all source code (including pre-generated proto files)
COPY . .

# Build Server, Worker, Seeder, and qsops
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server ./src/backend/api-server/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/worker ./src/backend/worker/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/qsops ./src/backend/api/cmd/qsops/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/seeder ./src/backend/api/cmd/seeder


# Stage 2: Build the Web UI
FROM node:22 AS web-builder
WORKDIR /app

# Copy package files
COPY src/frontend/web/package.json src/frontend/web/package-lock.json* ./

# Install dependencies
RUN --mount=type=cache,target=/root/.npm npm install

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
ENV PGDATA=/var/lib/postgresql/pgdata
ENV QS_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
ENV QS_SERVER_PORT=8080
ENV QS_SERVER_STATIC_DIR="/app/web/dist"
ENV QS_ENV=sandbox

EXPOSE 5432 8080 8081

# Copy binaries from builder
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/worker /app/worker
COPY --from=builder /app/bin/qsops /usr/local/bin/qsops
COPY --from=builder /app/bin/seeder /app/seeder

# Copy Web UI build from web-builder
COPY --from=web-builder /app/dist /app/web/dist

# Copy database schema, seeds, and configs
COPY src/backend/db/quadsmith.sql /app/src/backend/db/quadsmith.sql
COPY test/seed_data /app/test/seed_data
COPY src/backend/api-server/config /app/src/backend/api-server/config
COPY src/backend/worker/config /app/src/backend/worker/config
COPY src/frontend/config /app/src/frontend/config

# Pre-populate PostgreSQL database with schema and seeds at image build time
RUN docker-entrypoint.sh postgres & \
    pid=$! && \
    until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 0.1; done && \
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith < /app/src/backend/db/quadsmith.sql && \
    ./seeder --config /app/src/backend/api-server/config/sandbox.textproto && \
    gosu postgres pg_ctl -D "$PGDATA" -m fast -w stop && \
    wait $pid

# Inline the sandbox entrypoint script
RUN printf '#!/usr/bin/env bash\n\
set -e\n\
echo "Starting PostgreSQL..."\n\
docker-entrypoint.sh postgres > /var/log/postgres.log 2>&1 &\n\
echo "Waiting for PostgreSQL to be ready..."\n\
until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 0.1; done\n\
ROW_COUNT=$(psql -U postgres -d quadsmith -tAc "SELECT count(*) FROM motors;" 2>/dev/null || echo "0")\n\
if [ "$ROW_COUNT" = "0" ]; then\n\
    echo "Applying Schema & Seeding Database..."\n\
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith < /app/src/backend/db/quadsmith.sql\n\
    ./seeder --config /app/src/backend/api-server/config/sandbox.textproto\n\
fi\n\
echo "Starting Background Worker Daemon..."\n\
./worker --config /app/src/backend/worker/config/sandbox.textproto > /var/log/worker.log 2>&1 &\n\
echo "Pre-seeded users:"\n\
psql -U postgres -d quadsmith -tAc "SELECT concat(\047  - \047, email, \047 (@\047, id, \047)\047) FROM users WHERE deleted_at IS NULL ORDER BY email;"\n\
exec ./server --config /app/src/backend/api-server/config/sandbox.textproto\n' > /app/sandbox-entrypoint.sh

RUN chmod +x /app/sandbox-entrypoint.sh

ENTRYPOINT ["/app/sandbox-entrypoint.sh"]
