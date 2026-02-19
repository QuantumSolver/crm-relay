#!/bin/bash

# CRM Relay Server Startup Script
# This script generates secure keys and starts the server

set -e

echo "🚀 restarting and pulling latest CRM Relay Server Setup..."

docker compose -f docker-compose.server.yml pull

docker compose -f docker-compose.server.yml up -d

docker compose -f docker-compose.server.yml logs -f