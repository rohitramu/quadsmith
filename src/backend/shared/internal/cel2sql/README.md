# cel2sql

This package is responsible for translating Common Expression Language (CEL) filter queries from the frontend into native PostgreSQL `WHERE` clauses.

It parses the AST (Abstract Syntax Tree) using Google's `cel-go` library and prevents SQL injection by strictly validating identifiers and mapping all dynamic values to parameterized SQL arguments (e.g., `$1`, `$2`).
