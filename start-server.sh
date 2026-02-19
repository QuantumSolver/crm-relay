#!/bin/bash

# CRM Relay Server Startup Script
# This script generates secure keys and starts the server

set -e

echo "🚀 Starting CRM Relay Server Setup..."

# Generate secure keys
echo "🔑 Generating secure keys..."

API_KEY=$(openssl rand -hex 32)
REDIS_PASSWORD=$(openssl rand -base64 32)

# Create .env file
cat > .env << EOF
# Generated on $(date)
# DO NOT commit this file to version control

# Required: API Key for authentication
API_KEY=${API_KEY}

# Redis Configuration
REDIS_URL=redis:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0

# Stream Configuration
STREAM_NAME=webhook-stream
CONSUMER_GROUP=relay-group
CONSUMER_NAME=relay-client
DEAD_LETTER_QUEUE=webhook-dlq
MESSAGE_TTL=86400

# Webhook Configuration
LOCAL_WEBHOOK_URL=http://nginx:3000/webhook

# Retry Configuration
MAX_RETRIES=3
RETRY_DELAY=1000
RETRY_MULTIPLIER=2.0

# Health Check Configuration
HEALTH_CHECK_INTERVAL=30
EOF

echo "✅ Environment file created: .env"
echo ""
echo "📝 Generated Keys:"
echo "   API_KEY: ${API_KEY}"
echo "   REDIS_PASSWORD: ${REDIS_PASSWORD}"
echo ""
echo "⚠️  Save these keys securely! They are stored in .env"
echo ""

# Create dockernet network if it doesn't exist
echo "🌐 Setting up dockernet network..."
if ! docker network inspect dockernet >/dev/null 2>&1; then
    docker network create dockernet
    echo "✅ Created dockernet network"
else
    echo "✅ dockernet network already exists"
fi

# Start the server
echo ""
echo "🐳 Starting server services..."
docker compose -f docker-compose.server.yml up -d

echo ""
echo "✅ Server started successfully!"
echo ""
echo "📊 To view logs:"
echo "   docker compose -f docker-compose.server.yml logs -f"
echo ""
echo "🛑 To stop the server:"
echo "   docker compose -f docker-compose.server.yml down"
echo ""
echo "🔗 Server is accessible on dockernet network at:"
echo "   http://crm-relay-server:8080"
echo ""
