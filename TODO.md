# CRM Relay UI - TODO List

**Last Updated**: February 19, 2026

---

## High Priority Tasks

### Client UI Implementation

- [x] Create [web/client-ui/src/lib/api.ts](web/client-ui/src/lib/api.ts)
  - [x] Set up axios instance with baseURL
  - [x] Add auth interceptors (token injection, 401 handling)
  - [x] Implement authApi (login, getCurrentUser)
  - [x] Implement configApi (updateLocalEndpoint, updateRetryConfig)
  - [x] Implement dlqApi (getMessages, replayMessage, deleteMessage)
  - [x] Implement metricsApi (getMetrics)
  - [x] Add TypeScript interfaces for all request/response types

- [x] Create [web/client-ui/src/stores/authStore.ts](web/client-ui/src/stores/authStore.ts)
  - [x] Set up Zustand store with AuthState interface
  - [x] Implement login function
  - [x] Implement logout function
  - [x] Implement checkAuth function
  - [x] Add localStorage persistence middleware
  - [x] Add clearError function

- [x] Create [web/client-ui/src/App.tsx](web/client-ui/src/App.tsx)
  - [x] Set up BrowserRouter
  - [x] Define Routes for /login, /, /config, /retry-config, /dlq
  - [x] Implement protected route logic with Navigate
  - [x] Add useEffect for checkAuth on mount
  - [x] Wrap authenticated routes with Layout component

- [x] Create [web/client-ui/src/pages/Login.tsx](web/client-ui/src/pages/Login.tsx)
  - [x] Create login form with username/password fields
  - [x] Implement form submission with useAuthStore.login
  - [x] Add error display
  - [x] Add loading state
  - [x] Add note about default credentials

- [x] Create [web/client-ui/src/pages/Dashboard.tsx](web/client-ui/src/pages/Dashboard.tsx)
  - [x] Create metrics cards (processed, failed, retried, queue depth)
  - [x] Add system information section
  - [x] Implement auto-refresh every 5 seconds
  - [x] Add loading and error states
  - [x] Style to match server UI

- [x] Create [web/client-ui/src/pages/Config.tsx](web/client-ui/src/pages/Config.tsx)
  - [x] Create form to update LOCAL_WEBHOOK_URL
  - [x] Add test endpoint connectivity button
  - [x] Display current configuration
  - [x] Add success/error messages
  - [x] Add validation for URL format

- [x] Create [web/client-ui/src/pages/RetryConfig.tsx](web/client-ui/src/pages/RetryConfig.tsx)
  - [x] Create form to update MAX_RETRIES
  - [x] Create form to update RETRY_DELAY
  - [x] Create form to update RETRY_MULTIPLIER
  - [x] Add preview retry schedule
  - [x] Add reset to defaults button
  - [x] Display current configuration

- [x] Create [web/client-ui/src/pages/DLQ.tsx](web/client-ui/src/pages/DLQ.tsx)
  - [x] Create table to list failed messages
  - [x] Add view message details modal
  - [x] Add replay individual message button
  - [x] Add delete message button
  - [x] Add bulk replay functionality
  - [x] Add pagination or infinite scroll
  - [x] Add filter/search functionality

- [x] Create [web/client-ui/src/components/Layout.tsx](web/client-ui/src/components/Layout.tsx)
  - [x] Create navbar with "CRM Relay Client" title
  - [x] Add navigation links (Dashboard, Config, RetryConfig, DLQ)
  - [x] Implement active state highlighting
  - [x] Add user display
  - [x] Add logout button

- [x] Update [web/client-ui/vite.config.ts](web/client-ui/vite.config.ts)
  - [x] Set base: '/'
  - [x] Configure build.outDir: 'dist'
  - [x] Configure build.emptyOutDir: true
  - [x] Configure server.port: 3001
  - [x] Configure server.proxy for /api to http://localhost:8081

---

## Medium Priority Tasks

### Testing

- [ ] Write unit tests for JWT utilities
  - [ ] Test GenerateToken
  - [ ] Test ValidateToken
  - [ ] Test HashPassword
  - [ ] Test VerifyPassword

- [ ] Write unit tests for storage methods
  - [ ] Test StoreUser, GetUser, ValidateUser
  - [ ] Test CreateAPIKey, GetAPIKey, ListAPIKeys, UpdateAPIKey, DeleteAPIKey
  - [ ] Test CreateEndpoint, GetEndpoint, ListEndpoints, UpdateEndpoint, DeleteEndpoint
  - [ ] Test ReadDLQMessages, GetDLQMessage, DeleteDLQMessage

- [ ] Write integration tests for auth endpoints
  - [ ] Test POST /api/auth/login
  - [ ] Test GET /api/auth/me
  - [ ] Test JWT middleware

- [ ] Write integration tests for management APIs
  - [ ] Test API key CRUD endpoints
  - [ ] Test endpoint CRUD endpoints
  - [ ] Test metrics endpoints
  - [ ] Test client config endpoints
  - [ ] Test DLQ endpoints

- [ ] Write E2E tests for UI flows
  - [ ] Test login flow
  - [ ] Test API key creation
  - [ ] Test endpoint creation
  - [ ] Test config updates
  - [ ] Test DLQ replay

### Documentation

- [ ] Update [README.md](README.md)
  - [ ] Add UI features section
  - [ ] Add screenshots of UI
  - [ ] Add usage guide for server UI
  - [ ] Add usage guide for client UI
  - [ ] Update architecture diagram

- [ ] Update [QUICKSTART.md](QUICKSTART.md)
  - [ ] Add UI setup instructions
  - [ ] Add default credentials note
  - [ ] Add UI access URLs
  - [ ] Add troubleshooting section

- [ ] Create API documentation
  - [ ] Document all server endpoints
  - [ ] Document all client endpoints
  - [ ] Add request/response examples
  - [ ] Add authentication details

- [ ] Add security considerations
  - [ ] Document JWT secret management
  - [ ] Document HTTPS requirement for production
  - [ ] Document CORS configuration
  - [ ] Document rate limiting recommendations

- [ ] Update [.env.example](.env.example)
  - [ ] Add JWT_SECRET
  - [ ] Add ADMIN_USERNAME
  - [ ] Add ADMIN_PASSWORD
  - [ ] Add JWT_EXPIRATION

---

## Low Priority Tasks

### Enhancements

- [ ] Add webhook logs viewer to server UI
  - [ ] Create [web/server-ui/src/pages/Logs.tsx](web/server-ui/src/pages/Logs.tsx)
  - [ ] Add paginated table of webhook logs
  - [ ] Add filter by platform, status, date range
  - [ ] Add view detailed webhook payload
  - [ ] Add search functionality

- [ ] Add rate limiting
  - [ ] Implement rate limiting middleware
  - [ ] Add rate limit configuration
  - [ ] Add rate limit headers to responses

- [ ] Add audit logging
  - [ ] Log all API key operations
  - [ ] Log all endpoint operations
  - [ ] Log all config changes
  - [ ] Add audit log viewer

- [ ] Add user management (multi-user support)
  - [ ] Update User model with roles
  - [ ] Add user CRUD endpoints
  - [ ] Add user management UI
  - [ ] Add role-based access control

- [ ] Add webhook testing functionality
  - [ ] Add test webhook button in endpoint UI
  - [ ] Add webhook payload editor
  - [ ] Add test result display

- [ ] Add export/import configuration
  - [ ] Add export API keys button
  - [ ] Add export endpoints button
  - [ ] Add import configuration button
  - [ ] Support JSON format

- [ ] Add real-time notifications
  - [ ] Add WebSocket support
  - [ ] Notify on webhook failures
  - [ ] Notify on DLQ additions
  - [ ] Add notification center in UI

- [ ] Add dark mode support
  - [ ] Add theme toggle
  - [ ] Update all components for dark mode
  - [ ] Persist theme preference

---

## Technical Debt

- [ ] Upgrade Go version from 1.13.8 to latest
  - [ ] Update go.mod
  - [ ] Migrate from github.com/dgrijalva/jwt-go to golang-jwt/jwt/v5
  - [ ] Test all functionality after upgrade

- [ ] Add error handling improvements
  - [ ] Add more specific error messages
  - [ ] Add error logging
  - [ ] Add error recovery

- [ ] Add input validation
  - [ ] Validate API key names
  - [ ] Validate endpoint paths
  - [ ] Validate URLs
  - [ ] Validate retry configuration

- [ ] Add performance optimizations
  - [ ] Add caching for frequently accessed data
  - [ ] Optimize database queries
  - [ ] Add pagination for large datasets

- [ ] Add monitoring and alerting
  - [ ] Add Prometheus metrics
  - [ ] Add health check endpoints
  - [ ] Add alerting for failures

---

## Completed Tasks ✅

### Phase 1: Backend - Data Models & Storage
- [x] Add User, APIKey, WebhookEndpoint, JWTClaims models
- [x] Extend Redis storage with CRUD operations
- [x] Add default admin user initialization

### Phase 2: Backend - Authentication System
- [x] Implement JWT utilities
- [x] Add JWT middleware
- [x] Add auth endpoints

### Phase 3: Backend - Management APIs
- [x] Add API key management endpoints
- [x] Add webhook endpoint management endpoints
- [x] Add metrics and monitoring endpoints
- [x] Add client management endpoints

### Phase 4: Backend - Webhook Routing
- [x] Modify webhook handler for path-based routing
- [x] Update webhook data model with routing fields
- [x] Update client consumer to use routing metadata

### Phase 5: Backend - Static File Serving
- [x] Add static file serving to relay-server
- [x] Add static file serving to relay-client

### Phase 6: Frontend - Setup
- [x] Initialize React projects (server-ui, client-ui)
- [x] Set up Vite, dependencies, project structure

### Phase 7: Frontend - Server UI
- [x] Build authentication flow
- [x] Build dashboard with metrics
- [x] Build API key management page
- [x] Build webhook endpoint management page

### Phase 9: Build & Integration
- [x] Update build process in Makefile
- [x] Update Docker setup
- [x] Add environment variables

---

## Notes

- Client UI implementation should follow the same patterns as server UI for consistency
- Remember to run `make build-ui-client` after implementing client UI pages
- Consider adding tests after client UI is complete
- Update documentation after all features are complete
- Default credentials are logged to console on startup (check logs for admin password)

---

## Quick Reference

### Build Commands
```bash
make build              # Build everything
make build-ui-server    # Build server UI only
make build-ui-client    # Build client UI only
```

### Run Commands
```bash
./bin/relay-server      # Run server (UI at http://localhost:8080)
./bin/relay-client      # Run client (UI at http://localhost:8081)
```

### Docker Commands
```bash
docker-compose build    # Build Docker images
docker-compose up -d    # Start services
docker-compose logs -f  # View logs
```

### Test Commands
```bash
go test ./...           # Run all tests
go test -cover ./...    # Run tests with coverage
```
