---
title: "Deployment"
weight: 5
---

# Deployment Guide

Deploy embapi to production environments.

## Deployment Options

embapi can be deployed in several ways:

- **Docker Compose** - Simplest option, includes PostgreSQL
- **Docker with External Database** - Production-ready setup
- **Standalone Binary** - For custom environments
- **Kubernetes** - For orchestrated deployments

## Production Considerations

When deploying to production:

- Use strong, randomly generated keys
- Enable HTTPS/TLS for all API endpoints
- Configure database backups
- Set up monitoring and logging
- Restrict network access to database
- Use environment variables for sensitive configuration

## Guides

- [Docker Deployment](docker/) - Complete Docker guide
- [Database Setup](database/) - PostgreSQL configuration
- [Environment Variables](environment-variables/) - All configuration options
- [Security](security/) - Security best practices
