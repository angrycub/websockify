package rfb

import (
	"image/color"
	"testing"
)

func TestConvertPixelFormat(t *testing.T) {
	// Test data: 2x2 pixel BGRA image (8 bytes total)
	// Each pixel is 4 bytes: B, G, R, A
	bgraData := []byte{
		255, 0, 0, 255,   // Blue pixel (B=255, G=0, R=0, A=255)
		0, 255, 0, 255,   // Green pixel (B=0, G=255, R=0, A=255)
		0, 0, 255, 255,   // Red pixel (B=0, G=0, R=255, A=255)
		128, 128, 128, 255, // Gray pixel (B=128, G=128, R=128, A=255)
	}

	tests := []struct {
		name         string
		targetFormat PixelFormat
		width        int
		height       int
		expectLength int
	}{
		{
			name:         "Default 32bpp format (no conversion)",
			targetFormat: DefaultPixelFormat(),
			width:        2,
			height:       2,
			expectLength: 16, // 2x2 pixels * 4 bytes per pixel
		},
		{
			name:         "RGB565 16bpp format",
			targetFormat: RGB565PixelFormat(),
			width:        2,
			height:       2,
			expectLength: 8, // 2x2 pixels * 2 bytes per pixel
		},
		{
			name: "8bpp format",
			targetFormat: PixelFormat{
				BitsPerPixel:  8,
				Depth:         8,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        7,   // 3 bits
				GreenMax:      7,   // 3 bits
				BlueMax:       3,   // 2 bits
				RedShift:      5,   // bits 5-7
				GreenShift:    2,   // bits 2-4
				BlueShift:     0,   // bits 0-1
			},
			width:        2,
			height:       2,
			expectLength: 4, // 2x2 pixels * 1 byte per pixel
		},
		{
			name: "24bpp format",
			targetFormat: PixelFormat{
				BitsPerPixel:  24,
				Depth:         24,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        255,
				GreenMax:      255,
				BlueMax:       255,
				RedShift:      16,
				GreenShift:    8,
				BlueShift:     0,
			},
			width:        2,
			height:       2,
			expectLength: 12, // 2x2 pixels * 3 bytes per pixel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertPixelFormat(bgraData, tt.width, tt.height, tt.targetFormat)
			
			if len(result) != tt.expectLength {
				t.Errorf("ConvertPixelFormat() length = %d, want %d", len(result), tt.expectLength)
			}

			// For default format, should return same data
			if IsDefaultPixelFormat(tt.targetFormat) {
				for i := range result {
					if result[i] != bgraData[i] {
						t.Errorf("Default format conversion byte %d = %d, want %d", i, result[i], bgraData[i])
						break
					}
				}
			}
		})
	}
}

func TestWritePixelValue(t *testing.T) {
	tests := []struct {
		name      string
		bufSize   int
		value     uint32
		bigEndian uint8
		expected  []byte
	}{
		{
			name:      "8bpp little endian",
			bufSize:   1,
			value:     0xAB,
			bigEndian: 0,
			expected:  []byte{0xAB},
		},
		{
			name:      "16bpp little endian",
			bufSize:   2,
			value:     0xABCD,
			bigEndian: 0,
			expected:  []byte{0xCD, 0xAB}, // little endian: LSB first
		},
		{
			name:      "16bpp big endian",
			bufSize:   2,
			value:     0xABCD,
			bigEndian: 1,
			expected:  []byte{0xAB, 0xCD}, // big endian: MSB first
		},
		{
			name:      "24bpp little endian",
			bufSize:   3,
			value:     0xABCDEF,
			bigEndian: 0,
			expected:  []byte{0xEF, 0xCD, 0xAB}, // little endian: LSB first
		},
		{
			name:      "24bpp big endian",
			bufSize:   3,
			value:     0xABCDEF,
			bigEndian: 1,
			expected:  []byte{0xAB, 0xCD, 0xEF}, // big endian: MSB first
		},
		{
			name:      "32bpp little endian",
			bufSize:   4,
			value:     0x12345678,
			bigEndian: 0,
			expected:  []byte{0x78, 0x56, 0x34, 0x12}, // little endian: LSB first
		},
		{
			name:      "32bpp big endian",
			bufSize:   4,
			value:     0x12345678,
			bigEndian: 1,
			expected:  []byte{0x12, 0x34, 0x56, 0x78}, // big endian: MSB first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := make([]byte, tt.bufSize)
			WritePixelValue(buffer, tt.value, tt.bigEndian)
			
			for i, expected := range tt.expected {
				if buffer[i] != expected {
					t.Errorf("WritePixelValue() byte %d = 0x%02X, want 0x%02X", i, buffer[i], expected)
				}
			}
		})
	}
}

func TestReadPixelValue(t *testing.T) {
	tests := []struct {
		name      string
		buffer    []byte
		bigEndian uint8
		expected  uint32
	}{
		{
			name:      "8bpp",
			buffer:    []byte{0xAB},
			bigEndian: 0,
			expected:  0xAB,
		},
		{
			name:      "16bpp little endian",
			buffer:    []byte{0xCD, 0xAB}, // LSB first
			bigEndian: 0,
			expected:  0xABCD,
		},
		{
			name:      "16bpp big endian",
			buffer:    []byte{0xAB, 0xCD}, // MSB first
			bigEndian: 1,
			expected:  0xABCD,
		},
		{
			name:      "24bpp little endian",
			buffer:    []byte{0xEF, 0xCD, 0xAB}, // LSB first
			bigEndian: 0,
			expected:  0xABCDEF,
		},
		{
			name:      "24bpp big endian",
			buffer:    []byte{0xAB, 0xCD, 0xEF}, // MSB first
			bigEndian: 1,
			expected:  0xABCDEF,
		},
		{
			name:      "32bpp little endian",
			buffer:    []byte{0x78, 0x56, 0x34, 0x12}, // LSB first
			bigEndian: 0,
			expected:  0x12345678,
		},
		{
			name:      "32bpp big endian",
			buffer:    []byte{0x12, 0x34, 0x56, 0x78}, // MSB first
			bigEndian: 1,
			expected:  0x12345678,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReadPixelValue(tt.buffer, tt.bigEndian)
			if result != tt.expected {
				t.Errorf("ReadPixelValue() = 0x%08X, want 0x%08X", result, tt.expected)
			}
		})
	}
}

func TestConvertPixelToRGBA(t *testing.T) {
	tests := []struct {
		name        string
		pixelBytes  []byte
		pixelFormat PixelFormat
		expected    color.RGBA
	}{
		{
			name:       "32bpp BGRA to RGBA",
			pixelBytes: []byte{255, 0, 0, 255}, // Blue pixel in BGRA
			pixelFormat: PixelFormat{
				BitsPerPixel:  32,
				Depth:         24,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        255,
				GreenMax:      255,
				BlueMax:       255,
				RedShift:      16,
				GreenShift:    8,
				BlueShift:     0,
			},
			expected: color.RGBA{R: 0, G: 0, B: 255, A: 255}, // Blue pixel
		},
		{
			name:       "16bpp RGB565 red pixel",
			pixelBytes: []byte{0x00, 0xF8}, // Red in RGB565 little-endian
			pixelFormat: PixelFormat{
				BitsPerPixel:  16,
				Depth:         16,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        31,
				GreenMax:      63,
				BlueMax:       31,
				RedShift:      11,
				GreenShift:    5,
				BlueShift:     0,
			},
			expected: color.RGBA{R: 255, G: 0, B: 0, A: 255}, // Red pixel
		},
		{
			name:       "16bpp RGB565 green pixel",
			pixelBytes: []byte{0xE0, 0x07}, // Green in RGB565 little-endian
			pixelFormat: PixelFormat{
				BitsPerPixel:  16,
				Depth:         16,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        31,
				GreenMax:      63,
				BlueMax:       31,
				RedShift:      11,
				GreenShift:    5,
				BlueShift:     0,
			},
			expected: color.RGBA{R: 0, G: 255, B: 0, A: 255}, // Green pixel
		},
		{
			name:       "8bpp palette color",
			pixelBytes: []byte{0xFF}, // Max value for 8bpp
			pixelFormat: PixelFormat{
				BitsPerPixel:  8,
				Depth:         8,
				BigEndianFlag: 0,
				TrueColorFlag: 1,
				RedMax:        7,
				GreenMax:      7,
				BlueMax:       3,
				RedShift:      5,
				GreenShift:    2,
				BlueShift:     0,
			},
			expected: color.RGBA{R: 255, G: 255, B: 255, A: 255}, // White pixel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertPixelToRGBA(tt.pixelBytes, tt.pixelFormat)
			
			if result.R != tt.expected.R {
				t.Errorf("Red = %d, want %d", result.R, tt.expected.R)
			}
			if result.G != tt.expected.G {
				t.Errorf("Green = %d, want %d", result.G, tt.expected.G)
			}
			if result.B != tt.expected.B {
				t.Errorf("Blue = %d, want %d", result.B, tt.expected.B)
			}
			if result.A != tt.expected.A {
				t.Errorf("Alpha = %d, want %d", result.A, tt.expected.A)
			}
		})
	}
}

func TestIsDefaultPixelFormat(t *testing.T) {
	defaultPF := DefaultPixelFormat()
	
	if !IsDefaultPixelFormat(defaultPF) {
		t.Error("IsDefaultPixelFormat() should return true for DefaultPixelFormat()")
	}

	rgb565PF := RGB565PixelFormat()
	if IsDefaultPixelFormat(rgb565PF) {
		t.Error("IsDefaultPixelFormat() should return false for RGB565PixelFormat()")
	}

	// Test with modified default format
	modifiedPF := defaultPF
	modifiedPF.RedMax = 127 // Change one field
	if IsDefaultPixelFormat(modifiedPF) {
		t.Error("IsDefaultPixelFormat() should return false for modified pixel format")
	}
}

// Test round-trip conversion: BGRA -> target format -> BGRA
func TestPixelFormatRoundTrip(t *testing.T) {
	// Test with RGB565 format
	originalBGRA := []byte{255, 128, 64, 255} // Orange-ish color
	
	// Convert to RGB565
	rgb565Data := ConvertPixelFormat(originalBGRA, 1, 1, RGB565PixelFormat())
	
	// Convert back to RGBA and check if values are close
	rgba := ConvertPixelToRGBA(rgb565Data, RGB565PixelFormat())
	
	// Due to precision loss in RGB565, values won't be exact
	// Check if they're reasonably close (within expected precision loss)
	tolerance := uint8(8) // Allow some precision loss
	
	if abs(rgba.B, 255) > tolerance {
		t.Errorf("Blue round-trip error too large: got %d, want near 255", rgba.B)
	}
	if abs(rgba.G, 128) > tolerance {
		t.Errorf("Green round-trip error too large: got %d, want near 128", rgba.G)
	}
	if abs(rgba.R, 64) > tolerance {
		t.Errorf("Red round-trip error too large: got %d, want near 64", rgba.R)
	}
}

// Helper function for absolute difference
func abs(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}