module quadsmith/server-jobs

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	quadsmith/shared v0.0.0
)

require google.golang.org/protobuf v1.36.12 // indirect

replace quadsmith/shared => ../shared
