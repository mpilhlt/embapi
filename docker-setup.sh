#!/bin/bash
#
# Quick setup script for embapi Docker deployment
#
# This script helps you quickly set up environment variables for Docker deployment.
#

set -e

echo "====================================="
echo "embapi Docker Quick Setup"
echo "====================================="
echo ""

# Check if .env already exists
if [ -f .env ]; then
    echo "Warning: .env file already exists."
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Setup cancelled. Exiting."
        exit 0
    fi
fi

# Copy template
cp .env.docker.template .env
echo "✓ Created .env file from template"

# Generate secure keys
echo ""
echo "Generating secure keys..."

# Check if openssl is available
if command -v openssl &> /dev/null; then
    ADMIN_KEY=$(openssl rand -base64 32)
    ENCRYPTION_KEY=$(openssl rand -hex 32)
    
    # Update .env file with generated keys
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/SERVICE_ADMINKEY=.*/SERVICE_ADMINKEY=$ADMIN_KEY/" .env
        sed -i '' "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
        sed -i '' "s/SERVICE_DBPASSWORD=.*/SERVICE_DBPASSWORD=$(openssl rand -base64 16)/" .env
    else
        # Linux
        sed -i "s/SERVICE_ADMINKEY=.*/SERVICE_ADMINKEY=$ADMIN_KEY/" .env
        sed -i "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
        sed -i "s/SERVICE_DBPASSWORD=.*/SERVICE_DBPASSWORD=$(openssl rand -base64 16)/" .env
    fi
    
    echo "✓ Generated and set secure keys in .env"
else
    echo "⚠ openssl not found. Please manually set the following in .env:"
    echo "  - SERVICE_ADMINKEY"
    echo "  - ENCRYPTION_KEY"
    echo "  - SERVICE_DBPASSWORD"
fi

echo ""
echo "====================================="
echo "Setup complete!"
echo "====================================="
echo ""
echo "Next steps:"
echo "1. Review and adjust settings in .env file if needed"
echo "2. Start the services: docker-compose up -d"
echo "3. Check logs: docker-compose logs -f"
echo "4. Access API documentation: http://localhost:8880/docs"
echo ""
echo "IMPORTANT: Save your admin key securely!"
echo "Admin Key: $ADMIN_KEY"
echo ""
echo "See DOCKER.md for detailed documentation."
echo ""
