.PHONY: help build build-all clean run-all stop logs test lint format proto migrate-up migrate-down

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build-http: ## Build HTTP server
	go build -o bin/http-server ./cmd/http-server

build-grpc: ## Build gRPC server
	go build -o bin/grpc-server ./cmd/grpc-server

build-consumer: ## Build NATS consumer
	go build -o bin/consumer ./cmd/consumer

build-cron: ## Build cron server
	go build -o bin/cron-server ./cmd/cron-server

build-migrate: ## Build migration tool
	go build -o bin/migrate ./cmd/migrate

build-all: build-http build-grpc build-consumer build-cron build-migrate ## Build all services

build: build-all ## Alias for build-all

# Docker targets
docker-build-http: ## Build HTTP server Docker image
	docker build -f Dockerfile.http -t activity-log-http:latest .

docker-build-grpc: ## Build gRPC server Docker image
	docker build -f Dockerfile.grpc -t activity-log-grpc:latest .

docker-build-consumer: ## Build consumer Docker image
	docker build -f Dockerfile.consumer -t activity-log-consumer:latest .

docker-build-cron: ## Build cron server Docker image
	docker build -f Dockerfile.cron -t activity-log-cron:latest .

docker-build-all: docker-build-http docker-build-grpc docker-build-consumer docker-build-cron ## Build all Docker images

# Docker Compose targets
up: ## Start all services with Docker Compose
	docker-compose up -d

down: ## Stop all services
	docker-compose down

up-build: ## Build and start all services
	docker-compose up -d --build

logs: ## Show logs from all services
	docker-compose logs -f

logs-http: ## Show HTTP server logs
	docker-compose logs -f activity-log-http

logs-grpc: ## Show gRPC server logs
	docker-compose logs -f activity-log-grpc

logs-consumer: ## Show consumer logs
	docker-compose logs -f activity-log-consumer

logs-cron: ## Show cron server logs
	docker-compose logs -f activity-log-cron

# Development targets
run-http: ## Run HTTP server locally
	CONFIG_PATH=configs/config.yaml go run ./cmd/http-server

run-grpc: ## Run gRPC server locally
	CONFIG_PATH=configs/config.yaml go run ./cmd/grpc-server

run-consumer: ## Run NATS consumer locally
	CONFIG_PATH=configs/config.yaml go run ./cmd/consumer

run-cron: ## Run cron server locally
	CONFIG_PATH=configs/config.yaml go run ./cmd/cron-server

# Testing targets
test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit: ## Run unit tests only
	go test -v -short ./...

test-integration: ## Run integration tests
	go test -v -run Integration ./...

# Code quality targets
lint: ## Run linter
	golangci-lint run

format: ## Format code
	gofmt -s -w .
	goimports -w .

# Protocol Buffers
proto: ## Generate protobuf files
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/activity_log.proto

# Database migrations
migrate-up: ## Run database migrations
	go run ./cmd/migrate -command=up -config=configs/config.yaml

migrate-down: ## Rollback database migrations
	go run ./cmd/migrate -command=down -config=configs/config.yaml -version=0

migrate-status: ## Check migration status
	go run ./cmd/migrate -command=status -config=configs/config.yaml

# Cleanup
clean: ## Clean build artifacts
	rm -rf bin/
	docker-compose down --rmi all --volumes --remove-orphans

clean-volumes: ## Clean Docker volumes
	docker-compose down --volumes

# Health checks
health: ## Check service health
	@echo "Checking HTTP server..."
	@curl -f http://localhost:8080/health || echo "HTTP server not healthy"
	@echo "Checking gRPC server..."
	@grpcurl -plaintext localhost:9000 grpc.health.v1.Health/Check || echo "gRPC server not healthy"

# Service management
restart-http: ## Restart HTTP server
	docker-compose restart activity-log-http

restart-grpc: ## Restart gRPC server
	docker-compose restart activity-log-grpc

restart-consumer: ## Restart consumer
	docker-compose restart activity-log-consumer

restart-cron: ## Restart cron server
	docker-compose restart activity-log-cron

scale-http: ## Scale HTTP server (usage: make scale-http REPLICAS=3)
	docker-compose up -d --scale activity-log-http=${REPLICAS:-2}

# Documentation
docs: ## Generate Swagger documentation
	swag init -g ./cmd/http-server/main.go -o ./docs

# Environment setup
setup: ## Setup development environment
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Quick start
dev: ## Quick development setup
	docker-compose up -d arangodb nats redis mailhog jaeger prometheus grafana
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services are ready! You can now run individual services locally."

# Production deployment
deploy: ## Deploy to production (placeholder)
	@echo "Deployment target - implement based on your infrastructure"
	@echo "Example: kubectl apply -f k8s/ or docker stack deploy -c docker-stack.yml activity-log"