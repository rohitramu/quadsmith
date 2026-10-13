module quadsmith/server-api

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	github.com/jackc/pgx/v5 v5.11.0
	golang.org/x/net v0.59.0
	quadsmith/frontend/config v0.0.0
	quadsmith/shared v0.0.0
)

replace (
	quadsmith/frontend/config => ../../frontend/config/go
	quadsmith/shared => ../shared
)
