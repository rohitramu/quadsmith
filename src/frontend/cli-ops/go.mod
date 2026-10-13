module quadsmith/cli-ops

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	github.com/spf13/cobra v1.10.2
	google.golang.org/protobuf v1.36.12
	quadsmith/shared v0.0.0
)

replace quadsmith/shared => ../../backend/shared
