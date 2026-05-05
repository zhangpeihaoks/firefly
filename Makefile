.PHONY: build run test lint clean docker

# Binary name
BINARY_NAME=firefly

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) main.go

# Run the application
run:
	$(GOCMD) run main.go -config ./config/config.yaml

# Run tests
test:
	$(GOTEST) ./... -race -coverprofile=coverage.out

# Run tests with verbose output
test-verbose:
	$(GOTEST) ./... -race -v

# Run linter
lint:
	golangci-lint run ./... -v

# Run linter with auto-fix
lint-fix:
	golangci-lint run ./... -v --fix

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build Docker image
docker:
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run:
	docker-compose up -d

# Stop Docker container
docker-stop:
	docker-compose down

# Generate Wire dependencies
wire:
	cd internal/app && wire

# Format code
fmt:
	$(GOCMD) fmt ./...

# Check code
check: fmt lint test
