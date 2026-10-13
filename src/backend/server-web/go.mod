module quadsmith/server-web

go 1.27.1

require (
	golang.org/x/net v0.26.0
	quadsmith/shared v0.0.0
)

require (
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace quadsmith/shared => ../shared
