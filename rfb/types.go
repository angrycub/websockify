package rfb

import "net"

// PixelFormat represents the RFB pixel format structure
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

// ServerInit represents the server initialization message
type ServerInit struct {
	Width       uint16
	Height      uint16
	PixelFormat PixelFormat
	NameLength  uint32
	Name        string
}

// Connection represents an RFB connection with common state
type Connection struct {
	Conn        net.Conn
	PixelFormat PixelFormat
	Width       int
	Height      int
}

// DefaultPixelFormat returns the standard 32bpp BGRA pixel format
func DefaultPixelFormat() PixelFormat {
	return PixelFormat{
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
}

// RGB565PixelFormat returns a 16bpp RGB565 pixel format for testing
func RGB565PixelFormat() PixelFormat {
	return PixelFormat{
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
		Padding:       [3]uint8{0, 0, 0},
	}
}