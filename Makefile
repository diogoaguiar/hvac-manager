# Makefile for HVAC Manager
# Run 'make help' to see available commands

# Binary output
BINARY_NAME=hvac-manager
BIN_DIR=bin

# Database settings
DB_FILE=hvac.db

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

.PHONY: help build test run demo clean fmt vet check coverage db-init db-reset db-load db-status

# Default target - show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build      - Build the application binary"
	@echo "  make run        - Run the application directly"
	@echo "  make demo       - Run the database demo"
	@echo ""
	@echo "Testing:"
	@echo "  make test       - Run all tests"
	@echo "  make coverage   - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt        - Format all Go code"
	@echo "  make vet        - Run go vet for static analysis"
	@echo "  make check      - Run fmt, vet, and test"
	@echo ""
	@echo "Database Management:"
	@echo "  make db-init    - Initialize database schema"
	@echo "  make db-reset   - Reset database (delete and reinitialize)"
	@echo "  make db-load    - Load IR codes from SmartIR files"
	@echo "  make db-status  - Show database status"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean      - Remove build artifacts and database"

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

# Run the database demo
demo:
	@echo "Running IR code database demo..."
	$(GOCMD) run ./cmd/demo

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
	rm -f $(DB_FILE)
	@echo "Clean complete!"

# Database management commands

# Initialize database schema
db-init:
	@echo "Initializing database schema..."
	@$(GOCMD) run -tags dbtools ./tools/db init $(DB_FILE)

# Reset database (delete and reinitialize)
db-reset:
	@echo "Resetting database..."
	@rm -f $(DB_FILE)
	@echo "Database removed."
	@$(GOCMD) run -tags dbtools ./tools/db init $(DB_FILE)

# Load IR codes from SmartIR files
db-load:
	@echo "Loading IR codes from SmartIR files..."
	@$(GOCMD) run -tags dbtools ./tools/db load $(DB_FILE) docs/smartir/reference

# Show database status
db-status:
	@echo "Database status:"
	@if [ -f $(DB_FILE) ]; then \
		$(GOCMD) run -tags dbtools ./tools/db status $(DB_FILE); \
	else \
		echo "Database file not found: $(DB_FILE)"; \
		echo "Run 'make db-init' to create it."; \
	fi
