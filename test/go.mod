module quadsmith/test

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	github.com/jackc/pgx/v5 v5.11.0
	golang.org/x/net v0.59.0
	google.golang.org/protobuf v1.36.12
	quadsmith/server-ops v0.0.0
	quadsmith/shared v0.0.0
)

replace quadsmith/shared => ../src/backend/shared
replace quadsmith/server-ops => ../src/backend/server-ops
replace quadsmith/frontend/config => ../src/frontend/config/go
