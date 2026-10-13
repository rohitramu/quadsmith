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
#   - Database schemas (quadsmith_api.sql, quadsmith_ops.sql) and initial seed data
#   - Database Seeder binary (populates hardware catalog on startup)
#   - Go Backend API Server (ConnectRPC on port 8080)
#   - Go Backend Ops Server (ConnectRPC on port 8081)
#   - Go Backend Job Server (Periodic jobs daemon)
#   - React 19 SPA Web Frontend (served directly from the API server)
#
# Usage:
#   - Automatically initializes Postgres, applies schema with psqldef, runs
#     the seeder, and starts the API server, ops server, job server, and web UI.
# ==============================================================================

# Stage 1: Build the Go binaries
FROM golang:1.27 AS builder
WORKDIR /app
ENV GOWORK=/app/src/go.work

# Copy go.work, go.mod files and vendor directory
COPY src/go.work src/go.work.sum* ./src/
COPY src/build_deps/go.mod src/build_deps/go.sum* ./src/build_deps/
COPY src/backend/shared/go.mod src/backend/shared/go.sum* ./src/backend/shared/
COPY src/backend/server-api/go.mod src/backend/server-api/go.sum* ./src/backend/server-api/
COPY src/backend/server-ops/go.mod src/backend/server-ops/go.sum* ./src/backend/server-ops/
COPY src/backend/server-jobs/go.mod src/backend/server-jobs/go.sum* ./src/backend/server-jobs/
COPY src/backend/server-web/go.mod src/backend/server-web/go.sum* ./src/backend/server-web/
COPY src/frontend/cli/go.mod src/frontend/cli/go.sum* ./src/frontend/cli/
COPY src/frontend/cli-ops/go.mod src/frontend/cli-ops/go.sum* ./src/frontend/cli-ops/
COPY test/go.mod test/go.sum* ./test/
COPY src/vendor/ ./src/vendor/

# Copy all source code (including pre-generated proto files)
COPY . .

# Build Server, Ops Server, Job Server, Web Server, Seeder, and qsops
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server-api ./src/backend/server-api/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server-ops ./src/backend/server-ops/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server-jobs ./src/backend/server-jobs/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server-web ./src/backend/server-web/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/qsops ./src/frontend/cli-ops && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/seeder ./src/backend/shared/cmd/seeder


# Stage 2: Build the Web UI
FROM node:22 AS web-builder
WORKDIR /app

# Copy package files
COPY src/frontend/web/app/package.json src/frontend/web/app/package-lock.json* ./

# Install dependencies
RUN --mount=type=cache,target=/root/.npm npm install

# Copy source and build
COPY src/frontend/web/app/ .
RUN npm run build


# Stage 3: The Sandbox (Postgres + Go API + Ops Server + Job Server + Web UI)
FROM postgres:15
WORKDIR /app

# Install psqldef for schema management
RUN apt-get update && apt-get install -y wget && \
    wget -qO- https://github.com/sqldef/sqldef/releases/download/v0.17.15/psqldef_linux_amd64.tar.gz | tar xvz -C /usr/local/bin && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Environment Setup
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=quadsmith_api
ENV PGDATA=/var/lib/postgresql/pgdata
ENV QS_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api"
ENV QS_INTERNAL_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops"
ENV QS_SERVER_PORT=8080
ENV QS_OPS_SERVER_PORT=8081
ENV QS_WEB_SERVER_PORT=3000
ENV QS_ENV=sandbox

EXPOSE 5432 8080 8081 3000

# Copy binaries from builder
COPY --from=builder /app/bin/server-api /app/server-api
COPY --from=builder /app/bin/server-ops /app/server-ops
COPY --from=builder /app/bin/server-jobs /app/server-jobs
COPY --from=builder /app/bin/server-web /app/server-web
COPY --from=builder /app/bin/qsops /usr/local/bin/qsops
COPY --from=builder /app/bin/seeder /app/seeder

# Copy Web UI build from web-builder
COPY --from=web-builder /app/dist /app/src/frontend/web/app/dist

# Copy database schema, seeds, and configs
COPY src/backend/db/quadsmith_api.sql /app/src/backend/db/quadsmith_api.sql
COPY src/backend/db/quadsmith_ops.sql /app/src/backend/db/quadsmith_ops.sql
COPY test/seed_data /app/test/seed_data
COPY src/backend/server-api/config /app/src/backend/server-api/config
COPY src/backend/server-ops/config /app/src/backend/server-ops/config
COPY src/backend/server-jobs/config /app/src/backend/server-jobs/config
COPY src/frontend/web/config /app/src/frontend/web/config
COPY src/frontend/cli/config /app/src/frontend/cli/config
COPY src/frontend/cli-ops/config /app/src/frontend/cli-ops/config

# Pre-populate PostgreSQL database with schema and seeds at image build time
RUN docker-entrypoint.sh postgres & \
    pid=$! && \
    until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 0.1; done && \
    psql -U postgres -c "CREATE DATABASE quadsmith_ops;" && \
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_api < /app/src/backend/db/quadsmith_api.sql && \
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_ops < /app/src/backend/db/quadsmith_ops.sql && \
    ./seeder --config /app/src/backend/server-api/config/sandbox.textproto && \
    gosu postgres pg_ctl -D "$PGDATA" -m fast -w stop && \
    wait $pid

# Inline the sandbox entrypoint script
RUN printf '#!/usr/bin/env bash\n\
set -e\n\
echo "Starting PostgreSQL..."\n\
docker-entrypoint.sh postgres > /var/log/postgres.log 2>&1 &\n\
echo "Waiting for PostgreSQL to be ready..."\n\
until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 0.1; done\n\
psql -U postgres -c "CREATE DATABASE quadsmith_ops;" 2>/dev/null || true\n\
ROW_COUNT=$(psql -U postgres -d quadsmith_api -tAc "SELECT count(*) FROM motors;" 2>/dev/null || echo "0")\n\
if [ "$ROW_COUNT" = "0" ]; then\n\
    echo "Applying Schema & Seeding Database..."\n\
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_api < /app/src/backend/db/quadsmith_api.sql\n\
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_ops < /app/src/backend/db/quadsmith_ops.sql\n\
    ./seeder --config /app/src/backend/server-api/config/sandbox.textproto\n\
else\n\
    psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_ops < /app/src/backend/db/quadsmith_ops.sql 2>/dev/null || true\n\
fi\n\
echo "Starting Ops Server (:8081)..."\n\
./server-ops --config /app/src/backend/server-ops/config/sandbox.textproto > /var/log/server-ops.log 2>&1 &\n\
echo "Starting Job Server..."\n\
./server-jobs --config /app/src/backend/server-jobs/config/sandbox.textproto > /var/log/server-jobs.log 2>&1 &\n\
echo "Starting Web Server (:3000)..."\n\
./server-web --config /app/src/frontend/web/config/sandbox.textproto > /var/log/server-web.log 2>&1 &\n\
echo "Pre-seeded users:"\n\
psql -U postgres -d quadsmith_api -tAc "SELECT concat(\047  - \047, email, \047 (@\047, id, \047)\047) FROM users WHERE deleted_at IS NULL ORDER BY email;"\n\
exec ./server-api --config /app/src/backend/server-api/config/sandbox.textproto\n' > /app/sandbox-entrypoint.sh

RUN chmod +x /app/sandbox-entrypoint.sh

ENTRYPOINT ["/app/sandbox-entrypoint.sh"]
