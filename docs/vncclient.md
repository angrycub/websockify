# VNC Client

Basic VNC client for testing websockify with framebuffer capture and GUI display capabilities.

## Overview

The VNC client is a test application that implements RFB protocol client functionality with frame capture, animation generation, and real-time GUI display. It's designed for comprehensive testing of websockify and VNC server implementations.

## Features

- **RFB Protocol Client**: Full RFB 3.8 protocol implementation
- **Frame Capture**: Export framebuffer updates as PNG files
- **Animation Generation**: Create APNG and WebM animations from captures
- **GUI Viewer**: Real-time framebuffer display with transparency visualization
- **Pixel Format Testing**: Support for custom pixel format negotiation
- **Timeout Sessions**: Configurable test duration for automated testing

## Usage

```bash
bin/vncclient [OPTIONS]
```

### Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-apng` | `false` | Create APNG animation from captured frames |
| `-capture` | `false` | Capture framebuffer updates as PNG files |
| `-checkerboard` | `false` | Add checkerboard background for transparency visualization |
| `-duration` | `10` | Duration to run client in seconds |
| `-fps` | `2` | Frame rate for animations (frames per second) |
| `-gui` | `false` | Show framebuffer in GUI window |
| `-help` | `false` | Show help message |
| `-host` | `localhost:5900` | VNC server host:port |
| `-output` | `./test_output` | Output directory for captured frames |
| `-test-pixel-format` | `false` | Send test SetPixelFormat message (16bpp RGB565) |
| `-webm` | `false` | Create WebM video animation from captured frames |

## Examples

### Basic Connection

Connect to VNC server and display for 10 seconds:

```bash
bin/vncclient -host localhost:5900
```

### Frame Capture Testing

Capture frames during VNC session:

```bash
bin/vncclient -host localhost:5900 -capture -output ./test-frames -duration 15
```

### GUI Viewer with Transparency

Show framebuffer with checkerboard background:

```bash
bin/vncclient -host localhost:5900 -gui -checkerboard
```

### Animation Generation

Create both APNG and WebM animations:

```bash
bin/vncclient -host localhost:5900 -capture -webm -apng -fps 5 -duration 10
```

### Testing Through Websockify

Connect to VNC server through websockify proxy:

```bash
bin/vncclient -host localhost:8080 -duration 15
```

### Pixel Format Testing

Test custom pixel format negotiation:

```bash
bin/vncclient -host localhost:5900 -test-pixel-format -gui
```

## Testing Workflows

### Basic VNC Integration Test

1. **Start VNC server:**
   ```bash
   bin/vncserver -port 5900 -gui
   ```

2. **Start websockify proxy:**
   ```bash
   bin/websockify -listen :8080 -target localhost:5900
   ```

3. **Test with VNC client:**
   ```bash
   bin/vncclient -host localhost:8080 -gui
   ```

### Framebuffer Capture Testing

Capture and analyze frames during VNC session:

```bash
# Connect with frame capture
bin/vncclient -host localhost:5900 -capture -output ./test_output -duration 10

# Inspect captured frames
ls -la test_output/
# Output: frame_0001.png, frame_0002.png, etc.
```

### Side-by-Side Visual Comparison

Compare server and client framebuffers in real-time:

```bash
# Terminal 1: Server with GUI viewer
bin/vncserver -port 5900 -gui

# Terminal 2: Client with GUI viewer  
bin/vncclient -host localhost:5900 -gui
```

## Protocol Implementation

### Handshake Process

1. **Version Exchange**: Negotiates RFB protocol version
2. **Security Handling**: Supports "None" security type
3. **Client Initialization**: Sends shared desktop request
4. **Server Response**: Receives screen dimensions and pixel format

### Message Types Supported

- **FramebufferUpdate**: Processes Raw encoding framebuffer data
- **SetColorMapEntries**: Handles color palette updates
- **Bell**: Processes server bell notifications
- **ServerCutText**: Receives clipboard text from server

### Pixel Format Conversion

- **Server Format Detection**: Reads server's native pixel format
- **Format Conversion**: Converts to RGBA for display and capture
- **Endianness Support**: Handles both big-endian and little-endian formats
- **Bit Depth Support**: 8/16/24/32 bits per pixel

## Output Formats

### PNG Frame Capture

Individual frames saved as:
```text
test_output/
├── frame_0001.png
├── frame_0002.png
├── frame_0003.png
└── ...
```

### APNG Animation

Animated PNG with configurable frame rate:

- Lossless compression
- Supports transparency
- Wide browser compatibility
- Filename: `animation.apng`

### WebM Video

Compressed video format:

- Efficient file size
- High quality
- Web-optimized
- Filename: `animation.webm`

## GUI Features

### Real-time Display

- **Framebuffer Rendering**: Live VNC session display
- **Window Management**: Resizable window with scroll support
- **Performance**: Smooth rendering at configurable FPS

### Transparency Visualization

With `-checkerboard` option:

- Shows transparent areas with checkerboard pattern
- Helps identify alpha channel issues
- Useful for debugging pixel format problems

## Testing Integration

### Programmatic Access

The VNC client provides methods for automated testing:

```go
// Example: Access framebuffer data in tests
client := &VNCClient{}
// ... connect and receive updates ...
framebuffer := client.GetFramebuffer()
pixel := client.GetPixel(100, 100)  // Get pixel at coordinates
```

### Automated Testing

- **Duration Control**: Automatic session termination
- **Output Verification**: Programmatic frame analysis
- **Error Detection**: Connection and protocol error reporting
- **Performance Metrics**: Frame rate and latency measurement

## Troubleshooting

### Connection Issues

- **Host Resolution**: Verify hostname/IP address is correct
- **Port Accessibility**: Check if target port is open and accessible
- **Network Connectivity**: Test basic network connectivity to host

### Capture Problems

- **Directory Permissions**: Ensure write access to output directory
- **Disk Space**: Verify sufficient space for frame capture
- **File Conflicts**: Check for existing files in output directory

### GUI Display Issues

- **Graphics System**: Requires GUI environment (X11, Wayland, macOS, Windows)
- **OpenGL Support**: May require hardware acceleration for smooth rendering
- **Window Manager**: Some minimal environments may have compatibility issues

### Performance Issues

- **Frame Rate**: Reduce FPS for slower systems or networks
- **Capture Format**: PNG compression can be CPU intensive
- **Network Latency**: High latency affects real-time display smoothness

## Advanced Usage

### Custom Pixel Formats

Test different pixel formats for compatibility:

```bash
# Test 16bpp RGB565 format
bin/vncclient -host localhost:5900 -test-pixel-format -capture
```

### Long-running Tests

Extended testing sessions:

```bash
# Run for 5 minutes with periodic captures
bin/vncclient -host localhost:5900 -duration 300 -capture -fps 1
```

### Multi-format Output

Generate multiple output formats simultaneously:

```bash
bin/vncclient -host localhost:5900 -capture -webm -apng -checkerboard -gui -fps 3
```

This enables comprehensive testing and analysis of VNC protocol implementations across different scenarios and configurations.
