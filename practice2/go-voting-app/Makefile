.PHONY: help build run stop clean test deps docker-build docker-up docker-down logs

help:
	@echo "Voting System - Available targets:"
	@echo ""
	@echo "Local Development:"
	@echo "  make deps          - Download dependencies"
	@echo "  make build         - Build all services"
	@echo "  make run-gateway   - Run Gateway Service"
	@echo "  make run-pm        - Run Poll Manager Service"
	@echo "  make run-af        - Run Anti-Fraud Service"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build Docker images"
	@echo "  make docker-up     - Start all services with Docker Compose"
	@echo "  make docker-down   - Stop all services"
	@echo "  make logs          - Show service logs"
	@echo ""
	@echo "Development:"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"

deps:
	go mod download
	go mod tidy

build: deps
	CGO_ENABLED=0 GOOS=linux go build -o build/gateway ./cmd/gateway
	CGO_ENABLED=0 GOOS=linux go build -o build/poll-manager ./cmd/poll-manager
	CGO_ENABLED=0 GOOS=linux go build -o build/anti-fraud ./cmd/anti-fraud

run-gateway: deps
	@echo "Starting Gateway Service..."
	go run ./cmd/gateway/main.go

run-pm: deps
	@echo "Starting Poll Manager Service..."
	go run ./cmd/poll-manager/main.go

run-af: deps
	@echo "Starting Anti-Fraud Service..."
	go run ./cmd/anti-fraud/main.go

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-rebuild: docker-down docker-build docker-up

logs:
	docker-compose logs -f

logs-gateway:
	docker-compose logs -f gateway

logs-pm:
	docker-compose logs -f poll-manager

logs-af:
	docker-compose logs -f anti-fraud

clean:
	rm -rf build/
	go clean

test:
	go test ./... -v -cover

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

# API Endpoints for testing
api-health:
	@echo "Gateway Health:"
	curl -X GET http://localhost:8001/health | jq

api-create-poll:
	@echo "Creating a poll..."
	curl -X POST http://localhost:8001/polls \
	  -H "Content-Type: application/json" \
	  -d '{"title":"Favorite Language?","options":["Go","Python","JavaScript","Rust"]}' | jq

api-list-polls:
	@echo "Listing polls..."
	curl -X GET http://localhost:8001/polls | jq

# Example vote endpoint (requires POLL_ID)
api-vote:
	@echo "Voting (requires POLL_ID and OPTION_ID)..."
	@echo "Usage: make api-vote POLL_ID=<id> OPTION_ID=<id>"
	curl -X POST http://localhost:8001/polls/$(POLL_ID)/vote \
	  -H "Content-Type: application/json" \
	  -d '{"option_id":"$(OPTION_ID)"}' | jq

api-results:
	@echo "Getting results (requires POLL_ID)..."
	@echo "Usage: make api-results POLL_ID=<id>"
	curl -X GET http://localhost:8001/polls/$(POLL_ID)/results | jq

# Development with hot reload (requires entr or similar)
watch:
	find . -name "*.go" | entr make run-gateway
