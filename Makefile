# Makefile for Event Pipeline

.PHONY: help build run test clean docker-up docker-down docker-logs init-db

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all binaries
	@echo "Building consumer..."
	@go build -o bin/consumer ./cmd/consumer
	@echo "Building producer..."
	@go build -o bin/producer ./cmd/producer
	@echo "Build complete!"

run-consumer: ## Run consumer locally
	@go run cmd/consumer/main.go

run-producer: ## Run producer locally
	@go run cmd/producer/main.go

test: ## Run tests
	@go test -v ./internal/...

test-coverage: ## Run tests with coverage
	@go test -v -coverprofile=coverage.txt -covermode=atomic ./internal/...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -f coverage.txt coverage.html
	@echo "Cleaned!"

docker-build: ## Build Docker images
	@docker-compose build

docker-up: ## Start all services with Docker Compose
	@docker-compose up -d
	@echo "Services started! Check status with 'make docker-ps'"

docker-down: ## Stop all services
	@docker-compose down

docker-logs: ## Follow logs from all services
	@docker-compose logs -f

docker-ps: ## Show status of all services
	@docker-compose ps

docker-restart: ## Restart all services
	@docker-compose restart

init-db: ## Initialize database schema
	@docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d master -Q 'CREATE DATABASE IF NOT EXISTS eventdb'
	@docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -i /schema.sql
	@echo "Database initialized!"

kafka-topics: ## List Kafka topics
	@docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-consumer-groups: ## Show Kafka consumer groups
	@docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --list

kafka-reset-offsets: ## Reset Kafka consumer group offsets to earliest
	@docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --group event-consumer-group --reset-offsets --to-earliest --topic events --execute

redis-cli: ## Connect to Redis CLI
	@docker exec -it redis redis-cli

redis-dlq-count: ## Show DLQ count
	@docker exec -it redis redis-cli LLEN dlq:events

redis-dlq-view: ## View DLQ entries
	@docker exec -it redis redis-cli LRANGE dlq:events 0 -1

mssql-cli: ## Connect to MS SQL CLI
	@docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb

health: ## Check health of all endpoints
	@echo "Checking API health..."
	@curl -s http://localhost:8080/health | jq .
	@echo "\nChecking metrics..."
	@curl -s http://localhost:8080/metrics | grep -E "events_processed|dlq_entries"

metrics: ## View key metrics
	@curl -s http://localhost:8080/metrics | grep -E "events_processed_total|dlq_entries_total|db_operation_duration_seconds"

deps: ## Download Go dependencies
	@go mod download
	@go mod tidy

lint: ## Run linters (requires golangci-lint)
	@golangci-lint run

fmt: ## Format code
	@go fmt ./...

install-tools: ## Install development tools
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed!"
