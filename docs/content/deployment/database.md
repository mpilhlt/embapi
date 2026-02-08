---
title: "Database Setup"
weight: 2
---

# Database Setup

dhamps-vdb requires PostgreSQL 11 or later with the pgvector extension.

## Requirements

- **PostgreSQL**: Version 11 or higher
- **pgvector**: Extension for vector similarity search
- **Storage**: Depends on your embeddings volume (estimate: 4 bytes × dimensions × embeddings count)

## Installing PostgreSQL with pgvector

### Using Docker (Recommended)

The easiest way to get PostgreSQL with pgvector:

```bash
docker run -d \
  --name postgres-pgvector \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=secure_password \
  -v postgres_data:/var/lib/postgresql/data \
  pgvector/pgvector:0.7.4-pg16
```

### On Ubuntu/Debian

```bash
# Add PostgreSQL APT repository
sudo apt install curl ca-certificates
sudo install -d /usr/share/postgresql-common/pgdg
sudo curl -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
  https://www.postgresql.org/media/keys/ACCC4CF8.asc
echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] \
  https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" | \
  sudo tee /etc/apt/sources.list.d/pgdg.list

# Install PostgreSQL
sudo apt update
sudo apt install postgresql-16 postgresql-contrib-16

# Install pgvector
sudo apt install postgresql-16-pgvector
```

### On macOS

```bash
# Using Homebrew
brew install postgresql@16
brew install pgvector

# Start PostgreSQL
brew services start postgresql@16
```

### On RHEL/CentOS/Fedora

```bash
# Install PostgreSQL
sudo dnf install postgresql16-server postgresql16-contrib

# Install pgvector (build from source)
sudo dnf install postgresql16-devel git gcc make
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install PG_CONFIG=/usr/pgsql-16/bin/pg_config
```

## Database Configuration

### Step 1: Create Database

Connect to PostgreSQL as superuser:

```bash
# Local connection
sudo -u postgres psql

# Remote connection
psql -h db.example.com -U postgres
```

Create the database:

```sql
-- Create database
CREATE DATABASE dhamps_vdb;
```

### Step 2: Create User

Create a dedicated user for dhamps-vdb:

```sql
-- Create user with password
CREATE USER dhamps_user WITH PASSWORD 'secure_password_here';
```

**Best practices:**
- Use strong, randomly generated password (e.g., `openssl rand -base64 32`)
- Store password securely (password manager, secrets management)
- Rotate passwords regularly

### Step 3: Grant Privileges

Grant necessary permissions:

```sql
-- Grant database privileges
GRANT ALL PRIVILEGES ON DATABASE dhamps_vdb TO dhamps_user;

-- Connect to the database
\c dhamps_vdb

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO dhamps_user;

-- For PostgreSQL 15+, also grant:
GRANT CREATE ON DATABASE dhamps_vdb TO dhamps_user;
```

### Step 4: Enable pgvector Extension

Still connected to the `dhamps_vdb` database:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Verify extension is installed
\dx
```

Expected output should include:

```
List of installed extensions
  Name   | Version |   Schema   |         Description
---------+---------+------------+------------------------------
 vector  | 0.7.4   | public     | vector data type and ivfflat and hnsw access methods
```

### Step 5: Verify Setup

Test the setup:

```sql
-- Test vector creation
SELECT '[1,2,3]'::vector;

-- Should return: [1,2,3]

-- Test vector distance
SELECT '[1,2,3]'::vector <=> '[4,5,6]'::vector AS distance;

-- Should return a float (distance value)
```

Exit psql:

```sql
\q
```

## Connection String Format

dhamps-vdb connects using these environment variables:

```bash
SERVICE_DBHOST=localhost      # Database hostname
SERVICE_DBPORT=5432          # Database port
SERVICE_DBUSER=dhamps_user   # Database username
SERVICE_DBPASSWORD=password  # Database password
SERVICE_DBNAME=dhamps_vdb    # Database name
```

The connection string format used internally:

```
postgresql://username:password@host:port/database?sslmode=disable
```

## Production Configuration

### PostgreSQL Tuning

Edit `postgresql.conf` for better vector search performance:

```ini
# Memory settings
shared_buffers = 4GB                    # 25% of RAM
effective_cache_size = 12GB             # 75% of RAM
maintenance_work_mem = 1GB
work_mem = 50MB

# Parallel query settings
max_parallel_workers_per_gather = 4
max_parallel_workers = 8

# Connection settings
max_connections = 100

# For pgvector specifically
shared_preload_libraries = 'vector'     # Add if not present

# Write-ahead log
wal_level = replica                     # For replication
max_wal_senders = 3                     # For replicas
```

Restart PostgreSQL after changes:

```bash
# On systemd systems
sudo systemctl restart postgresql

# On Docker
docker restart postgres-container
```

### Connection Pooling

For high-load scenarios, use connection pooling with PgBouncer:

```bash
# Install PgBouncer
sudo apt install pgbouncer

# Configure /etc/pgbouncer/pgbouncer.ini
[databases]
dhamps_vdb = host=localhost port=5432 dbname=dhamps_vdb

[pgbouncer]
listen_addr = 127.0.0.1
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

Connect dhamps-vdb to PgBouncer instead of PostgreSQL directly.

### SSL/TLS Encryption

Enable SSL in `postgresql.conf`:

```ini
ssl = on
ssl_cert_file = '/etc/postgresql/server.crt'
ssl_key_file = '/etc/postgresql/server.key'
ssl_ca_file = '/etc/postgresql/ca.crt'
```

Update connection string:

```bash
# Update dhamps-vdb configuration
SERVICE_DBHOST=db.example.com
# Change connection to use SSL (requires code modification or PostgreSQL parameter)
```

### Network Security

Restrict access in `pg_hba.conf`:

```
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# Allow dhamps_user from application server only
host    dhamps_vdb      dhamps_user     10.0.1.0/24            md5

# Allow localhost connections
local   all             all                                    peer
host    all             all             127.0.0.1/32           md5
host    all             all             ::1/128                md5
```

Reload PostgreSQL:

```bash
sudo systemctl reload postgresql
```

## Backup and Recovery

### Automated Backups

Daily backup script:

```bash
#!/bin/bash
# /usr/local/bin/backup-dhamps-vdb.sh

BACKUP_DIR="/backups/dhamps-vdb"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="dhamps_vdb"

# Create backup
pg_dump -U dhamps_user -h localhost $DB_NAME | gzip > "$BACKUP_DIR/dhamps-vdb-$DATE.sql.gz"

# Keep last 30 days
find $BACKUP_DIR -name "dhamps-vdb-*.sql.gz" -mtime +30 -delete

# Verify backup
if [ $? -eq 0 ]; then
    echo "Backup successful: $DATE"
else
    echo "Backup failed: $DATE" >&2
    exit 1
fi
```

Set up cron job:

```bash
# Run daily at 2 AM
0 2 * * * /usr/local/bin/backup-dhamps-vdb.sh
```

### Restore from Backup

```bash
# Decompress and restore
gunzip -c /backups/dhamps-vdb/dhamps-vdb-20240208.sql.gz | \
  psql -U dhamps_user -h localhost dhamps_vdb
```

### Point-in-Time Recovery (PITR)

Enable WAL archiving in `postgresql.conf`:

```ini
wal_level = replica
archive_mode = on
archive_command = 'test ! -f /archive/%f && cp %p /archive/%f'
```

## Monitoring

### Check Database Size

```sql
-- Database size
SELECT pg_size_pretty(pg_database_size('dhamps_vdb'));

-- Table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Check Connection Status

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'dhamps_vdb';

-- Connection details
SELECT 
    pid,
    usename,
    application_name,
    client_addr,
    state,
    query
FROM pg_stat_activity 
WHERE datname = 'dhamps_vdb';
```

### Monitor Performance

```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- View slow queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_stat_statements%'
ORDER BY mean_time DESC
LIMIT 10;
```

## Troubleshooting

### Cannot Connect to Database

```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-16-main.log

# Test connection
psql -h localhost -U dhamps_user -d dhamps_vdb
```

### pgvector Extension Not Found

```sql
-- Check available extensions
SELECT * FROM pg_available_extensions WHERE name = 'vector';

-- If not listed, install pgvector package
-- See installation section above
```

### Permission Denied

```sql
-- Reconnect as superuser
\c dhamps_vdb postgres

-- Re-grant privileges
GRANT ALL PRIVILEGES ON DATABASE dhamps_vdb TO dhamps_user;
GRANT ALL ON SCHEMA public TO dhamps_user;
GRANT ALL ON ALL TABLES IN SCHEMA public TO dhamps_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO dhamps_user;

-- For future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO dhamps_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO dhamps_user;
```

### Out of Disk Space

```bash
# Check disk usage
df -h

# Check PostgreSQL data directory
du -sh /var/lib/postgresql/16/main/

# Clean up old WAL files (if safe)
# Check archive status first
```

### Slow Queries

```sql
-- Check missing indexes
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE schemaname = 'public'
ORDER BY n_distinct DESC;

-- Analyze tables
ANALYZE;

-- Vacuum tables
VACUUM ANALYZE;
```

## Migration from Other Databases

dhamps-vdb is designed specifically for PostgreSQL with pgvector. Migration from other databases requires:

1. Export data from source database
2. Set up PostgreSQL with pgvector
3. Transform data to match dhamps-vdb schema
4. Import using dhamps-vdb API or direct SQL

## Further Reading

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [pgvector GitHub](https://github.com/pgvector/pgvector)
- [PostgreSQL Tuning Guide](https://wiki.postgresql.org/wiki/Tuning_Your_PostgreSQL_Server)
- [Environment Variables](../environment-variables/)
- [Docker Deployment](../docker/)
- [Security Guide](../security/)
