.PHONY: help build run test clean docker-build docker-run lint fmt

# Variables
BINARY_NAME=broker
DOCKER_IMAGE=gitlab-harbor-token-broker
VERSION?=latest

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/broker

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	go run ./cmd/broker -config config.yaml

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml $(DOCKER_IMAGE):$(VERSION)

lint: ## Run linter
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

verify: fmt lint test ## Run all verification steps (format, lint, test)
	@echo "Verification complete!"
