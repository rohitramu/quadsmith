.PHONY: all build sandbox generate clean test vendor

# Default port for the sandbox API
PORT ?= 8080

all: build

build: generate
	@echo "--- Building all Go binaries to bin/ ---"
	@go build -mod=vendor -o bin/server ./backend/api/cmd/server/main.go
	
	@echo "--- Generating and building CLI to bin/qs ---"
	@cd frontend/cli && go run generate_cli.go && go build -mod=vendor -o ../../bin/qs main.go

sandbox: build
	@echo "--- Starting Quadsmith Sandbox (Docker) ---"
	@./backend/start_sandbox.sh $(PORT) || if [ $$? -eq 130 ]; then exit 0; else exit $$?; fi

generate:
	@echo "--- Installing protoc plugins from vendor ---"
	@go build -mod=vendor -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	@go build -mod=vendor -o bin/protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go
	@go build -mod=vendor -o bin/buf github.com/bufbuild/buf/cmd/buf
	@echo "--- Building custom protoc plugins ---"
	@go build -mod=vendor -o bin/protoc-gen-sql ./backend/api/cmd/protoc-gen-sql
	@go build -mod=vendor -o bin/protoc-gen-store ./backend/api/cmd/protoc-gen-store
	@go build -mod=vendor -o bin/protoc-gen-server ./backend/api/cmd/protoc-gen-server
	@echo "--- Generating Protobuf & ConnectRPC Code ---"
	@cd proto && PATH="$(shell pwd)/bin:$$PATH" buf generate

test: generate
	@echo "--- Running Go Tests ---"
	@cd backend/api && go test -mod=vendor ./...
	@cd frontend/cli && go test -mod=vendor ./...

clean:
	@echo "--- Cleaning Workspace ---"
	@rm -rf bin/

vendor:
	@echo "--- Updating and vendoring dependencies ---"
	@for mod in backend/api frontend/cli; do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy) || exit 1; \
	done
	@echo "Updating tools..."
	@cd tools && go get github.com/bufbuild/buf/cmd/buf@latest google.golang.org/protobuf/cmd/protoc-gen-go@latest connectrpc.com/connect/cmd/protoc-gen-connect-go@latest && go mod tidy
	@echo "--- Syncing vendor directory ---"
	@go work vendor
