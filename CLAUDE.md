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

## Integration Testing

The repository includes test servers for integration testing websockify:

### Test Servers

**Echo Server** (`cmd/echoserver`):
- Simple TCP server that echoes back all received data
- Useful for testing basic websockify functionality
- Default port: 5901

**Mock VNC Server** (`cmd/vncserver`):
- Implements basic VNC/RFB protocol handshake
- Sends animated framebuffer updates with multiple patterns (wheel, waves, plasma, orbits, gradient)
- Useful for testing websockify with VNC-like protocols
- Optional GUI viewer for real-time server framebuffer display (requires GUI environment)
- Default port: 5900

**VNC Client** (`cmd/vncclient`):
- Basic VNC client that connects to VNC servers (including through websockify)
- Captures framebuffer updates as Go `image.RGBA` objects
- Exports frames as PNG files for debugging
- Provides programmatic access to pixel data for integration testing
- Supports timeout-based testing sessions
- Optional GUI viewer for real-time framebuffer display (requires GUI environment)

### Testing Workflows

#### Basic VNC Integration Test

1. **Start VNC server:**
   ```bash
   # Terminal 1: Start mock VNC server with GUI viewer
   ./vncserver -port 5900 -gui
   ```

2. **Start websockify proxy:**
   ```bash
   # Terminal 2: Start websockify pointing to VNC server
   ./websockify -listen :8080 -target localhost:5900
   ```

3. **Test with VNC client:**
   ```bash
   # Terminal 3: Connect VNC client with GUI viewer for side-by-side comparison
   ./vncclient -host localhost:5900 -gui
   ```

This setup allows you to visually compare what the server is generating versus what the client is receiving to ensure coherence.

#### Framebuffer Capture Testing

1. **Capture frames during VNC session:**
   ```bash
   # Connect VNC client with frame capture enabled
   ./vncclient -host localhost:5900 -capture -output ./test_output -duration 10
   ```

2. **Inspect captured frames:**
   ```bash
   # View captured PNG files
   ls -la test_output/
   # frame_0001.png, frame_0002.png, etc.
   ```

#### Echo Server Testing

1. **Start echo server:**
   ```bash
   make run-echo
   ```

2. **Test through websockify:**
   ```bash
   ./websockify -listen :8080 -target localhost:5901
   # Connect via WebSocket to ws://localhost:8080/websockify
   ```

### Manual Testing Commands

```bash
# Build all components
make build-servers  # Build echo and VNC servers
make build-client   # Build VNC client

# Run servers on custom ports
./echoserver -port 5901
./vncserver -port 5900
./vncserver -port 5900 -gui                              # With GUI viewer
./vncserver -port 5900 -animation plasma -gui            # Different animation with GUI
./vncserver -port 5900 -gui -fps 60                      # High frame rate GUI
./vncserver -port 5900 -gui -fps 5                       # Low frame rate GUI

# Run VNC client with various options
./vncclient -host localhost:5900                                   # Basic connection
./vncclient -host localhost:5900 -capture -output ./test_output    # With frame capture
./vncclient -host localhost:5900 -gui                              # With GUI viewer
./vncclient -host localhost:5900 -gui -checkerboard               # GUI with transparency visualization
./vncclient -host localhost:8080 -duration 15                      # Through websockify

# Test websockify configurations
./websockify -listen :8080 -target localhost:5901  # Echo server
./websockify -listen :8080 -target localhost:5900  # VNC server
```

### Programmatic Testing

The VNC client provides methods for integration testing:

```go
// Example: Access framebuffer data in tests
client := &VNCClient{}
// ... connect and receive updates ...
framebuffer := client.GetFramebuffer()
pixel := client.GetPixel(100, 100)  // Get pixel at coordinates (100,100)
```

## Markdown Guidelines

- Common markdown formatting requires that headers, code-fences, and lists require one blank line of standoff above and below them. 
- All files MUST end in a blank line--you will have to run `echo "" >> FILENAME` where FILENAME is the file to which you need to add a final line feed. 