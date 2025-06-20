package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websockify/rfb"
	"github.com/coder/websockify/viewer"
)

const (
	SCREEN_WIDTH  = 800
	SCREEN_HEIGHT = 600
)

var (
	animationType string
	globalServer *VNCServer
)


type VNCConnection struct {
	conn        net.Conn
	frameNumber int // Frame number for 30fps animation
	animationType string // Type of animation to generate
	buffer      []byte   // Message buffer for proper framing
	pixelFormat rfb.PixelFormat // Client's requested pixel format
}

type VNCServer struct {
	viewer    *viewer.FramebufferViewer
	showGUI   bool
	animation string
	fps       int
}

type AnimationGenerator func(frameNumber, width, height int) []byte

func main() {
	var (
		port = flag.String("port", "5900", "Port to listen on")
		animation = flag.String("animation", "wheel", "Animation type: wheel, waves, plasma, orbits, gradient")
		gui = flag.Bool("gui", false, "Show server framebuffer in GUI window (requires GUI environment)")
		fps = flag.Int("fps", 30, "Frame rate for GUI animation (frames per second)")
		help = flag.Bool("help", false, "Show this help message")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "vncserver - Mock VNC server for testing websockify\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 5900\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 5900 -gui\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 5900 -animation plasma -gui\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 5900 -gui -fps 60\n", os.Args[0])
		os.Exit(0)
	}

	// Configuration
	config := VNCServerConfig{
		port:      *port,
		animation: *animation,
		showGUI:   *gui,
		fps:       *fps,
	}

	if *gui {
		// Run with GUI - this will block on main thread
		runWithGUI(config)
	} else {
		// Run without GUI
		runWithoutGUI(config)
	}
}

type VNCServerConfig struct {
	port      string
	animation string
	showGUI   bool
	fps       int
}

func runWithGUI(config VNCServerConfig) {
	// This will run on the main thread as required by macOS
	viewer.RunWithVNCClient(fmt.Sprintf("VNC Server - %s:%s", config.animation, config.port), SCREEN_WIDTH, SCREEN_HEIGHT, func(v *viewer.FramebufferViewer) {
		runVNCServer(config, v)
	})
}

func runWithoutGUI(config VNCServerConfig) {
	runVNCServer(config, nil)
}

func runVNCServer(config VNCServerConfig, guiViewer *viewer.FramebufferViewer) {
	animationType = config.animation
	
	globalServer = &VNCServer{
		viewer:    guiViewer,
		showGUI:   config.showGUI,
		animation: config.animation,
		fps:       config.fps,
	}

	listener, err := net.Listen("tcp", ":"+config.port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", config.port, err)
	}
	defer listener.Close()

	log.Printf("Mock VNC server listening on port %s", config.port)
	if globalServer.showGUI {
		log.Printf("GUI viewer enabled for server framebuffer")
		// Start continuous framebuffer generation for GUI
		go startFramebufferAnimation()
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down VNC server...")
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("Listener closed, stopping accept loop")
				return
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleVNCConnection(conn)
	}
}

func startFramebufferAnimation() {
	frameNumber := 0
	// Calculate frame interval from FPS (default 30 FPS = 33ms interval)
	frameInterval := time.Duration(1000/globalServer.fps) * time.Millisecond
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()
	
	log.Printf("Starting framebuffer animation for GUI viewer at %d FPS", globalServer.fps)
	
	for {
		select {
		case <-ticker.C:
			if globalServer != nil && globalServer.showGUI && globalServer.viewer != nil {
				// Generate frame data
				pixelData := generateAnimationFrame(globalServer.animation, frameNumber, SCREEN_WIDTH, SCREEN_HEIGHT)
				updateServerGUI(pixelData, SCREEN_WIDTH, SCREEN_HEIGHT)
				frameNumber++
			}
		}
	}
}

func handleVNCConnection(conn net.Conn) {
	defer conn.Close()
	
	clientAddr := conn.RemoteAddr().String()
	log.Printf("New VNC connection from %s", clientAddr)

	// Create VNC connection state with default pixel format (matches ServerInit)
	defaultPixelFormat := rfb.DefaultPixelFormat()
	
	vncConn := &VNCConnection{
		conn:        conn,
		frameNumber: 0,
		animationType: animationType,
		pixelFormat: defaultPixelFormat,
	}

	// RFB Protocol Handshake
	if err := doVNCHandshake(vncConn.conn); err != nil {
		log.Printf("VNC handshake failed for %s: %v", clientAddr, err)
		return
	}

	log.Printf("VNC handshake completed for %s", clientAddr)

	// Keep connection alive and handle client messages with proper framing
	readBuffer := make([]byte, 1024)
	for {
		vncConn.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := vncConn.conn.Read(readBuffer)
		if err != nil {
			log.Printf("VNC connection from %s ended: %v", clientAddr, err)
			return
		}

		if n > 0 {
			log.Printf("VNC client %s sent %d bytes", clientAddr, n)
			// Append new data to connection buffer
			vncConn.buffer = append(vncConn.buffer, readBuffer[:n]...)
			
			// Process complete messages from buffer
			if err := processCompleteMessages(vncConn); err != nil {
				log.Printf("VNC message processing failed for %s: %v", clientAddr, err)
				return
			}
		}
	}
}

func doVNCHandshake(conn net.Conn) error {
	// Step 1: Send RFB version
	if err := rfb.SendRFBVersion(conn); err != nil {
		return fmt.Errorf("failed to send RFB version: %v", err)
	}

	// Step 2: Read client version
	clientVersion, err := rfb.ReadRFBVersion(conn)
	if err != nil {
		return fmt.Errorf("failed to read client version: %v", err)
	}
	log.Printf("Client version: %s", clientVersion)

	// Step 3: Send security types (1 = None)
	if err := rfb.SendSecurityTypes(conn, []uint8{rfb.SecurityNone}); err != nil {
		return fmt.Errorf("failed to send security types: %v", err)
	}

	// Step 4: Read client security choice
	securityChoice := make([]byte, 1)
	if _, err := conn.Read(securityChoice); err != nil {
		return fmt.Errorf("failed to read security choice: %v", err)
	}

	// Step 5: Send security result (0 = OK)
	if err := rfb.SendSecurityResult(conn, 0); err != nil {
		return fmt.Errorf("failed to send security result: %v", err)
	}

	// Step 6: Read ClientInit
	clientInit := make([]byte, 1)
	if _, err := conn.Read(clientInit); err != nil {
		return fmt.Errorf("failed to read client init: %v", err)
	}

	// Step 7: Send ServerInit
	serverInit := rfb.ServerInit{
		Width:       SCREEN_WIDTH,
		Height:      SCREEN_HEIGHT,
		PixelFormat: rfb.DefaultPixelFormat(),
		Name:        "Test",
	}

	if err := rfb.SendServerInit(conn, serverInit); err != nil {
		return fmt.Errorf("failed to send server init: %v", err)
	}

	return nil
}

// getMessageLength returns the expected length of a VNC client message based on its type
func getMessageLength(messageType byte, data []byte) (int, error) {
	length, err := rfb.GetMessageLength(messageType, data)
	if err != nil {
		return -1, err
	}
	if length == 0 && len(data) < 8 {
		return -1, nil // Need more data to determine length
	}
	return length, nil
}

// processCompleteMessages processes all complete messages in the buffer
func processCompleteMessages(vncConn *VNCConnection) error {
	for len(vncConn.buffer) > 0 {
		// Need at least 1 byte to determine message type
		if len(vncConn.buffer) < 1 {
			break
		}
		
		messageType := vncConn.buffer[0]
		expectedLength, err := getMessageLength(messageType, vncConn.buffer)
		if err != nil {
			return fmt.Errorf("invalid message type %d: %v", messageType, err)
		}
		
		// If expectedLength is -1, we need more data to determine the full message length
		if expectedLength == -1 {
			log.Printf("Need more data to determine message length for type %d", messageType)
			break
		}
		
		// Check if we have the complete message
		if len(vncConn.buffer) < expectedLength {
			log.Printf("Incomplete message: have %d bytes, need %d for type %d", 
				len(vncConn.buffer), expectedLength, messageType)
			break
		}
		
		// We have a complete message, process it
		messageData := vncConn.buffer[:expectedLength]
		if err := handleVNCMessage(vncConn, messageData); err != nil {
			return err
		}
		
		// Remove processed message from buffer
		vncConn.buffer = vncConn.buffer[expectedLength:]
		log.Printf("Processed message type %d (%d bytes), %d bytes remaining in buffer", 
			messageType, expectedLength, len(vncConn.buffer))
	}
	
	return nil
}

func handleVNCMessage(vncConn *VNCConnection, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	messageType := data[0]
	log.Printf("Processing complete message type %d (%d bytes)", messageType, len(data))
	
	switch messageType {
	case rfb.SetPixelFormat: // SetPixelFormat (20 bytes total)
		return handleSetPixelFormat(vncConn, data)
		
	case rfb.SetEncodings: // SetEncodings (variable length)
		numEncodings := (int(data[2]) << 8) | int(data[3])
		log.Printf("Received SetEncodings message with %d encodings", numEncodings)
		return nil
		
	case rfb.FramebufferUpdateRequest: // FramebufferUpdateRequest (10 bytes total)
		log.Printf("Received FramebufferUpdateRequest message")
		sendFramebufferUpdate(vncConn)
		return nil
		
	case rfb.KeyEvent: // KeyEvent (8 bytes total)
		log.Printf("Received KeyEvent message")
		return nil
		
	case rfb.PointerEvent: // PointerEvent (6 bytes total)
		log.Printf("Received PointerEvent message")
		return nil
		
	case rfb.ClientCutText: // ClientCutText (variable length)
		textLength := (int(data[4]) << 24) | (int(data[5]) << 16) | (int(data[6]) << 8) | int(data[7])
		log.Printf("Received ClientCutText message with %d bytes of text", textLength)
		return nil
		
	default:
		log.Printf("Received invalid message type: %d (0x%02X) - closing connection", messageType, messageType)
		return fmt.Errorf("invalid message type: %d", messageType)
	}
}

func handleSetPixelFormat(vncConn *VNCConnection, data []byte) error {
	pf, err := rfb.ParseSetPixelFormat(data)
	if err != nil {
		return err
	}
	
	// Update connection's pixel format
	vncConn.pixelFormat = pf
	
	log.Printf("SetPixelFormat: %d bpp, depth %d, %s-endian, true-color=%d", 
		pf.BitsPerPixel, pf.Depth, 
		map[uint8]string{0: "little", 1: "big"}[pf.BigEndianFlag],
		pf.TrueColorFlag)
	log.Printf("Color maximums: R=%d G=%d B=%d, Shifts: R=%d G=%d B=%d",
		pf.RedMax, pf.GreenMax, pf.BlueMax,
		pf.RedShift, pf.GreenShift, pf.BlueShift)
	
	return nil
}


func sendFramebufferUpdate(vncConn *VNCConnection) {
	// Send a simple framebuffer update (solid color rectangle)
	update := make([]byte, 16)
	update[0] = 0 // FramebufferUpdate message type
	update[1] = 0 // padding
	// number-of-rectangles (16-bit big-endian)
	update[2] = 0
	update[3] = 1
	// rectangle: x, y, width, height (each 16-bit big-endian)
	update[4] = 0   // x high
	update[5] = 0   // x low
	update[6] = 0   // y high  
	update[7] = 0   // y low
	update[8] = byte(SCREEN_WIDTH >> 8)   // width high
	update[9] = byte(SCREEN_WIDTH & 0xFF) // width low
	update[10] = byte(SCREEN_HEIGHT >> 8)   // height high
	update[11] = byte(SCREEN_HEIGHT & 0xFF) // height low
	// encoding-type (32-bit big-endian) - 0 = Raw
	update[12] = 0
	update[13] = 0
	update[14] = 0  
	update[15] = 0

	if _, err := vncConn.conn.Write(update); err != nil {
		log.Printf("Failed to send framebuffer update header: %v", err)
		return
	}
	log.Printf("Sent FramebufferUpdate header: %v", update)

	// Generate animated pixel data in BGRA format
	bgraData := generateAnimationFrame(vncConn.animationType, vncConn.frameNumber, SCREEN_WIDTH, SCREEN_HEIGHT)
	
	// Convert to client's requested pixel format
	pixelData := rfb.ConvertPixelFormat(bgraData, SCREEN_WIDTH, SCREEN_HEIGHT, vncConn.pixelFormat)
	log.Printf("Sending pixel data: %d bytes (converted from BGRA to client format), first 16 bytes: %v", len(pixelData), pixelData[:16])

	if _, err := vncConn.conn.Write(pixelData); err != nil {
		log.Printf("Failed to send framebuffer update data: %v", err)
	}

	// Update GUI viewer if enabled (use original BGRA data for GUI)
	if globalServer != nil && globalServer.showGUI && globalServer.viewer != nil {
		updateServerGUI(bgraData, SCREEN_WIDTH, SCREEN_HEIGHT)
	}

	// Increment frame number for next frame (30fps)
	vncConn.frameNumber++
}

func updateServerGUI(pixelData []byte, width, height int) {
	// Convert raw pixel data (BGRA) to image.RGBA
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	for i := 0; i < len(pixelData); i += 4 {
		pixelIndex := i / 4
		y := pixelIndex / width
		x := pixelIndex % width
		
		if x < width && y < height {
			// VNC uses BGRA format, convert to RGBA
			b := pixelData[i]
			g := pixelData[i+1]
			r := pixelData[i+2]
			a := pixelData[i+3]
			
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	
	globalServer.viewer.UpdateFramebuffer(img)
}

func generateAnimationFrame(animationType string, frameNumber, width, height int) []byte {
	switch animationType {
	case "wheel":
		return generateColorWheel(frameNumber, width, height)
	case "waves":
		return generateAlphaWaves(frameNumber, width, height)
	case "plasma":
		return generatePlasma(frameNumber, width, height)
	case "orbits":
		return generateOrbitingCircles(frameNumber, width, height)
	case "gradient":
		return generateGradientSweep(frameNumber, width, height)
	default:
		return generateColorWheel(frameNumber, width, height)
	}
}

func generateColorWheel(frameNumber, width, height int) []byte {
	pixelData := make([]byte, width*height*4)
	centerX := float64(width) / 2
	centerY := float64(height) / 2
	maxRadius := math.Min(centerX, centerY) * 0.8
	
	// Rotation based on frame number (360 degrees over 120 frames = 3 seconds at 30fps)
	rotation := float64(frameNumber) * 2 * math.Pi / 120
	
	for i := 0; i < len(pixelData); i += 4 {
		pixel := i / 4
		row := pixel / width
		col := pixel % width
		
		// Calculate distance from center and angle
		dx := float64(col) - centerX
		dy := float64(row) - centerY
		distance := math.Sqrt(dx*dx + dy*dy)
		angle := math.Atan2(dy, dx) + rotation
		
		if distance <= maxRadius {
			// Convert angle to hue (0-360 degrees)
			hue := angle * 180 / math.Pi
			if hue < 0 {
				hue += 360
			}
			
			// Create saturation gradient from center to edge
			saturation := distance / maxRadius
			
			// Create alpha gradient (more transparent towards edge)
			alpha := 1.0 - (distance / maxRadius) * 0.7
			
			// Convert HSV to RGB
			r, g, b := hsvToRgb(hue, saturation, 1.0)
			
			pixelData[i] = uint8(b * 255)     // blue
			pixelData[i+1] = uint8(g * 255)   // green
			pixelData[i+2] = uint8(r * 255)   // red
			pixelData[i+3] = uint8(alpha * 255) // alpha
		} else {
			// Transparent outside the wheel
			pixelData[i] = 0
			pixelData[i+1] = 0
			pixelData[i+2] = 0
			pixelData[i+3] = 0
		}
	}
	
	return pixelData
}

func generateAlphaWaves(frameNumber, width, height int) []byte {
	pixelData := make([]byte, width*height*4)
	
	// Wave parameters
	timeOffset := float64(frameNumber) * 0.1
	
	for i := 0; i < len(pixelData); i += 4 {
		pixel := i / 4
		row := pixel / width
		col := pixel % width
		
		// Create multiple wave patterns
		x := float64(col) / float64(width) * 4 * math.Pi
		y := float64(row) / float64(height) * 3 * math.Pi
		
		// Combine multiple sine waves for complex patterns
		wave1 := math.Sin(x + timeOffset)
		wave2 := math.Sin(y + timeOffset*1.3)
		wave3 := math.Sin((x+y)*0.5 + timeOffset*0.7)
		
		// Create RGB values based on waves
		r := (wave1 + 1) / 2
		g := (wave2 + 1) / 2
		b := (wave3 + 1) / 2
		
		// Create alpha based on wave interference
		alpha := (wave1*wave2 + 1) / 2
		alpha = math.Max(0.1, alpha) // Minimum 10% alpha
		
		pixelData[i] = uint8(b * 255)     // blue
		pixelData[i+1] = uint8(g * 255)   // green
		pixelData[i+2] = uint8(r * 255)   // red
		pixelData[i+3] = uint8(alpha * 255) // alpha
	}
	
	return pixelData
}

func generatePlasma(frameNumber, width, height int) []byte {
	pixelData := make([]byte, width*height*4)
	
	time := float64(frameNumber) * 0.05
	
	for i := 0; i < len(pixelData); i += 4 {
		pixel := i / 4
		row := pixel / width
		col := pixel % width
		
		x := float64(col) / float64(width)
		y := float64(row) / float64(height)
		
		// Classic plasma effect
		v1 := math.Sin(x*10 + time)
		v2 := math.Sin(y*10 + time*1.2)
		v3 := math.Sin((x+y)*10 + time*0.8)
		v4 := math.Sin(math.Sqrt(x*x+y*y)*10 + time*1.5)
		
		plasma := (v1 + v2 + v3 + v4) / 4
		
		// Convert plasma value to color
		hue := (plasma + 1) * 180 // 0-360 degrees
		saturation := 0.8
		brightness := 0.9
		
		r, g, b := hsvToRgb(hue, saturation, brightness)
		
		// Alpha varies with plasma intensity
		alpha := (math.Abs(plasma) + 0.3) * 0.9
		
		pixelData[i] = uint8(b * 255)     // blue
		pixelData[i+1] = uint8(g * 255)   // green
		pixelData[i+2] = uint8(r * 255)   // red
		pixelData[i+3] = uint8(alpha * 255) // alpha
	}
	
	return pixelData
}

func generateOrbitingCircles(frameNumber, width, height int) []byte {
	pixelData := make([]byte, width*height*4)
	
	// Clear background (transparent)
	for i := 0; i < len(pixelData); i += 4 {
		pixelData[i+3] = 0 // alpha = 0 (transparent)
	}
	
	centerX := float64(width) / 2
	centerY := float64(height) / 2
	orbitRadius := math.Min(centerX, centerY) * 0.6
	
	// Multiple orbiting circles
	numCircles := 5
	time := float64(frameNumber) * 0.1
	
	for c := 0; c < numCircles; c++ {
		// Each circle has different orbit speed and phase
		phase := float64(c) * 2 * math.Pi / float64(numCircles)
		speed := 1.0 + float64(c)*0.3
		angle := time*speed + phase
		
		// Circle position
		circleX := centerX + math.Cos(angle)*orbitRadius
		circleY := centerY + math.Sin(angle)*orbitRadius
		circleRadius := 30.0 + float64(c)*10
		
		// Circle color (different hue for each circle)
		hue := float64(c) * 360 / float64(numCircles)
		r, g, b := hsvToRgb(hue, 0.8, 0.9)
		
		// Draw circle
		for i := 0; i < len(pixelData); i += 4 {
			pixel := i / 4
			row := pixel / width
			col := pixel % width
			
			dx := float64(col) - circleX
			dy := float64(row) - circleY
			distance := math.Sqrt(dx*dx + dy*dy)
			
			if distance <= circleRadius {
				// Soft edge with alpha falloff
				alpha := 1.0 - (distance / circleRadius) * 0.7
				alpha = math.Max(0, alpha)
				
				// Blend with existing pixel (additive blending)
				existingAlpha := float64(pixelData[i+3]) / 255.0
				newAlpha := alpha + existingAlpha*(1-alpha)
				
				if newAlpha > 0 {
					// Blend colors
					blendR := (r*alpha + (float64(pixelData[i+2])/255.0)*existingAlpha) / newAlpha
					blendG := (g*alpha + (float64(pixelData[i+1])/255.0)*existingAlpha) / newAlpha
					blendB := (b*alpha + (float64(pixelData[i])/255.0)*existingAlpha) / newAlpha
					
					pixelData[i] = uint8(blendB * 255)     // blue
					pixelData[i+1] = uint8(blendG * 255)   // green
					pixelData[i+2] = uint8(blendR * 255)   // red
					pixelData[i+3] = uint8(newAlpha * 255) // alpha
				}
			}
		}
	}
	
	return pixelData
}

func generateGradientSweep(frameNumber, width, height int) []byte {
	pixelData := make([]byte, width*height*4)
	
	// Rotating gradient
	rotation := float64(frameNumber) * 2 * math.Pi / 90 // 3-second rotation at 30fps
	
	centerX := float64(width) / 2
	centerY := float64(height) / 2
	
	for i := 0; i < len(pixelData); i += 4 {
		pixel := i / 4
		row := pixel / width
		col := pixel % width
		
		// Calculate angle from center
		dx := float64(col) - centerX
		dy := float64(row) - centerY
		angle := math.Atan2(dy, dx) + rotation
		
		// Normalize angle to 0-1
		normalizedAngle := (angle + math.Pi) / (2 * math.Pi)
		normalizedAngle = normalizedAngle - math.Floor(normalizedAngle) // Keep in 0-1 range
		
		// Create gradient colors
		hue := normalizedAngle * 360
		r, g, b := hsvToRgb(hue, 0.9, 0.8)
		
		// Distance-based alpha
		distance := math.Sqrt(dx*dx + dy*dy)
		maxDistance := math.Sqrt(centerX*centerX + centerY*centerY)
		alpha := 0.3 + 0.7*(1.0 - distance/maxDistance) // More opaque in center
		
		pixelData[i] = uint8(b * 255)     // blue
		pixelData[i+1] = uint8(g * 255)   // green
		pixelData[i+2] = uint8(r * 255)   // red
		pixelData[i+3] = uint8(alpha * 255) // alpha
	}
	
	return pixelData
}

// HSV to RGB conversion
func hsvToRgb(h, s, v float64) (float64, float64, float64) {
	h = math.Mod(h, 360) / 60
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h, 2) - 1))
	m := v - c
	
	var r, g, b float64
	
	switch int(h) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	
	return r + m, g + m, b + m
}