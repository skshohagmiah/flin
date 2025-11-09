.PHONY: build build-cli build-server docker-build docker-run docker-stop test benchmark clean help

# Variables
BINARY_CLI=flin
BINARY_SERVER=kvserver
DOCKER_IMAGE=flin-kv
VERSION=1.0.0

# Build targets
build: build-cli build-server

build-cli:
	@echo "Building CLI..."
	@go build -o $(BINARY_CLI) ./cmd/flin

build-server:
	@echo "Building server..."
	@go build -o $(BINARY_SERVER) ./cmd/kvserver

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

docker-run:
	@echo "Starting Flin with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Flin containers..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

docker-cli:
	@docker-compose run --rm flin-cli ./flin $(ARGS)

docker-benchmark:
	@docker-compose run --rm flin-cli ./flin benchmark

# Test targets
test:
	@echo "Running tests..."
	@go test -v ./...

benchmark:
	@echo "Running benchmark..."
	@cd scripts && ./run_throughput_test.sh

# Clean targets
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_CLI) $(BINARY_SERVER)
	@rm -rf data/ tmp/
	@docker-compose down -v 2>/dev/null || true

# Help
help:
	@echo "Flin KV Store - Makefile commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build CLI and server"
	@echo "  make build-cli       - Build CLI only"
	@echo "  make build-server    - Build server only"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-run      - Start with Docker Compose"
	@echo "  make docker-stop     - Stop Docker containers"
	@echo "  make docker-logs     - View container logs"
	@echo "  make docker-cli      - Run CLI in Docker (use ARGS='set key value')"
	@echo "  make docker-benchmark - Run benchmark in Docker"
	@echo ""
	@echo "Test:"
	@echo "  make test            - Run Go tests"
	@echo "  make benchmark       - Run performance benchmark"
	@echo ""
	@echo "Clean:"
	@echo "  make clean           - Remove binaries and data"
	@echo ""
	@echo "Examples:"
	@echo "  make docker-cli ARGS='set mykey hello'"
	@echo "  make docker-cli ARGS='get mykey'"
	@echo "  make docker-cli ARGS='benchmark'"
