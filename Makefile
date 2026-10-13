.PHONY: all build sandbox generate clean test test-e2e vendor tool breaking-change test-breaking warn-breaking

export GOWORK := $(shell pwd)/src/go.work

# Default port for the sandbox API
PORT ?= 8080

# Extra arguments to pass to the tool
TOOL_ARGS ?=

# Dynamic rule to extract the tool name and arguments for "make tool <tool_name> [args...]"
ifeq (tool,$(firstword $(MAKECMDGOALS)))
  TOOL_NAME := $(word 2,$(MAKECMDGOALS))
  # Capture all remaining arguments as TOOL_ARGS
  TOOL_ARGS := $(wordlist 3,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  
  # Turn the tool name into a do-nothing target so Make doesn't complain about "No rule to make target"
  $(eval $(TOOL_NAME):;@:)
  # Do the same for every single tool argument we captured
  $(foreach arg,$(TOOL_ARGS),$(eval $(arg):;@:))
endif

VENDOR_PLUGINS := bin/protoc-gen-go bin/protoc-gen-connect-go bin/buf bin/txtpbfmt
CUSTOM_PLUGINS := bin/protoc-gen-sql bin/protoc-gen-store bin/protoc-gen-server
PROTO_FILES    := $(shell find proto -type f 2>/dev/null)
CLI_FILES      := $(shell find src/frontend/cli -name "*.go" -not -name "*_test.go" 2>/dev/null)

all: build

build: generate bin/qs bin/qsops
	@echo "--- Formatting textproto files with txtpbfmt ---"
	@find src test -name "*.textproto" -exec ./bin/txtpbfmt -expand_all_children {} +
	@echo "--- Verifying Protobuf regex patterns (Go RE2) ---"
	@go run -mod=vendor ./src/backend/shared/cmd/check-regex
	@echo "--- Building all Go binaries to bin/ ---"
	@go build -mod=vendor -o bin/server-api ./src/backend/server-api/main.go
	@go build -mod=vendor -o bin/server-ops ./src/backend/server-ops/main.go
	@go build -mod=vendor -o bin/server-jobs ./src/backend/server-jobs/main.go
	@go build -mod=vendor -o bin/seeder ./src/backend/shared/cmd/seeder
	@echo "--- Building WebAssembly Core Engine ---"
	@cp -f $$(go env GOROOT)/lib/wasm/wasm_exec.js src/frontend/web/public/wasm_exec.js
	@GOOS=js GOARCH=wasm go build -mod=vendor -o src/frontend/web/public/quadsmith-engine.wasm ./src/backend/shared/cmd/wasm/main.go
	@echo "--- Building Web UI ---"
	@cd src/frontend/web && npm run build

bin/qsops: $(wildcard src/frontend/cli-ops/*.go) bin/.generate_proto.sentinel
	@mkdir -p bin
	@echo "--- Building Operations CLI to bin/qsops ---"
	@go build -mod=vendor -o bin/qsops ./src/frontend/cli-ops/main.go
	@echo "--- Generating qsops shell completions to bin/ ---"
	@./bin/qsops completion bash > bin/qsops_completion.bash || true
	@./bin/qsops completion zsh > bin/qsops_completion.zsh || true

bin/qs: $(CLI_FILES) bin/.generate_proto.sentinel
	@mkdir -p bin
	@echo "--- Generating and building CLI to bin/qs ---"
	@cd src/frontend/cli && go run generate_cli.go && gofmt -s -w main.go && go build -mod=vendor -o ../../../bin/qs .
	@echo "--- Generating shell completions to bin/ ---"
	@./bin/qs completion bash > bin/completion.bash || true
	@./bin/qs completion zsh > bin/completion.zsh || true

sandbox: generate bin/qs bin/qsops
	@echo "--- Starting Quadsmith Sandbox (Docker) ---"
	@./test/start_sandbox.sh $(PORT) || if [ $$? -eq 130 ]; then exit 0; else exit $$?; fi

db-migrate: generate
	@echo "--- Applying Database Schemas with psqldef ---"
	@psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_api < src/backend/db/quadsmith_api.sql
	@psqldef -U postgres -W postgres -h 127.0.0.1 quadsmith_ops < src/backend/db/quadsmith_ops.sql

warn-breaking:
	@if [ ! -s .tmp/breaking_warning.log ]; then \
		rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log; \
		cd src/backend/shared && go test -mod=vendor -count=1 ./internal/regression/... > /dev/null 2>&1 || true; \
	fi
	@if [ -s .tmp/breaking_warning.log ]; then \
		echo ""; \
		echo "================================================================================"; \
		echo "⚠️  WARNING: BREAKING CHANGES DETECTED"; \
		echo "================================================================================"; \
		cat .tmp/breaking_warning.log; \
		echo "================================================================================"; \
		echo ""; \
	fi

$(VENDOR_PLUGINS):
	@mkdir -p bin
	@echo "--- Installing protoc plugins from vendor ---"
	@cd src && go build -mod=vendor -o ../bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	@cd src && go build -mod=vendor -o ../bin/protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go
	@cd src && go build -mod=vendor -o ../bin/buf github.com/bufbuild/buf/cmd/buf
	@cd src && go build -mod=vendor -o ../bin/txtpbfmt github.com/protocolbuffers/txtpbfmt/cmd/txtpbfmt

bin/protoc-gen-sql: $(wildcard src/backend/shared/cmd/protoc-gen-sql/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-sql ---"
	@go build -mod=vendor -o bin/protoc-gen-sql ./src/backend/shared/cmd/protoc-gen-sql

bin/protoc-gen-store: $(wildcard src/backend/shared/cmd/protoc-gen-store/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-store ---"
	@go build -mod=vendor -o bin/protoc-gen-store ./src/backend/shared/cmd/protoc-gen-store

bin/protoc-gen-server: $(wildcard src/backend/shared/cmd/protoc-gen-server/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-server ---"
	@go build -mod=vendor -o bin/protoc-gen-server ./src/backend/shared/cmd/protoc-gen-server

bin/.generate_proto.sentinel: $(PROTO_FILES) $(VENDOR_PLUGINS) $(CUSTOM_PLUGINS)
	@mkdir -p bin
	@echo "--- Generating Protobuf & ConnectRPC Code ---"
	@cd proto && PATH="$(shell pwd)/bin:$$PATH" buf format -w
	@cd proto && PATH="$(shell pwd)/bin:$$PATH" buf generate
	@gofmt -s -w src/backend/shared/gen/
	@npx --prefix src/frontend/web prettier --write "src/frontend/web/src/gen/**/*.{ts,tsx}" || true
	@touch bin/.generate_proto.sentinel

bin/.format_sql.sentinel: $(wildcard src/backend/db/*.sql) .sql-formatter.json
	@mkdir -p bin
	@for f in src/backend/db/*.sql; do npx -y sql-formatter --config .sql-formatter.json --fix "$$f"; done
	@touch bin/.format_sql.sentinel

generate: bin/.generate_proto.sentinel bin/.format_sql.sentinel

test: generate
	@rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log
	@echo "--- Running Go Tests ---"
	@cd src/frontend/config/go && go test -mod=vendor -count=1 ./...
	@cd src/backend/shared && go test -mod=vendor -count=1 ./...
	@cd src/backend/server-api && go test -mod=vendor -count=1 ./...
	@cd src/backend/server-ops && go test -mod=vendor -count=1 ./...
	@cd src/backend/server-jobs && go test -mod=vendor -count=1 ./...
	@cd src/frontend/cli && go test -mod=vendor ./...
	@cd src/frontend/cli-ops && go test -mod=vendor ./...
	@echo "--- Running Web UI Tests ---"
	@cd src/frontend/web && npm test
	@$(MAKE) --no-print-directory test-e2e
	@rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log

breaking-change: generate
	@echo "--- Running Breaking Change Regression Tests ---"
	@cd src/backend/shared && STRICT_BREAKING=1 go test -mod=vendor -v -count=1 ./internal/regression/... ./pkg/schemacheck/...

test-breaking: breaking-change

test-e2e: bin/qs bin/qsops
	@echo "--- Running E2E Sandbox Tests ---"
	@./test/run_sandbox_tests.sh

clean:
	@echo "--- Cleaning Workspace ---"
	@rm -rf bin/ .tmp/

vendor:
	@echo "--- Updating and vendoring dependencies ---"
	@for mod in src/backend/shared src/backend/server-api src/backend/server-ops src/backend/server-jobs src/frontend/cli src/frontend/cli-ops src/frontend/config/go test; do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy) || exit 1; \
	done
	@echo "Updating tools..."
	@cd src/build_deps && go get github.com/bufbuild/buf/cmd/buf@latest google.golang.org/protobuf/cmd/protoc-gen-go@latest connectrpc.com/connect/cmd/protoc-gen-connect-go@latest github.com/protocolbuffers/txtpbfmt/cmd/txtpbfmt@latest && go mod tidy
	@echo "--- Syncing vendor directory ---"
	@cd src && go work vendor

tool:
	@if [ -z "$(TOOL_NAME)" ]; then echo "Usage: make tool <tool_name>"; exit 1; fi
	@if [ ! -d "tools/$(TOOL_NAME)" ]; then echo "Error: Directory 'tools/$(TOOL_NAME)' does not exist."; exit 1; fi
	@echo "--- Running tool: $(TOOL_NAME) $(TOOL_ARGS) ---"
	@if [ -f "tools/$(TOOL_NAME)/main.py" ]; then \
		cd tools/$(TOOL_NAME) && uv run main.py $(TOOL_ARGS); \
	elif [ -f "tools/$(TOOL_NAME)/main.go" ]; then \
		cd tools/$(TOOL_NAME) && GOWORK=off go run main.go $(TOOL_ARGS); \
	else \
		echo "Error: No main.py or main.go found in tools/$(TOOL_NAME)"; exit 1; \
	fi
