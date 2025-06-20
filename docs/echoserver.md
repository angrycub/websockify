# Echo Server

Simple TCP echo server for basic websockify proxy testing.

## Overview

The echo server is a lightweight TCP server that echoes back all received data. It's designed for fundamental websockify testing, protocol validation, and network connectivity verification.

## Features

- **Simple Protocol**: Echoes all received data back to the client
- **TCP Server**: Standard TCP socket implementation
- **Configurable Port**: Customizable listening port
- **Connection Logging**: Logs client connections and data transfers
- **Graceful Shutdown**: Handles interruption signals properly

## Usage

```bash
./echoserver [OPTIONS]
```

### Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-help` | `false` | Show help message |
| `-port` | `5901` | Port to listen on |

## Examples

### Basic Echo Server

Start echo server on default port (5901):

```bash
./echoserver
```

### Custom Port

Start echo server on port 8000:

```bash
./echoserver -port 8000
```

## Testing with Websockify

### Basic Echo Testing

Test websockify proxy functionality with simple echo protocol:

1. **Start echo server:**
   ```bash
   ./echoserver -port 5901
   ```

2. **Start websockify proxy:**
   ```bash
   ./websockify -listen :8080 -target localhost:5901
   ```

3. **Test with WebSocket client:**

   ```javascript
   // Browser console or Node.js
   const ws = new WebSocket('ws://localhost:8080/websockify');
   ws.onmessage = (event) => console.log('Echo:', event.data);
   ws.send('Hello, WebSocket!');
   ```

### Command Line Testing

Test with netcat or telnet:

```bash
# Direct connection to echo server
echo "test message" | nc localhost 5901

# Through websockify (requires WebSocket client)
# Use a WebSocket testing tool or browser
```

## Protocol Behavior

### Echo Mechanism

- **Input**: Any bytes received from client connection
- **Output**: Identical bytes sent back to the same client
- **Buffering**: No buffering - immediate echo response
- **Encoding**: Binary safe - handles any byte sequence

### Connection Handling

- **Concurrent Connections**: Supports multiple simultaneous clients
- **Connection Lifecycle**: Each client connection handled independently
- **Error Handling**: Graceful handling of client disconnections
- **Resource Cleanup**: Proper cleanup of connection resources

## Use Cases

### Websockify Validation

- **Basic Connectivity**: Verify websockify can establish TCP connections
- **Data Integrity**: Confirm data passes through proxy unchanged
- **Bidirectional Communication**: Test both send and receive paths
- **Connection Stability**: Validate proxy maintains stable connections

### Network Testing

- **Latency Measurement**: Round-trip time testing
- **Throughput Testing**: Data transfer rate validation
- **Connection Limits**: Test maximum concurrent connections
- **Error Scenarios**: Network interruption and recovery testing

### Development Testing

- **Quick Validation**: Fast verification of websockify functionality
- **Integration Testing**: Simple protocol for test automation
- **Debugging**: Isolate websockify issues from complex protocols
- **Load Testing**: Generate controlled network traffic

## Integration Examples

### Makefile Integration

The project Makefile includes echo server support:

```bash
# Build echo server
make build-servers

# Run echo server (if Makefile target exists)
make run-echo
```

### Automated Testing

Example test script:

```bash
#!/bin/bash
# Start echo server in background
./echoserver -port 5901 &
ECHO_PID=$!

# Start websockify proxy
./websockify -listen :8080 -target localhost:5901 &
PROXY_PID=$!

# Wait for services to start
sleep 2

# Run tests here...
# (WebSocket client tests)

# Cleanup
kill $ECHO_PID $PROXY_PID
```

## Logging and Monitoring

### Connection Logs

The echo server logs:

- Client connection events
- Data transfer volume
- Connection termination
- Error conditions

### Example Log Output

```text
2024/06/20 10:30:15 Echo server listening on port 5901
2024/06/20 10:30:20 New connection from 127.0.0.1:54321
2024/06/20 10:30:20 Echoed 13 bytes to 127.0.0.1:54321
2024/06/20 10:30:25 Connection closed: 127.0.0.1:54321
```

## Performance Characteristics

### Resource Usage

- **Memory**: Minimal memory footprint
- **CPU**: Low CPU usage - simple byte copying
- **Network**: Direct socket I/O without buffering
- **Scalability**: Limited by system socket limits

### Throughput

- **Latency**: Near-zero processing latency
- **Bandwidth**: Limited by network and system I/O capabilities
- **Concurrent Connections**: Handles multiple clients efficiently

## Troubleshooting

### Port Issues

- **Port in Use**: Change port or find conflicting process
- **Permission Denied**: Use port > 1024 for non-root execution
- **Firewall**: Ensure port is accessible through firewall

### Connection Problems

- **Client Connection Refused**: Verify server is running and port is correct
- **Data Not Echoed**: Check for network connectivity issues
- **Connection Drops**: Monitor for network stability problems

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `bind: address already in use` | Port conflict | Use different port or stop conflicting service |
| `permission denied` | Privileged port | Use port > 1024 or run as root |
| `connection refused` | Server not running | Start echo server first |

## Advanced Usage

### Multiple Instances

Run multiple echo servers for load testing:

```bash
./echoserver -port 5901 &
./echoserver -port 5902 &
./echoserver -port 5903 &
```

### Docker Integration

Example Dockerfile for containerized testing:

```dockerfile
FROM golang:alpine
COPY echoserver /usr/local/bin/
EXPOSE 5901
CMD ["echoserver", "-port", "5901"]
```

### Load Testing

Use with load testing tools:

```bash
# Example with 'ab' (Apache Bench) equivalent for TCP
# Custom TCP load testing script required
```

The echo server provides a simple but effective foundation for testing websockify proxy functionality and network communication patterns.
