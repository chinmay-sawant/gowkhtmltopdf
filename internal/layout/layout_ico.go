package layout

import (
	"encoding/binary"
	"image"
)

const (
	icoHeaderLen  = 6
	icoEntryLen   = 16
	icoTypeIcon   = 1
	bmpInfoLen    = 40
	bmpBitCount32 = 32
	pngSig        = "\x89PNG\r\n\x1a\n"
	icoMaxEntries = 64
	icoMaxDim     = 1024
	icoMaxPayload = 16 << 20
	icoDim256     = 256
	bmpRowAlign   = 3 // (width*4 + 3) &^ 3 DWORD-aligns BGRA rows
)

// icoToPNG extracts the best PNG (or 32-bpp BMP) payload from a Windows ICO
// and re-encodes it as PNG. Favicons and navbar brands ship as ICO; painters
// only embed PNG/JPEG, so this is the layout-side conversion (no new deps).
func icoToPNG(data []byte) ([]byte, int, int, bool) {
	entry, payload, found := icoBestPayload(data)
	if !found {
		return nil, 0, 0, false
	}

	if len(payload) >= 8 && string(payload[:8]) == pngSig {
		width, height, _, dimsOK := imageDims(payload)
		if !dimsOK || width <= 0 || height <= 0 {
			return nil, 0, 0, false
		}

		return payload, width, height, true
	}

	img, decoded := decodeICOBMP32(payload, entry)
	if !decoded {
		return nil, 0, 0, false
	}

	pngData, err := encodePNGImage(img)
	if err != nil {
		return nil, 0, 0, false
	}

	bounds := img.Bounds()

	return pngData, bounds.Dx(), bounds.Dy(), true
}

type icoDirEntry struct {
	width, height int
	bytesInRes    int
	imageOffset   int
}

func emptyICODirEntry() icoDirEntry {
	return icoDirEntry{
		width: 0, height: 0, bytesInRes: 0, imageOffset: 0,
	}
}

func icoBestPayload(data []byte) (icoDirEntry, []byte, bool) { //nolint:cyclop // ICO directory validation gates
	if len(data) < icoHeaderLen {
		return emptyICODirEntry(), nil, false
	}

	reserved := binary.LittleEndian.Uint16(data[0:2])
	icoType := binary.LittleEndian.Uint16(data[2:4])
	count := int(binary.LittleEndian.Uint16(data[4:6]))

	if reserved != 0 || icoType != icoTypeIcon || count <= 0 || count > icoMaxEntries {
		return emptyICODirEntry(), nil, false
	}

	if len(data) < icoHeaderLen+count*icoEntryLen {
		return emptyICODirEntry(), nil, false
	}

	bestIdx := -1
	bestArea := -1
	entries := make([]icoDirEntry, count)

	for entryIdx := range count {
		off := icoHeaderLen + entryIdx*icoEntryLen
		entry := parseICODirEntry(data[off : off+icoEntryLen])

		if entry.bytesInRes <= 0 || entry.imageOffset < icoHeaderLen {
			continue
		}

		if entry.imageOffset > len(data) || entry.bytesInRes > icoMaxPayload {
			continue
		}

		if entry.imageOffset+entry.bytesInRes > len(data) {
			continue
		}

		entries[entryIdx] = entry
		area := entry.width * entry.height

		if area > bestArea {
			bestArea = area
			bestIdx = entryIdx
		}
	}

	if bestIdx < 0 {
		return emptyICODirEntry(), nil, false
	}

	entry := entries[bestIdx]
	payload := data[entry.imageOffset : entry.imageOffset+entry.bytesInRes]

	return entry, payload, true
}

func parseICODirEntry(entryBytes []byte) icoDirEntry {
	width := int(entryBytes[0])
	height := int(entryBytes[1])

	if width == 0 {
		width = icoDim256
	}

	if height == 0 {
		height = icoDim256
	}

	if width > icoMaxDim {
		width = icoMaxDim
	}

	if height > icoMaxDim {
		height = icoMaxDim
	}

	return icoDirEntry{
		width:       width,
		height:      height,
		bytesInRes:  int(binary.LittleEndian.Uint32(entryBytes[8:12])),
		imageOffset: int(binary.LittleEndian.Uint32(entryBytes[12:16])),
	}
}

// decodeICOBMP32 decodes a 32-bpp BI_RGB DIB stored inside an ICO. The DIB
// height is typically 2x the icon height (XOR bitmap + AND mask).
//
//nolint:cyclop // DIB header and XOR decode gates
func decodeICOBMP32(payload []byte, entry icoDirEntry) (*image.NRGBA, bool) {
	if len(payload) < bmpInfoLen {
		return nil, false
	}

	headerSize := binary.LittleEndian.Uint32(payload[0:4])
	if headerSize != bmpInfoLen {
		return nil, false
	}

	width := int(int32(binary.LittleEndian.Uint32(payload[4:8])))      //nolint:gosec // DIB signed width
	heightRaw := int(int32(binary.LittleEndian.Uint32(payload[8:12]))) //nolint:gosec // DIB signed height
	planes := binary.LittleEndian.Uint16(payload[12:14])
	bitCount := binary.LittleEndian.Uint16(payload[14:16])
	compression := binary.LittleEndian.Uint32(payload[16:20])

	if width <= 0 || width > icoMaxDim || planes != 1 || bitCount != bmpBitCount32 || compression != 0 {
		return nil, false
	}

	height := heightRaw
	if height < 0 {
		height = -height
	}

	// ICO DIBs store XOR then AND; the header height is usually 2×.
	if height == entry.height*two && entry.height > 0 {
		height = entry.height
	}

	if height <= 0 || height > icoMaxDim {
		return nil, false
	}

	rowStride := (width*4 + bmpRowAlign) &^ bmpRowAlign
	xorBytes := rowStride * height
	pixOff := int(headerSize)

	if pixOff+xorBytes > len(payload) {
		return nil, false
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	copyICOBMP32XOR(img, payload, pixOff, rowStride, width, height, heightRaw >= 0)

	return img, true
}

// copyICOBMP32XOR copies the XOR bitmap into an NRGBA image, flipping
// bottom-up DIB rows when needed and swapping BGRA to RGBA.
func copyICOBMP32XOR(img *image.NRGBA, payload []byte, pixOff, rowStride, width, height int, bottomUp bool) {
	for rowIdx := range height {
		srcY := rowIdx
		if bottomUp {
			srcY = height - 1 - rowIdx
		}

		row := payload[pixOff+srcY*rowStride : pixOff+srcY*rowStride+width*4]
		dst := img.Pix[rowIdx*img.Stride : rowIdx*img.Stride+width*4]

		for colIdx := range width {
			blue := row[colIdx*4+0]
			green := row[colIdx*4+1]
			red := row[colIdx*4+2]
			alpha := row[colIdx*4+3]
			dst[colIdx*4+0] = red
			dst[colIdx*4+1] = green
			dst[colIdx*4+2] = blue
			dst[colIdx*4+3] = alpha
		}
	}
}
