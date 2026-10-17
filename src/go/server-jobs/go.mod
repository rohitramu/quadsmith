module quadsmith/server-jobs

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	github.com/robfig/cron/v3 v3.0.1
	google.golang.org/protobuf v1.36.12
	quadsmith/shared v0.0.0-00010101000000-000000000000
)

replace quadsmith/shared => ../shared
