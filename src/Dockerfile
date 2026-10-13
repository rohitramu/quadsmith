# ==============================================================================
# Quadsmith Production Container Image
# ==============================================================================
# Purpose:
#   Production-ready container image for deploying the Quadsmith web application.
#   Designed for cloud platforms (GCP Cloud Run, Fly.io, Render, Railway, VPS, etc.).
#
# Included Components:
#   - Go Backend API Server (ConnectRPC over HTTP/2 & HTTP/1.1)
#   - React 19 SPA Web Frontend (Vite build served directly by Go via static.ServeSPA)
#   - Root CA certificates and timezone data (Alpine Linux base)
#
# Requirements:
#   - Requires an external PostgreSQL database provided via the DATABASE_URL
#     environment variable (e.g. Neon, Cloud SQL, AWS RDS, or managed Postgres).
# ==============================================================================

# Stage 1: Build the Go server binary
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
COPY src/frontend/cli/go.mod src/frontend/cli/go.sum* ./src/frontend/cli/
COPY src/frontend/cli-ops/go.mod src/frontend/cli-ops/go.sum* ./src/frontend/cli-ops/
COPY src/frontend/config/go/go.mod src/frontend/config/go/go.sum* ./src/frontend/config/go/
COPY test/go.mod test/go.sum* ./test/
COPY src/vendor/ ./src/vendor/

# Copy all source code (including pre-generated proto files)
COPY . .

# Build Server binary
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o /app/bin/server ./src/backend/server-api/main.go


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


# Stage 3: Production Runtime
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

ENV QS_SERVER_PORT=443
ENV QS_SERVER_STATIC_DIR="/app/web/dist"

EXPOSE 443 8080

COPY --from=builder /app/bin/server /app/server
COPY --from=web-builder /app/dist /app/web/dist
COPY src/backend/server-api/config /app/src/backend/server-api/config
 
ENTRYPOINT ["/app/server", "--config", "/app/src/backend/server-api/config/prod.textproto"]
