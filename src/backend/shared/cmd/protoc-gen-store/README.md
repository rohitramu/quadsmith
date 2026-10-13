# protoc-gen-store

This directory contains the custom Protobuf-to-Go-Store compiler plugin.

It automatically generates heavily optimized, strongly typed CRUD operations (`Create`, `Get`, `Update`, `Delete`, `List`) using `github.com/jackc/pgx/v5`. It gracefully handles native Postgres features like mapping Protobuf repeated strings to Postgres Arrays (`UUID[]`) and scanning `NULL` database columns safely into Go pointers.
