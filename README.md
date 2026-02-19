# CRM Relay Server

A high-performance webhook relay system built with Go and Redis Streams. The relay server receives webhooks from external platforms (e.g., Meta) and forwards them to clients behind firewalls through a secure, reliable transport layer.

## Architecture

```
Meta Platform → Relay Server (public) → Redis Streams → Relay Client (private) → Local Webhook
```

### Components

- **Relay Server**: Public-facing HTTP server that receives webhooks and persists them to Redis Streams
- **Redis Streams**: Reliable, persistent message queue with consumer groups for delivery guarantees
- **Relay Client**: Consumes messages from Redis Streams and forwards them to local webhook endpoints
- **Dead Letter Queue**: Captures failed messages for inspection and replay

## AI Codebase Index

This project includes a comprehensive AI-optimized codebase index to help AI assistants understand the project structure and make effective modifications:

- **[AI_CODEBASE_INDEX.md](.github/AI_CODEBASE_INDEX.md)** - Detailed markdown documentation
- **[AI_CODEBASE_INDEX.json](.github/AI_CODEBASE_INDEX.json)** - Machine-readable JSON format
- **[AI_CODEBASE_INDEX.yaml](.github/AI_CODEBASE_INDEX.yaml)** - Machine-readable YAML format

These files provide:
- Complete architecture overview
- File structure and dependencies
- Data models and configuration
- Common modification patterns
- Quick reference for AI assistants

## Features

- ✅ High-performance webhook relay using Go and Redis Streams
- ✅ Multi-architecture support (AMD64, ARM64, ARMv7)
- ✅ Message persistence with configurable TTL
- ✅ Automatic retry with exponential backoff
- ✅ Dead letter queue for failed messages
- ✅ API key authentication
- ✅ Health check endpoints with metrics
- ✅ Graceful shutdown
- ✅ Docker support with multi-platform images
- ✅ Single binary deployment
- ✅ AI-optimized codebase index for easy modifications

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Redis 7 or higher
- Docker (optional, for containerized deployment)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/crm-relay.git
   cd crm-relay
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Start Redis**
   ```bash
   docker run -d -p 6379:6379 redis:7-alpine
   ```

4. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

5. **Start the relay server**
   ```bash
   export API_KEY=your-secret-api-key
   export LOCAL_WEBHOOK_URL=http://localhost:3000/webhook
   go run cmd/relay-server/main.go
   ```

6. **Start the relay client** (in another terminal)
   ```bash
   export API_KEY=your-secret-api-key
   export LOCAL_WEBHOOK_URL=http://localhost:3000/webhook
   go run cmd/relay-client/main.go
   ```

### Docker Deployment

#### Option 1: Separate Server and Client (Recommended for Production)

1. **Create external network for client**
   ```bash
   docker network create dockernet
   ```

2. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env and set your API_KEY
   ```

3. **Start server with Redis**
   ```bash
   docker-compose -f docker-compose.server.yml up -d
   ```

4. **Start client (attached to dockernet network)**
   ```bash
   docker-compose -f docker-compose.client.yml up -d
   ```

5. **View logs**
   ```bash
   docker-compose -f docker-compose.server.yml logs -f
   docker-compose -f docker-compose.client.yml logs -f
   ```

6. **Stop services**
   ```bash
   docker-compose -f docker-compose.server.yml down
   docker-compose -f docker-compose.client.yml down
   ```

#### Option 2: All-in-One (Development)

1. **Build and start all services**
   ```bash
   docker-compose up -d
   ```

2. **View logs**
   ```bash
   docker-compose logs -f
   ```

3. **Stop services**
   ```bash
   docker-compose down
   ```

### Using Pre-built Multi-Architecture Images

The separate Docker Compose files use pre-built images from GitHub Container Registry (supports AMD64, ARM64, ARMv7):

```bash
# Pull latest images
docker pull ghcr.io/QuantumSolver/crm-relay/relay-server:latest
docker pull ghcr.io/QuantumSolver/crm-relay/relay-client:latest

# Docker automatically selects the correct architecture for your system
```

The `docker-compose.server.yml` and `docker-compose.client.yml` files are already configured to use these pre-built images.

### Building Multi-Architecture Binaries Locally

```bash
# Build for all supported architectures
make build-multiarch

# Build for specific architecture
make build-server-arm64
make build-client-arm64
```

This creates binaries like:
- `bin/relay-server-linux-amd64`
- `bin/relay-server-linux-arm64`
- `bin/relay-server-linux-armv7`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `REDIS_URL` | Redis connection URL | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `REDIS_DB` | Redis database number | `0` |
| `STREAM_NAME` | Redis stream name | `webhook-stream` |
| `CONSUMER_GROUP` | Consumer group name | `relay-group` |
| `CONSUMER_NAME` | Consumer name | `relay-client` |
| `DEAD_LETTER_QUEUE` | Dead letter queue name | `webhook-dlq` |
| `MESSAGE_TTL` | Message TTL in seconds | `86400` (24h) |
| `API_KEY` | API key for authentication | (required) |
| `LOCAL_WEBHOOK_URL` | Local webhook endpoint URL | (required) |
| `MAX_RETRIES` | Maximum retry attempts | `3` |
| `RETRY_DELAY` | Initial retry delay in ms | `1000` |
| `RETRY_MULTIPLIER` | Retry delay multiplier | `2.0` |
| `HEALTH_CHECK_INTERVAL` | Health check interval in seconds | `30` |

## API Reference

### Webhook Endpoint

**POST** `/webhook`

Receives webhook payloads from external platforms.

**Headers:**
- `X-API-Key`: API key for authentication (required)
- `Content-Type`: `application/json` (recommended)

**Request Body:**
```json
{
  "event": "message_received",
  "data": {
    "id": "12345",
    "message": "Hello, World!"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message_id": "1678901234567-0",
  "webhook_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Health Check Endpoint

**GET** `/health`

Returns health status and metrics.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "redis": {
    "status": "ok",
    "queue_depth": 5
  },
  "metrics": {
    "webhooks_received": 100,
    "webhooks_processed": 95,
    "webhooks_failed": 2,
    "webhooks_retried": 8,
    "average_latency_ms": 45,
    "last_webhook_time": "2024-01-15T10:29:55Z"
  }
}
```

## Testing

### Manual Testing

1. **Send a test webhook**
   ```bash
   curl -X POST http://localhost:8080/webhook \
     -H "X-API-Key: your-secret-api-key" \
     -H "Content-Type: application/json" \
     -d '{"test": "data"}'
   ```

2. **Check health status**
   ```bash
   curl http://localhost:8080/health
   ```

3. **Test retry logic**
   - Stop your local webhook endpoint
   - Send a webhook
   - Restart your local webhook endpoint
   - Verify the webhook is delivered

4. **Test dead letter queue**
   - Exceed max retries by keeping local webhook down
   - Check Redis DLQ stream: `redis-cli XLEN webhook-dlq`

### Automated Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Monitoring

### Metrics

The system tracks the following metrics:

- **Webhooks Received**: Total number of webhooks received
- **Webhooks Processed**: Total number of webhooks successfully processed
- **Webhooks Failed**: Total number of webhooks that failed after max retries
- **Webhooks Retried**: Total number of retry attempts
- **Queue Depth**: Current number of messages in the stream
- **Average Latency**: Average processing latency in milliseconds
- **Last Webhook Time**: Timestamp of the last received webhook

### Logs

Both relay server and client log important events:

- Webhook receipt and processing
- Success/failure of webhook forwarding
- Retry attempts
- Errors and warnings
- Health check results

## Troubleshooting

### Redis Connection Issues

**Problem**: Relay server/client cannot connect to Redis

**Solutions**:
- Verify Redis is running: `redis-cli ping`
- Check Redis URL in configuration
- Verify network connectivity
- Check Redis logs

### Webhook Not Delivered

**Problem**: Webhook received but not delivered to local endpoint

**Solutions**:
- Check relay client logs for errors
- Verify local webhook URL is correct
- Ensure local webhook endpoint is accessible
- Check dead letter queue for failed messages
- Verify retry configuration

### High Memory Usage

**Problem**: Redis memory usage growing

**Solutions**:
- Reduce `MESSAGE_TTL` to expire old messages
- Monitor and clean dead letter queue
- Adjust Redis maxmemory settings
- Implement periodic cleanup of processed messages

## Development

### Project Structure

```
crm-relay/
├── cmd/
│   ├── relay-server/     # Relay server entry point
│   └── relay-client/     # Relay client entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── models/           # Data models
│   ├── relay-server/     # Relay server handlers and middleware
│   ├── relay-client/     # Relay client consumer and forwarder
│   └── storage/          # Redis integration
├── .github/
│   ├── instructions/     # Development guidelines
│   └── skills/           # Domain-specific skills
├── docker-compose.yml    # Docker Compose configuration
├── Dockerfile.relay-server
├── Dockerfile.relay-client
└── README.md
```

### Building Binaries

```bash
# Build relay server
go build -o relay-server ./cmd/relay-server

# Build relay client
go build -o relay-client ./cmd/relay-client

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o relay-server-linux ./cmd/relay-server
```

## Security Considerations

1. **API Key Authentication**: Always use strong, unique API keys
2. **TLS/SSL**: Enable HTTPS in production environments
3. **Redis Security**: Use Redis ACLs and authentication
4. **Network Security**: Use firewalls and network segmentation
5. **Input Validation**: Validate all incoming webhook payloads
6. **Rate Limiting**: Implement rate limiting to prevent abuse

## Performance Tuning

### Redis Optimization

- Use Redis persistence (AOF or RDB) for durability
- Configure appropriate `maxmemory` settings
- Use connection pooling (already configured)
- Monitor memory usage and eviction policies

### Go Optimization

- Adjust HTTP client timeouts based on network conditions
- Tune worker pool size for concurrent processing
- Use appropriate buffer sizes for I/O operations
- Monitor goroutine leaks

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please read the development guidelines in `.github/instructions/` before submitting pull requests.

## Support

For issues and questions, please open an issue on GitHub.
