---
title: "Docker Deployment"
weight: 2
---

# Docker Deployment

Deploy embapi using Docker containers. This is the recommended approach for most users.

## Quick Start

The fastest way to get embapi running with Docker:

```bash
# Clone the repository
git clone https://github.com/mpilhlt/embapi.git
cd embapi

# Run automated setup (generates secure keys)
./docker-setup.sh

# Start services with docker-compose
docker-compose up -d

# Check logs
docker-compose logs -f embapi

# Access the API
curl http://localhost:8880/docs
```

## What's Included

The Docker Compose setup includes:

- **embapi**: The vector database API service
- **PostgreSQL 16**: Database with pgvector extension
- **Persistent storage**: Named volume for database data

## Configuration Files

### .env File

All configuration is managed through environment variables. Copy the template:

```bash
cp .env.docker.template .env
```

Edit `.env` to set required values:

```bash
# Admin API key for administrative operations
SERVICE_ADMINKEY=your-secure-admin-key-here

# Encryption key for API keys (32+ characters)
ENCRYPTION_KEY=your-secure-encryption-key-min-32-chars

# Database password
SERVICE_DBPASSWORD=secure-database-password

# Optional: Debug mode
SERVICE_DEBUG=false

# Optional: Change ports
API_PORT=8880
POSTGRES_PORT=5432
```

### docker-compose.yml

The compose file defines two services:

```yaml
services:
  postgres:
    image: pgvector/pgvector:0.7.4-pg16
    # PostgreSQL with pgvector support
    
  embapi:
    build: .
    # The API service
    depends_on:
      - postgres
```

## Deployment Options

### Option 1: Docker Compose with Included Database (Recommended)

Use the provided `docker-compose.yml`:

```bash
docker-compose up -d
```

**Advantages:**
- Everything included
- Automatic networking
- Data persistence
- Easy to manage

**Use when:**
- Getting started
- Development/testing
- Small to medium deployments

### Option 2: Standalone Container with External Database

Run only the embapi container:

```bash
# Build the image
docker build -t embapi:latest .

# Run the container
docker run -d \
  --name embapi \
  -p 8880:8880 \
  -e SERVICE_DBHOST=your-db-host \
  -e SERVICE_DBPORT=5432 \
  -e SERVICE_DBUSER=dbuser \
  -e SERVICE_DBPASSWORD=dbpass \
  -e SERVICE_DBNAME=embapi \
  -e SERVICE_ADMINKEY=admin-key \
  -e ENCRYPTION_KEY=encryption-key \
  embapi:latest
```

**Use when:**
- You have an existing PostgreSQL server
- Production deployments
- Need database separation

### Option 3: Docker Compose with External Database

Modify `docker-compose.yml` to remove the postgres service:

```yaml
services:
  embapi:
    build: .
    ports:
      - "${API_PORT:-8880}:8880"
    environment:
      SERVICE_DBHOST: external-db.example.com
      # ... other variables
```

## Building the Image

### Standard Build

```bash
docker build -t embapi:latest .
```

### Custom Tag

```bash
docker build -t embapi:v0.1.0 .
```

### Clean Build (No Cache)

```bash
docker build --no-cache -t embapi:latest .
```

### Multi-Stage Build Details

The Dockerfile uses multi-stage builds for efficiency:

1. **Builder stage**: Compiles Go code with sqlc generation
2. **Runtime stage**: Minimal Alpine image with only the binary

Result: Small, secure image (~20MB vs 800MB+)

## Managing Services

### Start Services

```bash
# Start in background
docker-compose up -d

# Start with logs visible
docker-compose up

# Rebuild and start
docker-compose up -d --build
```

### View Logs

```bash
# All logs
docker-compose logs

# Follow logs in real-time
docker-compose logs -f

# Specific service
docker-compose logs -f embapi
docker-compose logs -f postgres
```

### Stop Services

```bash
# Stop containers (keeps data)
docker-compose stop

# Stop and remove containers (keeps data)
docker-compose down

# Stop and remove everything including data
docker-compose down -v
```

### Restart Services

```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart embapi
```

## Data Persistence

### Docker Volumes

The compose file creates a named volume:

```yaml
volumes:
  postgres_data:
```

This ensures database data persists across container restarts.

### View Volumes

```bash
docker volume ls
```

### Inspect Volume

```bash
docker volume inspect embapi_postgres_data
```

### Backup Database

```bash
# Create backup
docker-compose exec postgres pg_dump -U postgres embapi > backup.sql

# Restore from backup
docker-compose exec -T postgres psql -U postgres embapi < backup.sql
```

## Networking

### Access from Host

The API is accessible at:

```
http://localhost:8880
```

### Access from Other Containers

Use the service name as hostname:

```
http://embapi:8880
```

### Custom Network

To use an existing Docker network:

```yaml
networks:
  default:
    external:
      name: your-network-name
```

## Security

### Required Environment Variables

Two critical environment variables must be set:

1. **SERVICE_ADMINKEY**: Admin API key
2. **ENCRYPTION_KEY**: For encrypting user API keys (32+ chars)

### Generating Secure Keys

```bash
# Admin key
openssl rand -base64 32

# Encryption key
openssl rand -hex 32
```

### Production Checklist

- [ ] Use strong, randomly generated keys
- [ ] Never commit `.env` to version control
- [ ] Run behind reverse proxy (nginx, Traefik)
- [ ] Enable HTTPS/TLS
- [ ] Restrict database network access
- [ ] Set resource limits
- [ ] Enable logging and monitoring
- [ ] Use specific image tags (not `latest`)
- [ ] Regular security updates
- [ ] Backup database regularly

## Verification

### Check Service Status

```bash
# Check if containers are running
docker-compose ps

# Expected: both services "running" or "healthy"
```

### Test API Access

```bash
# Get OpenAPI documentation
curl http://localhost:8880/docs

# Should return HTML page
```

### Test Database Connection

```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U postgres -d embapi

# Check pgvector extension
\dx

# Should show vector extension
```

### Create Test User

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "testuser",
    "full_name": "Test User"
  }'
```

## Troubleshooting

### Container Won't Start

Check logs:
```bash
docker-compose logs embapi
```

Common issues:
- Missing `SERVICE_ADMINKEY` or `ENCRYPTION_KEY`
- Database connection failure
- Port already in use

### Database Connection Errors

```bash
# Check postgres health
docker-compose ps

# Check database logs
docker-compose logs postgres

# Test connection
docker-compose exec postgres psql -U postgres -d embapi -c "SELECT 1;"
```

### Can't Connect to API

```bash
# Check if container is running
docker ps

# Check port mapping
docker port embapi

# Test from inside container
docker-compose exec embapi wget -O- http://localhost:8880/docs

# Test from host
curl http://localhost:8880/docs
```

### Permission Issues

The container runs as non-root user `appuser` (UID 1000). If you have permission errors:

```bash
# Check volume permissions
docker volume inspect embapi_postgres_data
```

### Reset Everything

```bash
# Stop and remove everything
docker-compose down -v

# Remove images
docker rmi embapi:latest
docker rmi pgvector/pgvector:0.7.4-pg16

# Start fresh
docker-compose up -d --build
```

### Build Failures

If Docker build fails with network errors:

```bash
# Try with host network
docker build --network=host -t embapi:latest .
```

## Advanced Configuration

### Resource Limits

Add to `docker-compose.yml`:

```yaml
services:
  embapi:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Health Checks

The Dockerfile includes a health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8880/ || exit 1
```

View health status:

```bash
docker inspect --format='{{.State.Health.Status}}' embapi
```

### Custom Dockerfile Builds

You can customize the build:

```bash
docker build \
  --build-arg GO_VERSION=1.24 \
  -t embapi:custom .
```

## External Database Setup

If using an external PostgreSQL database:

### Prepare Database

```sql
-- Create database
CREATE DATABASE embapi;

-- Create user
CREATE USER embapi_user WITH PASSWORD 'secure_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE embapi TO embapi_user;

-- Connect to database
\c embapi

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO embapi_user;

-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;
```

### Configure embapi

Update `.env`:

```bash
SERVICE_DBHOST=external-db.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=embapi_user
SERVICE_DBPASSWORD=secure_password
SERVICE_DBNAME=embapi
```

Then run only the embapi service or use a standalone container.

## Next Steps

After successful deployment:

1. [Configure the service](../configuration/)
2. [Create your first user](../quick-start/)
3. [Set up a project](../first-project/)
4. Review [security best practices](../../deployment/security/)
