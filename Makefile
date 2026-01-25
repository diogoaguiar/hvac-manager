# Makefile for HVAC Manager
# Run 'make help' to see available commands

# Binary output
BINARY_NAME=hvac-manager
BIN_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

.PHONY: help build test run clean fmt vet check coverage

# Default target - show help
help:
	@echo "Available targets:"
	@echo "  make build      - Build the application binary"
	@echo "  make test       - Run all tests"
	@echo "  make run        - Run the application directly"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make fmt        - Format all Go code"
	@echo "  make vet        - Run go vet for static analysis"
	@echo "  make check      - Run fmt, vet, and test"
	@echo "  make coverage   - Run tests with coverage report"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the application directly (without building binary)
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run ./cmd

# Format all Go code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet for static analysis
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run all checks (format, vet, test)
check: fmt vet test
	@echo "All checks passed!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete!"
