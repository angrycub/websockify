package rfb

import (
	"fmt"
	"io"
	"net"
)

// GetMessageLength calculates the expected length of a VNC message based on its type
func GetMessageLength(messageType byte, data []byte) (int, error) {
	switch messageType {
	case SetPixelFormat:
		return SetPixelFormatLength, nil
	case SetEncodings:
		if len(data) < 4 {
			return 0, fmt.Errorf("insufficient data for SetEncodings message")
		}
		numEncodings := (int(data[2]) << 8) | int(data[3])
		return 4 + numEncodings*4, nil
	case FramebufferUpdateRequest:
		return 10, nil
	case KeyEvent:
		return 8, nil
	case PointerEvent:
		return 6, nil
	case ClientCutText:
		if len(data) < 8 {
			return 0, fmt.Errorf("insufficient data for ClientCutText message")
		}
		textLength := (int(data[4]) << 24) | (int(data[5]) << 16) | (int(data[6]) << 8) | int(data[7])
		return 8 + textLength, nil
	default:
		return 0, fmt.Errorf("unknown message type: %d", messageType)
	}
}

// ParseSetPixelFormat parses a SetPixelFormat message from raw bytes
func ParseSetPixelFormat(data []byte) (PixelFormat, error) {
	if len(data) != SetPixelFormatLength {
		return PixelFormat{}, fmt.Errorf("SetPixelFormat message must be exactly %d bytes, got %d", SetPixelFormatLength, len(data))
	}

	// Parse pixel format from bytes 4-19 (skip message type byte 0 and 3 padding bytes)
	pf := PixelFormat{
		BitsPerPixel:  data[4],  // byte 4
		Depth:         data[5],  // byte 5
		BigEndianFlag: data[6],  // byte 6
		TrueColorFlag: data[7],  // byte 7
		RedMax:        uint16(data[8])<<8 | uint16(data[9]),    // bytes 8-9
		GreenMax:      uint16(data[10])<<8 | uint16(data[11]),  // bytes 10-11
		BlueMax:       uint16(data[12])<<8 | uint16(data[13]),  // bytes 12-13
		RedShift:      data[14], // byte 14
		GreenShift:    data[15], // byte 15
		BlueShift:     data[16], // byte 16
		Padding:       [3]uint8{data[17], data[18], data[19]}, // bytes 17-19
	}

	return pf, nil
}

// CreateSetPixelFormat creates a SetPixelFormat message from a PixelFormat
func CreateSetPixelFormat(pf PixelFormat) []byte {
	msg := make([]byte, SetPixelFormatLength)

	// Message type (0 = SetPixelFormat)
	msg[0] = SetPixelFormat

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
	msg[17] = pf.Padding[0]
	msg[18] = pf.Padding[1]
	msg[19] = pf.Padding[2]

	return msg
}

// SendRFBVersion sends the RFB protocol version
func SendRFBVersion(conn net.Conn) error {
	_, err := conn.Write([]byte(RFBVersion))
	return err
}

// ReadRFBVersion reads and returns the RFB protocol version
func ReadRFBVersion(conn net.Conn) (string, error) {
	version := make([]byte, len(RFBVersion))
	if _, err := io.ReadFull(conn, version); err != nil {
		return "", err
	}
	return string(version), nil
}

// SendSecurityTypes sends the list of supported security types
func SendSecurityTypes(conn net.Conn, types []uint8) error {
	msg := make([]byte, 1+len(types))
	msg[0] = uint8(len(types))
	copy(msg[1:], types)
	_, err := conn.Write(msg)
	return err
}

// ReadSecurityTypes reads the list of supported security types
func ReadSecurityTypes(conn net.Conn) ([]uint8, error) {
	var numTypes uint8
	if err := readByte(conn, &numTypes); err != nil {
		return nil, err
	}
	
	if numTypes == 0 {
		return nil, fmt.Errorf("server sent no security types")
	}
	
	types := make([]uint8, numTypes)
	if _, err := io.ReadFull(conn, types); err != nil {
		return nil, err
	}
	
	return types, nil
}

// SendSecurityResult sends the security handshake result
func SendSecurityResult(conn net.Conn, result uint32) error {
	msg := make([]byte, 4)
	msg[0] = uint8(result >> 24)
	msg[1] = uint8(result >> 16)
	msg[2] = uint8(result >> 8)
	msg[3] = uint8(result)
	_, err := conn.Write(msg)
	return err
}

// ReadSecurityResult reads the security handshake result
func ReadSecurityResult(conn net.Conn) (uint32, error) {
	result := make([]byte, 4)
	if _, err := io.ReadFull(conn, result); err != nil {
		return 0, err
	}
	return uint32(result[0])<<24 | uint32(result[1])<<16 | uint32(result[2])<<8 | uint32(result[3]), nil
}

// SendServerInit sends the server initialization message
func SendServerInit(conn net.Conn, init ServerInit) error {
	msg := make([]byte, 24+len(init.Name))
	
	// Width and height (big-endian 16-bit)
	msg[0] = uint8(init.Width >> 8)
	msg[1] = uint8(init.Width & 0xFF)
	msg[2] = uint8(init.Height >> 8)
	msg[3] = uint8(init.Height & 0xFF)
	
	// Pixel format (16 bytes)
	msg[4] = init.PixelFormat.BitsPerPixel
	msg[5] = init.PixelFormat.Depth
	msg[6] = init.PixelFormat.BigEndianFlag
	msg[7] = init.PixelFormat.TrueColorFlag
	msg[8] = uint8(init.PixelFormat.RedMax >> 8)
	msg[9] = uint8(init.PixelFormat.RedMax & 0xFF)
	msg[10] = uint8(init.PixelFormat.GreenMax >> 8)
	msg[11] = uint8(init.PixelFormat.GreenMax & 0xFF)
	msg[12] = uint8(init.PixelFormat.BlueMax >> 8)
	msg[13] = uint8(init.PixelFormat.BlueMax & 0xFF)
	msg[14] = init.PixelFormat.RedShift
	msg[15] = init.PixelFormat.GreenShift
	msg[16] = init.PixelFormat.BlueShift
	msg[17] = init.PixelFormat.Padding[0]
	msg[18] = init.PixelFormat.Padding[1]
	msg[19] = init.PixelFormat.Padding[2]
	
	// Name length (big-endian 32-bit)
	nameLen := uint32(len(init.Name))
	msg[20] = uint8(nameLen >> 24)
	msg[21] = uint8(nameLen >> 16)
	msg[22] = uint8(nameLen >> 8)
	msg[23] = uint8(nameLen & 0xFF)
	
	// Name
	copy(msg[24:], init.Name)
	
	_, err := conn.Write(msg)
	return err
}

// ReadServerInit reads the server initialization message
func ReadServerInit(conn net.Conn) (ServerInit, error) {
	var init ServerInit
	header := make([]byte, 24)
	
	if _, err := io.ReadFull(conn, header); err != nil {
		return init, err
	}
	
	// Parse width and height
	init.Width = uint16(header[0])<<8 | uint16(header[1])
	init.Height = uint16(header[2])<<8 | uint16(header[3])
	
	// Parse pixel format
	init.PixelFormat = PixelFormat{
		BitsPerPixel:  header[4],
		Depth:         header[5],
		BigEndianFlag: header[6],
		TrueColorFlag: header[7],
		RedMax:        uint16(header[8])<<8 | uint16(header[9]),
		GreenMax:      uint16(header[10])<<8 | uint16(header[11]),
		BlueMax:       uint16(header[12])<<8 | uint16(header[13]),
		RedShift:      header[14],
		GreenShift:    header[15],
		BlueShift:     header[16],
		Padding:       [3]uint8{header[17], header[18], header[19]},
	}
	
	// Parse name length
	nameLen := uint32(header[20])<<24 | uint32(header[21])<<16 | uint32(header[22])<<8 | uint32(header[23])
	
	// Read name
	if nameLen > 0 {
		nameBytes := make([]byte, nameLen)
		if _, err := io.ReadFull(conn, nameBytes); err != nil {
			return init, err
		}
		init.Name = string(nameBytes)
	}
	
	init.NameLength = nameLen
	return init, nil
}

// Helper function to read a single byte
func readByte(conn net.Conn, b *uint8) error {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	*b = buf[0]
	return nil
}