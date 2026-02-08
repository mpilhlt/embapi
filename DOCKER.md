# Docker Deployment Guide

This guide explains how to deploy dhamps-vdb using Docker containers.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Deployment Options](#deployment-options)
- [Building the Image](#building-the-image)
- [Running with Docker Compose](#running-with-docker-compose)
- [Running Standalone Container](#running-standalone-container)
- [Connecting to External Database](#connecting-to-external-database)
- [Data Persistence](#data-persistence)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

## Quick Start

The fastest way to get started with dhamps-vdb using Docker:

### Automated Setup (Recommended)

```bash
# Run the quick setup script (generates secure keys automatically)
./docker-setup.sh

# Start the services
docker-compose up -d

# Check logs
docker-compose logs -f dhamps-vdb

# Access the API
curl http://localhost:8880/docs
```

### Manual Setup

```bash
# 1. Copy the environment template
cp .env.docker.template .env

# 2. Edit .env and set required values:
#    - SERVICE_ADMINKEY (admin API key)
#    - ENCRYPTION_KEY (at least 32 characters)
#    - SERVICE_DBPASSWORD (database password)

# 3. Start the services
docker-compose up -d

# 4. Check logs
docker-compose logs -f dhamps-vdb

# 5. Access the API
curl http://localhost:8880/docs
```

## Configuration

All configuration is done via environment variables. You can set them:

1. **In a `.env` file** (recommended) - see `.env.docker.template`
2. **In docker-compose.yml** - under the `environment` section
3. **Via command line** - using `-e` flag with `docker run`

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `SERVICE_ADMINKEY` | Admin API key for administrative operations | `Ch4ngeM3SecureKey!` |
| `ENCRYPTION_KEY` | Encryption key for API keys (min 32 chars) | `openssl rand -hex 32` |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVICE_DEBUG` | Enable debug logging | `false` |
| `SERVICE_HOST` | Hostname to bind to | `0.0.0.0` |
| `SERVICE_PORT` | Internal service port | `8880` |
| `SERVICE_DBHOST` | Database hostname | `postgres` |
| `SERVICE_DBPORT` | Database port | `5432` |
| `SERVICE_DBUSER` | Database username | `postgres` |
| `SERVICE_DBPASSWORD` | Database password | `postgres` |
| `SERVICE_DBNAME` | Database name | `dhamps_vdb` |
| `API_PORT` | External port to expose API | `8880` |
| `POSTGRES_PORT` | External port to expose PostgreSQL | `5432` |

## Deployment Options

### Option 1: Docker Compose with Included PostgreSQL (Recommended)

This is the easiest option for getting started. It includes both dhamps-vdb and a PostgreSQL database with pgvector support.

```bash
docker-compose up -d
```

This will:
- Start a PostgreSQL 16 container with pgvector extension
- Build and start the dhamps-vdb container
- Set up networking between the containers
- Persist database data in a Docker volume

### Option 2: Standalone Container with External Database

If you have an existing PostgreSQL database with pgvector support, you can run just the dhamps-vdb container:

```bash
# Build the image
docker build -t dhamps-vdb:latest .

# Run the container
docker run -d \
  --name dhamps-vdb \
  -p 8880:8880 \
  -e SERVICE_DBHOST=your-db-host \
  -e SERVICE_DBPORT=5432 \
  -e SERVICE_DBUSER=your-db-user \
  -e SERVICE_DBPASSWORD=your-db-password \
  -e SERVICE_DBNAME=your-db-name \
  -e SERVICE_ADMINKEY=your-admin-key \
  -e ENCRYPTION_KEY=your-encryption-key \
  dhamps-vdb:latest
```

### Option 3: Docker Compose with External Database

Modify `docker-compose.yml` to comment out or remove the `postgres` service, then set `SERVICE_DBHOST` to your external database hostname in the `.env` file:

```yaml
# Comment out the postgres service in docker-compose.yml
# services:
#   postgres:
#     ...

services:
  dhamps-vdb:
    # ... rest of the configuration
    environment:
      SERVICE_DBHOST: external-db.example.com  # Your external database
      # ... other environment variables
```

## Building the Image

To build the Docker image manually:

```bash
# Build with default tag
docker build -t dhamps-vdb:latest .

# Build with specific tag
docker build -t dhamps-vdb:v0.1.0 .

# Build with no cache (clean build)
docker build --no-cache -t dhamps-vdb:latest .
```

The Dockerfile uses a multi-stage build:
1. **Builder stage**: Compiles the Go application and generates sqlc code
2. **Runtime stage**: Creates a minimal Alpine-based image with just the binary

## Running with Docker Compose

### Starting Services

```bash
# Start in background (detached mode)
docker-compose up -d

# Start with logs visible
docker-compose up

# Rebuild and start (after code changes)
docker-compose up -d --build
```

### Viewing Logs

```bash
# View all logs
docker-compose logs

# Follow logs in real-time
docker-compose logs -f

# View logs for specific service
docker-compose logs -f dhamps-vdb
docker-compose logs -f postgres
```

### Stopping Services

```bash
# Stop containers (keeps volumes)
docker-compose stop

# Stop and remove containers (keeps volumes)
docker-compose down

# Stop, remove containers, and remove volumes (deletes data!)
docker-compose down -v
```

### Restarting Services

```bash
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart dhamps-vdb
```

## Running Standalone Container

### Basic Run

```bash
docker run -d \
  --name dhamps-vdb \
  -p 8880:8880 \
  --env-file .env \
  dhamps-vdb:latest
```

### Run with Individual Environment Variables

```bash
docker run -d \
  --name dhamps-vdb \
  -p 8880:8880 \
  -e SERVICE_DEBUG=false \
  -e SERVICE_HOST=0.0.0.0 \
  -e SERVICE_PORT=8880 \
  -e SERVICE_DBHOST=postgres-host \
  -e SERVICE_DBPORT=5432 \
  -e SERVICE_DBUSER=dbuser \
  -e SERVICE_DBPASSWORD=dbpass \
  -e SERVICE_DBNAME=dhamps_vdb \
  -e SERVICE_ADMINKEY=admin-key \
  -e ENCRYPTION_KEY=encryption-key-32-chars-min \
  dhamps-vdb:latest
```

### Interactive Run (for debugging)

```bash
docker run -it --rm \
  --name dhamps-vdb \
  -p 8880:8880 \
  --env-file .env \
  dhamps-vdb:latest
```

## Connecting to External Database

To use an external PostgreSQL database with pgvector:

### Prerequisites

1. PostgreSQL 11+ with pgvector extension installed
2. Database and user created with appropriate permissions
3. Network access from the container to the database

### Setup External Database

Connect to your PostgreSQL server and run:

```sql
-- Create database
CREATE DATABASE dhamps_vdb;

-- Create user
CREATE USER dhamps_user WITH PASSWORD 'secure_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE dhamps_vdb TO dhamps_user;

-- Connect to the database
\c dhamps_vdb

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO dhamps_user;

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
```

### Configure dhamps-vdb

In your `.env` file:

```bash
SERVICE_DBHOST=your-database-host.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_user
SERVICE_DBPASSWORD=secure_password
SERVICE_DBNAME=dhamps_vdb
```

For docker-compose with external database, you can either:

1. **Keep the postgres service commented out:**
   ```yaml
   # services:
   #   postgres:
   #     ...
   ```

2. **Or skip docker-compose** and run just the dhamps-vdb container with `docker run`.

## Data Persistence

### Docker Compose Volumes

The docker-compose setup creates a named volume `postgres_data` that persists database data:

```yaml
volumes:
  postgres_data:  # Named volume for database
```

View volumes:
```bash
docker volume ls
```

Inspect volume:
```bash
docker volume inspect dhamps-vdb_postgres_data
```

### Backup Database

```bash
# Backup
docker-compose exec postgres pg_dump -U postgres dhamps_vdb > backup.sql

# Restore
docker-compose exec -T postgres psql -U postgres dhamps_vdb < backup.sql
```

## Security Considerations

### Production Deployment Checklist

- [ ] Use strong, randomly generated passwords and keys
- [ ] Never commit `.env` files to version control
- [ ] Use Docker secrets for sensitive values in production
- [ ] Run containers behind a reverse proxy (nginx, Traefik)
- [ ] Enable HTTPS/TLS for API endpoints
- [ ] Restrict database network access
- [ ] Keep Docker images updated
- [ ] Use specific image tags (not `latest`) in production
- [ ] Implement proper logging and monitoring
- [ ] Set resource limits for containers

### Generating Secure Keys

```bash
# Generate admin key
openssl rand -base64 32

# Generate encryption key
openssl rand -hex 32
```

### Using Docker Secrets (Swarm/Kubernetes)

For Docker Swarm:

```yaml
services:
  dhamps-vdb:
    secrets:
      - admin_key
      - encryption_key
    environment:
      SERVICE_ADMINKEY_FILE: /run/secrets/admin_key
      ENCRYPTION_KEY_FILE: /run/secrets/encryption_key

secrets:
  admin_key:
    external: true
  encryption_key:
    external: true
```

## Troubleshooting

### Container won't start

Check logs:
```bash
docker-compose logs dhamps-vdb
docker logs dhamps-vdb
```

Common issues:
- Missing required environment variables (`SERVICE_ADMINKEY`, `ENCRYPTION_KEY`)
- Database connection failure
- Port already in use

### Database connection errors

```bash
# Check if postgres is healthy
docker-compose ps

# Check database logs
docker-compose logs postgres

# Test database connection
docker-compose exec postgres psql -U postgres -d dhamps_vdb -c "SELECT 1;"
```

### Can't connect to API

```bash
# Check if container is running
docker ps

# Check port mapping
docker port dhamps-vdb

# Test from inside container
docker-compose exec dhamps-vdb wget -O- http://localhost:8880/docs

# Test from host
curl http://localhost:8880/docs
```

### Permission issues

If you see permission errors, check:
- Container runs as non-root user `appuser` (UID 1000)
- Volume permissions match container user

### Reset everything

```bash
# Stop and remove everything (including data!)
docker-compose down -v

# Remove images
docker rmi dhamps-vdb:latest
docker rmi pgvector/pgvector:0.7.4-pg16

# Start fresh
docker-compose up -d --build
```

### Health check failing

```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' dhamps-vdb

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' dhamps-vdb
```

## Advanced Topics

### Custom Build Arguments

```bash
docker build \
  --build-arg GO_VERSION=1.24 \
  -t dhamps-vdb:custom .
```

### Resource Limits

Add to docker-compose.yml:

```yaml
services:
  dhamps-vdb:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Networks

Connect to existing network:

```bash
docker network create app-network
```

```yaml
services:
  dhamps-vdb:
    networks:
      - app-network

networks:
  app-network:
    external: true
```

## Getting Help

- Check the [main README](README.md) for general usage
- Review [API documentation](http://localhost:8880/docs) once running
- Check container logs for errors
- Verify environment variables are set correctly
