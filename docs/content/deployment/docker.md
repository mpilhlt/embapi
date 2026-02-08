---
title: "Docker Deployment"
weight: 1
---

# Docker Deployment

This guide covers production-focused Docker deployment for embapi.

## Overview

For detailed Docker setup instructions, see [Getting Started with Docker](../../getting-started/docker/). This page focuses on production deployment considerations.

## Production Deployment

### Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+ (or docker-compose 1.29+)
- PostgreSQL 11+ with pgvector extension (included in compose setup)

### Quick Production Setup

```bash
# Clone repository
git clone https://github.com/mpilhlt/embapi.git
cd embapi

# Generate secure keys
./docker-setup.sh

# Review and customize .env
nano .env

# Deploy
docker-compose up -d
```

## Production Considerations

### Use Reverse Proxy

Always run behind a reverse proxy (nginx, Traefik, Caddy) for:

- **HTTPS/TLS termination**
- **Request filtering**
- **Rate limiting**
- **Load balancing**

Example nginx configuration:

```nginx
upstream embapi {
    server localhost:8880;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://embapi;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Container Resource Limits

Set resource limits in `docker-compose.yml`:

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
  
  postgres:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

### Use Specific Image Tags

Avoid `latest` in production:

```yaml
services:
  postgres:
    image: pgvector/pgvector:0.7.4-pg16  # Specific version
  
  embapi:
    image: embapi:v0.1.0  # Tag your builds
```

### Health Monitoring

The image includes health checks. Monitor with:

```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' embapi

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' embapi

# Integrate with monitoring (Prometheus, etc.)
```

### Logging

Configure logging drivers in `docker-compose.yml`:

```yaml
services:
  embapi:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Or use centralized logging:

```yaml
services:
  embapi:
    logging:
      driver: "syslog"
      options:
        syslog-address: "tcp://logserver:514"
```

## External Database Deployment

For production, consider using a managed PostgreSQL service or separate database server.

### Requirements

- PostgreSQL 11+ with pgvector extension
- Network connectivity from container to database
- Database user with appropriate privileges

See [Database Setup](../database/) for detailed instructions.

### Configuration

Update `.env`:

```bash
SERVICE_DBHOST=db.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=embapi_user
SERVICE_DBPASSWORD=secure_password
SERVICE_DBNAME=embapi
```

Modify `docker-compose.yml` to remove postgres service:

```yaml
services:
  embapi:
    build: .
    ports:
      - "8880:8880"
    env_file: .env
    restart: unless-stopped
```

## Scaling and High Availability

### Horizontal Scaling

Run multiple instances behind a load balancer:

```yaml
services:
  embapi:
    build: .
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
```

**Note:** All instances must connect to the same database.

### Database Replication

For high availability:

- Use PostgreSQL replication (streaming or logical)
- Consider read replicas for read-heavy workloads
- Point write operations to primary, reads to replicas

## Backup Strategy

### Database Backups

```bash
# Automated daily backups
0 2 * * * docker-compose exec -T postgres pg_dump -U postgres embapi | gzip > /backups/embapi-$(date +\%Y\%m\%d).sql.gz

# Keep last 30 days
find /backups -name "embapi-*.sql.gz" -mtime +30 -delete
```

### Volume Backups

```bash
# Backup Docker volume
docker run --rm -v embapi_postgres_data:/data -v /backups:/backup alpine tar czf /backup/postgres-data-$(date +\%Y\%m\%d).tar.gz /data
```

### Environment Configuration Backups

```bash
# Backup .env (securely!)
gpg --encrypt --recipient admin@example.com .env > .env.gpg
```

## Troubleshooting

### Container Performance Issues

```bash
# Check resource usage
docker stats

# Check container logs
docker-compose logs -f --tail=100 embapi

# Check slow queries (if database is slow)
docker-compose exec postgres psql -U postgres -d embapi -c "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
```

### Network Connectivity Issues

```bash
# Test database connection from container
docker-compose exec embapi nc -zv postgres 5432

# Check DNS resolution
docker-compose exec embapi nslookup postgres

# Test API from inside container
docker-compose exec embapi wget -O- http://localhost:8880/docs
```

### Update and Rollback

```bash
# Update to new version
docker-compose pull
docker-compose up -d

# Rollback if needed
docker-compose down
docker-compose up -d --force-recreate
```

## Security Best Practices

- See [Security Guide](../security/) for comprehensive security recommendations
- Use [Environment Variables Guide](../environment-variables/) for proper configuration
- Never expose database port publicly
- Use strong, randomly generated keys (see `./docker-setup.sh`)
- Keep Docker and images updated
- Run containers as non-root (already configured)

## Further Reading

- [Docker Official Documentation](https://docs.docker.com/)
- [PostgreSQL Docker Hub](https://hub.docker.com/_/postgres)
- [pgvector Extension](https://github.com/pgvector/pgvector)
- [Database Setup Guide](../database/)
- [Environment Variables Reference](../environment-variables/)
- [Security Best Practices](../security/)
