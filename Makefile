.PHONY: all build clean run test lint proto docker-build docker-up docker-down

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOBUILD := go build

# Build the applications
all: build

build: build-crawler build-analyzer build-notification

build-crawler:
	$(GOBUILD) -o $(GOBIN)/crawler ./cmd/crawler/main.go

build-analyzer:
	$(GOBUILD) -o $(GOBIN)/analyzer ./cmd/analyzer/main.go

build-notification:
	$(GOBUILD) -o $(GOBIN)/notification ./cmd/notification/main.go

# Clean builds
clean:
	rm -rf $(GOBIN)/*

# Run individual services
run-crawler:
	go run ./cmd/crawler/main.go

run-analyzer:
	go run ./cmd/analyzer/main.go

run-notification:
	go run ./cmd/notification/main.go

# Test the applications
test:
	go test -v ./...

# Lint the code
lint:
	golangci-lint run

# Generate code from proto files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/common/proto/*.proto

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Database migrations
db-migrate:
	go run ./scripts/migrate.go

# Initialize the database with sample data
db-seed:
	go run ./scripts/seed.go

# Help command
help:
	@echo "Available commands:"
	@echo "  make all            - Build all services"
	@echo "  make build          - Build all services"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run-crawler    - Run the crawler service"
	@echo "  make run-analyzer   - Run the analyzer service"
	@echo "  make run-notification - Run the notification service"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linter"
	@echo "  make proto          - Generate code from proto files"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make docker-up      - Start Docker containers"
	@echo "  make docker-down    - Stop Docker containers"
	@echo "  make docker-logs    - Show Docker container logs"
	@echo "  make db-migrate     - Run database migrations"
	@echo "  make db-seed        - Seed the database with sample data"