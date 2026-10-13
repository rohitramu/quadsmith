module quadsmith/server-web

go 1.27.1

require (
	github.com/jackc/pgx/v5 v5.11.0
	quadsmith/shared v0.0.0
)

replace quadsmith/shared => ../shared
