# Build stage
FROM golang:1.24-alpine3.21 AS builder

# Install build dependencies (with retry logic for network issues)
RUN apk update && apk add --no-cache git make || \
    (sleep 5 && apk update && apk add --no-cache git make)

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate sqlc code
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    sqlc generate --no-remote

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dhamps-vdb main.go

# Runtime stage
FROM alpine:3.21

# Update package index and install ca-certificates for HTTPS (with retry logic)
RUN apk update && apk add --no-cache ca-certificates wget || \
    (sleep 5 && apk update && apk add --no-cache ca-certificates wget)

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/dhamps-vdb .

# Copy migrations (needed for database schema management)
COPY --from=builder /build/internal/database/migrations ./internal/database/migrations

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8880

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8880/docs || exit 1

# Run the application
CMD ["./dhamps-vdb"]
