package layout

import (
	"encoding/binary"
	"testing"
)

func TestICOToPNGExtractsEmbeddedPNG(t *testing.T) {
	t.Parallel()

	pngPayload := tinyPNG(8, 8)
	ico := buildTestPNGICO(pngPayload, 8, 8)

	out, width, height, ok := icoToPNG(ico)
	if !ok {
		t.Fatal("icoToPNG rejected a PNG-in-ICO payload")
	}

	if width != 8 || height != 8 {
		t.Fatalf("dims = %dx%d, want 8x8", width, height)
	}

	if string(out[:8]) != pngSig {
		t.Fatalf("output missing PNG signature: %q", out[:8])
	}
}

func TestICOToPNGExtractsBMP32(t *testing.T) {
	t.Parallel()

	ico := buildTestBMP32ICO(2, 2)
	out, width, height, ok := icoToPNG(ico)

	if !ok {
		t.Fatal("icoToPNG rejected a 32-bpp BMP-in-ICO payload")
	}

	if width != 2 || height != 2 {
		t.Fatalf("dims = %dx%d, want 2x2", width, height)
	}

	if string(out[:8]) != pngSig {
		t.Fatalf("output missing PNG signature: %q", out[:8])
	}
}

func TestICOToPNGRejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, _, _, ok := icoToPNG([]byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}); ok {
		t.Fatal("truncated ICO header was accepted")
	}

	if _, _, _, ok := icoToPNG([]byte("not an ico")); ok {
		t.Fatal("non-ICO bytes were accepted")
	}
}

// buildTestPNGICO wraps pngBytes in a single-entry Windows ICO.
func buildTestPNGICO(pngBytes []byte, width, height int) []byte {
	out := make([]byte, icoHeaderLen+icoEntryLen+len(pngBytes))
	binary.LittleEndian.PutUint16(out[0:2], 0)
	binary.LittleEndian.PutUint16(out[2:4], icoTypeIcon)
	binary.LittleEndian.PutUint16(out[4:6], 1)

	wByte := byte(width)
	hByte := byte(height)

	if width >= 256 {
		wByte = 0
	}

	if height >= 256 {
		hByte = 0
	}

	entry := out[icoHeaderLen : icoHeaderLen+icoEntryLen]
	entry[0] = wByte
	entry[1] = hByte
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngBytes))) //nolint:gosec // test-sized
	binary.LittleEndian.PutUint32(entry[12:16], uint32(icoHeaderLen+icoEntryLen))
	copy(out[icoHeaderLen+icoEntryLen:], pngBytes)

	return out
}

// buildTestBMP32ICO builds a 32-bpp BI_RGB ICO with a solid red XOR bitmap.
func buildTestBMP32ICO(width, height int) []byte {
	rowStride := (width*4 + 3) &^ 3
	xorBytes := rowStride * height
	andRow := ((width + 31) / 32) * 4
	andBytes := andRow * height
	dib := make([]byte, bmpInfoLen+xorBytes+andBytes)

	binary.LittleEndian.PutUint32(dib[0:4], bmpInfoLen)
	binary.LittleEndian.PutUint32(dib[4:8], uint32(width))     //nolint:gosec // test-sized
	binary.LittleEndian.PutUint32(dib[8:12], uint32(height*2)) //nolint:gosec // XOR+AND
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], bmpBitCount32)

	for y := range height {
		for x := range width {
			off := bmpInfoLen + y*rowStride + x*4
			dib[off+0] = 0   // B
			dib[off+1] = 0   // G
			dib[off+2] = 255 // R
			dib[off+3] = 255 // A
		}
	}

	out := make([]byte, icoHeaderLen+icoEntryLen+len(dib))
	binary.LittleEndian.PutUint16(out[0:2], 0)
	binary.LittleEndian.PutUint16(out[2:4], icoTypeIcon)
	binary.LittleEndian.PutUint16(out[4:6], 1)

	entry := out[icoHeaderLen : icoHeaderLen+icoEntryLen]
	entry[0] = byte(width)
	entry[1] = byte(height)
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(dib))) //nolint:gosec // test-sized
	binary.LittleEndian.PutUint32(entry[12:16], uint32(icoHeaderLen+icoEntryLen))
	copy(out[icoHeaderLen+icoEntryLen:], dib)

	return out
}
