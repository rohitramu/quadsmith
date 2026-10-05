.PHONY: all build sandbox generate clean test vendor

# Default port for the sandbox API
PORT ?= 8080

all: build

build: generate
	@echo "--- Building all Go binaries to bin/ ---"
	@GOBIN=$(shell pwd)/bin go install -mod=vendor quadsmith/api/cmd/... quadsmith/cli
	@echo "--- Generating shell completions to bin/ ---"
	@./bin/cli completion bash > bin/completion.bash || true
	@./bin/cli completion zsh > bin/completion.zsh || true

sandbox: build
	@echo "--- Starting Quadsmith Sandbox (Docker) ---"
	@./backend/start_sandbox.sh $(PORT) || if [ $$? -eq 130 ]; then exit 0; else exit $$?; fi

generate:
	@echo "--- Installing protoc plugins from vendor ---"
	@go build -mod=vendor -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	@go build -mod=vendor -o bin/protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go
	@go build -mod=vendor -o bin/buf github.com/bufbuild/buf/cmd/buf
	@echo "--- Generating Protobuf & ConnectRPC Code ---"
	@cd proto && PATH="$(shell pwd)/bin:$$PATH" buf generate

test: generate
	@echo "--- Running Go Tests ---"
	@cd backend/api && go test -mod=vendor ./...
	@cd backend/engines && go test -mod=vendor ./...
	@cd frontend/cli && go test -mod=vendor ./...

clean:
	@echo "--- Cleaning Workspace ---"
	@rm -rf bin/

vendor:
	@echo "--- Updating and vendoring dependencies ---"
	@for mod in backend/api backend/engines frontend/cli; do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy) || exit 1; \
	done
	@echo "Updating tools..."
	@cd tools && go get github.com/bufbuild/buf/cmd/buf@latest google.golang.org/protobuf/cmd/protoc-gen-go@latest connectrpc.com/connect/cmd/protoc-gen-connect-go@latest && go mod tidy
	@echo "--- Syncing vendor directory ---"
	@go work vendor
