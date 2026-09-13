//nolint:cyclop,mnd,varnamelen,wsl // EXIF and image metadata byte parsers
package layout

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register the JPEG decoder for EXIF-bearing content images.
	"image/png"
	"math"
)

// Image metadata parsing and raster transforms for image-orientation,
// image-resolution, and object-view-box. The ResolvedStyle field reads live in
// layout_images.go; this file is pure bytes in, bytes out.
const (
	jpegSOIMarker  = 0xD8
	jpegEOIMarker  = 0xD9
	jpegSOSMarker  = 0xDA
	jpegAPP0Marker = 0xE0
	jpegAPP1Marker = 0xE1

	exifHeaderBytes = 6
	exifTIFFMinLen  = 8
	exifMagic       = 42
	exifIFDEntryLen = 12

	exifTagOrientation    = 0x0112
	exifTagXResolution    = 0x011A
	exifTagYResolution    = 0x011B
	exifTagResolutionUnit = 0x0128

	exifTypeShort    = 3
	exifTypeRational = 5

	exifUnitInch       = 2
	exifUnitCentimeter = 3

	inchesToCentimeters = 2.54
	metersToInches      = 0.0254

	jfifUnitsOffset           = 7
	jfifDensityOffset         = 8
	jfifDensityMinLen         = 12
	defaultImageResolutionDPI = 96
	pngPhysMeterUnit          = 1
	degreesQuarterTurn        = 90
	degreesHalfTurn           = 180
	degreesThreeQuarts        = 270
	maxEXIFOrientation        = 8
)

// jpegEXIFOrientation returns the EXIF orientation tag (1-8) or 0 when the
// JPEG carries none.
func jpegEXIFOrientation(data []byte) int {
	orientation, _ := jpegEXIFMetadata(data)

	return orientation
}

// jpegEXIFResolutionDPI returns the EXIF resolution in DPI, or 0 when absent.
func jpegEXIFResolutionDPI(data []byte) float64 {
	_, dpi := jpegEXIFMetadata(data)

	return dpi
}

// jpegEXIFMetadata reads orientation and X/Y resolution from the first Exif
// APP1 segment's IFD0.
func jpegEXIFMetadata(data []byte) (int, float64) {
	tiff := jpegEXIFPayload(data)
	if len(tiff) < exifTIFFMinLen {
		return 0, 0
	}

	order, ok := exifByteOrder(tiff)
	if !ok || order.Uint16(tiff[2:4]) != exifMagic {
		return 0, 0
	}

	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < exifTIFFMinLen || ifd+2 > len(tiff) {
		return 0, 0
	}

	orientation := 0
	xResolution, yResolution := 0.0, 0.0
	unit := uint16(exifUnitInch)
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	pos := ifd + 2

	for range count {
		if pos+exifIFDEntryLen > len(tiff) {
			break
		}

		tag := order.Uint16(tiff[pos : pos+2])
		typ := order.Uint16(tiff[pos+2 : pos+4])
		n := order.Uint32(tiff[pos+4 : pos+8])
		valueField := tiff[pos+8 : pos+12]

		switch tag {
		case exifTagOrientation:
			if typ == exifTypeShort && n >= 1 {
				orientation = int(order.Uint16(valueField[:2]))
			}
		case exifTagXResolution:
			xResolution = exifRational(order, tiff, typ, n, valueField)
		case exifTagYResolution:
			yResolution = exifRational(order, tiff, typ, n, valueField)
		case exifTagResolutionUnit:
			if typ == exifTypeShort && n >= 1 {
				unit = order.Uint16(valueField[:2])
			}
		}

		pos += exifIFDEntryLen
	}

	return orientation, exifResolutionToDPI(xResolution, yResolution, unit)
}

// jpegEXIFPayload returns the TIFF block after the "Exif\0\0" identifier of
// the first Exif APP1 segment, or nil.
func jpegEXIFPayload(data []byte) []byte {
	if len(data) < 4 || data[0] != jpegMarkerPrefix || data[1] != jpegSOIMarker {
		return nil
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			pos++

			continue
		}

		marker := data[pos+1]
		if marker == jpegEOIMarker || marker == jpegSOSMarker {
			return nil
		}

		segLen := int(data[pos+2])<<jpegLengthShift | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			return nil
		}

		if marker == jpegAPP1Marker {
			payload := data[pos+4 : pos+2+segLen]
			if len(payload) > exifHeaderBytes && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				return payload[exifHeaderBytes:]
			}
		}

		pos += 2 + segLen
	}

	return nil
}

// jpegJFIFResolutionDPI returns the APP0 JFIF density in DPI, or 0 when the
// segment is absent or declares only an aspect ratio.
func jpegJFIFResolutionDPI(data []byte) float64 {
	if len(data) < 4 || data[0] != jpegMarkerPrefix || data[1] != jpegSOIMarker {
		return 0
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			pos++

			continue
		}

		marker := data[pos+1]
		if marker == jpegEOIMarker || marker == jpegSOSMarker {
			return 0
		}

		segLen := int(data[pos+2])<<jpegLengthShift | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			return 0
		}

		if marker == jpegAPP0Marker {
			if dpi := jfifSegmentDPI(data[pos+4 : pos+2+segLen]); dpi > 0 {
				return dpi
			}
		}

		pos += 2 + segLen
	}

	return 0
}

func jfifSegmentDPI(payload []byte) float64 {
	if len(payload) < jfifDensityMinLen || !bytes.HasPrefix(payload, []byte("JFIF\x00")) {
		return 0
	}

	x := int(payload[jfifDensityOffset])<<jpegLengthShift | int(payload[jfifDensityOffset+1])
	y := int(payload[jfifDensityOffset+2])<<jpegLengthShift | int(payload[jfifDensityOffset+3])
	density := math.Max(float64(x), float64(y))
	if density <= 0 {
		return 0
	}

	switch payload[jfifUnitsOffset] {
	case 1:
		return density
	case 2:
		return density * inchesToCentimeters
	default:
		return 0
	}
}

// pngResolutionDPI reads a pHYs chunk's pixels-per-meter value. The chunk must
// precede IDAT; a unitless pHYs stores only an aspect ratio and returns 0.
func pngResolutionDPI(data []byte) float64 {
	const (
		pngHeaderBytes   = 8
		chunkHeaderBytes = 8
		chunkCRCBytes    = 4
		physChunkBytes   = 9
	)

	if len(data) < pngHeaderBytes || string(data[:pngHeaderBytes]) != "\x89PNG\r\n\x1a\n" {
		return 0
	}

	pos := pngHeaderBytes
	for pos+chunkHeaderBytes+chunkCRCBytes <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		if pos+chunkHeaderBytes+length+chunkCRCBytes > len(data) {
			return 0
		}

		chunkType := string(data[pos+4 : pos+8])
		if chunkType == "pHYs" && length >= physChunkBytes {
			x := binary.BigEndian.Uint32(data[pos+8 : pos+12])
			unit := data[pos+16]
			if unit == pngPhysMeterUnit && x > 0 {
				return float64(x) * metersToInches
			}

			return 0
		}

		if chunkType == "IDAT" || chunkType == "IEND" {
			return 0
		}

		pos += chunkHeaderBytes + length + chunkCRCBytes
	}

	return 0
}

// intrinsicImageResolutionDPI returns the image's own resolution in DPI, or 0
// when the bytes declare none. PNG pHYs wins for PNG; for JPEG the EXIF
// resolution wins over the JFIF density.
func intrinsicImageResolutionDPI(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	if dpi := pngResolutionDPI(data); dpi > 0 {
		return dpi
	}

	if dpi := jpegEXIFResolutionDPI(data); dpi > 0 {
		return dpi
	}

	return jpegJFIFResolutionDPI(data)
}

func exifByteOrder(tiff []byte) (binary.ByteOrder, bool) {
	switch string(tiff[:2]) {
	case "II":
		return binary.LittleEndian, true
	case "MM":
		return binary.BigEndian, true
	default:
		return nil, false
	}
}

// exifRational reads one RATIONAL value, which lives at an offset when it
// does not fit in the entry's 4-byte value field.
func exifRational(order binary.ByteOrder, tiff []byte, typ uint16, count uint32, valueField []byte) float64 {
	if typ != exifTypeRational || count < 1 || len(valueField) < 4 {
		return 0
	}

	offset := int(order.Uint32(valueField))
	if offset < 0 || offset+8 > len(tiff) {
		return 0
	}

	numerator := order.Uint32(tiff[offset : offset+4])
	denominator := order.Uint32(tiff[offset+4 : offset+8])
	if denominator == 0 {
		return 0
	}

	return float64(numerator) / float64(denominator)
}

func exifResolutionToDPI(xResolution, yResolution float64, unit uint16) float64 {
	dpi := xResolution
	if dpi <= 0 {
		dpi = yResolution
	}

	switch unit {
	case exifUnitInch:
		return dpi
	case exifUnitCentimeter:
		return dpi * inchesToCentimeters
	default:
		// Unit 1 means "no absolute unit": the values are only a pixel aspect
		// ratio, which CSS does not turn into a DPI.
		return 0
	}
}

// exifOrientationTransform maps EXIF orientation 2-8 to clockwise quarter
// turns plus a post-rotation horizontal mirror.
func exifOrientationTransform(orientation int) (int, bool) {
	switch orientation {
	case 2:
		return 0, true
	case 3:
		return 2, false
	case 4:
		return 2, true
	case 5:
		return 1, true
	case 6:
		return 1, false
	case 7:
		return 3, true
	case 8:
		return 3, false
	default:
		return 0, false
	}
}

// normalizeDegrees folds an angle into [0, 360).
func normalizeDegrees(deg float64) float64 {
	deg = math.Mod(deg, fullTurnDegrees)
	if deg < 0 {
		deg += fullTurnDegrees
	}

	if deg == 0 {
		return 0
	}

	return deg
}

// quarterTurns reports how many clockwise quarter turns deg is, if exact.
func quarterTurns(deg float64) (int, bool) {
	switch deg {
	case 0:
		return 0, true
	case degreesQuarterTurn:
		return 1, true
	case degreesHalfTurn:
		return 2, true
	case degreesThreeQuarts:
		return 3, true
	default:
		return 0, false
	}
}

// quarterTurnSwapsAxes reports whether a rotation exchanges width and height.
func quarterTurnSwapsAxes(deg float64) bool {
	quarters, exact := quarterTurns(deg)

	return exact && quarters%2 == 1
}

// applyRasterOrientation rotates img clockwise by deg and optionally mirrors
// it horizontally after the rotation. Quarter turns preserve pixels exactly;
// other angles rotate about the center inside the original canvas, so content
// outside the canvas is dropped.
func applyRasterOrientation(img image.Image, deg float64, mirrorX bool) image.Image {
	deg = normalizeDegrees(deg)
	if deg == 0 && !mirrorX {
		return img
	}

	quarters, exact := quarterTurns(deg)
	if exact {
		return rotateQuarters(img, quarters, mirrorX)
	}

	return rotateArbitrary(img, deg, mirrorX)
}

func rotateQuarters(img image.Image, quarters int, mirrorX bool) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dstW, dstH := w, h
	if quarters == 1 || quarters == 3 {
		dstW, dstH = h, w
	}

	src := copyToRGBA(img)
	if quarters == 0 && !mirrorX {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	for y := range dstH {
		for x := range dstW {
			sx, sy := quarterSourcePixel(x, y, w, h, dstW, quarters, mirrorX)
			dst.Set(x, y, src.At(sx, sy))
		}
	}

	return dst
}

// quarterSourcePixel is the inverse of a clockwise quarter rotation with an
// optional destination-axis mirror.
func quarterSourcePixel(x, y, w, h, dstW, quarters int, mirrorX bool) (int, int) {
	if mirrorX {
		x = dstW - 1 - x
	}

	switch quarters {
	case 1:
		return y, h - 1 - x
	case 2:
		return w - 1 - x, h - 1 - y
	case 3:
		return w - 1 - y, x
	default:
		return x, y
	}
}

// rotateArbitrary rotates about the center with nearest-neighbor sampling and
// keeps the original canvas size.
func rotateArbitrary(img image.Image, deg float64, mirrorX bool) image.Image {
	src := copyToRGBA(img)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	radians := deg * math.Pi / degreesInHalfCircle
	cosine, sine := math.Cos(radians), math.Sin(radians)
	centerX, centerY := float64(w-1)/2, float64(h-1)/2

	for y := range h {
		for x := range w {
			px := float64(x)
			if mirrorX {
				px = float64(w-1) - px
			}

			offsetX, offsetY := px-centerX, float64(y)-centerY
			sourceX := offsetX*cosine + offsetY*sine + centerX
			sourceY := -offsetX*sine + offsetY*cosine + centerY

			sampleX, sampleY := int(math.Round(sourceX)), int(math.Round(sourceY))
			if sampleX < 0 || sampleX >= w || sampleY < 0 || sampleY >= h {
				continue
			}

			dst.Set(x, y, src.At(sampleX, sampleY))
		}
	}

	return dst
}

func copyToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)

	return dst
}

func encodePNGImage(img image.Image) ([]byte, error) {
	var out bytes.Buffer

	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("encode oriented image: %w", err)
	}

	return out.Bytes(), nil
}
