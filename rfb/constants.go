package rfb

const (
	RFBVersion = "RFB 003.008\n"

	// Client-to-server message types
	SetPixelFormat         = 0
	SetEncodings          = 2
	FramebufferUpdateRequest = 3
	KeyEvent              = 4
	PointerEvent          = 5
	ClientCutText         = 6

	// Server-to-client message types
	FramebufferUpdate     = 0
	SetColorMapEntries    = 1
	Bell                  = 2
	ServerCutText         = 3

	// Encoding types
	RawEncoding = 0

	// Security types
	SecurityNone = 1

	// Message lengths
	SetPixelFormatLength = 20
	ClientInitLength     = 1
)