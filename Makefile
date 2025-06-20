.PHONY: build clean install run test fmt vet deps help build-servers run-echo run-vnc build-client run-client

# Binary names and directories
BIN_DIR=bin
BINARY_NAME=$(BIN_DIR)/websockify
CMD_DIR=./cmd/websockify
ECHO_BINARY=$(BIN_DIR)/echoserver
VNC_BINARY=$(BIN_DIR)/vncserver
VNC_CLIENT_BINARY=$(BIN_DIR)/vncclient

# Default target
all: build

# Build the binary
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY_NAME) $(CMD_DIR)

# Clean build artifacts
clean:
	go clean
	rm -rf $(BIN_DIR)
	rm -rf frames/ test-frames/

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

# Build test servers
build-servers:
	mkdir -p $(BIN_DIR)
	go build -o $(ECHO_BINARY) ./cmd/echoserver
	go build -o $(VNC_BINARY) ./cmd/vncserver

# Build VNC client
build-client:
	mkdir -p $(BIN_DIR)
	go build -o $(VNC_CLIENT_BINARY) ./cmd/vncclient

# Run echo server (for websockify testing)
run-echo: build-servers
	./$(ECHO_BINARY) -port 5901

# Run VNC server (for websockify testing)  
run-vnc: build-servers
	./$(VNC_BINARY) -port 5900

# Run VNC client (connect to VNC server)
run-client: build-client
	./$(VNC_CLIENT_BINARY) -host localhost:5900 -duration 5

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the websockify binary"
	@echo "  build-servers - Build test servers (echo and VNC)"
	@echo "  build-client  - Build VNC client"
	@echo "  clean         - Remove build artifacts and frame captures"
	@echo "  install       - Install binary to \$$GOPATH/bin"
	@echo "  run           - Build and run websockify with default settings"
	@echo "  run-echo      - Build and run echo server on port 5901"
	@echo "  run-vnc       - Build and run mock VNC server on port 5900"
	@echo "  run-client    - Build and run VNC client connecting to localhost:5900"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  help          - Show this help message"