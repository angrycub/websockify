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

	"github.com/coder/websockify/rfb"
	"github.com/coder/websockify/version"
	"github.com/coder/websockify/viewer"
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
	serverPixelFormat rfb.PixelFormat // Server's pixel format from handshake
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
		showVersion    = flag.Bool("version", false, "Show version information")
		help           = flag.Bool("help", false, "Show this help message")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("vncclient %s\n", version.Version())
		os.Exit(0)
	}

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
		testFormat := rfb.RGB565PixelFormat()
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
	serverVersion, err := rfb.ReadRFBVersion(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read server version: %v", err)
	}
	log.Printf("Server version: %s", serverVersion)

	// Send client version
	if err := rfb.SendRFBVersion(c.conn); err != nil {
		return fmt.Errorf("failed to send client version: %v", err)
	}

	// Read security types
	securityTypes, err := rfb.ReadSecurityTypes(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read security types: %v", err)
	}
	log.Printf("Available security types: %v", securityTypes)

	// Choose security type (1 = None)
	securityChoice := uint8(rfb.SecurityNone)
	if err := binary.Write(c.conn, binary.BigEndian, securityChoice); err != nil {
		return fmt.Errorf("failed to send security choice: %v", err)
	}

	// Read security result
	securityResult, err := rfb.ReadSecurityResult(c.conn)
	if err != nil {
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
	serverInit, err := rfb.ReadServerInit(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read server init: %v", err)
	}

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
func (c *VNCClient) sendSetPixelFormat(pf rfb.PixelFormat) error {
	msg := rfb.CreateSetPixelFormat(pf)
	
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
	msg[0] = rfb.FramebufferUpdateRequest
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
	case rfb.FramebufferUpdate: // FramebufferUpdate
		return c.handleFramebufferUpdate()
	case rfb.SetColorMapEntries: // SetColorMapEntries
		log.Printf("Received SetColorMapEntries (not implemented)")
		return c.skipMessage(6) // Skip the rest of the message
	case rfb.Bell: // Bell
		log.Printf("Received Bell")
		return nil
	case rfb.ServerCutText: // ServerCutText
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
				rgba := rfb.ConvertPixelToRGBA(pixelData[pixelOffset:pixelOffset+bytesPerPixel], c.serverPixelFormat)
				c.framebuffer.Set(x+col, y+row, rgba)
			}
		}
	}

	return nil
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