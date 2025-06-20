package rfb

import (
	"net"
	"testing"
)

func TestDefaultPixelFormat(t *testing.T) {
	pf := DefaultPixelFormat()

	// Test standard 32bpp BGRA format per RFC 6143
	expected := PixelFormat{
		BitsPerPixel:  32,
		Depth:         24,
		BigEndianFlag: 0, // little-endian
		TrueColorFlag: 1,
		RedMax:        255,
		GreenMax:      255,
		BlueMax:       255,
		RedShift:      16,
		GreenShift:    8,
		BlueShift:     0,
		Padding:       [3]uint8{0, 0, 0},
	}

	if pf.BitsPerPixel != expected.BitsPerPixel {
		t.Errorf("BitsPerPixel = %d, want %d", pf.BitsPerPixel, expected.BitsPerPixel)
	}
	if pf.Depth != expected.Depth {
		t.Errorf("Depth = %d, want %d", pf.Depth, expected.Depth)
	}
	if pf.BigEndianFlag != expected.BigEndianFlag {
		t.Errorf("BigEndianFlag = %d, want %d", pf.BigEndianFlag, expected.BigEndianFlag)
	}
	if pf.TrueColorFlag != expected.TrueColorFlag {
		t.Errorf("TrueColorFlag = %d, want %d", pf.TrueColorFlag, expected.TrueColorFlag)
	}
	if pf.RedMax != expected.RedMax {
		t.Errorf("RedMax = %d, want %d", pf.RedMax, expected.RedMax)
	}
	if pf.GreenMax != expected.GreenMax {
		t.Errorf("GreenMax = %d, want %d", pf.GreenMax, expected.GreenMax)
	}
	if pf.BlueMax != expected.BlueMax {
		t.Errorf("BlueMax = %d, want %d", pf.BlueMax, expected.BlueMax)
	}
	if pf.RedShift != expected.RedShift {
		t.Errorf("RedShift = %d, want %d", pf.RedShift, expected.RedShift)
	}
	if pf.GreenShift != expected.GreenShift {
		t.Errorf("GreenShift = %d, want %d", pf.GreenShift, expected.GreenShift)
	}
	if pf.BlueShift != expected.BlueShift {
		t.Errorf("BlueShift = %d, want %d", pf.BlueShift, expected.BlueShift)
	}
}

func TestRGB565PixelFormat(t *testing.T) {
	pf := RGB565PixelFormat()

	// Test 16bpp RGB565 format
	expected := PixelFormat{
		BitsPerPixel:  16,
		Depth:         16,
		BigEndianFlag: 0, // little-endian
		TrueColorFlag: 1,
		RedMax:        31,  // 5 bits (2^5 - 1)
		GreenMax:      63,  // 6 bits (2^6 - 1)
		BlueMax:       31,  // 5 bits (2^5 - 1)
		RedShift:      11,  // bits 11-15
		GreenShift:    5,   // bits 5-10
		BlueShift:     0,   // bits 0-4
		Padding:       [3]uint8{0, 0, 0},
	}

	if pf.BitsPerPixel != expected.BitsPerPixel {
		t.Errorf("BitsPerPixel = %d, want %d", pf.BitsPerPixel, expected.BitsPerPixel)
	}
	if pf.RedMax != expected.RedMax {
		t.Errorf("RedMax = %d, want %d", pf.RedMax, expected.RedMax)
	}
	if pf.GreenMax != expected.GreenMax {
		t.Errorf("GreenMax = %d, want %d", pf.GreenMax, expected.GreenMax)
	}
	if pf.BlueMax != expected.BlueMax {
		t.Errorf("BlueMax = %d, want %d", pf.BlueMax, expected.BlueMax)
	}
	if pf.RedShift != expected.RedShift {
		t.Errorf("RedShift = %d, want %d", pf.RedShift, expected.RedShift)
	}
	if pf.GreenShift != expected.GreenShift {
		t.Errorf("GreenShift = %d, want %d", pf.GreenShift, expected.GreenShift)
	}
	if pf.BlueShift != expected.BlueShift {
		t.Errorf("BlueShift = %d, want %d", pf.BlueShift, expected.BlueShift)
	}
}

func TestServerInit(t *testing.T) {
	init := ServerInit{
		Width:       800,
		Height:      600,
		PixelFormat: DefaultPixelFormat(),
		Name:        "Test Server",
	}

	if init.Width != 800 {
		t.Errorf("Width = %d, want %d", init.Width, 800)
	}
	if init.Height != 600 {
		t.Errorf("Height = %d, want %d", init.Height, 600)
	}
	if init.Name != "Test Server" {
		t.Errorf("Name = %q, want %q", init.Name, "Test Server")
	}
}

func TestConnection(t *testing.T) {
	// Create a mock connection for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := Connection{
		Conn:        client,
		PixelFormat: DefaultPixelFormat(),
		Width:       800,
		Height:      600,
	}

	if conn.Width != 800 {
		t.Errorf("Width = %d, want %d", conn.Width, 800)
	}
	if conn.Height != 600 {
		t.Errorf("Height = %d, want %d", conn.Height, 600)
	}
	if conn.Conn == nil {
		t.Error("Conn should not be nil")
	}
}