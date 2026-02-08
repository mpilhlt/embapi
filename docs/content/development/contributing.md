---
title: "Contributing"
weight: 2
---

# Contributing Guide

Thank you for your interest in contributing to dhamps-vdb! This guide will help you get started.

## Development Setup

### Prerequisites

- **Go 1.21+**: [Download Go](https://go.dev/dl/)
- **PostgreSQL 16+**: With pgvector extension
- **sqlc**: For generating type-safe database code
- **Docker/Podman**: For running tests
- **Git**: For version control

### Clone and Build

```bash
# Clone repository
git clone https://github.com/mpilhlt/dhamps-vdb.git
cd dhamps-vdb

# Install dependencies
go get ./...

# Generate sqlc code
sqlc generate --no-remote

# Build application
go build -o build/dhamps-vdb main.go

# Or run directly
go run main.go
```

### Environment Setup

Create a `.env` file for local development:

```bash
# Copy template
cp template.env .env

# Edit with your settings
SERVICE_DEBUG=true
SERVICE_HOST=localhost
SERVICE_PORT=8880
SERVICE_ADMINKEY=your-secure-admin-key-here

# Database settings
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=password
SERVICE_DBNAME=dhamps_vdb_dev

# Encryption key (32+ characters)
ENCRYPTION_KEY=your-secure-encryption-key-min-32-chars
```

**Important**: Never commit `.env` files (already in `.gitignore`)

### Database Setup

For local development:

```bash
# Start PostgreSQL with pgvector
podman run -p 5432:5432 \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=dhamps_vdb_dev \
  pgvector/pgvector:0.7.4-pg16

# Or use docker-compose
docker-compose up -d

# Application auto-migrates on startup
go run main.go
```

### Verify Setup

```bash
# Check application starts
go run main.go

# In another terminal, test API
curl http://localhost:8880/docs

# Run tests
go test -v ./...
```

## Code Style

### Go Formatting

dhamps-vdb follows standard Go conventions:

```bash
# Format all code
go fmt ./...

# Check for common issues
go vet ./...

# Run linter (if installed)
golangci-lint run
```

### Code Organization

Follow existing patterns:

```go
// Package comment at top of file
package handlers

import (
	// Standard library first
	"context"
	"fmt"
	
	// External packages
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	
	// Internal packages
	"github.com/mpilhlt/dhamps-vdb/internal/database"
	"github.com/mpilhlt/dhamps-vdb/internal/models"
)

// Exported function with doc comment
// GetUsers retrieves all users from the database
func GetUsers(ctx context.Context, pool *pgxpool.Pool) ([]models.User, error) {
	// Implementation
}
```

### Naming Conventions

**Files:**
- `handlers/users.go` - Implementation
- `handlers/users_test.go` - Tests
- `handlers/users_sharing_test.go` - Specific feature tests

**Functions:**
- `GetUsers()` - List/retrieve multiple
- `GetUser()` - Retrieve single
- `CreateUser()` - Create new
- `UpdateUser()` - Update existing
- `DeleteUser()` - Delete
- `LinkUserToProject()` - Create association
- `IsProjectOwner()` - Boolean check

**Database Queries (in `queries.sql`):**
- `-- name: GetAllUsers :many`
- `-- name: RetrieveUserByHandle :one`
- `-- name: UpsertUser :one`
- `-- name: DeleteUser :exec`

### Error Handling

```go
// Return errors, don't panic
func CreateUser(ctx context.Context, pool *pgxpool.Pool, user models.User) error {
	if user.Handle == "" {
		return fmt.Errorf("user handle is required")
	}
	
	// Wrap errors with context
	err := db.InsertUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	
	return nil
}

// Use specific error responses in handlers
func handleCreateUser(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
	err := CreateUser(ctx, pool, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid user data", err)
	}
	
	return &CreateUserOutput{Body: result}, nil
}
```

### Comments

Comment public APIs and complex logic:

```go
// GetAccessibleProjects returns all projects the user can access.
// This includes projects owned by the user and projects shared with them.
// Results are paginated using limit and offset.
func GetAccessibleProjects(ctx context.Context, pool *pgxpool.Pool, 
	userHandle string, limit, offset int) ([]models.Project, error) {
	
	// Use UNION ALL for better query performance
	// See docs/PERFORMANCE_OPTIMIZATION.md for details
	query := `
		SELECT * FROM projects WHERE owner = $1
		UNION ALL
		SELECT p.* FROM projects p
		INNER JOIN projects_shared_with ps ON p.project_id = ps.project_id
		WHERE ps.user_handle = $1
	`
	
	// Implementation...
}
```

Don't over-comment obvious code:

```go
// Bad - obvious
// i is set to 0
i := 0

// Good - explains why
// Start from second element (first is header)
i := 1
```

## Git Workflow

### Branching Strategy

```bash
# Create feature branch from main
git checkout main
git pull origin main
git checkout -b feature/your-feature-name

# Create fix branch
git checkout -b fix/issue-description

# Create docs branch
git checkout -b docs/update-contributing-guide
```

Branch naming:
- `feature/*` - New features
- `fix/*` - Bug fixes
- `docs/*` - Documentation updates
- `test/*` - Test improvements
- `refactor/*` - Code refactoring

### Commit Messages

Write clear, descriptive commit messages:

```bash
# Good commit messages
git commit -m "Add project sharing functionality"
git commit -m "Fix dimension validation for embeddings"
git commit -m "Update contributing guide with git workflow"

# Multi-line for complex changes
git commit -m "Refactor similarity search query

- Use UNION ALL for better performance
- Add dimension filtering to subqueries
- Update tests to verify performance improvement

Closes #123"
```

**Commit message format:**
- First line: Brief summary (50 chars or less)
- Blank line
- Detailed explanation if needed
- Reference issues: `Closes #123` or `Fixes #456`

### Pull Request Process

1. **Create feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make changes and commit**
   ```bash
   git add .
   git commit -m "Add feature description"
   ```

3. **Keep branch updated**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

4. **Push to your fork**
   ```bash
   git push origin feature/my-feature
   ```

5. **Create Pull Request**
   - Go to GitHub repository
   - Click "New Pull Request"
   - Select your branch
   - Fill in PR template

6. **PR Review Process**
   - Automated tests run
   - Code review by maintainers
   - Address feedback
   - Merge when approved

### PR Title Format

```
[Type] Brief description

Examples:
[Feature] Add project ownership transfer
[Fix] Correct dimension validation in embeddings
[Docs] Update API documentation for similarity search
[Test] Add integration tests for sharing workflow
[Refactor] Simplify authentication middleware
```

### PR Description Template

```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- Change 1
- Change 2
- Change 3

## Testing
How was this tested?
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing performed

## Related Issues
Closes #123
Relates to #456

## Checklist
- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes (or clearly documented)
```

## Testing Requirements

All contributions must include tests:

### For New Features

```go
// Add tests in same package
// File: handlers/projects.go
func CreateProject(...) { ... }

// File: handlers/projects_test.go
func TestCreateProject(t *testing.T) {
	// Test happy path
	// Test error cases
	// Test edge cases
}
```

### For Bug Fixes

```go
// Add regression test
func TestBugFix_Issue123(t *testing.T) {
	// Reproduce the bug
	// Verify it's fixed
}
```

### Test Coverage

Aim for reasonable coverage:

```bash
# Check coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Target coverage: **70%+** for new code

### Running Tests

```bash
# Before submitting PR
go test -v ./...

# With race detection
go test -race ./...

# Specific package
go test -v ./internal/handlers
```

See [Testing Guide](../testing/) for detailed information.

## Documentation Updates

### When to Update Docs

Update documentation for:

- **New features**: Add usage examples
- **API changes**: Update endpoint documentation
- **Breaking changes**: Clearly document migration path
- **Configuration**: New environment variables or options

### Documentation Structure

```
docs/content/
├── getting-started/     # Installation, quickstart
├── concepts/            # Core concepts
├── guides/              # How-to guides
├── api/                 # API reference
└── development/         # Development docs (this section)
```

### Adding Documentation

```bash
# Create new doc file
cd docs/content/guides
cat > new-guide.md <<EOF
---
title: "Your Guide Title"
weight: 5
---

# Your Guide

Content here...
EOF
```

### Hugo Front Matter

All docs need front matter:

```yaml
---
title: "Page Title"
weight: 1          # Determines order in menu
draft: false       # Set true for work in progress
---
```

### Code Examples in Docs

Include working examples:

````markdown
```bash
# Example command
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer admin-key" \
  -d '{"user_handle":"alice"}'
```

```go
// Example Go code
user := models.User{
	Handle: "alice",
	Name:   "Alice Smith",
}
err := CreateUser(ctx, pool, user)
```
````

## Issue Reporting

### Before Creating an Issue

1. Search existing issues
2. Check documentation
3. Try latest version
4. Prepare reproduction steps

### Bug Report Template

```markdown
## Bug Description
Clear description of the bug

## Steps to Reproduce
1. Start application with these settings...
2. Send this request...
3. Observe error...

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- dhamps-vdb version: v0.1.0
- Go version: 1.21.5
- PostgreSQL version: 16.1
- OS: Ubuntu 22.04

## Logs
```
Relevant error logs here
```

## Additional Context
Any other relevant information
```

### Feature Request Template

```markdown
## Feature Description
Clear description of proposed feature

## Use Case
Why is this needed? Who benefits?

## Proposed Solution
How should it work?

## Alternatives Considered
Other approaches you've thought about

## Additional Context
Examples, mockups, related features
```

## Code Review Guidelines

### For Reviewers

- **Be constructive**: Suggest improvements, don't just criticize
- **Ask questions**: "Could we handle this case?" vs "This is wrong"
- **Explain reasoning**: Help author understand why
- **Approve when ready**: Don't be too pedantic on style

### For Authors

- **Respond to feedback**: Address all comments
- **Ask for clarification**: If feedback unclear
- **Don't take it personally**: Focus on improving code
- **Update PR**: Push changes after addressing feedback

### What Reviewers Check

- [ ] Code follows style guidelines
- [ ] Tests are comprehensive
- [ ] Documentation is updated
- [ ] No obvious bugs or security issues
- [ ] Error handling is appropriate
- [ ] Performance considerations addressed
- [ ] Breaking changes documented

## Release Process

Maintainers handle releases, but contributors should:

1. **Document breaking changes** in PR description
2. **Update CHANGELOG** if applicable
3. **Tag related issues** with milestone

## Development Best Practices

### 1. Start Small

- Small PRs are easier to review
- Focus on one feature/fix per PR
- Break large changes into multiple PRs

### 2. Write Tests First

```go
// Write test first (TDD approach)
func TestNewFeature(t *testing.T) {
	// This will fail initially
	result := NewFeature()
	assert.Equal(t, expected, result)
}

// Then implement
func NewFeature() Result {
	// Implementation
}
```

### 3. Use Type Safety

```go
// Generate type-safe queries with sqlc
// Edit: internal/database/queries/queries.sql
-- name: GetUserByHandle :one
SELECT * FROM users WHERE user_handle = $1;

// Then regenerate
sqlc generate --no-remote

// Use generated code
user, err := db.GetUserByHandle(ctx, handle)
```

### 4. Handle Errors Properly

```go
// Don't ignore errors
result, err := SomeFunction()
if err != nil {
	return fmt.Errorf("operation failed: %w", err)
}

// Provide context
if err := ValidateInput(input); err != nil {
	return huma.Error400BadRequest("invalid input", err)
}
```

### 5. Keep It Simple

- Prefer clarity over cleverness
- Use standard library when possible
- Don't over-engineer solutions

## Getting Help

### Resources

- **Documentation**: Browse `/docs` folder
- **API Docs**: Run locally and visit `/docs` endpoint
- **Code Examples**: Check `testdata/` for examples
- **Test Files**: See how features are tested

### Communication

- **GitHub Issues**: For bugs and feature requests
- **Pull Requests**: For code discussions
- **Code Comments**: For implementation questions

### Asking Questions

Good questions include:

- What you're trying to accomplish
- What you've already tried
- Relevant code snippets
- Error messages (full stack trace)

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](../../../LICENSE).

## Recognition

Contributors are acknowledged in:
- Git commit history
- GitHub contributors page
- Release notes (for significant contributions)

Thank you for contributing to dhamps-vdb! 🎉
