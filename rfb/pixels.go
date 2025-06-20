package rfb

import "image/color"

// ConvertPixelFormat converts pixel data from one pixel format to another
func ConvertPixelFormat(bgraData []byte, width, height int, targetFormat PixelFormat) []byte {
	// If target format matches our default (32bpp BGRA), no conversion needed
	if IsDefaultPixelFormat(targetFormat) {
		return bgraData
	}

	pixelCount := width * height
	bytesPerPixel := int(targetFormat.BitsPerPixel) / 8
	outputData := make([]byte, pixelCount*bytesPerPixel)

	for i := 0; i < pixelCount; i++ {
		// Extract BGRA components from input
		srcOffset := i * 4
		b := uint16(bgraData[srcOffset])
		g := uint16(bgraData[srcOffset+1])
		r := uint16(bgraData[srcOffset+2])
		// a := uint16(bgraData[srcOffset+3]) // Alpha not used in conversion

		// Scale color components to target maximums
		scaledR := (r * uint16(targetFormat.RedMax)) / 255
		scaledG := (g * uint16(targetFormat.GreenMax)) / 255
		scaledB := (b * uint16(targetFormat.BlueMax)) / 255

		// Combine into target pixel value
		pixelValue := uint32(scaledR)<<targetFormat.RedShift |
			uint32(scaledG)<<targetFormat.GreenShift |
			uint32(scaledB)<<targetFormat.BlueShift

		// Write pixel in target format
		dstOffset := i * bytesPerPixel
		WritePixelValue(outputData[dstOffset:dstOffset+bytesPerPixel], pixelValue, targetFormat.BigEndianFlag)
	}

	return outputData
}

// WritePixelValue writes a pixel value to the buffer in the specified endianness
func WritePixelValue(buffer []byte, value uint32, bigEndian uint8) {
	switch len(buffer) {
	case 1: // 8 bits per pixel
		buffer[0] = uint8(value)
	case 2: // 16 bits per pixel
		if bigEndian == 1 {
			buffer[0] = uint8(value >> 8)
			buffer[1] = uint8(value)
		} else {
			buffer[0] = uint8(value)
			buffer[1] = uint8(value >> 8)
		}
	case 3: // 24 bits per pixel
		if bigEndian == 1 {
			buffer[0] = uint8(value >> 16)
			buffer[1] = uint8(value >> 8)
			buffer[2] = uint8(value)
		} else {
			buffer[0] = uint8(value)
			buffer[1] = uint8(value >> 8)
			buffer[2] = uint8(value >> 16)
		}
	case 4: // 32 bits per pixel
		if bigEndian == 1 {
			buffer[0] = uint8(value >> 24)
			buffer[1] = uint8(value >> 16)
			buffer[2] = uint8(value >> 8)
			buffer[3] = uint8(value)
		} else {
			buffer[0] = uint8(value)
			buffer[1] = uint8(value >> 8)
			buffer[2] = uint8(value >> 16)
			buffer[3] = uint8(value >> 24)
		}
	}
}

// ReadPixelValue reads a pixel value from the buffer considering endianness
func ReadPixelValue(buffer []byte, bigEndian uint8) uint32 {
	var pixelValue uint32
	switch len(buffer) {
	case 1: // 8 bits per pixel
		pixelValue = uint32(buffer[0])
	case 2: // 16 bits per pixel
		if bigEndian == 1 {
			pixelValue = uint32(buffer[0])<<8 | uint32(buffer[1])
		} else {
			pixelValue = uint32(buffer[1])<<8 | uint32(buffer[0])
		}
	case 3: // 24 bits per pixel
		if bigEndian == 1 {
			pixelValue = uint32(buffer[0])<<16 | uint32(buffer[1])<<8 | uint32(buffer[2])
		} else {
			pixelValue = uint32(buffer[2])<<16 | uint32(buffer[1])<<8 | uint32(buffer[0])
		}
	case 4: // 32 bits per pixel
		if bigEndian == 1 {
			pixelValue = uint32(buffer[0])<<24 | uint32(buffer[1])<<16 | uint32(buffer[2])<<8 | uint32(buffer[3])
		} else {
			pixelValue = uint32(buffer[3])<<24 | uint32(buffer[2])<<16 | uint32(buffer[1])<<8 | uint32(buffer[0])
		}
	}
	return pixelValue
}

// ConvertPixelToRGBA converts a pixel from the server's format to RGBA
func ConvertPixelToRGBA(pixelBytes []byte, pf PixelFormat) color.RGBA {
	// Read pixel value from bytes considering endianness
	pixelValue := ReadPixelValue(pixelBytes, pf.BigEndianFlag)

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

// IsDefaultPixelFormat checks if a pixel format matches the default 32bpp BGRA format
func IsDefaultPixelFormat(pf PixelFormat) bool {
	defaultPF := DefaultPixelFormat()
	return pf.BitsPerPixel == defaultPF.BitsPerPixel &&
		pf.Depth == defaultPF.Depth &&
		pf.BigEndianFlag == defaultPF.BigEndianFlag &&
		pf.TrueColorFlag == defaultPF.TrueColorFlag &&
		pf.RedMax == defaultPF.RedMax &&
		pf.GreenMax == defaultPF.GreenMax &&
		pf.BlueMax == defaultPF.BlueMax &&
		pf.RedShift == defaultPF.RedShift &&
		pf.GreenShift == defaultPF.GreenShift &&
		pf.BlueShift == defaultPF.BlueShift
}