# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

Use the provided Makefile for common development tasks:

```bash
# Build the executable
make build

# Build and run with default settings
make run

# Install to $GOPATH/bin
make install

# Format code
make fmt

# Run static analysis
make vet

# Get dependencies
make deps

# Run tests
make test

# Clean build artifacts
make clean

# Show available make targets
make help
```

Direct Go commands are also available:

```bash
# Build the standalone executable
go build -o websockify ./cmd/websockify

# Run from source
go run ./cmd/websockify -listen :8080 -target localhost:5900

# Show help
./websockify -help
```

## Architecture

This is a Go library that implements a websockify server - a WebSocket to TCP proxy that allows web browsers to connect to TCP services (like VNC servers) through WebSocket connections.

### Core Components

- **Server struct**: Main server that handles HTTP requests and WebSocket upgrades
- **Config struct**: Configuration for listener address, target TCP address, and optional web root
- **WebSocket Handler**: Located at `/websockify` endpoint, upgrades HTTP connections to WebSocket
- **Bidirectional Forwarding**: Two goroutines handle data forwarding between WebSocket and TCP connections

### Key Architecture Details

- Uses Gorilla WebSocket library for WebSocket handling
- Implements bidirectional proxy with separate goroutines for each direction:
  - `forwardTCP()`: TCP → WebSocket (reads from TCP, writes to WebSocket as binary messages)
  - `forwardWeb()`: WebSocket → TCP (reads WebSocket messages, writes to TCP)
- Includes safety check to prevent serving static files from current working directory
- Supports optional static file serving from specified web root directory
- Uses graceful shutdown with context cancellation

### Module Information
- Module path: `github.com/coder/websockify`
- Go version: 1.24
- Primary dependency: `github.com/gorilla/websocket v1.5.3`