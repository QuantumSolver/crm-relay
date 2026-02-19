.PHONY: build build-server build-client test test-coverage clean docker-build docker-up docker-down docker-logs run-server run-client

# Build targets
build: build-server build-client

build-server:
	@echo "Building relay server..."
	@export PATH=/snap/go/current/bin:$PATH && go build -o bin/relay-server ./cmd/relay-server

build-client:
	@echo "Building relay client..."
	@export PATH=/snap/go/current/bin:$PATH && go build -o bin/relay-client ./cmd/relay-client

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
	@docker-compose build

docker-up:
	@echo "Starting services..."
	@docker-compose up -d

docker-down:
	@echo "Stopping services..."
	@docker-compose down

docker-logs:
	@echo "Showing logs..."
	@docker-compose logs -f

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
