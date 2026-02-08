---
title: "Security Best Practices"
weight: 4
---

# Security Best Practices

Comprehensive guide for securing your dhamps-vdb production deployment.

## Security Overview

dhamps-vdb handles sensitive data including embeddings, metadata, and API credentials. This guide covers essential security measures for production deployments.

## HTTPS/TLS Configuration

### Why HTTPS is Required

- **Encryption in transit**: Protects API keys and data from interception
- **Authentication**: Verifies server identity
- **Compliance**: Required by most security standards

### Using a Reverse Proxy

**Never expose dhamps-vdb directly to the internet.** Always use a reverse proxy with TLS termination.

#### Nginx with Let's Encrypt

Install Certbot:

```bash
sudo apt install nginx certbot python3-certbot-nginx
```

Configure nginx (`/etc/nginx/sites-available/dhamps-vdb`):

```nginx
upstream dhamps_backend {
    server 127.0.0.1:8880;
    keepalive 32;
}

server {
    listen 80;
    server_name api.example.com;
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # SSL certificates (managed by Certbot)
    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;

    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS (HTTP Strict Transport Security)
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Proxy settings
    location / {
        proxy_pass http://dhamps_backend;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Buffer settings
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }

    # Rate limiting (optional, see below)
    limit_req zone=api_limit burst=20 nodelay;
    limit_req_status 429;
}
```

Enable site and get certificate:

```bash
sudo ln -s /etc/nginx/sites-available/dhamps-vdb /etc/nginx/sites-enabled/
sudo certbot --nginx -d api.example.com
sudo systemctl reload nginx
```

Auto-renewal:

```bash
# Certbot creates a systemd timer automatically
sudo systemctl status certbot.timer

# Test renewal
sudo certbot renew --dry-run
```

#### Traefik (Docker)

`docker-compose.yml`:

```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt

  dhamps-vdb:
    build: .
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dhamps-vdb.rule=Host(`api.example.com`)"
      - "traefik.http.routers.dhamps-vdb.entrypoints=websecure"
      - "traefik.http.routers.dhamps-vdb.tls.certresolver=letsencrypt"
      # Redirect HTTP to HTTPS
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https"
      - "traefik.http.routers.dhamps-vdb-http.rule=Host(`api.example.com`)"
      - "traefik.http.routers.dhamps-vdb-http.entrypoints=web"
      - "traefik.http.routers.dhamps-vdb-http.middlewares=redirect-to-https"
```

#### Caddy (Automatic HTTPS)

`Caddyfile`:

```
api.example.com {
    reverse_proxy localhost:8880
    
    # Automatic HTTPS via Let's Encrypt
    tls admin@example.com
    
    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
    }
}
```

## API Key Management

### Admin Key Security

The `SERVICE_ADMINKEY` has full control over the system.

**Best practices:**

1. **Generate securely:**
   ```bash
   openssl rand -base64 32
   ```

2. **Store securely:**
   - Password manager (1Password, Bitwarden)
   - Secrets manager (Vault, AWS Secrets Manager)
   - Never in version control
   - Never in logs or error messages

3. **Rotate regularly:**
   - Every 90 days minimum
   - After team member departure
   - After suspected compromise

4. **Audit usage:**
   - Log all admin operations
   - Monitor for unusual activity
   - Review regularly

### User API Keys

User API keys are automatically generated and encrypted before database storage.

**Security features:**

- **One-time display:** Keys shown only at creation
- **Encrypted storage:** AES-256-GCM encryption
- **Non-recoverable:** Lost keys cannot be retrieved

**For users:**

1. **Secure storage:**
   - Use environment variables
   - Never hardcode in applications
   - Never commit to repositories

2. **Use HTTPS:**
   - Always transmit over TLS
   - Validate server certificates

3. **Rotate if compromised:**
   - Delete old user and create new one
   - Update all client applications

### Bearer Token Authentication

All API requests require authentication:

```bash
curl -X GET https://api.example.com/v1/projects/alice \
  -H "Authorization: Bearer your-api-key-here"
```

**Security:**

- Use `Authorization` header (not query parameters)
- Never log full API keys
- Validate on every request
- Use HTTPS to prevent interception

## Encryption Key Security

The `ENCRYPTION_KEY` protects all user API keys in the database.

### Critical Security Points

⚠️ **CRITICAL:** If the encryption key is lost or changed, **all user API keys become permanently unrecoverable.**

**Best practices:**

1. **Generate securely:**
   ```bash
   openssl rand -hex 32
   ```

2. **Backup separately:**
   - Store in multiple secure locations
   - Separate from database backups
   - Document recovery procedure
   - Test recovery process

3. **Never rotate in production:**
   - Cannot be changed without re-encrypting all keys
   - Requires database migration
   - Risk of data loss

4. **Protect at rest:**
   - Encrypt .env files: `gpg --encrypt .env`
   - Use secrets management (Vault, AWS Secrets Manager)
   - Restrict file permissions: `chmod 600 .env`

5. **Protect in transit:**
   - Never send over unencrypted channels
   - Use secure channels for team sharing
   - Avoid email/chat

### Encryption Details

- **Algorithm:** AES-256-GCM
- **Key derivation:** SHA-256 hash of input key
- **Nonce:** Unique per encryption
- **Authentication:** GCM provides authentication
- **Storage format:** Base64-encoded ciphertext

### Disaster Recovery

Document and test recovery procedure:

```markdown
## Encryption Key Recovery Procedure

1. Retrieve backup encryption key from [location]
2. Verify key integrity: [checksum/hash]
3. Update deployment configuration
4. Restart services
5. Verify user authentication works
6. Document incident
```

## Database Security

### Network Security

1. **Restrict access:**
   ```sql
   -- In pg_hba.conf
   # Allow only from application server
   host    dhamps_vdb    dhamps_user    10.0.1.0/24    md5
   ```

2. **Firewall rules:**
   ```bash
   # UFW example
   sudo ufw allow from 10.0.1.0/24 to any port 5432
   sudo ufw deny 5432
   ```

3. **Use VPC/private network:**
   - Keep database on private network
   - No public internet exposure
   - VPN for remote administration

### Database Authentication

1. **Strong passwords:**
   ```bash
   # Generate secure password
   openssl rand -base64 32
   ```

2. **Dedicated user:**
   ```sql
   -- Not superuser
   CREATE USER dhamps_user WITH PASSWORD 'secure_password';
   GRANT ALL PRIVILEGES ON DATABASE dhamps_vdb TO dhamps_user;
   ```

3. **SSL/TLS connections:**
   ```ini
   # postgresql.conf
   ssl = on
   ssl_cert_file = 'server.crt'
   ssl_key_file = 'server.key'
   ssl_ca_file = 'ca.crt'
   ```

### Database Encryption

1. **Encryption at rest:**
   - Use encrypted filesystems (LUKS, dm-crypt)
   - Cloud provider encryption (AWS RDS encryption)
   - Transparent Data Encryption (TDE)

2. **Backup encryption:**
   ```bash
   # Encrypt backup
   pg_dump -U postgres dhamps_vdb | gzip | \
     gpg --encrypt --recipient admin@example.com > backup.sql.gz.gpg
   ```

### Database Auditing

Enable audit logging:

```sql
-- Install pgaudit
CREATE EXTENSION pgaudit;

-- Configure logging
ALTER SYSTEM SET pgaudit.log = 'write, ddl';
ALTER SYSTEM SET pgaudit.log_catalog = off;
ALTER SYSTEM SET pgaudit.log_parameter = on;

-- Reload config
SELECT pg_reload_conf();
```

## Network Security

### Firewall Configuration

```bash
# UFW example
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS (reverse proxy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Application port (if not behind proxy)
# sudo ufw allow from trusted_network to any port 8880

# Database (from app server only)
sudo ufw allow from 10.0.1.0/24 to any port 5432

sudo ufw enable
```

### Network Segmentation

Deploy in isolated networks:

```
Internet
    ↓
[Reverse Proxy] ← (DMZ/Public subnet)
    ↓
[dhamps-vdb]    ← (Application subnet)
    ↓
[PostgreSQL]    ← (Database subnet)
```

### Rate Limiting

Protect against abuse and DoS attacks.

**Nginx:**

```nginx
# Define rate limit zone
http {
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=api_burst:10m rate=100r/s;
}

server {
    location /v1/ {
        limit_req zone=api_limit burst=20 nodelay;
        limit_req_status 429;
        proxy_pass http://dhamps_backend;
    }
}
```

**Application-level** (future enhancement):
- Implement token bucket or leaky bucket
- Per-user rate limits
- Different limits for different endpoints

## Backup Strategies

### Database Backups

1. **Automated daily backups:**
   ```bash
   #!/bin/bash
   # /usr/local/bin/backup-dhamps.sh
   
   DATE=$(date +%Y%m%d_%H%M%S)
   BACKUP_DIR="/backups/dhamps-vdb"
   
   # Create backup
   pg_dump -U dhamps_user dhamps_vdb | gzip > \
     "$BACKUP_DIR/db-$DATE.sql.gz"
   
   # Encrypt backup
   gpg --encrypt --recipient admin@example.com \
     "$BACKUP_DIR/db-$DATE.sql.gz"
   
   # Remove unencrypted backup
   rm "$BACKUP_DIR/db-$DATE.sql.gz"
   
   # Verify backup
   gpg --decrypt "$BACKUP_DIR/db-$DATE.sql.gz.gpg" | gunzip | head -n 5
   
   # Upload to offsite storage
   aws s3 cp "$BACKUP_DIR/db-$DATE.sql.gz.gpg" \
     s3://backups/dhamps-vdb/ --storage-class GLACIER
   
   # Cleanup old local backups (keep 7 days)
   find $BACKUP_DIR -name "db-*.sql.gz.gpg" -mtime +7 -delete
   ```

2. **Cron schedule:**
   ```cron
   0 2 * * * /usr/local/bin/backup-dhamps.sh
   ```

3. **Test restores regularly:**
   ```bash
   # Monthly restore test
   0 3 1 * * /usr/local/bin/test-restore.sh
   ```

### Configuration Backups

```bash
# Backup environment configuration
cp .env .env.backup
gpg --encrypt --recipient admin@example.com .env.backup

# Backup with timestamp
tar czf config-$(date +%Y%m%d).tar.gz .env docker-compose.yml
gpg --encrypt --recipient admin@example.com config-*.tar.gz
```

### Backup Retention Policy

- **Daily backups:** Keep 7 days locally
- **Weekly backups:** Keep 4 weeks offsite
- **Monthly backups:** Keep 12 months in cold storage
- **Yearly backups:** Keep 7 years (compliance dependent)

### Offsite Backups

Always maintain offsite backups:

- **Cloud storage:** AWS S3, Azure Blob Storage, GCP Cloud Storage
- **Different geographic region**
- **Encrypted before upload**
- **Test restoration procedures**

## Monitoring and Alerting

### What to Monitor

1. **Authentication failures:**
   ```bash
   # Parse logs for 401 responses
   grep "401" /var/log/nginx/access.log | wc -l
   ```

2. **Unusual API usage:**
   - Spike in requests
   - Requests to non-existent endpoints
   - Large data transfers

3. **Database health:**
   - Connection count
   - Query performance
   - Disk usage
   - Replication lag (if applicable)

4. **System resources:**
   - CPU usage
   - Memory usage
   - Disk I/O
   - Network throughput

### Log Management

1. **Centralized logging:**
   - ELK Stack (Elasticsearch, Logstash, Kibana)
   - Splunk
   - Graylog
   - CloudWatch Logs (AWS)

2. **Log retention:**
   - Application logs: 30 days minimum
   - Access logs: 90 days minimum
   - Audit logs: 1 year minimum (compliance dependent)

3. **Log security:**
   - Never log API keys in full (mask: `api_key=abc...xyz`)
   - Never log passwords
   - Encrypt archived logs
   - Restrict access to logs

### Alerting

Set up alerts for:

- Failed authentication attempts (>10/minute)
- Database connection failures
- Disk space >80% full
- Service downtime
- Unusual network traffic
- SSL certificate expiration (30 days before)

## Security Checklist

Use this checklist for production deployments:

### Pre-Deployment

- [ ] Strong, random `SERVICE_ADMINKEY` generated
- [ ] Strong, random `ENCRYPTION_KEY` generated (32+ chars)
- [ ] Strong database password set
- [ ] `.env` file encrypted or in secrets manager
- [ ] `.env` not in version control
- [ ] Encryption key backed up separately from database

### Network & Access

- [ ] HTTPS/TLS configured with valid certificate
- [ ] Reverse proxy deployed (nginx/Traefik/Caddy)
- [ ] Firewall configured and enabled
- [ ] Database on private network only
- [ ] Rate limiting configured
- [ ] HSTS header enabled
- [ ] Security headers configured

### Database

- [ ] PostgreSQL 11+ with pgvector installed
- [ ] Dedicated database user (not superuser)
- [ ] Strong database password
- [ ] SSL/TLS for database connections
- [ ] Database firewall rules configured
- [ ] pg_hba.conf restricts access by IP
- [ ] Regular backups configured
- [ ] Backup encryption enabled
- [ ] Restore procedure tested

### Application

- [ ] Service runs as non-root user
- [ ] Debug logging disabled (`SERVICE_DEBUG=false`)
- [ ] Resource limits configured
- [ ] Health checks enabled
- [ ] Logging configured
- [ ] Monitoring configured
- [ ] Alerting configured

### Operations

- [ ] Backup procedure documented
- [ ] Restore procedure documented and tested
- [ ] Disaster recovery plan created
- [ ] Security incident response plan created
- [ ] Key rotation schedule defined
- [ ] Access control documented (who has admin key)
- [ ] Regular security updates scheduled

### Compliance

- [ ] Data retention policy defined
- [ ] Privacy policy reviewed
- [ ] Terms of service reviewed
- [ ] GDPR compliance (if applicable)
- [ ] Data processing agreements in place
- [ ] Audit logging enabled

## Incident Response

### Suspected API Key Compromise

1. **Immediate actions:**
   - Identify compromised user
   - Delete user (invalidates API key)
   - Create new user with new API key
   - Review access logs for unauthorized access

2. **Investigation:**
   - Determine scope of compromise
   - Check for data exfiltration
   - Document incident

3. **Notification:**
   - Notify affected parties (if required)
   - Document lessons learned
   - Update security procedures

### Database Breach

1. **Immediate actions:**
   - Isolate database server
   - Revoke network access
   - Change all database credentials
   - Activate incident response team

2. **Assessment:**
   - Determine data accessed
   - Check encryption effectiveness
   - Review audit logs

3. **Recovery:**
   - Restore from clean backup
   - Apply security patches
   - Update firewall rules
   - Notify affected parties (if required)
   - Document and learn

### Encryption Key Loss

⚠️ **CRITICAL SITUATION:** If encryption key is lost, all user API keys are unrecoverable.

1. **Recovery attempt:**
   - Check all backup locations
   - Review documentation
   - Contact all team members

2. **If unrecoverable:**
   - All users must be deleted and recreated
   - New API keys issued to all users
   - All client applications must be updated
   - Communicate timeline to users

3. **Prevention:**
   - Review backup procedures
   - Add redundant backup locations
   - Document key locations
   - Test recovery process

## Security Updates

### Regular Updates

- **Weekly:** Review security advisories
- **Monthly:** Apply security patches
- **Quarterly:** Security audit and penetration testing
- **Yearly:** Full security review and compliance audit

### Update Procedure

1. **Test in staging environment**
2. **Backup production database and configuration**
3. **Schedule maintenance window**
4. **Apply updates**
5. **Verify functionality**
6. **Monitor for issues**

### Subscribe to Security Advisories

- PostgreSQL security announcements
- pgvector security updates
- Docker security advisories
- Go security advisories
- Operating system security updates

## Further Reading

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [PostgreSQL Security](https://www.postgresql.org/docs/current/security.html)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Environment Variables Reference](../environment-variables/)
- [Database Setup Guide](../database/)
- [Docker Deployment](../docker/)
