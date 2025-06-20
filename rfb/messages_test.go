package rfb

import (
	"net"
	"testing"
)

func TestGetMessageLength(t *testing.T) {
	tests := []struct {
		name        string
		messageType byte
		data        []byte
		expected    int
		expectError bool
	}{
		{
			name:        "SetPixelFormat",
			messageType: SetPixelFormat,
			data:        make([]byte, 20),
			expected:    SetPixelFormatLength,
			expectError: false,
		},
		{
			name:        "SetEncodings with 2 encodings",
			messageType: SetEncodings,
			data:        []byte{2, 0, 0, 2}, // 2 encodings
			expected:    4 + 2*4,           // header + 2 * 4 bytes per encoding
			expectError: false,
		},
		{
			name:        "SetEncodings insufficient data",
			messageType: SetEncodings,
			data:        []byte{2, 0}, // Not enough data to read encoding count
			expected:    0,
			expectError: true,
		},
		{
			name:        "FramebufferUpdateRequest",
			messageType: FramebufferUpdateRequest,
			data:        make([]byte, 10),
			expected:    10,
			expectError: false,
		},
		{
			name:        "KeyEvent",
			messageType: KeyEvent,
			data:        make([]byte, 8),
			expected:    8,
			expectError: false,
		},
		{
			name:        "PointerEvent",
			messageType: PointerEvent,
			data:        make([]byte, 6),
			expected:    6,
			expectError: false,
		},
		{
			name:        "ClientCutText with 10 bytes text",
			messageType: ClientCutText,
			data:        []byte{6, 0, 0, 0, 0, 0, 0, 10}, // 10 bytes of text
			expected:    8 + 10,                            // header + text length
			expectError: false,
		},
		{
			name:        "ClientCutText insufficient data",
			messageType: ClientCutText,
			data:        []byte{6, 0, 0}, // Not enough data to read text length
			expected:    0,
			expectError: true,
		},
		{
			name:        "Unknown message type",
			messageType: 255,
			data:        []byte{},
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, err := GetMessageLength(tt.messageType, tt.data)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if length != tt.expected {
				t.Errorf("GetMessageLength() = %d, want %d", length, tt.expected)
			}
		})
	}
}

func TestParseSetPixelFormat(t *testing.T) {
	// Create a valid SetPixelFormat message per RFC 6143
	data := make([]byte, 20)
	data[0] = SetPixelFormat // message type
	// 3 bytes padding (bytes 1-3)
	data[1] = 0
	data[2] = 0
	data[3] = 0
	// Pixel format (16 bytes starting at byte 4)
	data[4] = 32  // bits-per-pixel
	data[5] = 24  // depth
	data[6] = 0   // big-endian-flag (little-endian)
	data[7] = 1   // true-colour-flag
	data[8] = 0   // red-max high byte
	data[9] = 255 // red-max low byte (255)
	data[10] = 0  // green-max high byte
	data[11] = 255 // green-max low byte (255)
	data[12] = 0  // blue-max high byte
	data[13] = 255 // blue-max low byte (255)
	data[14] = 16 // red-shift
	data[15] = 8  // green-shift
	data[16] = 0  // blue-shift
	data[17] = 0  // padding
	data[18] = 0  // padding
	data[19] = 0  // padding

	pf, err := ParseSetPixelFormat(data)
	if err != nil {
		t.Fatalf("ParseSetPixelFormat() error = %v", err)
	}

	expected := PixelFormat{
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
		Padding:       [3]uint8{0, 0, 0},
	}

	if pf.BitsPerPixel != expected.BitsPerPixel {
		t.Errorf("BitsPerPixel = %d, want %d", pf.BitsPerPixel, expected.BitsPerPixel)
	}
	if pf.Depth != expected.Depth {
		t.Errorf("Depth = %d, want %d", pf.Depth, expected.Depth)
	}
	if pf.RedMax != expected.RedMax {
		t.Errorf("RedMax = %d, want %d", pf.RedMax, expected.RedMax)
	}

	// Test invalid message length
	shortData := make([]byte, 19)
	_, err = ParseSetPixelFormat(shortData)
	if err == nil {
		t.Error("Expected error for short message, but got none")
	}
}

func TestCreateSetPixelFormat(t *testing.T) {
	pf := PixelFormat{
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
		Padding:       [3]uint8{0, 0, 0},
	}

	msg := CreateSetPixelFormat(pf)

	if len(msg) != SetPixelFormatLength {
		t.Errorf("Message length = %d, want %d", len(msg), SetPixelFormatLength)
	}

	if msg[0] != SetPixelFormat {
		t.Errorf("Message type = %d, want %d", msg[0], SetPixelFormat)
	}

	// Check padding bytes 1-3
	for i := 1; i <= 3; i++ {
		if msg[i] != 0 {
			t.Errorf("Padding byte %d = %d, want 0", i, msg[i])
		}
	}

	// Check pixel format fields
	if msg[4] != pf.BitsPerPixel {
		t.Errorf("BitsPerPixel = %d, want %d", msg[4], pf.BitsPerPixel)
	}
	if msg[5] != pf.Depth {
		t.Errorf("Depth = %d, want %d", msg[5], pf.Depth)
	}

	// Check color maximums (big-endian 16-bit values)
	redMax := uint16(msg[8])<<8 | uint16(msg[9])
	if redMax != pf.RedMax {
		t.Errorf("RedMax = %d, want %d", redMax, pf.RedMax)
	}

	greenMax := uint16(msg[10])<<8 | uint16(msg[11])
	if greenMax != pf.GreenMax {
		t.Errorf("GreenMax = %d, want %d", greenMax, pf.GreenMax)
	}

	blueMax := uint16(msg[12])<<8 | uint16(msg[13])
	if blueMax != pf.BlueMax {
		t.Errorf("BlueMax = %d, want %d", blueMax, pf.BlueMax)
	}
}

func TestRFBVersionHandshake(t *testing.T) {
	// Test sending and receiving RFB version
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Test sending RFB version
	go func() {
		err := SendRFBVersion(server)
		if err != nil {
			t.Errorf("SendRFBVersion() error = %v", err)
		}
	}()

	// Test receiving RFB version
	version, err := ReadRFBVersion(client)
	if err != nil {
		t.Fatalf("ReadRFBVersion() error = %v", err)
	}

	if version != RFBVersion {
		t.Errorf("ReadRFBVersion() = %q, want %q", version, RFBVersion)
	}
}

func TestSecurityHandshake(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Test sending security types
	go func() {
		err := SendSecurityTypes(server, []uint8{SecurityNone})
		if err != nil {
			t.Errorf("SendSecurityTypes() error = %v", err)
		}
	}()

	// Test receiving security types
	types, err := ReadSecurityTypes(client)
	if err != nil {
		t.Fatalf("ReadSecurityTypes() error = %v", err)
	}

	if len(types) != 1 || types[0] != SecurityNone {
		t.Errorf("ReadSecurityTypes() = %v, want [%d]", types, SecurityNone)
	}
}

func TestSecurityResult(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Test sending security result
	go func() {
		err := SendSecurityResult(server, 0) // Success
		if err != nil {
			t.Errorf("SendSecurityResult() error = %v", err)
		}
	}()

	// Test receiving security result
	result, err := ReadSecurityResult(client)
	if err != nil {
		t.Fatalf("ReadSecurityResult() error = %v", err)
	}

	if result != 0 {
		t.Errorf("ReadSecurityResult() = %d, want 0", result)
	}
}

func TestServerInitHandshake(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	expected := ServerInit{
		Width:       800,
		Height:      600,
		PixelFormat: DefaultPixelFormat(),
		Name:        "Test Server",
	}

	// Test sending ServerInit
	go func() {
		err := SendServerInit(server, expected)
		if err != nil {
			t.Errorf("SendServerInit() error = %v", err)
		}
	}()

	// Test receiving ServerInit
	received, err := ReadServerInit(client)
	if err != nil {
		t.Fatalf("ReadServerInit() error = %v", err)
	}

	if received.Width != expected.Width {
		t.Errorf("Width = %d, want %d", received.Width, expected.Width)
	}
	if received.Height != expected.Height {
		t.Errorf("Height = %d, want %d", received.Height, expected.Height)
	}
	if received.Name != expected.Name {
		t.Errorf("Name = %q, want %q", received.Name, expected.Name)
	}
	if received.PixelFormat.BitsPerPixel != expected.PixelFormat.BitsPerPixel {
		t.Errorf("BitsPerPixel = %d, want %d", received.PixelFormat.BitsPerPixel, expected.PixelFormat.BitsPerPixel)
	}
}

// Test unimplemented message types that should be added later
func TestUnimplementedMessages(t *testing.T) {
	t.Run("CopyRect encoding", func(t *testing.T) {
		t.Skip("CopyRect encoding not yet implemented")
	})

	t.Run("RRE encoding", func(t *testing.T) {
		t.Skip("RRE encoding not yet implemented")
	})

	t.Run("Hextile encoding", func(t *testing.T) {
		t.Skip("Hextile encoding not yet implemented")
	})

	t.Run("TRLE encoding", func(t *testing.T) {
		t.Skip("TRLE encoding not yet implemented")
	})

	t.Run("ZRLE encoding", func(t *testing.T) {
		t.Skip("ZRLE encoding not yet implemented")
	})

	t.Run("VNC Authentication", func(t *testing.T) {
		t.Skip("VNC Authentication not yet implemented")
	})

	t.Run("Cursor pseudo-encoding", func(t *testing.T) {
		t.Skip("Cursor pseudo-encoding not yet implemented")
	})

	t.Run("DesktopSize pseudo-encoding", func(t *testing.T) {
		t.Skip("DesktopSize pseudo-encoding not yet implemented")
	})
}