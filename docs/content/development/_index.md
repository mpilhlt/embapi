---
title: "Development"
weight: 6
---

# Development Guide

Information for developers contributing to embapi.

## Getting Started with Development

This section covers:

- Setting up a development environment
- Running tests
- Understanding the codebase architecture
- Contributing guidelines
- Performance optimization

## Project Structure

```
embapi/
├── main.go                    # Application entry point
├── internal/
│   ├── auth/                  # Authentication logic
│   ├── database/              # Database layer (sqlc)
│   ├── handlers/              # HTTP handlers
│   └── models/                # Data models
├── testdata/                  # Test fixtures
└── docs/                      # Documentation
```

## Development Workflow

1. Make changes to code
2. Generate sqlc code if database queries changed: `sqlc generate`
3. Run tests: `go test -v ./...`
4. Build: `go build -o embapi main.go`
5. Submit pull request

## Resources

- [Testing](testing/) - How to run tests
- [Contributing](contributing/) - Contribution guidelines
- [Architecture](architecture/) - Technical deep-dive
- [Performance](performance/) - Optimization notes
