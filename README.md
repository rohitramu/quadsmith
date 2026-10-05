# Quadsmith

## Development

### Build dependencies

Before running `make` or building the project, ensure you have the following installed on your system:

- **Go** (1.22 or higher)
- **Docker** & **Docker Compose** (for the database / sandbox)
- **Make**

*Note: All Go dependencies and build tools (like `buf` and `protoc` plugins) are hermetically vendored in the `vendor/` directory and will be built automatically during the compilation process. No internet connection is required to fetch Go modules during `make build`.*
