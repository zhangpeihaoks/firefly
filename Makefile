# =============================================================================
# Firefly Makefile
# =============================================================================
.PHONY: help build run test test-short test-race test-verbose lint lint-fix \
        clean deps tidy fmt vet check docker docker-run docker-stop \
        coverage coverage-html security audit

# Binary name
BINARY_NAME=firefly

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags="-s -w"

## help: Show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | sort | awk -F ': ' '{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) main.go

## run: Run the application
run:
	$(GOCMD) run main.go -config ./config/config.yaml

## test: Run all tests with race detection and coverage
test:
	$(GOTEST) ./... -race -coverprofile=coverage.out -count=1

## test-short: Run short tests only (no property-based tests)
test-short:
	$(GOTEST) ./... -short -count=1

## test-race: Run tests with race detection (no coverage)
test-race:
	$(GOTEST) ./... -race -count=1

## test-verbose: Run all tests with verbose output
test-verbose:
	$(GOTEST) ./... -race -v -count=1

## lint: Run golangci-lint
lint:
	golangci-lint run ./... -v

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	golangci-lint run ./... -v --fix

## vet: Run go vet
vet:
	$(GOVET) ./...

## fmt: Format code with gofmt
fmt:
	$(GOCMD) fmt ./...

## tidy: Tidy Go module dependencies
tidy:
	$(GOMOD) tidy

## deps: Download and tidy dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## coverage: Show test coverage report
coverage:
	$(GOTEST) ./... -coverprofile=coverage.out -count=1
	$(GOCMD) tool cover -func=coverage.out

## coverage-html: Open coverage report in browser
coverage-html:
	$(GOTEST) ./... -coverprofile=coverage.out -count=1
	$(GOCMD) tool cover -html=coverage.out

## security: Run security checks (govulncheck)
security:
	@govulncheck ./... 2>/dev/null || echo "Install govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest"

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

## check: Run fmt, vet, lint and tests
check: fmt vet lint test

## audit: Full audit - tidy, fmt, vet, lint, test, security
audit: tidy fmt vet lint test security

## docker: Build Docker image
docker:
	docker build -t $(BINARY_NAME):latest .

## docker-run: Run Docker containers
docker-run:
	docker-compose up -d

## docker-stop: Stop Docker containers
docker-stop:
	docker-compose down
