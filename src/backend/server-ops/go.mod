module quadsmith/server-ops

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	github.com/jackc/pgx/v5 v5.11.0
	golang.org/x/net v0.59.0
	quadsmith/shared v0.0.0
)

replace (
	quadsmith/shared => ../shared
)
