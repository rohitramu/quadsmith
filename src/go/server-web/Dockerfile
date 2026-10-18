FROM golang:1.27 AS builder
WORKDIR /app
ENV GOWORK=/app/src/go/go.work

COPY src/go/go.work src/go/go.work.sum* ./src/go/
COPY src/go/build_deps/go.mod src/go/build_deps/go.sum* ./src/go/build_deps/
COPY src/go/shared/go.mod src/go/shared/go.sum* ./src/go/shared/
COPY src/go/server-api/go.mod src/go/server-api/go.sum* ./src/go/server-api/
COPY src/go/server-ops-api/go.mod src/go/server-ops-api/go.sum* ./src/go/server-ops-api/
COPY src/go/server-ops-web/go.mod src/go/server-ops-web/go.sum* ./src/go/server-ops-web/
COPY src/go/server-cron/go.mod src/go/server-cron/go.sum* ./src/go/server-cron/
COPY src/go/server-web/go.mod src/go/server-web/go.sum* ./src/go/server-web/
COPY src/go/cli/go.mod src/go/cli/go.sum* ./src/go/cli/
COPY src/go/cli-ops/go.mod src/go/cli-ops/go.sum* ./src/go/cli-ops/
COPY test/go.mod test/go.sum* ./test/
COPY src/go/vendor/ ./src/go/vendor/

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o /app/bin/server-web ./src/go/server-web/main.go && \
    GOOS=js GOARCH=wasm go build -mod=vendor -ldflags="-w -s" -o /app/bin/quadsmith-engine.wasm ./src/go/shared/cmd/wasm/main.go && \
    cp -f "$(go env GOROOT)/lib/wasm/wasm_exec.js" /app/bin/wasm_exec.js

FROM node:22 AS web-builder
WORKDIR /app

COPY src/react/package.json src/react/package-lock.json* ./src/react/
COPY src/react/shared/package.json ./src/react/shared/
COPY src/react/app-web/package.json src/react/app-web/package-lock.json* ./src/react/app-web/
COPY src/react/app-ops-web/package.json ./src/react/app-ops-web/
RUN cd src/react && npm install

COPY src/react/shared ./src/react/shared
COPY src/react/app-web ./src/react/app-web
COPY --from=builder /app/bin/quadsmith-engine.wasm ./src/react/app-web/public/quadsmith-engine.wasm
COPY --from=builder /app/bin/wasm_exec.js ./src/react/app-web/public/wasm_exec.js
RUN cd src/react/app-web && npm run build

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

EXPOSE 8080

COPY --from=builder /app/bin/server-web /app/server-web
COPY --from=web-builder /app/src/react/app-web/dist /app/src/react/app-web/dist
COPY src/go/server-web/config /app/src/go/server-web/config

ENV QS_CONFIG=/app/src/go/server-web/config/prod.textproto
ENTRYPOINT ["/app/server-web"]
