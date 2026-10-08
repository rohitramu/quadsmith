# Stage 1: Build the Go server binary
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

# Build Server binary
RUN cd src/backend/api && CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o /app/bin/server ./cmd/server/main.go


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

ENV PORT=8080
ENV STATIC_DIR="/app/web/dist"

EXPOSE 8080

COPY --from=builder /app/bin/server /app/server
COPY --from=web-builder /app/dist /app/web/dist

ENTRYPOINT ["/app/server"]
