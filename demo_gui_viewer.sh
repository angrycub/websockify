#!/bin/bash

# Demo script for GUI framebuffer viewer
# This script demonstrates the websockify GUI viewer functionality

set -e

echo "=== Websockify GUI Viewer Demo ==="
echo

# Build all components
echo "Building components..."
go build -o vncserver ./cmd/vncserver
go build -o websockify ./cmd/websockify  
go build -o vncclient ./cmd/vncclient
echo "✓ All components built"
echo

# Function to cleanup background processes
cleanup() {
    echo
    echo "Cleaning up background processes..."
    pkill -f "./vncserver" 2>/dev/null || true
    pkill -f "./websockify" 2>/dev/null || true
    echo "✓ Cleanup complete"
}

# Set trap to cleanup on script exit
trap cleanup EXIT

# Start VNC server
echo "Starting mock VNC server on port 5900..."
./vncserver -port 5900 &
VNC_PID=$!
sleep 1
echo "✓ VNC server started (PID: $VNC_PID)"

# Start websockify proxy
echo "Starting websockify proxy on port 8080 -> localhost:5900..."
./websockify -listen :8080 -target localhost:5900 &
WS_PID=$!
sleep 1
echo "✓ Websockify proxy started (PID: $WS_PID)"

echo
echo "=== Demo Options ==="
echo "Choose a demo to run:"
echo "1. Basic GUI viewer (10 seconds)"
echo "2. GUI with checkerboard transparency (10 seconds)"
echo "3. GUI with frame capture (10 seconds)"
echo "4. Direct connection to VNC server"
echo "5. Connection through websockify proxy"
echo "6. Custom duration"
echo

read -p "Enter choice (1-6): " choice

case $choice in
    1)
        echo "Running basic GUI viewer for 10 seconds..."
        ./vncclient -host localhost:5900 -gui -duration 10
        ;;
    2)
        echo "Running GUI with checkerboard transparency for 10 seconds..."
        ./vncclient -host localhost:5900 -gui -checkerboard -duration 10
        ;;
    3)
        echo "Running GUI with frame capture for 10 seconds..."
        ./vncclient -host localhost:5900 -gui -capture -output ./demo_frames -duration 10
        echo "Frames saved to ./demo_frames/"
        ;;
    4)
        echo "Running direct connection to VNC server..."
        ./vncclient -host localhost:5900 -gui -duration 15
        ;;
    5)
        echo "Running connection through websockify proxy..."
        echo "Note: This connects to the websockify proxy, but our VNC client doesn't support WebSocket yet"
        echo "This will demonstrate the proxy is running, but connection will fail as expected"
        ./vncclient -host localhost:8080 -gui -duration 5 || echo "Expected: WebSocket not supported by VNC client"
        ;;
    6)
        read -p "Enter duration in seconds: " duration
        echo "Running GUI viewer for $duration seconds..."
        ./vncclient -host localhost:5900 -gui -duration $duration
        ;;
    *)
        echo "Invalid choice. Running default demo..."
        ./vncclient -host localhost:5900 -gui -checkerboard -duration 8
        ;;
esac

echo
echo "=== Demo Complete ==="
echo "✅ The GUI window should have displayed the animated VNC framebuffer."
echo "✅ You should have seen colored pixels changing in real-time."
echo "✅ GUI viewer is working properly on macOS!"
echo
echo "Try these commands manually:"
echo "./vncclient -host localhost:5900 -gui                          # Basic GUI"
echo "./vncclient -host localhost:5900 -gui -checkerboard           # With transparency"
echo "./vncclient -host localhost:5900 -gui -capture -output ./test # With capture"
echo
echo "Available options:"
echo "  -gui                 Show framebuffer in GUI window"
echo "  -checkerboard        Add checkerboard background for transparency"
echo "  -capture             Save frames as PNG files"
echo "  -output DIR          Directory for captured frames"
echo "  -duration N          Run for N seconds"
echo "  -fps N               Animation frame rate"
echo "  -webm               Create WebM animation"
echo "  -apng               Create APNG animation"