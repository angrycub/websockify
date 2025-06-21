# WebSockify Library Examples

This directory contains examples demonstrating how to use the websockify library in different scenarios.

## Examples

### [Basic Usage](./basic/)
The simplest possible usage - create a WebSocket to TCP proxy with minimal configuration.

**Features:**
- Basic websockify proxy setup
- Signal handling for graceful shutdown
- Default logging

**Run:**
```bash
cd examples/basic
go run main.go
```

### [Custom Logger](./custom-logger/)
Shows how to integrate a custom structured logger (using Go's `log/slog`).

**Features:**
- Custom logger implementation
- Structured JSON logging
- Logger adapter pattern

**Run:**
```bash
cd examples/custom-logger  
go run main.go
```

### [HTTP Integration](./http-integration/)
Demonstrates integrating websockify into an existing HTTP server with multiple endpoints.

**Features:**
- Mount websockify as an HTTP handler
- Multiple HTTP endpoints
- Health checks and status API
- Graceful shutdown with timeout
- Silent websockify logging (using NoOpLogger)

**Run:**
```bash
cd examples/http-integration
go run main.go
```

Visit:
- http://localhost:8080/ - Home page
- http://localhost:8080/health - Health check
- http://localhost:8080/api/status - Status API
- ws://localhost:8080/vnc - WebSocket endpoint

### [Silent Mode](./silent/)
Shows running websockify without any logging output.

**Features:**
- Silent operation with `NoOpLogger`
- Timeout-based execution
- Signal handling

**Run:**
```bash
cd examples/silent
go run main.go
```

## Testing the Examples

To test any of these examples, you'll need a TCP service running on the target port. You can use the included test servers:

### Option 1: Echo Server
```bash
# Terminal 1: Start echo server
make run-echo

# Terminal 2: Run example
cd examples/basic && go run main.go
```

### Option 2: Mock VNC Server  
```bash
# Terminal 1: Start VNC server
make run-vnc

# Terminal 2: Run example  
cd examples/basic && go run main.go
```

## Library API Overview

### Basic Configuration
```go
config := websockify.Config{
    Listener: ":8080",           // WebSocket listen address
    Target:   "localhost:5900",  // TCP target address  
    WebRoot:  "",                // Optional static files directory
    Logger:   nil,               // Optional custom logger
}

server := websockify.New(config)
```

### Starting the Server
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Blocks until context is cancelled
err := server.Serve(ctx)
```

### Using as HTTP Handler
```go
mux := http.NewServeMux()
mux.Handle("/websocket", server) // server implements http.Handler
```

### Custom Logger Interface
```go
type Logger interface {
    Printf(format string, v ...interface{})
    Println(v ...interface{})
}
```

The library provides `websockify.NoOpLogger{}` for silent operation.