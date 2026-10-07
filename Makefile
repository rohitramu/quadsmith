.PHONY: all build sandbox generate clean test vendor tool

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

all: build

build: generate
	@echo "--- Building all Go binaries to bin/ ---"
	@go build -mod=vendor -o bin/server ./src/backend/api/cmd/server/main.go

	@echo "--- Generating and building CLI to bin/qs ---"
	@cd src/frontend/cli && go run generate_cli.go && gofmt -w main.go && go build -mod=vendor -o ../../../bin/qs main.go
	@echo "--- Generating shell completions to bin/ ---"
	@./bin/qs completion bash > bin/completion.bash || true
	@./bin/qs completion zsh > bin/completion.zsh || true

sandbox: build
	@echo "--- Starting Quadsmith Sandbox (Docker) ---"
	@./src/backend/start_sandbox.sh $(PORT) || if [ $$? -eq 130 ]; then exit 0; else exit $$?; fi

generate:
	@echo "--- Installing protoc plugins from vendor ---"
	@cd src && go build -mod=vendor -o ../bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	@cd src && go build -mod=vendor -o ../bin/protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go
	@cd src && go build -mod=vendor -o ../bin/buf github.com/bufbuild/buf/cmd/buf
	@echo "--- Building custom protoc plugins ---"
	@go build -mod=vendor -o bin/protoc-gen-sql ./src/backend/api/cmd/protoc-gen-sql
	@go build -mod=vendor -o bin/protoc-gen-store ./src/backend/api/cmd/protoc-gen-store
	@go build -mod=vendor -o bin/protoc-gen-server ./src/backend/api/cmd/protoc-gen-server
	@echo "--- Generating Protobuf & ConnectRPC Code ---"
	@cd proto && PATH="$(shell pwd)/bin:$$PATH" buf generate
	@gofmt -w src/backend/api/gen/
	@npx -y sql-formatter -l postgresql --fix src/backend/db/schema.sql

test: generate
	@echo "--- Running Go Tests ---"
	@cd src/backend/api && go test -mod=vendor ./...
	@cd src/frontend/cli && go test -mod=vendor ./...

clean:
	@echo "--- Cleaning Workspace ---"
	@rm -rf bin/

vendor:
	@echo "--- Updating and vendoring dependencies ---"
	@for mod in src/backend/api src/frontend/cli; do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy) || exit 1; \
	done
	@echo "Updating tools..."
	@cd src/build_deps && go get github.com/bufbuild/buf/cmd/buf@latest google.golang.org/protobuf/cmd/protoc-gen-go@latest connectrpc.com/connect/cmd/protoc-gen-connect-go@latest && go mod tidy
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
