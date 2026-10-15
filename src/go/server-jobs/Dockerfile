FROM golang:1.27 AS builder
WORKDIR /app
ENV GOWORK=/app/src/go/go.work

COPY src/go/go.work src/go/go.work.sum* ./src/go/
COPY src/go/build_deps/go.mod src/go/build_deps/go.sum* ./src/go/build_deps/
COPY src/go/shared/go.mod src/go/shared/go.sum* ./src/go/shared/
COPY src/go/server-api/go.mod src/go/server-api/go.sum* ./src/go/server-api/
COPY src/go/server-ops/go.mod src/go/server-ops/go.sum* ./src/go/server-ops/
COPY src/go/server-jobs/go.mod src/go/server-jobs/go.sum* ./src/go/server-jobs/
COPY src/go/server-web/go.mod src/go/server-web/go.sum* ./src/go/server-web/
COPY src/go/cli/go.mod src/go/cli/go.sum* ./src/go/cli/
COPY src/go/cli-ops/go.mod src/go/cli-ops/go.sum* ./src/go/cli-ops/
COPY test/go.mod test/go.sum* ./test/
COPY src/go/vendor/ ./src/go/vendor/

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o /app/bin/server-jobs ./src/go/server-jobs/main.go

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/server-jobs /app/server-jobs
COPY src/go/server-jobs/config /app/src/go/server-jobs/config

ENTRYPOINT ["/app/server-jobs", "--config", "/app/src/go/server-jobs/config/prod.textproto"]
