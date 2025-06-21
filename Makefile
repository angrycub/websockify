.PHONY: build clean install run test fmt vet deps help build-servers run-echo run-vnc build-client run-client build-examples

# Binary names and directories
BIN_DIR=bin
BINARY_NAME=$(BIN_DIR)/websockify
CMD_DIR=./cmd/websockify
ECHO_BINARY=$(BIN_DIR)/echoserver
VNC_BINARY=$(BIN_DIR)/vncserver
VNC_CLIENT_BINARY=$(BIN_DIR)/vncclient

# Version information
VERSION := $(shell ./scripts/version.sh)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags for version injection
LDFLAGS := -X 'github.com/coder/websockify/version.tag=$(VERSION)' \
           -X 'github.com/coder/websockify/version.commit=$(COMMIT)' \
           -X 'github.com/coder/websockify/version.date=$(DATE)'

# Default target
all: build

# Build the binary
build:
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)

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

# Build test servers (without GUI support by default)
build-servers:
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(ECHO_BINARY) ./cmd/echoserver
	go build -ldflags="$(LDFLAGS)" -o $(VNC_BINARY) ./cmd/vncserver

# Build test servers with GUI support (requires GUI dependencies)
build-servers-gui:
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(ECHO_BINARY) ./cmd/echoserver
	CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" go build -tags=gui -ldflags="$(LDFLAGS)" -o $(VNC_BINARY) ./cmd/vncserver

# Build VNC client (without GUI support by default)
build-client:
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(VNC_CLIENT_BINARY) ./cmd/vncclient

# Build VNC client with GUI support (requires GUI dependencies)
build-client-gui:
	mkdir -p $(BIN_DIR)
	CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" go build -tags=gui -ldflags="$(LDFLAGS)" -o $(VNC_CLIENT_BINARY) ./cmd/vncclient

# Run echo server (for websockify testing)
run-echo: build-servers
	./$(ECHO_BINARY) -port 5901

# Run VNC server (for websockify testing)  
run-vnc: build-servers
	./$(VNC_BINARY) -port 5900

# Run VNC client (connect to VNC server)
run-client: build-client
	./$(VNC_CLIENT_BINARY) -host localhost:5900 -duration 5

# Build all examples
build-examples:
	@echo "Building examples..."
	@cd examples/basic && go mod tidy && go build -o basic main.go
	@cd examples/custom-logger && go mod tidy && go build -o custom-logger main.go
	@cd examples/http-integration && go mod tidy && go build -o http-integration main.go
	@cd examples/silent && go mod tidy && go build -o silent main.go
	@echo "All examples built successfully"

# Show help
help:
	@echo "Available targets:"
	@echo "  build             - Build the websockify binary"
	@echo "  build-servers     - Build test servers (echo and VNC) without GUI"
	@echo "  build-servers-gui - Build test servers with GUI support"
	@echo "  build-client      - Build VNC client without GUI"
	@echo "  build-client-gui  - Build VNC client with GUI support"
	@echo "  build-examples    - Build all library usage examples"
	@echo "  clean             - Remove build artifacts and frame captures"
	@echo "  install           - Install binary to \$$GOPATH/bin"
	@echo "  run               - Build and run websockify with default settings"
	@echo "  run-echo          - Build and run echo server on port 5901"
	@echo "  run-vnc           - Build and run mock VNC server on port 5900"
	@echo "  run-client        - Build and run VNC client connecting to localhost:5900"
	@echo "  test              - Run tests"
	@echo "  fmt               - Format Go code"
	@echo "  vet               - Run go vet"
	@echo "  deps              - Download and tidy dependencies"
	@echo "  help              - Show this help message"
	@echo ""
	@echo "Version information:"
	@echo "  Current version: $(VERSION)"
	@echo "  All binaries support -version flag to show version info"
	@echo ""
	@echo "Build modes:"
	@echo "  Default: Lean builds without GUI dependencies (only gorilla/websocket)"
	@echo "  GUI:     Use -gui targets to enable GUI features (adds fyne.io dependencies)"
	@echo ""
	@echo "Examples:"
	@echo "  See examples/ directory for library usage examples"
	@echo "  Run 'make build-examples' to build all examples"