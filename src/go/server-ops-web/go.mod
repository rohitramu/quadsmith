module quadsmith/server-ops-web

go 1.27.1

require (
	connectrpc.com/connect v1.21.0
	google.golang.org/protobuf v1.36.12
	quadsmith/shared v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace quadsmith/shared => ../shared
