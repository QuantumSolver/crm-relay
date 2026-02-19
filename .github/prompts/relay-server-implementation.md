# Plan: Go Relay Server with Redis Streams

**TL;DR** — Build a high-performance webhook relay system in Go using Redis Streams as the transport layer. The relay server (public-facing) receives Meta webhooks, persists them to Redis Streams, and the relay client (behind firewall) consumes and forwards to local endpoints. Includes retry logic, dead letter queue, and API key authentication. Single client deployment with self-hosted Redis for low-scale (< 100 req/min) operation.

**Architecture Overview**
```
Meta Platform → Relay Server (public) → Redis Streams → Relay Client (private) → Local Webhook
```

**Steps**

### Phase 1: Project Setup & Configuration
1. Initialize Go module with `go mod init github.com/yourusername/crm-relay`
2. Create project structure following [Go instructions](../instructions/go.instructions.md#L1-L373):
   - [cmd/relay-server/main.go](../../cmd/relay-server/main.go) - Relay server entry point
   - [cmd/relay-client/main.go](../../cmd/relay-client/main.go) - Relay client entry point
   - [internal/config/config.go](../../internal/config/config.go) - Configuration management (env vars, validation)
   - [internal/models/types.go](../../internal/models/types.go) - Data models (Webhook, RelayMessage, etc.)
3. Add `.gitignore` for Go (binaries, test coverage, IDE files)
4. Create `README.md` with architecture overview and setup instructions

### Phase 2: Core Data Models & Configuration
5. Define core types in [internal/models/types.go](../../internal/models/types.go):
   - `Webhook` struct (headers, body, timestamp, signature)
   - `RelayMessage` struct (message ID, webhook data, retry count)
   - `Config` struct (server port, Redis URL, API keys, local webhook URL)
6. Implement configuration loader in [internal/config/config.go](../../internal/config/config.go):
   - Load from environment variables
   - Validate required fields
   - Provide sensible defaults

### Phase 3: Redis Integration
7. Create [internal/storage/redis.go](../../internal/storage/redis.go) following [Redis skill](../skills/redis-development/SKILL.md):
   - Redis connection pooling with timeouts
   - Stream operations: `XADD` (produce), `XREAD`/`XREADGROUP` (consume)
   - Consumer group management for single client
   - Message acknowledgment (`XACK`)
   - Dead letter queue stream
8. Implement retry logic with exponential backoff
9. Add TTL for message persistence (configurable, default 24h)

### Phase 4: Relay Server Implementation
10. Create [internal/relay-server/handler.go](../../internal/relay-server/handler.go):
    - HTTP webhook endpoint (`POST /webhook`)
    - API key authentication middleware
    - Meta signature verification (optional, can add later)
    - Request validation and parsing
    - Write to Redis Stream
11. Implement middleware in [internal/relay-server/middleware.go](../../internal/relay-server/middleware.go):
    - Logging middleware
    - Recovery middleware (panic handling)
    - API key authentication
12. Set up HTTP server in [cmd/relay-server/main.go](../../cmd/relay-server/main.go):
    - Use Go 1.22+ enhanced `ServeMux` (per [Go instructions](../instructions/go.instructions.md#L1-L373))
    - Graceful shutdown
    - Health check endpoint (`GET /health`)

### Phase 5: Relay Client Implementation
13. Create [internal/relay-client/consumer.go](../../internal/relay-client/consumer.go):
    - Redis Stream consumer using consumer group
    - Blocking read with timeout
    - Process messages sequentially or with worker pool
14. Implement webhook forwarding in [internal/relay-client/forwarder.go](../../internal/relay-client/forwarder.go):
    - HTTP client to local webhook endpoint
    - Retry logic with exponential backoff
    - Move to dead letter queue after max retries
15. Set up client in [cmd/relay-client/main.go](../../cmd/relay-client/main.go):
    - Graceful shutdown handling
    - Connection recovery on Redis failures
    - Logging and metrics

### Phase 6: Error Handling & Resilience
16. Implement comprehensive error handling:
    - Wrap errors with context (per [Go instructions](../instructions/go.instructions.md#L1-L373))
    - Custom error types for relay-specific errors
    - Circuit breaker for local webhook failures
17. Add dead letter queue handling:
    - Failed messages after max retries → DLQ stream
    - Optional: Admin endpoint to inspect/replay DLQ messages

### Phase 7: Testing
18. Write unit tests following [Go instructions](../instructions/go.instructions.md#L1-L373):
    - Table-driven tests for handlers
    - Mock Redis client for storage tests
    - Test retry logic and error scenarios
19. Add integration tests:
    - Test full flow: webhook → Redis → client → local endpoint
    - Test Redis connection failures
    - Test concurrent requests

### Phase 8: Docker & Deployment
20. Create `Dockerfile` for relay server (multi-stage build)
21. Create `Dockerfile` for relay client
22. Create `docker-compose.yml` with:
    - Redis service
    - Relay server service
    - Relay client service
    - Environment variables configuration
23. Add `.env.example` with all required configuration

### Phase 9: Documentation & Monitoring
24. Update [README.md](../../README.md) with:
    - Architecture diagram
    - Setup instructions
    - Configuration reference
    - Troubleshooting guide
25. Add basic metrics:
    - Webhooks received/processed
    - Success/failure rates
    - Queue depth
    - Latency metrics

**Verification**

### Manual Testing
1. Start Redis: `docker-compose up redis`
2. Start relay server: `go run cmd/relay-server/main.go`
3. Start relay client: `go run cmd/relay-client/main.go`
4. Send test webhook: `curl -X POST http://localhost:8080/webhook -H "X-API-Key: test-key" -d '{"test": "data"}'`
5. Verify local webhook receives the payload
6. Test retry logic: stop local webhook, send webhook, restart local webhook
7. Test dead letter queue: exceed max retries, check DLQ stream

### Automated Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Load Testing (Optional)
```bash
# Use vegeta or similar tool
echo "POST http://localhost:8080/webhook" | vegeta attack -rate=10 -duration=30s | vegeta report
```

**Decisions**
- **Go over Python**: Chosen for superior concurrency, performance, and single binary deployment. Comprehensive Go instructions already available.
- **Redis Streams over Pub/Sub**: Streams provide persistence, consumer groups, and replayability — critical for reliable webhook delivery.
- **Single client architecture**: Simplifies consumer group management. Can scale to multiple clients later using stream partitioning.
- **Self-hosted Redis**: Cost-effective for low scale, gives full control. Can migrate to managed Redis (Redis Cloud, ElastiCache) if scale increases.
- **API key authentication**: Simple, effective for single-client scenario. Can add TLS/IP whitelisting later if needed.
- **Retry with exponential backoff**: Standard pattern for handling transient failures in local webhook.
- **Dead letter queue**: Essential for debugging failed webhooks without losing data.

**Additional Skills to Consider**
Based on this architecture, you may want to add skills for:
- Docker containerization best practices
- Go testing patterns (beyond what's in instructions)
- Monitoring and observability (Prometheus, OpenTelemetry)
- CI/CD pipelines for Go projects
