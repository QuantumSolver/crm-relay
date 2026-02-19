# Quick Start Guide

This guide will help you get the CRM Relay Server up and running in minutes.

## Prerequisites

- Go 1.21+ (or use the provided binaries)
- Redis 7+ (or Docker)
- A local webhook endpoint to receive forwarded webhooks

## Option 1: Using Pre-built Binaries

### 1. Start Redis

```bash
docker run -d -p 6379:6379 --name crm-redis redis:7-alpine
```

### 2. Start Test Webhook Server

```bash
./bin/test-webhook
```

This starts a simple webhook server on `http://localhost:3000` that logs all received webhooks.

### 3. Start Relay Server

```bash
export API_KEY=your-secret-api-key
export LOCAL_WEBHOOK_URL=http://localhost:3000/webhook
./bin/relay-server
```

### 4. Start Relay Client

In a new terminal:

```bash
export API_KEY=your-secret-api-key
export LOCAL_WEBHOOK_URL=http://localhost:3000/webhook
./bin/relay-client
```

### 5. Test the Relay

```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"message": "Hello, World!"}}'
```

You should see the webhook logged in the test webhook server terminal.

## Option 2: Using Docker Compose (Separate Server and Client)

### Prerequisites

Ensure you have an external network named `dockernet` for the client to communicate with nginx:

```bash
docker network create dockernet
```

### 1. Start Server Services

```bash
# Copy and configure environment variables
cp .env.example .env
# Edit .env and set your API_KEY

# Start server with Redis
docker-compose -f docker-compose.server.yml up -d
```

This starts:
- Redis on port 6379
- Relay Server on port 8080

### 2. Start Client Service

```bash
# Start client (attached to dockernet network)
docker-compose -f docker-compose.client.yml up -d
```

This starts:
- Relay Client (configured to forward to nginx on dockernet network)

### 3. Test the Relay

```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"message": "Hello, World!"}}'
```

### 4. View Logs

```bash
# View server logs
docker-compose -f docker-compose.server.yml logs -f

# View client logs
docker-compose -f docker-compose.client.yml logs -f
```

### 5. Stop Services

```bash
# Stop server and Redis
docker-compose -f docker-compose.server.yml down

# Stop client
docker-compose -f docker-compose.client.yml down
```

## Option 3: Using Docker Compose (All-in-One)

### 1. Start All Services

```bash
docker-compose up -d
```

This starts:
- Redis on port 6379
- Relay Server on port 8080
- Relay Client (configured to forward to host.docker.internal:3000)

### 2. Start Test Webhook Server

```bash
./bin/test-webhook
```

### 3. Test the Relay

```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"message": "Hello, World!"}}'
```

### 4. View Logs

```bash
docker-compose logs -f
```

### 5. Stop Services

```bash
docker-compose down
```

## Option 3: Building from Source

### 1. Install Dependencies

```bash
go mod download
```

### 2. Build Binaries

```bash
make build
```

Or build individually:

```bash
go build -o bin/relay-server ./cmd/relay-server
go build -o bin/relay-client ./cmd/relay-client
go build -o bin/test-webhook ./cmd/test-webhook
```

### 3. Follow Option 1 steps

## Testing

### Check Health Status

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "redis": {
    "status": "ok",
    "queue_depth": 0
  },
  "metrics": {
    "webhooks_received": 1,
    "webhooks_processed": 1,
    "webhooks_failed": 0,
    "webhooks_retried": 0,
    "average_latency_ms": 45,
    "last_webhook_time": "2024-01-15T10:29:55Z"
  }
}
```

### Test Retry Logic

1. Stop the test webhook server
2. Send a webhook
3. Start the test webhook server again
4. The webhook should be delivered after retry

### Test Dead Letter Queue

1. Keep the test webhook server stopped
2. Send multiple webhooks (more than MAX_RETRIES)
3. Check Redis DLQ:

```bash
docker exec -it crm-redis redis-cli XLEN webhook-dlq
```

## Configuration

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key configuration options:

- `API_KEY`: Your secret API key (required)
- `LOCAL_WEBHOOK_URL`: Your local webhook endpoint (required)
- `MAX_RETRIES`: Maximum retry attempts (default: 3)
- `RETRY_DELAY`: Initial retry delay in ms (default: 1000)
- `MESSAGE_TTL`: Message TTL in seconds (default: 86400)

## Troubleshooting

### Redis Connection Failed

- Verify Redis is running: `docker ps` or `redis-cli ping`
- Check Redis URL in configuration
- Ensure Redis port 6379 is accessible

### Webhook Not Delivered

- Check relay client logs for errors
- Verify local webhook URL is correct
- Ensure local webhook endpoint is running
- Check dead letter queue for failed messages

### Permission Denied

- Ensure binaries are executable: `chmod +x bin/*`
- Check file permissions

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Configure your actual webhook endpoint
- Set up monitoring and alerting
- Deploy to production using Docker or Kubernetes

## Support

For issues and questions, please open an issue on GitHub.
