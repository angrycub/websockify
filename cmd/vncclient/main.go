package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/coder/websockify/viewer"
)

const (
	RFB_VERSION = "RFB 003.008\n"
)

type VNCClient struct {
	conn            net.Conn
	width           int
	height          int
	framebuffer     *image.RGBA
	frameCount      int
	captureFrames   bool
	outputDir       string
	useCheckerboard bool
	createWebM      bool
	createAPNG      bool
	frameRate       int
	capturedFrames  []*image.RGBA // Store frames for animation
	viewer          *viewer.FramebufferViewer
	showGUI         bool
	serverPixelFormat PixelFormat // Server's pixel format from handshake
}

type ServerInit struct {
	Width      uint16
	Height     uint16
	PixelFormat PixelFormat
	NameLength uint32
	Name       string
}

type PixelFormat struct {
	BitsPerPixel  uint8
	Depth         uint8
	BigEndianFlag uint8
	TrueColorFlag uint8
	RedMax        uint16
	GreenMax      uint16
	BlueMax       uint16
	RedShift      uint8
	GreenShift    uint8
	BlueShift     uint8
	Padding       [3]uint8
}

func main() {
	var (
		host           = flag.String("host", "localhost:5900", "VNC server host:port")
		capture        = flag.Bool("capture", false, "Capture framebuffer updates as PNG files")
		output         = flag.String("output", "./test_output", "Output directory for captured frames")
		duration       = flag.Int("duration", 10, "Duration to run client in seconds")
		checkerboard   = flag.Bool("checkerboard", false, "Add checkerboard background to show transparency")
		animateWebM    = flag.Bool("webm", false, "Create WebM video animation from captured frames")
		animateAPNG    = flag.Bool("apng", false, "Create APNG animation from captured frames")
		frameRate      = flag.Int("fps", 2, "Frame rate for animations (frames per second)")
		gui            = flag.Bool("gui", false, "Show framebuffer in GUI window (requires GUI environment)")
		testPixelFormat = flag.Bool("test-pixel-format", false, "Send a test SetPixelFormat message (16bpp RGB565)")
		help           = flag.Bool("help", false, "Show this help message")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "vncclient - Basic VNC client for testing websockify\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -host localhost:5900\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host localhost:8080 -capture -output ./test-frames\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host localhost:5900 -capture -checkerboard\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host localhost:5900 -webm -apng -fps 5 -duration 10\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host localhost:5900 -capture -checkerboard -webm -fps 2\n", os.Args[0])
		os.Exit(0)
	}

	// Configuration for VNC client
	config := VNCConfig{
		host:            *host,
		captureFrames:   *capture,
		outputDir:       *output,
		duration:        *duration,
		useCheckerboard: *checkerboard,
		createWebM:      *animateWebM,
		createAPNG:      *animateAPNG,
		frameRate:       *frameRate,
		showGUI:         *gui,
		testPixelFormat: *testPixelFormat,
	}

	if *gui {
		// Run with GUI - this will block on main thread
		runWithGUI(config)
	} else {
		// Run without GUI
		runWithoutGUI(config)
	}
}

type VNCConfig struct {
	host            string
	captureFrames   bool
	outputDir       string
	duration        int
	useCheckerboard bool
	createWebM      bool
	createAPNG      bool
	frameRate       int
	showGUI         bool
	testPixelFormat bool
}

func runWithGUI(config VNCConfig) {
	// This will run on the main thread as required by macOS
	viewer.RunWithVNCClient("VNC Client", 800, 600, func(v *viewer.FramebufferViewer) {
		runVNCClient(config, v)
	})
}

func runWithoutGUI(config VNCConfig) {
	runVNCClient(config, nil)
}

func runVNCClient(config VNCConfig, guiViewer *viewer.FramebufferViewer) {
	client := &VNCClient{
		captureFrames:   config.captureFrames,
		outputDir:       config.outputDir,
		useCheckerboard: config.useCheckerboard,
		createWebM:      config.createWebM,
		createAPNG:      config.createAPNG,
		frameRate:       config.frameRate,
		capturedFrames:  make([]*image.RGBA, 0),
		showGUI:         config.showGUI,
		viewer:          guiViewer,
	}

	if client.captureFrames {
		if err := os.MkdirAll(client.outputDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	log.Printf("Connecting to VNC server at %s", config.host)
	conn, err := net.Dial("tcp", config.host)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client.conn = conn

	if err := client.handshake(); err != nil {
		log.Fatalf("Handshake failed: %v", err)
	}

	log.Printf("VNC handshake completed. Screen: %dx%d", client.width, client.height)
	
	// Test SetPixelFormat if requested
	if config.testPixelFormat {
		// Send a 16bpp RGB565 pixel format
		testFormat := PixelFormat{
			BitsPerPixel:  16,
			Depth:         16,
			BigEndianFlag: 0, // little-endian
			TrueColorFlag: 1,
			RedMax:        31,  // 5 bits
			GreenMax:      63,  // 6 bits
			BlueMax:       31,  // 5 bits
			RedShift:      11,  // bits 11-15
			GreenShift:    5,   // bits 5-10
			BlueShift:     0,   // bits 0-4
		}
		log.Printf("Sending test SetPixelFormat message (16bpp RGB565)")
		if err := client.sendSetPixelFormat(testFormat); err != nil {
			log.Printf("Failed to send SetPixelFormat: %v", err)
		}
	}

	// If GUI viewer was passed, reinitialize it with actual dimensions
	if client.showGUI && client.viewer != nil {
		client.viewer.Initialize(fmt.Sprintf("VNC Client - %s", config.host), client.width, client.height)
		client.viewer.Show()
		log.Printf("GUI viewer initialized with actual screen size")
	}

	// Request initial framebuffer update
	if err := client.requestFramebufferUpdate(false, 0, 0, uint16(client.width), uint16(client.height)); err != nil {
		log.Printf("Failed to request framebuffer update: %v", err)
	}

	// Run for specified duration
	timeout := time.After(time.Duration(config.duration) * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Printf("Running VNC client for %d seconds...", config.duration)

	for {
		select {
		case <-timeout:
			log.Printf("Client finished. Captured %d frames.", client.frameCount)
			
			// Create animations if requested
			if client.createWebM {
				if err := client.createWebMAnimation(); err != nil {
					log.Printf("Failed to create WebM animation: %v", err)
				}
			}
			if client.createAPNG {
				if err := client.createAPNGAnimation(); err != nil {
					log.Printf("Failed to create APNG animation: %v", err)
				}
			}
			return
		case <-ticker.C:
			// Request periodic framebuffer updates
			if err := client.requestFramebufferUpdate(true, 0, 0, uint16(client.width), uint16(client.height)); err != nil {
				log.Printf("Failed to request framebuffer update: %v", err)
			}
		default:
			// Handle incoming messages
			if err := client.handleMessage(); err != nil {
				if err == io.EOF {
					log.Printf("Connection closed by server")
					return
				}
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (c *VNCClient) handshake() error {
	// Read server version
	serverVersion := make([]byte, 12)
	if _, err := io.ReadFull(c.conn, serverVersion); err != nil {
		return fmt.Errorf("failed to read server version: %v", err)
	}
	log.Printf("Server version: %s", string(serverVersion))

	// Send client version
	if _, err := c.conn.Write([]byte(RFB_VERSION)); err != nil {
		return fmt.Errorf("failed to send client version: %v", err)
	}

	// Read security types
	var numSecurityTypes uint8
	if err := binary.Read(c.conn, binary.BigEndian, &numSecurityTypes); err != nil {
		return fmt.Errorf("failed to read security types count: %v", err)
	}

	securityTypes := make([]uint8, numSecurityTypes)
	if _, err := io.ReadFull(c.conn, securityTypes); err != nil {
		return fmt.Errorf("failed to read security types: %v", err)
	}
	log.Printf("Available security types: %v", securityTypes)

	// Choose security type (1 = None)
	securityChoice := uint8(1)
	if err := binary.Write(c.conn, binary.BigEndian, securityChoice); err != nil {
		return fmt.Errorf("failed to send security choice: %v", err)
	}

	// Read security result
	var securityResult uint32
	if err := binary.Read(c.conn, binary.BigEndian, &securityResult); err != nil {
		return fmt.Errorf("failed to read security result: %v", err)
	}
	if securityResult != 0 {
		return fmt.Errorf("security handshake failed: %d", securityResult)
	}

	// Send ClientInit (shared = 1)
	clientInit := uint8(1)
	if err := binary.Write(c.conn, binary.BigEndian, clientInit); err != nil {
		return fmt.Errorf("failed to send client init: %v", err)
	}

	// Read ServerInit
	var serverInit ServerInit
	if err := binary.Read(c.conn, binary.BigEndian, &serverInit.Width); err != nil {
		return fmt.Errorf("failed to read width: %v", err)
	}
	if err := binary.Read(c.conn, binary.BigEndian, &serverInit.Height); err != nil {
		return fmt.Errorf("failed to read height: %v", err)
	}
	if err := binary.Read(c.conn, binary.BigEndian, &serverInit.PixelFormat); err != nil {
		return fmt.Errorf("failed to read pixel format: %v", err)
	}
	if err := binary.Read(c.conn, binary.BigEndian, &serverInit.NameLength); err != nil {
		return fmt.Errorf("failed to read name length: %v", err)
	}

	nameBytes := make([]byte, serverInit.NameLength)
	if _, err := io.ReadFull(c.conn, nameBytes); err != nil {
		return fmt.Errorf("failed to read server name: %v", err)
	}
	serverInit.Name = string(nameBytes)

	c.width = int(serverInit.Width)
	c.height = int(serverInit.Height)
	c.framebuffer = image.NewRGBA(image.Rect(0, 0, c.width, c.height))
	c.serverPixelFormat = serverInit.PixelFormat

	log.Printf("Server: %s, %dx%d, %d bpp", serverInit.Name, c.width, c.height, serverInit.PixelFormat.BitsPerPixel)
	log.Printf("Server pixel format: depth=%d, true-color=%d, endian=%s", 
		serverInit.PixelFormat.Depth, serverInit.PixelFormat.TrueColorFlag,
		map[uint8]string{0: "little", 1: "big"}[serverInit.PixelFormat.BigEndianFlag])
	log.Printf("Color maximums: R=%d G=%d B=%d, Shifts: R=%d G=%d B=%d",
		serverInit.PixelFormat.RedMax, serverInit.PixelFormat.GreenMax, serverInit.PixelFormat.BlueMax,
		serverInit.PixelFormat.RedShift, serverInit.PixelFormat.GreenShift, serverInit.PixelFormat.BlueShift)

	return nil
}

// sendSetPixelFormat sends a SetPixelFormat message to the server
func (c *VNCClient) sendSetPixelFormat(pf PixelFormat) error {
	msg := make([]byte, 20)
	
	// Message type (0 = SetPixelFormat)
	msg[0] = 0
	
	// 3 bytes of padding (bytes 1-3)
	msg[1] = 0
	msg[2] = 0
	msg[3] = 0
	
	// Pixel format (16 bytes starting at byte 4)
	msg[4] = pf.BitsPerPixel
	msg[5] = pf.Depth
	msg[6] = pf.BigEndianFlag
	msg[7] = pf.TrueColorFlag
	
	// Color maximums (16-bit big-endian)
	msg[8] = uint8(pf.RedMax >> 8)
	msg[9] = uint8(pf.RedMax & 0xFF)
	msg[10] = uint8(pf.GreenMax >> 8)
	msg[11] = uint8(pf.GreenMax & 0xFF)
	msg[12] = uint8(pf.BlueMax >> 8)
	msg[13] = uint8(pf.BlueMax & 0xFF)
	
	// Color shifts
	msg[14] = pf.RedShift
	msg[15] = pf.GreenShift
	msg[16] = pf.BlueShift
	
	// 3 bytes of padding (bytes 17-19)
	msg[17] = 0
	msg[18] = 0
	msg[19] = 0
	
	if _, err := c.conn.Write(msg); err != nil {
		return fmt.Errorf("failed to send SetPixelFormat message: %v", err)
	}
	
	log.Printf("Sent SetPixelFormat: %d bpp, depth %d, %s-endian, true-color=%d", 
		pf.BitsPerPixel, pf.Depth, 
		map[uint8]string{0: "little", 1: "big"}[pf.BigEndianFlag],
		pf.TrueColorFlag)
	
	return nil
}

func (c *VNCClient) requestFramebufferUpdate(incremental bool, x, y, width, height uint16) error {
	msg := make([]byte, 10)
	msg[0] = 3 // FramebufferUpdateRequest
	if incremental {
		msg[1] = 1
	} else {
		msg[1] = 0
	}
	binary.BigEndian.PutUint16(msg[2:4], x)
	binary.BigEndian.PutUint16(msg[4:6], y)
	binary.BigEndian.PutUint16(msg[6:8], width)
	binary.BigEndian.PutUint16(msg[8:10], height)

	_, err := c.conn.Write(msg)
	return err
}

func (c *VNCClient) handleMessage() error {
	c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	
	var messageType uint8
	if err := binary.Read(c.conn, binary.BigEndian, &messageType); err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil // Timeout is expected
		}
		return err
	}

	c.conn.SetReadDeadline(time.Time{}) // Clear deadline

	switch messageType {
	case 0: // FramebufferUpdate
		return c.handleFramebufferUpdate()
	case 1: // SetColorMapEntries
		log.Printf("Received SetColorMapEntries (not implemented)")
		return c.skipMessage(6) // Skip the rest of the message
	case 2: // Bell
		log.Printf("Received Bell")
		return nil
	case 3: // ServerCutText
		return c.handleServerCutText()
	default:
		log.Printf("Unknown message type: %d", messageType)
		return fmt.Errorf("unknown message type: %d", messageType)
	}
}

func (c *VNCClient) handleFramebufferUpdate() error {
	var padding uint8
	var numRects uint16

	if err := binary.Read(c.conn, binary.BigEndian, &padding); err != nil {
		return err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &numRects); err != nil {
		return err
	}

	log.Printf("Framebuffer update: %d rectangles", numRects)

	for i := uint16(0); i < numRects; i++ {
		var x, y, width, height uint16
		var encoding int32

		if err := binary.Read(c.conn, binary.BigEndian, &x); err != nil {
			return err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &y); err != nil {
			return err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &width); err != nil {
			return err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &height); err != nil {
			return err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &encoding); err != nil {
			return err
		}

		log.Printf("Rectangle %d: %dx%d at (%d,%d), encoding %d", i, width, height, x, y, encoding)

		if encoding == 0 { // Raw encoding
			if err := c.handleRawRectangle(int(x), int(y), int(width), int(height)); err != nil {
				return err
			}
		} else {
			log.Printf("Unsupported encoding: %d", encoding)
			// Skip unknown encoding data - this is a simplified approach
			pixelBytes := int(width) * int(height) * 4 // Assume 32-bit pixels
			if _, err := io.CopyN(io.Discard, c.conn, int64(pixelBytes)); err != nil {
				return err
			}
		}
	}

	// Update GUI viewer if enabled
	if c.showGUI && c.viewer != nil {
		var displayImage image.Image = c.framebuffer
		if c.useCheckerboard {
			displayImage = c.compositeWithCheckerboard()
		}
		c.viewer.UpdateFramebuffer(displayImage)
	}

	// Save frame if capturing
	if c.captureFrames {
		if err := c.saveFrame(); err != nil {
			log.Printf("Failed to save frame: %v", err)
		}
	}

	return nil
}

func (c *VNCClient) handleRawRectangle(x, y, width, height int) error {
	// Calculate bytes per pixel based on server's pixel format
	bytesPerPixel := int(c.serverPixelFormat.BitsPerPixel) / 8
	pixelDataSize := width * height * bytesPerPixel
	pixelData := make([]byte, pixelDataSize)
	
	if _, err := io.ReadFull(c.conn, pixelData); err != nil {
		return err
	}

	// Update framebuffer by converting server pixel format to RGBA
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			pixelOffset := (row*width + col) * bytesPerPixel
			if pixelOffset+bytesPerPixel <= len(pixelData) {
				rgba := c.convertPixelToRGBA(pixelData[pixelOffset:pixelOffset+bytesPerPixel])
				c.framebuffer.Set(x+col, y+row, rgba)
			}
		}
	}

	return nil
}

// convertPixelToRGBA converts a pixel from the server's format to RGBA
func (c *VNCClient) convertPixelToRGBA(pixelBytes []byte) color.RGBA {
	pf := c.serverPixelFormat
	
	// Read pixel value from bytes considering endianness
	var pixelValue uint32
	switch len(pixelBytes) {
	case 1: // 8 bits per pixel
		pixelValue = uint32(pixelBytes[0])
	case 2: // 16 bits per pixel
		if pf.BigEndianFlag == 1 {
			pixelValue = uint32(pixelBytes[0])<<8 | uint32(pixelBytes[1])
		} else {
			pixelValue = uint32(pixelBytes[1])<<8 | uint32(pixelBytes[0])
		}
	case 3: // 24 bits per pixel
		if pf.BigEndianFlag == 1 {
			pixelValue = uint32(pixelBytes[0])<<16 | uint32(pixelBytes[1])<<8 | uint32(pixelBytes[2])
		} else {
			pixelValue = uint32(pixelBytes[2])<<16 | uint32(pixelBytes[1])<<8 | uint32(pixelBytes[0])
		}
	case 4: // 32 bits per pixel
		if pf.BigEndianFlag == 1 {
			pixelValue = uint32(pixelBytes[0])<<24 | uint32(pixelBytes[1])<<16 | uint32(pixelBytes[2])<<8 | uint32(pixelBytes[3])
		} else {
			pixelValue = uint32(pixelBytes[3])<<24 | uint32(pixelBytes[2])<<16 | uint32(pixelBytes[1])<<8 | uint32(pixelBytes[0])
		}
	}
	
	// Extract color components using shifts and maximums
	redBits := (pixelValue >> pf.RedShift) & uint32(pf.RedMax)
	greenBits := (pixelValue >> pf.GreenShift) & uint32(pf.GreenMax)
	blueBits := (pixelValue >> pf.BlueShift) & uint32(pf.BlueMax)
	
	// Scale to 8-bit values
	var r, g, b uint8
	if pf.RedMax > 0 {
		r = uint8((redBits * 255) / uint32(pf.RedMax))
	}
	if pf.GreenMax > 0 {
		g = uint8((greenBits * 255) / uint32(pf.GreenMax))
	}
	if pf.BlueMax > 0 {
		b = uint8((blueBits * 255) / uint32(pf.BlueMax))
	}
	
	// For simplicity, assume full opacity (alpha = 255)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func (c *VNCClient) handleServerCutText() error {
	var padding [3]uint8
	var length uint32

	if err := binary.Read(c.conn, binary.BigEndian, &padding); err != nil {
		return err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
		return err
	}

	text := make([]byte, length)
	if _, err := io.ReadFull(c.conn, text); err != nil {
		return err
	}

	log.Printf("Server cut text: %s", string(text))
	return nil
}

func (c *VNCClient) skipMessage(bytes int) error {
	_, err := io.CopyN(io.Discard, c.conn, int64(bytes))
	return err
}

func (c *VNCClient) saveFrame() error {
	c.frameCount++
	
	var imageToSave *image.RGBA = c.framebuffer
	
	// Composite with checkerboard background if requested
	if c.useCheckerboard {
		imageToSave = c.compositeWithCheckerboard()
	}

	// Store frame for animation if needed
	if c.createWebM || c.createAPNG {
		// Create a copy of the frame for animation
		frameCopy := image.NewRGBA(imageToSave.Bounds())
		copy(frameCopy.Pix, imageToSave.Pix)
		c.capturedFrames = append(c.capturedFrames, frameCopy)
	}

	// Save individual PNG if capture is enabled
	if c.captureFrames {
		filename := filepath.Join(c.outputDir, fmt.Sprintf("frame_%04d.png", c.frameCount))
		
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := png.Encode(file, imageToSave); err != nil {
			return err
		}

		log.Printf("Saved frame %d to %s", c.frameCount, filename)
	}

	return nil
}

func (c *VNCClient) compositeWithCheckerboard() *image.RGBA {
	// Create a new image with checkerboard background
	composite := image.NewRGBA(image.Rect(0, 0, c.width, c.height))
	
	// Checkerboard square size
	squareSize := 20
	
	// Light and dark gray colors for checkerboard
	lightGray := color.RGBA{240, 240, 240, 255}
	darkGray := color.RGBA{200, 200, 200, 255}
	
	// Draw checkerboard background
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			// Determine checkerboard square
			squareX := x / squareSize
			squareY := y / squareSize
			
			var bgColor color.RGBA
			if (squareX+squareY)%2 == 0 {
				bgColor = lightGray
			} else {
				bgColor = darkGray
			}
			
			// Get the framebuffer pixel
			fbPixel := c.framebuffer.RGBAAt(x, y)
			
			// Alpha blend the framebuffer pixel over the checkerboard
			alpha := float64(fbPixel.A) / 255.0
			invAlpha := 1.0 - alpha
			
			finalR := uint8(float64(fbPixel.R)*alpha + float64(bgColor.R)*invAlpha)
			finalG := uint8(float64(fbPixel.G)*alpha + float64(bgColor.G)*invAlpha)
			finalB := uint8(float64(fbPixel.B)*alpha + float64(bgColor.B)*invAlpha)
			
			// Preserve the original alpha channel
			composite.Set(x, y, color.RGBA{finalR, finalG, finalB, fbPixel.A})
		}
	}
	
	return composite
}

// GetFramebuffer returns the current framebuffer for programmatic access
func (c *VNCClient) GetFramebuffer() *image.RGBA {
	return c.framebuffer
}

// GetPixel returns the color at the specified coordinates
func (c *VNCClient) GetPixel(x, y int) color.RGBA {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return color.RGBA{}
	}
	return c.framebuffer.RGBAAt(x, y)
}

func (c *VNCClient) createWebMAnimation() error {
	if len(c.capturedFrames) == 0 {
		return fmt.Errorf("no frames captured for WebM animation")
	}

	// For WebM, we'll need to use external tools like ffmpeg
	// For now, let's create a simple approach using individual PNGs and ffmpeg
	log.Printf("WebM creation requires ffmpeg. Use: ffmpeg -r %d -i %s/frame_%%04d.png -c:v libvpx-vp9 -pix_fmt yuva420p animation.webm", 
		c.frameRate, c.outputDir)
	
	return nil
}

func (c *VNCClient) createAPNGAnimation() error {
	if len(c.capturedFrames) == 0 {
		return fmt.Errorf("no frames captured for APNG animation")
	}

	filename := filepath.Join(c.outputDir, "animation.apng")
	
	// For APNG, we'll need to use external tools like apngasm
	// For now, let's save instructions and create a simple multi-frame PNG approach
	log.Printf("APNG creation with full transparency requires apngasm tool.")
	log.Printf("Use: apngasm %s %s/frame_*.png 1/%d", filename, c.outputDir, c.frameRate)
	log.Printf("Or install apngasm: brew install apngasm (macOS) or apt-get install apngasm (Linux)")
	
	// Alternative: Create a simple animated approach by saving all frames in sequence
	// This won't be a true APNG but will demonstrate the concept
	return c.createFrameSequenceFile()
}

func (c *VNCClient) createFrameSequenceFile() error {
	filename := filepath.Join(c.outputDir, "frame_sequence_info.txt")
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(file, "Animation Info:\n")
	fmt.Fprintf(file, "Total frames: %d\n", len(c.capturedFrames))
	fmt.Fprintf(file, "Frame rate: %d fps\n", c.frameRate)
	fmt.Fprintf(file, "Duration: %.2f seconds\n", float64(len(c.capturedFrames))/float64(c.frameRate))
	fmt.Fprintf(file, "Frame size: %dx%d\n", c.width, c.height)
	fmt.Fprintf(file, "\nTo create APNG: apngasm animation.apng frame_*.png 1/%d\n", c.frameRate)
	fmt.Fprintf(file, "To create WebM: ffmpeg -r %d -i frame_%%04d.png -c:v libvpx-vp9 -pix_fmt yuva420p animation.webm\n", c.frameRate)
	
	log.Printf("Created animation info file: %s", filename)
	return nil
}