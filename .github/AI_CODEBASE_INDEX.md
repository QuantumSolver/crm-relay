# AI Codebase Index - CRM Relay Server

**Purpose**: This index provides AI assistants with a comprehensive understanding of the CRM Relay Server codebase structure, architecture, and modification patterns.

## Quick Reference

- **Language**: Go 1.21+
- **Architecture**: Relay Server + Relay Client with Redis Streams
- **Key Technologies**: Go, Redis Streams, Docker, GitHub Actions
- **Deployment**: Multi-architecture Docker images (AMD64, ARM64, ARMv7)
- **Repository**: https://github.com/QuantumSolver/crm-relay

---

## Project Architecture

```
┌─────────────────┐
│  Meta Platform  │
│   (Webhooks)    │
└────────┬────────┘
         │ HTTP POST
         ↓
┌─────────────────────────────────────────────────────────────┐
│                  Relay Server (Public)                      │
│  - Receives webhooks on /webhook endpoint                   │
│  - Validates API key authentication                         │
│  - Persists to Redis Streams                                │
│  - Returns 202 Accepted                                     │
└────────────────────────┬────────────────────────────────────┘
                         │ Redis Streams
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    Redis Streams                            │
│  - Persistent message queue                                 │
│  - Consumer groups for reliable delivery                    │
│  - Dead letter queue for failed messages                    │
└────────────────────────┬────────────────────────────────────┘
                         │ Consumer Group
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                  Relay Client (Private)                     │
│  - Consumes messages from Redis Streams                     │
│  - Forwards to local webhook endpoint                       │
│  - Implements retry logic with exponential backoff          │
│  - Moves failed messages to DLQ after max retries           │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTP POST
                         ↓
┌─────────────────┐
│  Local Webhook  │
│   (Behind       │
│   Firewall)     │
└─────────────────┘
```

---

## File Structure

```
crm-relay/
├── cmd/                                    # Entry points (binaries)
│   ├── relay-server/main.go               # Relay server HTTP server
│   ├── relay-client/main.go               # Relay client consumer
│   └── test-webhook/main.go               # Test webhook server
│
├── internal/                              # Private application code
│   ├── config/                            # Configuration management
│   │   ├── config.go                     # Load and validate config
│   │   └── config_test.go                # Configuration tests
│   │
│   ├── models/                            # Data models and types
│   │   ├── types.go                      # Core data structures
│   │   └── types_test.go                 # Model tests
│   │
│   ├── relay-server/                      # Relay server implementation
│   │   ├── handler.go                    # HTTP request handlers
│   │   └── middleware.go                 # HTTP middleware
│   │
│   ├── relay-client/                      # Relay client implementation
│   │   ├── consumer.go                   # Redis Stream consumer
│   │   └── forwarder.go                  # Webhook forwarder
│   │
│   └── storage/                           # Redis integration
│       └── redis.go                      # Redis client and operations
│
├── .github/                               # GitHub-specific files
│   ├── instructions/                      # Development guidelines
│   │   ├── go.instructions.md           # Go coding standards
│   │   └── python.instructions.md       # Python coding standards
│   │
│   ├── skills/                            # Domain-specific skills
│   │   └── redis-development/SKILL.md   # Redis best practices
│   │
│   ├── prompts/                           # AI prompts
│   │   └── relay-server-implementation.md # Implementation plan
│   │
│   └── workflows/                         # GitHub Actions workflows
│       ├── docker-build-push.yml         # Multi-arch Docker builds
│       └── README.md                     # Workflow documentation
│
├── bin/                                   # Compiled binaries (gitignored)
│   ├── relay-server                      # Server binary
│   ├── relay-client                      # Client binary
│   └── test-webhook                      # Test server binary
│
├── Dockerfile.relay-server                # Multi-arch server image
├── Dockerfile.relay-client                # Multi-arch client image
├── docker-compose.yml                     # Local development stack
├── Makefile                              # Build commands
├── go.mod                                 # Go module definition
├── go.sum                                 # Go module checksums
├── .env.example                          # Configuration template
├── .gitignore                            # Git ignore patterns
│
├── README.md                             # Main documentation
├── QUICKSTART.md                         # Quick start guide
├── IMPLEMENTATION_SUMMARY.md             # Implementation details
└── AI_CODEBASE_INDEX.md                  # This file
```

---

## Core Components

### 1. Relay Server (`internal/relay-server/`)

**Purpose**: Public-facing HTTP server that receives webhooks and persists them to Redis Streams.

**Key Files**:
- `handler.go`: HTTP request handlers
  - `HandleWebhook()`: POST /webhook endpoint
  - `HandleHealth()`: GET /health endpoint
- `middleware.go`: HTTP middleware
  - `LoggingMiddleware`: Request/response logging
  - `RecoveryMiddleware`: Panic recovery
  - `AuthenticationMiddleware`: API key validation
  - `CORSMiddleware`: CORS headers

**Entry Point**: `cmd/relay-server/main.go`

**Dependencies**:
- `internal/config`: Configuration loading
- `internal/storage`: Redis client
- `internal/models`: Data types

**Key Operations**:
1. Receive HTTP POST to `/webhook`
2. Validate API key via middleware
3. Parse request body and headers
4. Create `Webhook` struct
5. Persist to Redis Stream via `storage.AddWebhook()`
6. Return 202 Accepted with message ID

**Configuration**:
- `SERVER_PORT`: HTTP server port (default: 8080)
- `API_KEY`: Authentication key (required)
- `REDIS_URL`: Redis connection string
- `STREAM_NAME`: Redis stream name

---

### 2. Relay Client (`internal/relay-client/`)

**Purpose**: Consumes messages from Redis Streams and forwards them to local webhook endpoints.

**Key Files**:
- `consumer.go`: Redis Stream consumer
  - `Start()`: Start consuming messages
  - `Stop()`: Stop consuming
  - `processMessage()`: Process individual message
  - `forwardWithRetry()`: Forward with retry logic
- `forwarder.go`: Webhook forwarder
  - `Forward()`: Forward webhook to local endpoint
  - HTTP client with connection pooling

**Entry Point**: `cmd/relay-client/main.go`

**Dependencies**:
- `internal/config`: Configuration loading
- `internal/storage`: Redis client
- `internal/models`: Data types

**Key Operations**:
1. Connect to Redis Stream consumer group
2. Block and read messages via `storage.ReadMessages()`
3. Parse message into `RelayMessage`
4. Forward to local webhook via `forwarder.Forward()`
5. Implement retry logic with exponential backoff
6. Acknowledge successful messages via `storage.AcknowledgeMessage()`
7. Move failed messages to DLQ after max retries

**Configuration**:
- `LOCAL_WEBHOOK_URL`: Local webhook endpoint (required)
- `MAX_RETRIES`: Maximum retry attempts (default: 3)
- `RETRY_DELAY`: Initial retry delay in ms (default: 1000)
- `RETRY_MULTIPLIER`: Backoff multiplier (default: 2.0)

---

### 3. Redis Storage (`internal/storage/`)

**Purpose**: Redis client with Stream operations for reliable message delivery.

**Key File**: `redis.go`

**Key Functions**:
- `NewRedisClient()`: Create Redis client with connection pooling
- `AddWebhook()`: Add webhook to stream (XADD)
- `ReadMessages()`: Read messages from consumer group (XREADGROUP)
- `AcknowledgeMessage()`: Acknowledge processed message (XACK)
- `MoveToDeadLetterQueue()`: Move failed message to DLQ
- `GetQueueDepth()`: Get current queue length
- `GetPendingMessages()`: Get pending message count

**Redis Operations**:
- **XADD**: Add message to stream
- **XREADGROUP**: Read messages from consumer group
- **XACK**: Acknowledge message as processed
- **XLEN**: Get stream length
- **XPENDING**: Get pending messages

**Configuration**:
- `REDIS_URL`: Redis connection string
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Redis database number
- `STREAM_NAME`: Stream name
- `CONSUMER_GROUP`: Consumer group name
- `CONSUMER_NAME`: Consumer name
- `DEAD_LETTER_QUEUE`: DLQ stream name
- `MESSAGE_TTL`: Message TTL in seconds

---

### 4. Configuration (`internal/config/`)

**Purpose**: Load and validate configuration from environment variables.

**Key File**: `config.go`

**Key Functions**:
- `Load()`: Load configuration from environment
- `validate()`: Validate required fields

**Environment Variables**:
All configuration is loaded from environment variables. See `.env.example` for complete list.

**Validation Rules**:
- `API_KEY`: Required
- `LOCAL_WEBHOOK_URL`: Required
- `MAX_RETRIES`: Must be >= 0
- `RETRY_MULTIPLIER`: Must be > 0
- `MESSAGE_TTL`: Must be > 0

---

### 5. Data Models (`internal/models/`)

**Purpose**: Core data structures and types.

**Key File**: `types.go`

**Key Types**:
- `Webhook`: Incoming webhook data
  - `ID`: Unique identifier
  - `Headers`: HTTP headers
  - `Body`: Request body
  - `Timestamp`: Reception time
  - `Signature`: Optional signature

- `RelayMessage`: Message in Redis Stream
  - `MessageID`: Stream message ID
  - `Webhook`: Webhook data
  - `RetryCount`: Current retry count
  - `CreatedAt`: Creation time

- `Config`: Configuration struct
  - All configuration fields

- `Metrics`: Runtime metrics
  - `WebhooksReceived`: Total received
  - `WebhooksProcessed`: Total processed
  - `WebhooksFailed`: Total failed
  - `WebhooksRetried`: Total retries
  - `QueueDepth`: Current queue length
  - `AverageLatency`: Average latency in ms
  - `LastWebhookTime`: Last webhook timestamp

- `RelayError`: Custom error type
  - `Code`: Error code
  - `Message`: Error message
  - `Err`: Wrapped error

**Error Codes**:
- `ErrCodeInvalidRequest`: Invalid request
- `ErrCodeAuthentication`: Authentication failed
- `ErrCodeRedisConnection`: Redis connection error
- `ErrCodeStreamWrite`: Stream write error
- `ErrCodeStreamRead`: Stream read error
- `ErrCodeWebhookForward`: Webhook forward error
- `ErrCodeMaxRetriesExceeded`: Max retries exceeded
- `ErrCodeInvalidConfig`: Invalid configuration

---

## Data Flow

### Webhook Reception Flow

```
1. Meta Platform sends POST /webhook
   ↓
2. AuthenticationMiddleware validates X-API-Key
   ↓
3. LoggingMiddleware logs request
   ↓
4. Handler.HandleWebhook() processes request
   ↓
5. Create Webhook struct with headers, body, timestamp
   ↓
6. storage.AddWebhook() adds to Redis Stream
   ↓
7. Return 202 Accepted with message ID
```

### Webhook Forwarding Flow

```
1. Consumer reads messages from Redis Stream
   ↓
2. Parse message into RelayMessage
   ↓
3. Forwarder.Forward() sends to local webhook
   ↓
4. If success:
   - storage.AcknowledgeMessage()
   - Update metrics
   ↓
5. If failure:
   - Increment retry count
   - Calculate backoff delay
   - Retry after delay
   ↓
6. If max retries exceeded:
   - storage.MoveToDeadLetterQueue()
   - Update failure metrics
```

---

## Dependencies

### Go Modules

```
github.com/redis/go-redis/v9    # Redis client
github.com/google/uuid          # UUID generation
```

### External Services

- **Redis 7+**: Message queue and persistence
- **GitHub Container Registry**: Docker image registry

---

## Configuration

### Environment Variables

**Server Configuration**:
- `SERVER_PORT`: HTTP server port (default: 8080)

**Redis Configuration**:
- `REDIS_URL`: Redis connection URL (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_DB`: Redis database number (default: 0)

**Stream Configuration**:
- `STREAM_NAME`: Stream name (default: webhook-stream)
- `CONSUMER_GROUP`: Consumer group name (default: relay-group)
- `CONSUMER_NAME`: Consumer name (default: relay-client)
- `DEAD_LETTER_QUEUE`: DLQ name (default: webhook-dlq)
- `MESSAGE_TTL`: Message TTL in seconds (default: 86400)

**Authentication**:
- `API_KEY`: API key for authentication (required)

**Client Configuration**:
- `LOCAL_WEBHOOK_URL`: Local webhook URL (required)

**Retry Configuration**:
- `MAX_RETRIES`: Maximum retries (default: 3)
- `RETRY_DELAY`: Initial delay in ms (default: 1000)
- `RETRY_MULTIPLIER`: Backoff multiplier (default: 2.0)

**Health Check**:
- `HEALTH_CHECK_INTERVAL`: Interval in seconds (default: 30)

---

## Testing

### Unit Tests

**Location**: `internal/*/test.go`

**Run Tests**:
```bash
go test ./...
go test -cover ./...
```

**Test Coverage**:
- `internal/config/config_test.go`: Configuration loading and validation
- `internal/models/types_test.go`: Data types and errors

### Manual Testing

**Test Webhook Server**:
```bash
./bin/test-webhook
```

**Send Test Webhook**:
```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

**Check Health**:
```bash
curl http://localhost:8080/health
```

---

## Build and Deployment

### Local Build

```bash
# Build all binaries
make build

# Build specific binary
make build-server
make build-client

# Build multi-architecture
make build-multiarch
```

### Docker Build

```bash
# Build images
docker-compose build

# Start services
docker-compose up -d

# View logs
docker-compose logs -f
```

### GitHub Actions

**Workflow**: `.github/workflows/docker-build-push.yml`

**Triggers**:
- Push to `main` branch
- Tag push (e.g., `v1.0.0`)
- Pull request

**Platforms**: linux/amd64, linux/arm64, linux/arm/v7

**Images**:
- `ghcr.io/QuantumSolver/crm-relay/relay-server`
- `ghcr.io/QuantumSolver/crm-relay/relay-client`

---

## Common Modification Patterns

### Adding a New HTTP Endpoint

1. Add handler in `internal/relay-server/handler.go`
2. Register route in `cmd/relay-server/main.go`
3. Add middleware if needed
4. Update documentation

Example:
```go
// In handler.go
func (h *Handler) HandleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}

// In main.go
mux.HandleFunc("GET /new-endpoint", handler.HandleNewEndpoint)
```

### Adding a New Configuration Variable

1. Add field to `Config` struct in `internal/models/types.go`
2. Add loading logic in `internal/config/config.go`
3. Add validation in `internal/config/config.go`
4. Add to `.env.example`
5. Update documentation

### Modifying Retry Logic

1. Update `forwardWithRetry()` in `internal/relay-client/consumer.go`
2. Adjust backoff calculation
3. Update `Config` struct if needed
4. Add tests

### Adding a New Middleware

1. Create middleware function in `internal/relay-server/middleware.go`
2. Apply in `cmd/relay-server/main.go`
3. Add tests

### Adding Metrics

1. Add field to `Metrics` struct in `internal/models/types.go`
2. Update metric in relevant handler/consumer
3. Update health check endpoint
4. Update documentation

---

## File Dependencies

### Relay Server Dependencies

```
cmd/relay-server/main.go
├── internal/config/config.go
├── internal/relay-server/handler.go
│   ├── internal/models/types.go
│   └── internal/storage/redis.go
└── internal/relay-server/middleware.go
    └── internal/models/types.go
```

### Relay Client Dependencies

```
cmd/relay-client/main.go
├── internal/config/config.go
├── internal/relay-client/consumer.go
│   ├── internal/models/types.go
│   ├── internal/storage/redis.go
│   └── internal/relay-client/forwarder.go
│       └── internal/models/types.go
```

### Storage Dependencies

```
internal/storage/redis.go
├── internal/models/types.go
└── github.com/redis/go-redis/v9
```

---

## Key Design Decisions

### Why Redis Streams?

- **Persistence**: Messages persist until consumed
- **Consumer Groups**: Enable reliable delivery with acknowledgments
- **Replayability**: Can replay messages if needed
- **Scalability**: Can scale to multiple consumers

### Why Go?

- **Performance**: Excellent concurrency and low memory footprint
- **Single Binary**: Easy deployment
- **Type Safety**: Compile-time error checking
- **Standard Library**: Comprehensive HTTP and networking support

### Why Multi-Architecture?

- **Flexibility**: Deploy on various platforms
- **Cost**: Use ARM instances for cost savings
- **Edge**: Deploy on edge devices (Raspberry Pi)
- **Future-Proof**: Support emerging architectures

---

## Development Guidelines

### Code Style

Follow guidelines in `.github/instructions/go.instructions.md`:
- Idiomatic Go patterns
- Error handling with context
- Early returns
- Clear naming conventions
- Comprehensive documentation

### Testing

- Write unit tests for all new code
- Use table-driven tests
- Mock external dependencies
- Test error scenarios

### Documentation

- Update README.md for user-facing changes
- Update this index for structural changes
- Add inline comments for complex logic
- Keep examples up to date

---

## Troubleshooting

### Common Issues

**Redis Connection Failed**:
- Check Redis is running
- Verify REDIS_URL
- Check network connectivity

**Webhook Not Delivered**:
- Check relay client logs
- Verify LOCAL_WEBHOOK_URL
- Check dead letter queue

**Build Fails**:
- Check Go version (1.21+)
- Run `go mod tidy`
- Verify dependencies

---

## AI Assistant Guidelines

When asked to modify this codebase:

1. **Read this index first** to understand the architecture
2. **Identify affected components** using the file structure
3. **Follow the modification patterns** for common tasks
4. **Update tests** for any code changes
5. **Update documentation** for user-facing changes
6. **Consider multi-architecture** when making changes
7. **Follow Go guidelines** in `.github/instructions/go.instructions.md`
8. **Follow Redis guidelines** in `.github/skills/redis-development/SKILL.md`

### Quick Reference for Common Tasks

| Task | File(s) to Modify |
|------|-------------------|
| Add HTTP endpoint | `internal/relay-server/handler.go`, `cmd/relay-server/main.go` |
| Add config variable | `internal/models/types.go`, `internal/config/config.go`, `.env.example` |
| Modify retry logic | `internal/relay-client/consumer.go` |
| Add middleware | `internal/relay-server/middleware.go`, `cmd/relay-server/main.go` |
| Add metrics | `internal/models/types.go`, relevant handler/consumer |
| Modify Docker build | `Dockerfile.*`, `.github/workflows/docker-build-push.yml` |
| Update documentation | `README.md`, `QUICKSTART.md`, this file |

---

## Contact

- **Repository**: https://github.com/QuantumSolver/crm-relay
- **Issues**: https://github.com/QuantumSolver/crm-relay/issues

---

**Last Updated**: 2026-02-19
**Version**: 1.0.0
