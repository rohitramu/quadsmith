# Stage 1: Build the Go binaries
FROM golang:1.27 AS builder
WORKDIR /app

# Copy go.work, go.mod files and vendor directory
COPY go.work go.work.sum ./
COPY tools/go.mod tools/go.sum ./tools/
COPY backend/api/go.mod backend/api/go.sum ./backend/api/
COPY frontend/cli/go.mod frontend/cli/go.sum* ./frontend/cli/
COPY vendor/ ./vendor/

# Copy all source code (including pre-generated proto files)
COPY . .

# Build Server and Seeder
RUN cd backend/api && CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/server ./cmd/server/main.go && CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/seeder ./cmd/seeder/main.go


# Stage 2: The Sandbox (Postgres + Go API)
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

EXPOSE 5432 8080

# Inline the sandbox entrypoint script securely using printf (legacy builder compatible)
RUN printf '#!/usr/bin/env bash\n\
set -e\n\
echo "Starting PostgreSQL..."\n\
docker-entrypoint.sh postgres &\n\
echo "Waiting for PostgreSQL to be ready..."\n\
until pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; do sleep 1; done\n\
echo "Applying Schema..."\n\
psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith < /app/backend/db/schema.sql\n\
echo "Seeding Database..."\n\
./seeder\n\
exec ./server\n' > /app/sandbox-entrypoint.sh

RUN chmod +x /app/sandbox-entrypoint.sh

# Copy binaries from builder
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/seeder /app/seeder

# Copy database schema and seeds at the very end
COPY backend/db/schema.sql /app/backend/db/schema.sql
COPY backend/db/seeds /app/backend/db/seeds

ENTRYPOINT ["/app/sandbox-entrypoint.sh"]
