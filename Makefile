.PHONY: build clean install run test fmt vet deps help

# Binary name
BINARY_NAME=websockify
CMD_DIR=./cmd/websockify

# Default target
all: build

# Build the binary
build:
	go build -o $(BINARY_NAME) $(CMD_DIR)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Install to $GOPATH/bin
install:
	go install $(CMD_DIR)

# Run the application (with default settings)
run: build
	./$(BINARY_NAME) -listen :8080 -target localhost:5900

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Run static analysis
vet:
	go vet ./...

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the websockify binary"
	@echo "  clean    - Remove build artifacts"
	@echo "  install  - Install binary to \$$GOPATH/bin"
	@echo "  run      - Build and run with default settings"
	@echo "  test     - Run tests"
	@echo "  fmt      - Format Go code"
	@echo "  vet      - Run go vet"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  help     - Show this help message"