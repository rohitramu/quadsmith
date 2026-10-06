# protoc-gen-sql

This directory contains the custom Protobuf-to-SQL compiler plugin.

It reads `.proto` files via standard `buf generate` execution, inspects custom message and field extensions (defined in `quadsmith/sql.proto`), and automatically outputs a strictly typed Postgres `schema.sql` definition containing the primary keys, foreign keys, and indexes.
