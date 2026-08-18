.PHONY: help build run test clean lint docker-build docker-run

help:
	@echo "Available commands:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make lint          - Run linter"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"

build:
	@echo "Building application..."
	@go build -o app ./cmd

run: build
	@echo "Running application..."
	@./app

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -f app
	@go clean

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

docker-build:
	@echo "Building Docker image..."
	@docker build -t myapp:latest .

docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 myapp:latest

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...
