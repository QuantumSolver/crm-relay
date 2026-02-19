# Plan: React UI with JWT Auth and Dynamic Webhook Management

**TL;DR**: Add two React frontends (server-ui and client-ui) embedded in Go binaries with JWT authentication. Implement path-based webhook routing, API key management, and webhook endpoint configuration. Use Redis for storing users, keys, and endpoint configs. Both UIs share design patterns but have separate auth flows and features.

---

## Architecture Overview

**Server UI Features**:
- JWT-based authentication (single admin)
- API key management (create, list, update, delete)
- Webhook endpoint configuration per platform/service
- Path-based routing (`/webhook/meta`, `/webhook/slack`, etc.)
- Real-time metrics and monitoring
- Webhook logs viewer

**Client UI Features**:
- JWT-based authentication (single admin)
- Local webhook endpoint configuration
- Retry settings management
- Real-time metrics and monitoring
- Dead letter queue viewer
- Consumer group management

**Data Flow**:
```
Platform → POST /webhook/{platform} → Server (validates key, routes) 
→ Redis Stream (with routing info) → Client (reads routing, forwards to configured endpoint)
```

---

## Steps

### Phase 1: Backend - Data Models & Storage

1. **Add new data models** in [internal/models/types.go](internal/models/types.go)
   - `User` struct (username, password hash, role)
   - `APIKey` struct (id, name, key, platform, created_at, is_active)
   - `WebhookEndpoint` struct (id, platform, path, http_method, headers, retry_config)
   - `JWTClaims` struct (user_id, username, role, exp)
   - `LoginRequest` and `LoginResponse` structs

2. **Extend Redis storage** in [internal/storage/redis.go](internal/storage/redis.go)
   - Add `StoreUser()`, `GetUser()`, `ValidateUser()` methods
   - Add `CreateAPIKey()`, `GetAPIKey()`, `ListAPIKeys()`, `UpdateAPIKey()`, `DeleteAPIKey()` methods
   - Add `CreateEndpoint()`, `GetEndpoint()`, `ListEndpoints()`, `UpdateEndpoint()`, `DeleteEndpoint()` methods
   - Add `GetEndpointByPath()` for routing lookup
   - Use Redis JSON for structured data (following Redis best practices)
   - Set appropriate TTLs and use consistent key naming

3. **Add user initialization** in [internal/storage/redis.go](internal/storage/redis.go)
   - Create default admin user on first startup
   - Generate random password and log it
   - Allow password reset via API

### Phase 2: Backend - Authentication System

4. **Implement JWT utilities** in new file [internal/auth/jwt.go](internal/auth/jwt.go)
   - `GenerateToken()` - create JWT with claims
   - `ValidateToken()` - parse and validate JWT
   - `HashPassword()` - bcrypt password hashing
   - `VerifyPassword()` - verify password against hash
   - Use environment variable for JWT secret

5. **Add authentication middleware** in [internal/relay-server/middleware.go](internal/relay-server/middleware.go)
   - `JWTMiddleware()` - validate JWT token from Authorization header
   - Skip auth for health check and login endpoint
   - Add user context to request

6. **Add auth endpoints** in [internal/relay-server/handler.go](internal/relay-server/handler.go)
   - `HandleLogin()` - POST /api/auth/login
   - `HandleLogout()` - POST /api/auth/logout
   - `HandleGetCurrentUser()` - GET /api/auth/me

### Phase 3: Backend - Management APIs

7. **Add API key management endpoints** in [internal/relay-server/handler.go](internal/relay-server/handler.go)
   - `HandleListAPIKeys()` - GET /api/keys
   - `HandleCreateAPIKey()` - POST /api/keys
   - `HandleUpdateAPIKey()` - PUT /api/keys/:id
   - `HandleDeleteAPIKey()` - DELETE /api/keys/:id

8. **Add webhook endpoint management** in [internal/relay-server/handler.go](internal/relay-server/handler.go)
   - `HandleListEndpoints()` - GET /api/endpoints
   - `HandleCreateEndpoint()` - POST /api/endpoints
   - `HandleUpdateEndpoint()` - PUT /api/endpoints/:id
   - `HandleDeleteEndpoint()` - DELETE /api/endpoints/:id

9. **Add metrics and monitoring endpoints** in [internal/relay-server/handler.go](internal/relay-server/handler.go)
   - `HandleGetMetrics()` - GET /api/metrics (enhanced with endpoint-specific metrics)
   - `HandleGetQueueDepth()` - GET /api/queue-depth
   - `HandleGetPendingMessages()` - GET /api/pending-messages

10. **Add client management endpoints** in new file [internal/relay-client/handler.go](internal/relay-client/handler.go)
    - `HandleUpdateLocalEndpoint()` - PUT /api/config/local-endpoint
    - `HandleUpdateRetryConfig()` - PUT /api/config/retry
    - `HandleGetDLQMessages()` - GET /api/dlq
    - `HandleReplayDLQMessage()` - POST /api/dlq/:id/replay

### Phase 4: Backend - Webhook Routing

11. **Modify webhook handler** in [internal/relay-server/handler.go](internal/relay-server/handler.go)
    - Update `HandleWebhook()` to support path-based routing
    - Extract platform from URL path: `/webhook/{platform}`
    - Look up endpoint configuration using `GetEndpointByPath()`
    - Validate API key against platform-specific key
    - Add routing metadata to webhook message (platform, endpoint_id, http_method)

12. **Update webhook data model** in [internal/models/types.go](internal/models/types.go)
    - Add `Platform` field to `Webhook` struct
    - Add `EndpointID` field to `Webhook` struct
    - Add `HTTPMethod` field to `Webhook` struct
    - Add `TargetEndpoint` field to `RelayMessage` struct

13. **Update client consumer** in [internal/relay-client/consumer.go](internal/relay-client/consumer.go)
    - Read routing metadata from message
    - Use `TargetEndpoint` for forwarding (override LOCAL_WEBHOOK_URL)
    - Use `HTTPMethod` for the request (default to POST if not specified)
    - Log routing information

### Phase 5: Backend - Static File Serving

14. **Add static file serving** in [cmd/relay-server/main.go](cmd/relay-server/main.go)
    - Use Go embed to bundle React build files
    - Add file server for `/` path
    - Handle SPA routing (fallback to index.html for non-API routes)
    - Serve API routes before static files

15. **Add static file serving** in [cmd/relay-client/main.go](cmd/relay-client/main.go)
    - Same approach as server
    - Separate embedded UI for client

### Phase 6: Frontend - Setup

16. **Initialize React projects**
    - Create `web/server-ui/` directory
    - Create `web/client-ui/` directory
    - Set up Vite for both projects
    - Install dependencies: React, React Router, Axios, Zustand, shadcn/ui

17. **Set up shadcn/ui** in both projects
    - Initialize shadcn/ui
    - Install required components: Button, Input, Card, Table, Dialog, Form, etc.
    - Set up theme configuration
    - Create shared design tokens

18. **Set up project structure** for both UIs
    ```
    web/server-ui/
    ├── src/
    │   ├── components/
    │   │   ├── ui/ (shadcn components)
    │   │   ├── layout/
    │   │   ├── auth/
    │   │   └── dashboard/
    │   ├── pages/
    │   ├── hooks/
    │   ├── lib/
    │   │   ├── api.ts
    │   │   └── auth.ts
    │   ├── stores/
    │   └── App.tsx
    ```

### Phase 7: Frontend - Server UI

19. **Build authentication flow** in `web/server-ui/src/`
    - Login page with username/password
    - Auth context for managing JWT token
    - Protected route wrapper
    - Auto-refresh token before expiry
    - Logout functionality

20. **Build dashboard** in `web/server-ui/src/pages/Dashboard.tsx`
    - Real-time metrics cards (webhooks received, processed, failed, queue depth)
    - Charts for webhook volume over time
    - Recent webhook logs table
    - Auto-refresh every 5 seconds

21. **Build API key management** in `web/server-ui/src/pages/APIKeys.tsx`
    - List all API keys in a table
    - Create new API key dialog (name, platform, auto-generate key)
    - Edit API key dialog
    - Delete API key with confirmation
    - Toggle active/inactive status
    - Copy key to clipboard

22. **Build webhook endpoint management** in `web/server-ui/src/pages/Endpoints.tsx`
    - List all endpoints in a table
    - Create new endpoint dialog (platform, path, http_method, headers, retry_config)
    - Edit endpoint dialog
    - Delete endpoint with confirmation
    - Test endpoint functionality

23. **Build webhook logs viewer** in `web/server-ui/src/pages/Logs.tsx`
    - Paginated table of webhook logs
    - Filter by platform, status, date range
    - View detailed webhook payload
    - Search functionality

### Phase 8: Frontend - Client UI

24. **Build authentication flow** in `web/client-ui/src/`
    - Same as server UI but separate auth context
    - Separate JWT token storage

25. **Build client dashboard** in `web/client-ui/src/pages/Dashboard.tsx`
    - Real-time metrics (processed, failed, retried, queue depth)
    - Charts for processing rate
    - Recent activity log
    - Auto-refresh every 5 seconds

26. **Build local endpoint configuration** in `web/client-ui/src/pages/Config.tsx`
    - Form to update LOCAL_WEBHOOK_URL
    - Test endpoint connectivity
    - View current configuration

27. **Build retry settings** in `web/client-ui/src/pages/RetryConfig.tsx`
    - Form to update MAX_RETRIES, RETRY_DELAY, RETRY_MULTIPLIER
    - Preview retry schedule
    - Reset to defaults

28. **Build dead letter queue viewer** in `web/client-ui/src/pages/DLQ.tsx`
    - List failed messages in DLQ
    - View message details
    - Replay individual messages
    - Bulk replay functionality
    - Delete messages

### Phase 9: Build & Integration

29. **Update build process** in [Makefile](Makefile)
    - Add `build-ui-server` target (build React app for server)
    - Add `build-ui-client` target (build React app for client)
    - Update `build` target to include UI builds
    - Add `watch-ui` target for development

30. **Update Docker setup**
    - Update [Dockerfile.relay-server](Dockerfile.relay-server) to build UI
    - Update [Dockerfile.relay-client](Dockerfile.relay-client) to build UI
    - Update [docker-compose.yml](docker-compose.yml) if needed

31. **Add environment variables** to [internal/models/types.go](internal/models/types.go) and [.env.example](.env.example)
    - `JWT_SECRET` - JWT signing secret
    - `ADMIN_USERNAME` - Default admin username
    - `ADMIN_PASSWORD` - Default admin password (optional, auto-generate if not set)
    - `JWT_EXPIRATION` - Token expiration time (default: 24h)

### Phase 10: Testing & Documentation

32. **Add tests**
    - Unit tests for JWT utilities
    - Unit tests for new storage methods
    - Integration tests for auth endpoints
    - Integration tests for management APIs
    - E2E tests for UI flows

33. **Update documentation**
    - Update [README.md](README.md) with UI features
    - Add UI screenshots and usage guide
    - Document API endpoints
    - Update [QUICKSTART.md](QUICKSTART.md) with UI setup
    - Add troubleshooting section for UI

34. **Add security considerations**
    - Document JWT secret management
    - Document HTTPS requirement for production
    - Document CORS configuration
    - Document rate limiting recommendations

---

## Verification

**Manual Testing**:
1. Start server and client with UIs
2. Login to server UI with default admin credentials
3. Create API key for Meta platform
4. Create webhook endpoint for Meta at `/webhook/meta`
5. Send test webhook to `/webhook/meta` with new API key
6. Verify webhook appears in server logs
7. Login to client UI
8. Verify webhook was received and processed
9. Check metrics in both UIs
10. Test retry configuration changes
11. Test DLQ replay functionality

**Automated Testing**:
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run UI tests
cd web/server-ui && npm test
cd web/client-ui && npm test
```

**Build Verification**:
```bash
# Build everything
make build

# Verify binaries include UI
./bin/relay-server --help
./bin/relay-client --help

# Test with Docker
docker-compose up -d
docker-compose logs -f
```

---

## Decisions

- **JWT over sessions**: Stateless, easier to scale, works well with SPA
- **Single admin over multi-user**: Simpler, sufficient for current use case, can be extended later
- **Path-based routing**: Clear separation of platforms, easy to understand, RESTful
- **Embedded UI over separate deployment**: Single binary deployment, simpler operations, no extra infrastructure
- **Redis JSON for structured data**: Flexible schema, efficient queries, follows Redis best practices
- **shadcn/ui over other UI libraries**: Modern, customizable, built on Radix UI and Tailwind CSS
- **Vite over CRA**: Faster builds, better DX, modern tooling

---

## File Structure After Changes

```
crm-relay/
├── cmd/
│   ├── relay-server/main.go (updated with static file serving)
│   └── relay-client/main.go (updated with static file serving)
├── internal/
│   ├── auth/ (new)
│   │   └── jwt.go
│   ├── models/types.go (updated with new models)
│   ├── relay-server/
│   │   ├── handler.go (updated with new endpoints)
│   │   └── middleware.go (updated with JWT middleware)
│   ├── relay-client/
│   │   ├── handler.go (new)
│   │   └── consumer.go (updated for routing)
│   └── storage/redis.go (updated with new storage methods)
├── web/
│   ├── server-ui/ (new React app)
│   │   ├── src/
│   │   ├── package.json
│   │   └── vite.config.ts
│   └── client-ui/ (new React app)
│       ├── src/
│       ├── package.json
│       └── vite.config.ts
├── Makefile (updated)
├── Dockerfile.relay-server (updated)
├── Dockerfile.relay-client (updated)
└── .env.example (updated)
```

---

This plan provides a comprehensive roadmap for adding full-featured UIs to your relay server and client. The implementation follows Go best practices, Redis best practices, and modern React patterns with shadcn/ui.
