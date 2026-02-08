---
title: "Installation"
weight: 1
---

# Installation

Install dhamps-vdb by compiling from source.

## Prerequisites

- **Go 1.21 or later**
- **PostgreSQL 11+** with pgvector extension
- **sqlc** for code generation

## Quick Install

```bash
# Clone the repository
git clone https://github.com/mpilhlt/dhamps-vdb.git
cd dhamps-vdb

# Install dependencies and generate code
go get ./...
sqlc generate --no-remote

# Build the binary
go build -o build/dhamps-vdb main.go
```

## Detailed Steps

### 1. Install Dependencies

Download all Go module dependencies:

```bash
go get ./...
```

### 2. Generate Database Code

Generate type-safe database queries using sqlc:

```bash
sqlc generate --no-remote
```

This creates Go code from SQL queries in `internal/database/queries/`.

### 3. Build the Application

Compile the application:

```bash
go build -o build/dhamps-vdb main.go
```

The binary will be created at `build/dhamps-vdb`.

## Running Without Building

You can run the application directly without building a binary:

```bash
go run main.go
```

This is useful during development but slower than running a pre-built binary.

## Verify Installation

Check that the binary was created successfully:

```bash
./build/dhamps-vdb --help
```

You should see the available command-line options.

## Next Steps

After installation, you need to:

1. [Set up the database](../deployment/database/)
2. [Configure environment variables](configuration/)
3. [Run the service](quick-start/)

## System Requirements

- **Memory**: Minimum 512MB RAM (2GB+ recommended for production)
- **Disk**: Minimal (< 50MB for binary, database size varies)
- **CPU**: Any modern CPU (multi-core recommended for concurrent requests)

## Troubleshooting

### sqlc Command Not Found

Install sqlc:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Make sure `$GOPATH/bin` is in your PATH.

### Build Errors

Ensure you're using Go 1.21 or later:

```bash
go version
```

Clean the build cache if you encounter issues:

```bash
go clean -cache
go build -o build/dhamps-vdb main.go
```

### Missing Dependencies

Force update all dependencies:

```bash
go mod download
go get -u ./...
```
