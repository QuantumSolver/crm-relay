.PHONY: build build-server build-client build-ui-server build-ui-client build-multiarch test test-coverage clean docker-build docker-up docker-down docker-logs run-server run-client

# Build targets
build: build-server build-client build-ui-server build-ui-client

build-server:
	@echo "Building relay server..."
	@export PATH=/snap/go/current/bin:$PATH && go build -o bin/relay-server ./cmd/relay-server

build-client:
	@echo "Building relay client..."
	@export PATH=/snap/go/current/bin:$PATH && go build -o bin/relay-client ./cmd/relay-client

build-ui-server:
	@echo "Building server UI..."
	@cd web/server-ui && npm install && npm run build

build-ui-client:
	@echo "Building client UI..."
	@cd web/client-ui && npm install && npm run build

# Multi-architecture build targets
build-multiarch: build-server-amd64 build-server-arm64 build-server-armv7 build-client-amd64 build-client-arm64 build-client-armv7

build-server-amd64:
	@echo "Building relay server for linux/amd64..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=amd64 go build -o bin/relay-server-linux-amd64 ./cmd/relay-server

build-server-arm64:
	@echo "Building relay server for linux/arm64..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=arm64 go build -o bin/relay-server-linux-arm64 ./cmd/relay-server

build-server-armv7:
	@echo "Building relay server for linux/arm/v7..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=arm GOARM=7 go build -o bin/relay-server-linux-armv7 ./cmd/relay-server

build-client-amd64:
	@echo "Building relay client for linux/amd64..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=amd64 go build -o bin/relay-client-linux-amd64 ./cmd/relay-client

build-client-arm64:
	@echo "Building relay client for linux/arm64..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=arm64 go build -o bin/relay-client-linux-arm64 ./cmd/relay-client

build-client-armv7:
	@echo "Building relay client for linux/arm/v7..."
	@export PATH=/snap/go/current/bin:$PATH && GOOS=linux GOARCH=arm GOARM=7 go build -o bin/relay-client-linux-armv7 ./cmd/relay-client

# Test targets
test:
	@echo "Running tests..."
	@export PATH=/snap/go/current/bin:$PATH && go test ./...

test-coverage:
	@echo "Running tests with coverage..."
	@export PATH=/snap/go/current/bin:$PATH && go test -cover ./...

# Clean targets
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@export PATH=/snap/go/current/bin:$PATH && go clean

# Docker targets
docker-build:
	@echo "Building Docker images..."
	@docker compose build

docker-up:
	@echo "Starting services..."
	@docker compose up -d

docker-down:
	@echo "Stopping services..."
	@docker compose down

docker-logs:
	@echo "Showing logs..."
	@docker compose logs -f

# Server Docker targets
docker-server-up:
	@echo "Starting server services..."
	@docker compose -f docker-compose.server.yml up -d

docker-server-down:
	@echo "Stopping server services..."
	@docker compose -f docker-compose.server.yml down

docker-server-logs:
	@echo "Showing server logs..."
	@docker compose -f docker-compose.server.yml logs -f

# Client Docker targets
docker-client-up:
	@echo "Starting client services..."
	@docker compose -f docker-compose.client.yml up -d

docker-client-down:
	@echo "Stopping client services..."
	@docker compose -f docker-compose.client.yml down

docker-client-logs:
	@echo "Showing client logs..."
	@docker compose -f docker-compose.client.yml logs -f

# Run targets
run-server:
	@echo "Running relay server..."
	@export PATH=/snap/go/current/bin:$PATH && go run cmd/relay-server/main.go

run-client:
	@echo "Running relay client..."
	@export PATH=/snap/go/current/bin:$PATH && go run cmd/relay-client/main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@export PATH=/snap/go/current/bin:$PATH && go mod download
	@export PATH=/snap/go/current/bin:$PATH && go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@export PATH=/snap/go/current/bin:$PATH && go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@export PATH=/snap/go/current/bin:$PATH && go vet ./...
