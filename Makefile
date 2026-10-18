.PHONY: all build sandbox deploy db-migrate check-breaking generate fmt verify clean test update-snapshots vendor tool breaking-change test-breaking warn-breaking

export GOWORK := $(shell pwd)/src/go/go.work

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

MODULES        := src/go/shared src/go/server-api src/go/server-ops-api src/go/server-ops-web src/go/server-cron src/go/server-web src/go/cli src/go/cli-ops test src/go/build_deps
VENDOR_PLUGINS := bin/protoc-gen-go bin/protoc-gen-connect-go bin/buf bin/txtpbfmt
CUSTOM_PLUGINS := bin/protoc-gen-sql bin/protoc-gen-store bin/protoc-gen-server
PROTO_FILES    := $(shell find src/proto -type f 2>/dev/null)
CLI_FILES      := $(shell find src/go/cli -name "*.go" -not -name "*_test.go" 2>/dev/null)
OPS_CLI_FILES  := $(shell find src/go/cli-ops -name "*.go" -not -name "*_test.go" 2>/dev/null)

all: build

tidy:
	@echo "--- Running go mod tidy on all modules ---"
	@for mod in $(MODULES); do \
		(cd $$mod && go mod tidy) || exit 1; \
	done
	@echo "--- Syncing vendor directory ---"
	@cd src/go && go work vendor

fmt: $(VENDOR_PLUGINS)
	@echo "--- Formatting Python code ---"
	@if command -v uvx >/dev/null 2>&1; then \
		uvx ruff check --select I --fix tools/; \
		uvx ruff format tools/; \
	elif command -v ruff >/dev/null 2>&1; then \
		ruff check --select I --fix tools/; \
		ruff format tools/; \
	fi
	@echo "--- Formatting Go code with gofmt ---"
	@gofmt -s -w $(shell find . -name "*.go" -not -path "*/vendor/*" -not -path "*/.tmp/*")
	@echo "--- Formatting TypeScript, HTML, CSS with Prettier ---"
	@npx --prefix src/react/app-web prettier --write "src/react/app-web/index.html" "src/react/app-web/src/**/*.{ts,tsx,css,html}" "src/react/app-ops-web/index.html" "src/react/app-ops-web/src/**/*.{ts,tsx,css,html}" "src/react/shared/src/**/*.{ts,tsx,css,html}" >/dev/null
	@echo "--- Formatting and linting Markdown ---"
	@npx --prefix src/react/app-web prettier --prose-wrap always --ignore-path .prettierignore --write "**/*.md" >/dev/null
	@npx --prefix src/react/app-web markdownlint -c .markdownlint.json --ignore-path .markdownlintignore --fix "**/*.md"
	@echo "--- Formatting Protobuf with buf ---"
	@cd src/proto && PATH="$(shell pwd)/bin:$$PATH" buf format -w
	@echo "--- Formatting textproto files with txtpbfmt ---"
	@find src test deploy -name "*.textproto" -exec ./bin/txtpbfmt -expand_all_children {} +
	@echo "--- Formatting SQL files with sql-formatter ---"
	@for f in src/db/*.sql; do npx -y sql-formatter --config .sql-formatter.json --fix "$$f"; done

build: verify
	@echo "--- Building all Go server binaries to bin/ ---"
	@mkdir -p bin
	@go build -mod=vendor -o bin/server-api ./src/go/server-api/main.go
	@go build -mod=vendor -o bin/server-ops-api ./src/go/server-ops-api/main.go
	@go build -mod=vendor -o bin/server-ops-web ./src/go/server-ops-web/main.go
	@go build -mod=vendor -o bin/server-cron ./src/go/server-cron/main.go
	@go build -mod=vendor -o bin/server-web ./src/go/server-web/main.go
	@go build -mod=vendor -o bin/seeder ./src/go/shared/cmd/seeder
	@echo "--- Building CLI (qs) binaries to bin/ ---"
	@go build -mod=vendor -tags sandbox -o bin/qs-sandbox ./src/go/cli
	@go build -mod=vendor -tags staging -o bin/qs-staging ./src/go/cli
	@go build -mod=vendor -tags prod -o bin/qs-prod ./src/go/cli
	@cp -f bin/qs-sandbox bin/qs
	@echo "--- Building Operations CLI (qsops) binaries to bin/ ---"
	@go build -mod=vendor -tags sandbox -o bin/qsops-sandbox ./src/go/cli-ops
	@go build -mod=vendor -tags staging -o bin/qsops-staging ./src/go/cli-ops
	@go build -mod=vendor -tags prod -o bin/qsops-prod ./src/go/cli-ops
	@cp -f bin/qsops-sandbox bin/qsops
	@echo "--- Generating shell completions to bin/ ---"
	@./bin/qs completion bash > bin/completion.bash || true
	@./bin/qs completion zsh > bin/completion.zsh || true
	@./bin/qsops completion bash > bin/qsops_completion.bash || true
	@./bin/qsops completion zsh > bin/qsops_completion.zsh || true
	@echo "--- Building WebAssembly Core Engine ---"
	@cp -f $$(go env GOROOT)/lib/wasm/wasm_exec.js src/react/app-web/public/wasm_exec.js
	@GOOS=js GOARCH=wasm go build -mod=vendor -ldflags="-w -s" -o src/react/app-web/public/quadsmith-engine.wasm ./src/go/shared/cmd/wasm/main.go
	@echo "--- Building Web UI & Ops Console UI ---"
	@cd src/react/app-web && npm run build
	@cd src/react/app-ops-web && npm run build

bin/qsops: $(OPS_CLI_FILES) bin/.generate_proto.sentinel src/go/vendor/modules.txt
	@mkdir -p bin
	@echo "--- Building Operations CLI to bin/qsops ---"
	@go build -mod=vendor -tags sandbox -o bin/qsops-sandbox ./src/go/cli-ops
	@cp -f bin/qsops-sandbox bin/qsops
	@echo "--- Generating qsops shell completions to bin/ ---"
	@./bin/qsops completion bash > bin/qsops_completion.bash || true
	@./bin/qsops completion zsh > bin/qsops_completion.zsh || true

bin/qs: $(CLI_FILES) bin/.generate_proto.sentinel src/go/vendor/modules.txt
	@mkdir -p bin
	@echo "--- Generating and building CLI to bin/qs ---"
	@cd src/go/cli && go run generate_cli.go && gofmt -s -w main.go
	@go build -mod=vendor -tags sandbox -o bin/qs-sandbox ./src/go/cli
	@cp -f bin/qs-sandbox bin/qs
	@echo "--- Generating shell completions to bin/ ---"
	@./bin/qs completion bash > bin/completion.bash || true
	@./bin/qs completion zsh > bin/completion.zsh || true

sandbox: generate bin/qs bin/qsops
	@echo "--- Starting Quadsmith Sandbox (Docker) ---"
	@./test/start_sandbox.sh $(PORT) || if [ $$? -eq 130 ]; then exit 0; else exit $$?; fi

deploy:
	@./deploy/deploy.sh --env=$(or $(ENV),prod) $(TOOL_ARGS)

db-migrate:
	@echo "--- Applying Database Schemas with psqldef ---"
	@API_URL="$${QS_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable}"; \
	OPS_URL="$${QS_INTERNAL_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable}"; \
	run_psqldef() { \
		local url="$$1" file="$$2"; \
		local no_scheme="$${url#*://}"; \
		local userpass="$${no_scheme%%@*}"; \
		local user="$${userpass%%:*}"; \
		local pass="$${userpass#*:}"; \
		local hostportdb="$${no_scheme#*@}"; \
		local hostport="$${hostportdb%%/*}"; \
		local host="$${hostport%%:*}"; \
		local port="$${hostport#*:}"; \
		[ "$$port" = "$$host" ] && port="5432"; \
		local db_and_query="$${hostportdb#*/}"; \
		local db="$${db_and_query%%\?*}"; \
		psqldef -U "$$user" -W "$$pass" -h "$$host" -p "$$port" $$([ "$${DRY_RUN:-}" = "true" ] && echo "--dry-run") "$$db" < "$$file"; \
	}; \
	run_psqldef "$$API_URL" src/db/quadsmith_api.sql && \
	run_psqldef "$$OPS_URL" src/db/quadsmith_ops.sql

check-breaking:
	@echo "--- Running Strict Backwards-Compatibility Gate ---"
	@cd src/go/shared && STRICT_BREAKING=1 go test -mod=vendor -count=1 -v ./internal/regression/...

warn-breaking:
	@if [ ! -s .tmp/breaking_warning.log ]; then \
		rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log; \
		cd src/go/shared && go test -mod=vendor -count=1 ./internal/regression/... > /dev/null 2>&1 || true; \
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

src/go/vendor/modules.txt: $(shell find src/go -name "go.mod" -o -name "go.sum" 2>/dev/null) src/go/go.work
	@echo "--- Syncing vendor directory ---"
	@cd src/go && go work vendor

$(VENDOR_PLUGINS): src/go/vendor/modules.txt
	@mkdir -p bin
	@echo "--- Installing protoc plugins from vendor ---"
	@cd src/go && go build -mod=vendor -o $(CURDIR)/bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	@cd src/go && go build -mod=vendor -o $(CURDIR)/bin/protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go
	@cd src/go && go build -mod=vendor -o $(CURDIR)/bin/buf github.com/bufbuild/buf/cmd/buf
	@cd src/go && go build -mod=vendor -o $(CURDIR)/bin/txtpbfmt github.com/protocolbuffers/txtpbfmt/cmd/txtpbfmt

bin/protoc-gen-sql: src/go/vendor/modules.txt $(wildcard src/go/shared/cmd/protoc-gen-sql/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-sql ---"
	@go build -mod=vendor -o bin/protoc-gen-sql ./src/go/shared/cmd/protoc-gen-sql

bin/protoc-gen-store: src/go/vendor/modules.txt $(wildcard src/go/shared/cmd/protoc-gen-store/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-store ---"
	@go build -mod=vendor -o bin/protoc-gen-store ./src/go/shared/cmd/protoc-gen-store

bin/protoc-gen-server: src/go/vendor/modules.txt $(wildcard src/go/shared/cmd/protoc-gen-server/*.go)
	@mkdir -p bin
	@echo "--- Building custom protoc plugin: protoc-gen-server ---"
	@go build -mod=vendor -o bin/protoc-gen-server ./src/go/shared/cmd/protoc-gen-server

bin/.generate_proto.sentinel: $(PROTO_FILES) $(VENDOR_PLUGINS) $(CUSTOM_PLUGINS)
	@mkdir -p bin
	@echo "--- Generating Protobuf & ConnectRPC Code ---"
	@cd src/proto && PATH="$(shell pwd)/bin:$$PATH" buf format -w
	@cd src/proto && PATH="$(shell pwd)/bin:$$PATH" buf generate
	@gofmt -s -w src/go/shared/gen/
	@npx --prefix src/react/app-web prettier --write "src/react/shared/src/gen/**/*.{ts,tsx}" || true
	@touch bin/.generate_proto.sentinel

bin/.format_sql.sentinel: $(wildcard src/db/*.sql) .sql-formatter.json
	@mkdir -p bin
	@for f in src/db/*.sql; do npx -y sql-formatter --config .sql-formatter.json --fix "$$f"; done
	@touch bin/.format_sql.sentinel

bin/.generate_cli.sentinel: $(CLI_FILES) bin/.generate_proto.sentinel
	@mkdir -p bin
	@echo "--- Generating CLI code ---"
	@cd src/go/cli && go run generate_cli.go && gofmt -s -w main.go
	@touch bin/.generate_cli.sentinel

generate: bin/.generate_proto.sentinel bin/.format_sql.sentinel bin/.generate_cli.sentinel

verify: tidy generate fmt
	@echo "--- Verifying Protobuf regex patterns (Go RE2) ---"
	@go run -mod=vendor ./src/go/shared/cmd/check-regex
	@echo "--- Validating all application configurations ---"
	@go run -mod=vendor ./src/go/shared/cmd/check-config
	@echo "--- Verifying shell scripts syntax ---"
	@bash -n deploy/deploy.sh deploy/deploy_blue_green.sh test/run_sandbox_tests.sh test/start_sandbox.sh .githooks/pre-commit
	@if [ "$${CI:-false}" = "true" ]; then \
		echo "--- Verifying Clean Git Working Tree (CI) ---"; \
		git diff --exit-code || (echo "Code generation or formatting is out of sync. Please run 'make generate fmt' and commit the changes." && exit 1); \
	fi

test: verify build
	@rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log
	@echo "--- Running Go Tests ---"
	@cd src/go/shared && go test -mod=vendor -count=1 ./...
	@cd src/go/server-api && go test -mod=vendor -count=1 ./...
	@cd src/go/server-ops-api && go test -mod=vendor -count=1 ./...
	@cd src/go/server-ops-web && go test -mod=vendor -count=1 ./...
	@cd src/go/server-cron && go test -mod=vendor -count=1 ./...
	@cd src/go/server-web && go test -mod=vendor -count=1 ./...
	@cd src/go/cli && go test -mod=vendor ./...
	@cd src/go/cli-ops && go test -mod=vendor ./...
	@echo "--- Running Web UI & Ops Console UI Tests ---"
	@cd src/react/app-web && npm test
	@cd src/react/app-ops-web && npm test
	@echo "--- Running Full E2E Sandbox Tests (All Browsers & Profiles) ---"
	@./test/run_sandbox_tests.sh $(if $(PROJECT),--project=$(PROJECT)) $(if $(SHARD),--shard=$(SHARD)) $(TOOL_ARGS)
	@rm -f .tmp/breaking_warning.log .tmp/breaking_*.log .tmp/clean_*.log

update-snapshots:
	@echo "--- Updating Web Visual Regression Snapshots Across All Browsers ---"
	@UPDATE_SNAPSHOTS=true ./test/run_sandbox_tests.sh $(if $(PROJECT),--project=$(PROJECT)) $(TOOL_ARGS)

clean:
	@echo "--- Cleaning Workspace ---"
	@rm -rf bin/ .tmp/

vendor:
	@echo "--- Updating and vendoring dependencies ---"
	@for mod in src/go/shared src/go/server-api src/go/server-ops-api src/go/server-ops-web src/go/server-cron src/go/server-web src/go/cli src/go/cli-ops test; do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy) || exit 1; \
	done
	@echo "Updating tools..."
	@cd src/go/build_deps && go get github.com/bufbuild/buf/cmd/buf@latest google.golang.org/protobuf/cmd/protoc-gen-go@latest connectrpc.com/connect/cmd/protoc-gen-connect-go@latest github.com/protocolbuffers/txtpbfmt/cmd/txtpbfmt@latest && go mod tidy
	@echo "--- Syncing vendor directory ---"
	@cd src/go && go work vendor

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
