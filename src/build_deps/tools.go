//go:build tools
// +build tools

package tools

import (
	_ "connectrpc.com/connect/cmd/protoc-gen-connect-go"
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/protocolbuffers/txtpbfmt/cmd/txtpbfmt"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
