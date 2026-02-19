# Docker Compose Setup Guide

This guide explains the two separate Docker Compose files for deploying the CRM Relay Server and Client.

## Overview

The project now includes three Docker Compose files:

1. **`docker-compose.server.yml`** - Server and Redis (public-facing)
2. **`docker-compose.client.yml`** - Client only (private, attached to dockernet)
3. **`docker-compose.yml`** - All-in-one (development only)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Public Network                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  docker-compose.server.yml                            │  │
│  │  ┌──────────────┐      ┌──────────────────────────┐  │  │
│  │  │   Redis      │◄─────┤   Relay Server (8080)    │  │  │
│  │  │   (6379)     │      │   ghcr.io/.../server     │  │  │
│  │  └──────────────┘      └──────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Redis Streams
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Private Network (dockernet)              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  docker-compose.client.yml                            │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │   Relay Client                                 │  │  │
│  │  │   ghcr.io/.../client                           │  │  │
│  │  │   (no exposed ports)                           │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                          │                            │  │
│  │                          ▼                            │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │   Nginx (on dockernet)                         │  │  │
│  │  │   Exposes webhooks to external services        │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Server Compose File (`docker-compose.server.yml`)

### Services

- **Redis**: Redis 7 Alpine with persistence
  - Port: 6379 (exposed)
  - Volume: redis-data
  - Health check enabled

- **Relay Server**: Public-facing webhook receiver
  - Image: `ghcr.io/quantumsolver/crm-relay/relay-server:latest`
  - Port: 8080 (exposed)
  - Depends on Redis (healthy)
  - Environment variables from `.env`

### Environment Variables

Required:
- `API_KEY` - Authentication key (must be set in `.env`)

Optional (with defaults):
- `REDIS_URL` - Redis connection URL (default: `redis:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `STREAM_NAME` - Redis stream name (default: `webhook-stream`)
- `CONSUMER_GROUP` - Consumer group name (default: `relay-group`)
- `CONSUMER_NAME` - Consumer name (default: `relay-client`)
- `DEAD_LETTER_QUEUE` - DLQ name (default: `webhook-dlq`)
- `MESSAGE_TTL` - Message TTL in seconds (default: 86400)
- `LOCAL_WEBHOOK_URL` - Local webhook URL (default: `http://localhost:3000/webhook`)
- `MAX_RETRIES` - Max retry attempts (default: 3)
- `RETRY_DELAY` - Initial retry delay in ms (default: 1000)
- `RETRY_MULTIPLIER` - Retry delay multiplier (default: 2.0)
- `HEALTH_CHECK_INTERVAL` - Health check interval in seconds (default: 30)

## Client Compose File (`docker-compose.client.yml`)

### Services

- **Relay Client**: Private webhook forwarder
  - Image: `ghcr.io/quantumsolver/crm-relay/relay-client:latest`
  - No exposed ports (communicates via dockernet network)
  - Attached to external `dockernet` network
  - Environment variables from `.env`

### Network

- **dockernet**: External network for communication with nginx
  - Must be created before starting client: `docker network create dockernet`
  - Client can reach nginx via `http://nginx:3000/webhook`

### Environment Variables

Same as server, with `LOCAL_WEBHOOK_URL` defaulting to `http://nginx:3000/webhook` for dockernet communication.

## Quick Start

### 1. Setup Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env and set your API_KEY
nano .env
```

### 2. Create External Network

```bash
docker network create dockernet
```

### 3. Start Server

```bash
# Start server and Redis
docker-compose -f docker-compose.server.yml up -d

# View logs
docker-compose -f docker-compose.server.yml logs -f
```

### 4. Start Client

```bash
# Start client (attached to dockernet)
docker-compose -f docker-compose.client.yml up -d

# View logs
docker-compose -f docker-compose.client.yml logs -f
```

### 5. Test the Setup

```bash
# Send a webhook to the server
curl -X POST http://localhost:8080/webhook \
  -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"message": "Hello, World!"}}'
```

### 6. Stop Services

```bash
# Stop server and Redis
docker-compose -f docker-compose.server.yml down

# Stop client
docker-compose -f docker-compose.client.yml down
```

## Production Considerations

### Security

1. **API Key**: Always set a strong, unique `API_KEY` in `.env`
2. **Redis Password**: Set `REDIS_PASSWORD` in production
3. **Network Isolation**: Client is isolated on dockernet network
4. **No Client Ports**: Client has no exposed ports, only accessible via nginx

### Scaling

1. **Server**: Can be scaled horizontally behind a load balancer
2. **Client**: Multiple clients can be run with different `CONSUMER_NAME` values
3. **Redis**: Use Redis Cluster for high availability

### Monitoring

1. **Health Checks**: Both services expose `/health` endpoints
2. **Metrics**: Health endpoints include message counts and error rates
3. **Logs**: Use `docker-compose logs -f` to monitor

### Persistence

1. **Redis Data**: Stored in `redis-data` volume
2. **Dead Letter Queue**: Failed messages persist in Redis
3. **Message TTL**: Configurable via `MESSAGE_TTL`

## Troubleshooting

### Client Cannot Connect to Redis

- Ensure server is running: `docker-compose -f docker-compose.server.yml ps`
- Check Redis is healthy: `docker-compose -f docker-compose.server.yml logs redis`
- Verify `REDIS_URL` in client environment

### Client Cannot Reach Nginx

- Ensure dockernet network exists: `docker network ls | grep dockernet`
- Check nginx is on dockernet: `docker network inspect dockernet`
- Verify `LOCAL_WEBHOOK_URL` points to nginx: `http://nginx:3000/webhook`

### API Key Errors

- Verify `API_KEY` is set in `.env`
- Check API key in request header: `X-API-Key: your-secret-api-key`
- Restart services after changing `.env`: `docker-compose -f docker-compose.server.yml restart`

## Migration from All-in-One

If you're currently using `docker-compose.yml` (all-in-one):

1. **Stop existing services**:
   ```bash
   docker-compose down
   ```

2. **Create dockernet network**:
   ```bash
   docker network create dockernet
   ```

3. **Start server**:
   ```bash
   docker-compose -f docker-compose.server.yml up -d
   ```

4. **Start client**:
   ```bash
   docker-compose -f docker-compose.client.yml up -d
   ```

5. **Verify everything works**:
   ```bash
   docker-compose -f docker-compose.server.yml logs -f
   docker-compose -f docker-compose.client.yml logs -f
   ```

## Additional Resources

- [Quick Start Guide](QUICKSTART.md)
- [README](README.md)
- [AI Codebase Index](.github/AI_CODEBASE_INDEX.md)
