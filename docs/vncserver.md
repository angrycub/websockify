# VNC Server

Mock VNC server for testing websockify with animated framebuffer patterns.

## Overview

The VNC server is a test application that implements a basic VNC/RFB protocol server with animated framebuffer generation. It's designed specifically for testing websockify functionality and VNC client implementations.

## Features

- **RFB Protocol Support**: Implements RFB 3.8 protocol with proper handshake
- **Animated Patterns**: Multiple animated framebuffer patterns for visual testing
- **Pixel Format Negotiation**: Supports multiple pixel formats (8/16/24/32 bpp)
- **GUI Viewer**: Optional real-time framebuffer display window
- **Configurable Frame Rate**: Adjustable animation speed for testing

## Usage

```bash
bin/vncserver [OPTIONS]
```

### Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-animation` | `wheel` | Animation type: wheel, waves, plasma, orbits, gradient |
| `-fps` | `30` | Frame rate for GUI animation (frames per second) |
| `-gui` | `false` | Show server framebuffer in GUI window |
| `-help` | `false` | Show help message |
| `-port` | `5900` | Port to listen on |

### Animation Types

- **wheel**: Rotating color wheel pattern
- **waves**: Animated wave interference patterns
- **plasma**: Flowing plasma effect with color gradients
- **orbits**: Circular orbital motion patterns
- **gradient**: Animated color gradients

## Examples

### Basic Server

Start a VNC server on the default port (5900):

```bash
bin/vncserver
```

### Server with GUI Viewer

Start server with real-time framebuffer display:

```bash
bin/vncserver -gui
```

### Custom Animation and Port

Start server with plasma animation on port 5901:

```bash
bin/vncserver -port 5901 -animation plasma
```

### High Frame Rate Testing

Start server with high frame rate for performance testing:

```bash
bin/vncserver -gui -fps 60
```

## Testing with Websockify

### Basic Setup

1. Start the VNC server:
   ```bash
   bin/vncserver -port 5900
   ```

2. Start websockify proxy:
   ```bash
   bin/websockify -listen :8080 -target localhost:5900
   ```

3. Connect VNC client through websockify:
   ```bash
   bin/vncclient -host localhost:8080
   ```

### Visual Comparison Testing

Use GUI viewers on both server and client for side-by-side comparison:

```bash
# Terminal 1: Server with GUI
bin/vncserver -port 5900 -gui

# Terminal 2: Websockify proxy  
bin/websockify -listen :8080 -target localhost:5900

# Terminal 3: Client with GUI
bin/vncclient -host localhost:8080 -gui
```

## Protocol Implementation

### Handshake Sequence

1. **Version Negotiation**: Exchanges RFB version string
2. **Security Selection**: Supports "None" security type
3. **Client Initialization**: Receives client init message
4. **Server Initialization**: Sends screen dimensions and pixel format

### Message Handling

- **SetPixelFormat**: Updates client's requested pixel format
- **SetEncodings**: Acknowledges client encoding preferences
- **FramebufferUpdateRequest**: Responds with animated framebuffer data
- **Input Events**: Logs key and pointer events (no action taken)

### Pixel Format Support

- **8 bpp**: Color palette mode with reduced color depth
- **16 bpp**: RGB565 format for mobile/embedded testing
- **24 bpp**: True color without alpha channel
- **32 bpp**: Full BGRA format (default)

## Technical Details

### Screen Resolution

- **Width**: 800 pixels
- **Height**: 600 pixels
- **Default Format**: 32bpp BGRA little-endian

### Frame Generation

Animated frames are generated using mathematical functions:
- Real-time calculation based on frame number

- Smooth animation loops for continuous testing
- Color space utilization for visual verification

### GUI Integration

When `-gui` flag is enabled:

- Opens cross-platform window using Fyne framework
- Real-time framebuffer display at specified FPS
- Window title shows current animation type and frame rate

## Troubleshooting

### Connection Issues

- Verify port is not in use: `netstat -an | grep :5900`
- Check firewall settings for the specified port
- Ensure client connects to correct host:port combination

### Performance Issues

- Reduce frame rate for slower systems: `-fps 15`
- Disable GUI viewer if not needed
- Use simpler animations like "gradient" for basic testing

### Protocol Errors

- Check client RFB version compatibility (3.8 supported)
- Verify pixel format negotiation in server logs
- Monitor message framing and buffer handling

## Integration with Test Suite

The VNC server integrates with the broader test infrastructure:

- Works with `vncclient` for end-to-end testing
- Supports `websockify` proxy validation
- Provides visual patterns for manual verification
- Enables automated framebuffer capture testing

For complete testing workflows, see the main project documentation.
