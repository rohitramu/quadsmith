//go:build tools
// +build tools

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "connectrpc.com/connect/cmd/protoc-gen-connect-go"
)
