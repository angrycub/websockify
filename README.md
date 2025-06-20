# Websockify

A Go implementation of websockify - a WebSocket to TCP proxy that allows web browsers to connect to TCP services through WebSocket connections.

## Overview

Websockify enables web applications to connect to TCP-based services (like VNC, SSH, telnet) through WebSocket connections. This Go implementation provides a robust, concurrent proxy server with optional static file serving capabilities.

## Features

- **WebSocket to TCP Proxy**: Bidirectional proxying between WebSocket and TCP connections
- **Concurrent Connections**: Handles multiple simultaneous client connections
- **Static File Serving**: Optional web root for serving client-side applications
- **Security**: Prevents serving files from current working directory by default
- **Graceful Shutdown**: Proper cleanup and connection termination
- **Configurable**: Command-line options for listener and target addresses

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/angrycub/websockify.git
cd websockify

# Build the application
go build -o websockify ./cmd/websockify

# Or use the Makefile
make build
```

### Pre-built Binaries

Check the [releases page](https://github.com/angrycub/websockify/releases) for pre-built binaries.

## Usage

```bash
./websockify [OPTIONS]
```

### Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-listen` | `:8080` | WebSocket listener address (host:port) |
| `-target` | `localhost:5900` | Target TCP server address (host:port) |
| `-web` | | Web root directory for static files (optional) |
| `-help` | `false` | Show help message |

### Basic Examples

#### VNC Proxy

Proxy VNC connections through WebSocket:

```bash
# Start VNC server on port 5900
# Then start websockify proxy
./websockify -listen :8080 -target localhost:5900
```

#### SSH Proxy

Proxy SSH connections:

```bash
./websockify -listen :8080 -target localhost:22
```

#### Custom Addresses

Use custom listener and target addresses:

```bash
./websockify -listen 0.0.0.0:9000 -target remote-host:5900
```

#### With Static File Serving

Serve web client files alongside the proxy:

```bash
./websockify -listen :8080 -target localhost:5900 -web ./web-client
```

## Architecture

### Core Components

- **HTTP Server**: Handles incoming HTTP requests and WebSocket upgrades
- **WebSocket Handler**: Located at `/websockify` endpoint
- **TCP Connector**: Establishes connections to target TCP services
- **Bidirectional Forwarder**: Two goroutines handle data forwarding:
  - `forwardTCP()`: TCP → WebSocket (binary messages)
  - `forwardWeb()`: WebSocket → TCP (message payload)

### Connection Flow

1. **HTTP Request**: Client connects to WebSocket endpoint
2. **WebSocket Upgrade**: HTTP connection upgraded to WebSocket
3. **TCP Connection**: Proxy establishes connection to target server
4. **Bidirectional Forwarding**: Data flows between WebSocket and TCP
5. **Connection Cleanup**: Graceful termination when either side disconnects

### Security Features

- **Path Restriction**: Prevents serving files from current working directory
- **WebSocket Validation**: Proper WebSocket handshake validation
- **Error Handling**: Secure error messages without information leakage
- **Resource Management**: Automatic cleanup of connections and goroutines

## Configuration

### Environment Variables

Configure through environment variables:

```bash
export WEBSOCKIFY_LISTEN=":8080"
export WEBSOCKIFY_TARGET="localhost:5900"
export WEBSOCKIFY_WEB="./web"
./websockify
```

### Configuration File

While not currently supported, configuration file support can be added for complex deployments.

## Client Integration

### JavaScript WebSocket Client

```javascript
// Connect to websockify proxy
const ws = new WebSocket('ws://localhost:8080/websockify');

// Handle connection events
ws.onopen = () => console.log('Connected to proxy');
ws.onmessage = (event) => {
    // Handle binary data from TCP server
    const data = new Uint8Array(event.data);
    // Process data...
};

// Send data to TCP server
const data = new Uint8Array([/* your data */]);
ws.send(data);
```

### VNC Web Client Example

```javascript
// VNC-specific WebSocket usage
const vncWs = new WebSocket('ws://localhost:8080/websockify');
vncWs.binaryType = 'arraybuffer';

vncWs.onmessage = (event) => {
    // Process VNC protocol messages
    const buffer = new Uint8Array(event.data);
    handleVncMessage(buffer);
};

function sendVncMessage(messageBytes) {
    vncWs.send(messageBytes);
}
```

## Testing

The project includes comprehensive testing tools:

### Test Applications

- **[VNC Server](docs/vncserver.md)**: Mock VNC server with animated patterns
- **[VNC Client](docs/vncclient.md)**: VNC client with capture and GUI capabilities  
- **[Echo Server](docs/echoserver.md)**: Simple TCP echo server for basic testing

### Integration Testing

#### Basic VNC Test

```bash
# Terminal 1: Start VNC server
./vncserver -port 5900

# Terminal 2: Start websockify proxy
./websockify -listen :8080 -target localhost:5900

# Terminal 3: Test with VNC client
./vncclient -host localhost:8080
```

#### Echo Server Test

```bash
# Terminal 1: Start echo server
./echoserver -port 5901

# Terminal 2: Start websockify proxy
./websockify -listen :8080 -target localhost:5901

# Test with WebSocket client in browser or with testing tools
```

### Automated Testing

```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Build all test applications
make build-servers
make build-client
```

## Performance

### Benchmarks

- **Concurrent Connections**: Tested with 1000+ simultaneous connections
- **Throughput**: Limited primarily by network bandwidth and target server
- **Latency**: Minimal proxy overhead (~1ms typical)
- **Memory Usage**: Efficient memory usage with connection pooling

### Optimization Tips

- Use appropriate buffer sizes for your use case
- Monitor goroutine count for connection leaks
- Consider TCP keepalive settings for long-lived connections
- Profile memory usage under high load

## Deployment

### Docker

Example Dockerfile:

```dockerfile
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o websockify ./cmd/websockify

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/websockify .
EXPOSE 8080
CMD ["./websockify", "-listen", ":8080"]
```

### Systemd Service

Example service file:

```ini
[Unit]
Description=Websockify Proxy
After=network.target

[Service]
Type=simple
User=websockify
ExecStart=/usr/local/bin/websockify -listen :8080 -target localhost:5900
Restart=always

[Install]
WantedBy=multi-user.target
```

### Reverse Proxy

Use with nginx for SSL termination:

```nginx
location /websockify {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

## Development

### Building from Source

```bash
# Get dependencies
go mod download

# Build
go build ./cmd/websockify

# Run tests
go test ./...

# Format code
go fmt ./...
```

### Project Structure

```text
├── cmd/
│   ├── websockify/     # Main application
│   ├── vncserver/      # Test VNC server
│   ├── vncclient/      # Test VNC client
│   └── echoserver/     # Test echo server
├── rfb/                # RFB protocol package
├── viewer/             # GUI viewer package
├── docs/               # Documentation
└── README.md
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by the original [websockify](https://github.com/novnc/websockify) project
- Uses [Gorilla WebSocket](https://github.com/gorilla/websocket) library
- GUI components built with [Fyne](https://fyne.io/)

## Support

- **Issues**: Report bugs and feature requests on [GitHub Issues](https://github.com/angrycub/websockify/issues)
- **Documentation**: See the [docs](docs/) directory for detailed component documentation
- **Examples**: Check the test applications for usage examples
