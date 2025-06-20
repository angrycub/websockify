package rfb

import "testing"

func TestRFBConstants(t *testing.T) {
	// Test RFB protocol version string
	expected := "RFB 003.008\n"
	if RFBVersion != expected {
		t.Errorf("RFBVersion = %q, want %q", RFBVersion, expected)
	}

	// Test client-to-server message types per RFC 6143
	tests := []struct {
		name     string
		constant uint8
		expected uint8
	}{
		{"SetPixelFormat", SetPixelFormat, 0},
		{"SetEncodings", SetEncodings, 2},
		{"FramebufferUpdateRequest", FramebufferUpdateRequest, 3},
		{"KeyEvent", KeyEvent, 4},
		{"PointerEvent", PointerEvent, 5},
		{"ClientCutText", ClientCutText, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}

	// Test server-to-client message types per RFC 6143
	serverTests := []struct {
		name     string
		constant uint8
		expected uint8
	}{
		{"FramebufferUpdate", FramebufferUpdate, 0},
		{"SetColorMapEntries", SetColorMapEntries, 1},
		{"Bell", Bell, 2},
		{"ServerCutText", ServerCutText, 3},
	}

	for _, tt := range serverTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}

	// Test encoding types per RFC 6143
	if RawEncoding != 0 {
		t.Errorf("RawEncoding = %d, want %d", RawEncoding, 0)
	}

	// Test security types per RFC 6143
	if SecurityNone != 1 {
		t.Errorf("SecurityNone = %d, want %d", SecurityNone, 1)
	}

	// Test message length constants
	if SetPixelFormatLength != 20 {
		t.Errorf("SetPixelFormatLength = %d, want %d", SetPixelFormatLength, 20)
	}

	if ClientInitLength != 1 {
		t.Errorf("ClientInitLength = %d, want %d", ClientInitLength, 1)
	}
}