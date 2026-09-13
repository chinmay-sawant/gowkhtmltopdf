//nolint:cyclop,funlen,varnamelen // image adjustment layout tests
package layout

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"testing"
)

const imgAdjustFull = 0xFFFF

// imgAdjustLayout renders one <img> through the real Layout pipeline with the
// given declaration body applied to the img rule.
func imgAdjustLayout(t *testing.T, cssText string, data []byte) *Result {
	t.Helper()

	root := mustParse(t, `<html><body><img src="probe.png" alt="probe"></body></html>`)

	opts := Options{
		Width: testViewport, Height: 800, Background: true, Media: "print",
		Images: func(string) ([]byte, error) { return data, nil },
	}
	if cssText != "" {
		opts.Sheets = append(opts.Sheets, sheet(t, "img{"+cssText+"}"))
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

func imgAdjustSingleImage(t *testing.T, res *Result) Op {
	t.Helper()

	images := opsOfKind(res, OpImage)
	if len(images) != 1 {
		t.Fatalf("OpImage count = %d, want 1", len(images))
	}

	return images[0]
}

func imgAdjustDecodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode emitted PNG: %v", err)
	}

	return img
}

func imgAdjustAssertPixel(t *testing.T, img image.Image, x, y int, name string, wantR, wantG, wantB uint32) {
	t.Helper()

	r, g, b, _ := img.At(x, y).RGBA()
	if !imgAdjustNearChannel(r, wantR) || !imgAdjustNearChannel(g, wantG) || !imgAdjustNearChannel(b, wantB) {
		t.Fatalf("pixel (%d,%d) = #%04x%04x%04x, want %s", x, y, r, g, b, name)
	}
}

// imgAdjustNearChannel tolerates JPEG chroma bleed and nearest-neighbor edges.
func imgAdjustNearChannel(got, want uint32) bool {
	const tolerance = 0x2000 // 32/255

	return int64(got)-int64(want) <= tolerance && int64(want)-int64(got) <= tolerance
}

func TestImageAdjustOrientationEXIF(t *testing.T) {
	t.Parallel()

	jpegData := imgAdjustEXIFJPEG(t, 6, 4)
	if got := jpegEXIFOrientation(jpegData); got != 6 {
		t.Fatalf("jpegEXIFOrientation = %d, want 6", got)
	}

	t.Run("from-image rotates and swaps axes", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "", jpegData))
		if op.IsJPEG {
			t.Fatal("oriented output kept IsJPEG; EXIF rotation must re-encode")
		}

		if op.ImgW != 4 || op.ImgH != 8 {
			t.Fatalf("intrinsic = %dx%d, want 4x8", op.ImgW, op.ImgH)
		}

		if !near(op.W, 3) || !near(op.H, 6) {
			t.Fatalf("used size = %vx%v, want 3x6", op.W, op.H)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
		imgAdjustAssertPixel(t, img, 0, 7, "green", 0, imgAdjustFull, 0)
	})

	t.Run("none ignores EXIF", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "image-orientation: none", jpegData))
		if !op.IsJPEG {
			t.Fatal("image-orientation: none must keep the raw JPEG fast path")
		}

		if op.ImgW != 8 || op.ImgH != 4 {
			t.Fatalf("intrinsic = %dx%d, want 8x4", op.ImgW, op.ImgH)
		}

		if !near(op.W, 6) || !near(op.H, 3) {
			t.Fatalf("used size = %vx%v, want 6x3", op.W, op.H)
		}
	})

	t.Run("zero angle still suppresses EXIF", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "image-orientation: 0deg", jpegData))
		if !op.IsJPEG || op.ImgW != 8 || op.ImgH != 4 {
			t.Fatalf("0deg = jpeg=%v %dx%d, want raw 8x4", op.IsJPEG, op.ImgW, op.ImgH)
		}
	})

	t.Run("block image rotates through the replaced paint path", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "display: block", jpegData))
		if op.IsJPEG || op.ImgW != 4 || op.ImgH != 8 {
			t.Fatalf("block from-image = jpeg=%v %dx%d, want oriented 4x8", op.IsJPEG, op.ImgW, op.ImgH)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
		imgAdjustAssertPixel(t, img, 0, 7, "green", 0, imgAdjustFull, 0)
	})
}

func TestImageAdjustOrientationAnglePNG(t *testing.T) {
	t.Parallel()

	pngData := imgAdjustHalfPNG(t, 4)

	op := imgAdjustSingleImage(t, imgAdjustLayout(t, "image-orientation: 90deg", pngData))
	if op.IsJPEG {
		t.Fatal("explicit rotation must re-encode")
	}

	if op.ImgW != 4 || op.ImgH != 8 {
		t.Fatalf("intrinsic = %dx%d, want 4x8", op.ImgW, op.ImgH)
	}

	img := imgAdjustDecodePNG(t, op.Image)
	imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
	imgAdjustAssertPixel(t, img, 0, 7, "green", 0, imgAdjustFull, 0)

	t.Run("flip mirrors after rotation", func(t *testing.T) {
		t.Parallel()

		quadrants := imgAdjustQuadrantPNG(t, 4)
		rotated := imgAdjustSingleImage(t, imgAdjustLayout(t, "image-orientation: 90deg", quadrants))
		flipped := imgAdjustSingleImage(t, imgAdjustLayout(t, "image-orientation: 90deg flip", quadrants))

		rotatedImg := imgAdjustDecodePNG(t, rotated.Image)
		flippedImg := imgAdjustDecodePNG(t, flipped.Image)

		// 90deg alone puts the source top-left (red) at the bottom-left; the
		// flip moves it to the top-left and the source top-right (green) to
		// the bottom-left.
		imgAdjustAssertPixel(t, rotatedImg, 0, 0, "blue", 0, 0, imgAdjustFull)
		imgAdjustAssertPixel(t, flippedImg, 0, 0, "red", imgAdjustFull, 0, 0)
		imgAdjustAssertPixel(t, flippedImg, 3, 0, "blue", 0, 0, imgAdjustFull)
		imgAdjustAssertPixel(t, flippedImg, 3, 3, "yellow", imgAdjustFull, imgAdjustFull, 0)
	})
}

func TestImageAdjustResolution(t *testing.T) {
	t.Parallel()

	plain := tinyPNG(10, 20)
	hidpi := imgAdjustPNGWithDPI(t, 10, 20, 300)

	if got := intrinsicImageResolutionDPI(hidpi); got < 299.9 || got > 300.1 {
		t.Fatalf("intrinsicImageResolutionDPI = %v, want ~300", got)
	}

	tests := []struct {
		name  string
		css   string
		data  []byte
		wantW float64
		wantH float64
	}{
		{name: "from-image uses intrinsic 300dpi", data: hidpi, wantW: 2.4, wantH: 4.8},
		{name: "explicit 96dpi overrides intrinsic", css: "image-resolution: 96dpi", data: hidpi, wantW: 7.5, wantH: 15},
		{name: "explicit 300dpi on plain png", css: "image-resolution: 300dpi", data: plain, wantW: 2.4, wantH: 4.8},
		{name: "2dppx is 192dpi", css: "image-resolution: 2dppx", data: plain, wantW: 3.75, wantH: 7.5},
		{name: "unset stays 96dpi", data: plain, wantW: 7.5, wantH: 15},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			op := imgAdjustSingleImage(t, imgAdjustLayout(t, tc.css, tc.data))
			if !near(op.W, tc.wantW) || !near(op.H, tc.wantH) {
				t.Fatalf("used size = %vx%v, want %vx%v", op.W, op.H, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestImageAdjustObjectViewBox(t *testing.T) {
	t.Parallel()

	quadrants := imgAdjustQuadrantPNG(t, 4)

	t.Run("xywh crops the bottom right quadrant", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "object-view-box: xywh(50% 50% 50% 50%)", quadrants))
		if op.ImgW != 2 || op.ImgH != 2 {
			t.Fatalf("crop intrinsic = %dx%d, want 2x2", op.ImgW, op.ImgH)
		}

		// The element keeps the full intrinsic box; the crop scales into it.
		if !near(op.W, 3) || !near(op.H, 3) {
			t.Fatalf("used size = %vx%v, want 3x3", op.W, op.H)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "yellow", imgAdjustFull, imgAdjustFull, 0)
	})

	t.Run("inset crops the center", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "object-view-box: inset(25% 25% 25% 25%)", quadrants))
		if op.ImgW != 2 || op.ImgH != 2 {
			t.Fatalf("crop intrinsic = %dx%d, want 2x2", op.ImgW, op.ImgH)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
		imgAdjustAssertPixel(t, img, 1, 0, "green", 0, imgAdjustFull, 0)
		imgAdjustAssertPixel(t, img, 0, 1, "blue", 0, 0, imgAdjustFull)
		imgAdjustAssertPixel(t, img, 1, 1, "yellow", imgAdjustFull, imgAdjustFull, 0)
	})

	t.Run("block image crops through the replaced paint path", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(
			t, "display: block; object-view-box: inset(25% 25% 25% 25%)", quadrants))
		if op.ImgW != 2 || op.ImgH != 2 {
			t.Fatalf("crop intrinsic = %dx%d, want 2x2", op.ImgW, op.ImgH)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
		imgAdjustAssertPixel(t, img, 1, 1, "yellow", imgAdjustFull, imgAdjustFull, 0)
	})

	t.Run("rect matches inset edges", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "object-view-box: rect(25% 25% 25% 25%)", quadrants))
		if op.ImgW != 2 || op.ImgH != 2 {
			t.Fatalf("crop intrinsic = %dx%d, want 2x2", op.ImgW, op.ImgH)
		}

		img := imgAdjustDecodePNG(t, op.Image)
		imgAdjustAssertPixel(t, img, 0, 0, "red", imgAdjustFull, 0, 0)
	})

	t.Run("circle stays uncropped", func(t *testing.T) {
		t.Parallel()

		op := imgAdjustSingleImage(t, imgAdjustLayout(t, "object-view-box: circle(50% at 50% 50%)", quadrants))
		if op.ImgW != 4 || op.ImgH != 4 {
			t.Fatalf("unsupported shape cropped to %dx%d, want 4x4", op.ImgW, op.ImgH)
		}
	})
}

// imgAdjustEXIFJPEG builds a JPEG whose left half is red and right half is
// green, prefixed with an APP1 Exif segment carrying an orientation tag.
func imgAdjustEXIFJPEG(t *testing.T, orientation, half int) []byte {
	t.Helper()

	const bandHeight = 4

	img := image.NewRGBA(image.Rect(0, 0, half*2, bandHeight))

	for y := range bandHeight {
		for x := range half * 2 {
			if x < half {
				img.Set(x, y, color.RGBA{R: 0xFF, A: 0xFF})
			} else {
				img.Set(x, y, color.RGBA{G: 0xFF, A: 0xFF})
			}
		}
	}

	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, img, nil); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}

	segment := imgAdjustEXIFSegment(orientation)
	data := raw.Bytes()

	out := make([]byte, 0, len(data)+len(segment))
	out = append(out, data[:2]...)
	out = append(out, segment...)
	out = append(out, data[2:]...)

	return out
}

func imgAdjustHalfPNG(t *testing.T, half int) []byte {
	t.Helper()

	const bandHeight = 4

	img := image.NewRGBA(image.Rect(0, 0, half*2, bandHeight))

	for y := range bandHeight {
		for x := range half * 2 {
			if x < half {
				img.Set(x, y, color.RGBA{R: 0xFF, A: 0xFF})
			} else {
				img.Set(x, y, color.RGBA{G: 0xFF, A: 0xFF})
			}
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}

	return out.Bytes()
}

// imgAdjustEXIFSegment builds one APP1 Exif segment with a little-endian IFD0
// holding only the orientation tag.
func imgAdjustEXIFSegment(orientation int) []byte {
	tiff := []byte{
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, // TIFF header, IFD0 at offset 8.
		0x01, 0x00, // One IFD0 entry.
		0x12, 0x01, // Tag 0x0112 image-orientation.
		0x03, 0x00, // Type SHORT.
		0x01, 0x00, 0x00, 0x00, // Count 1.
		0x00, 0x00, 0x00, 0x00, // Value, patched below.
		0x00, 0x00, 0x00, 0x00, // No next IFD.
	}
	tiff[18] = byte(orientation)

	payload := append([]byte("Exif\x00\x00"), tiff...)

	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2)) //nolint:gosec // tiny test payload
	segment = append(segment, payload...)

	return segment
}

// imgAdjustPNGWithDPI splices a pHYs chunk after IHDR so the PNG declares its
// own resolution. The chunk must precede IDAT.
func imgAdjustPNGWithDPI(t *testing.T, w, h, dpi int) []byte {
	t.Helper()

	const (
		physPayloadBytes = 9
		ihdrEndOffset    = 33 // PNG signature + IHDR header + IHDR data + CRC.
	)

	ppm := uint32(math.Round(float64(dpi) / metersToInches))
	payload := make([]byte, physPayloadBytes)
	binary.BigEndian.PutUint32(payload[0:4], ppm)
	binary.BigEndian.PutUint32(payload[4:8], ppm)
	payload[8] = pngPhysMeterUnit

	chunk := make([]byte, 0, 4+4+physPayloadBytes+4)
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(physPayloadBytes))
	chunk = append(chunk, "pHYs"...)
	chunk = append(chunk, payload...)

	crc := crc32.NewIEEE()
	_, _ = crc.Write([]byte("pHYs"))
	_, _ = crc.Write(payload)
	chunk = binary.BigEndian.AppendUint32(chunk, crc.Sum32())

	base := tinyPNG(w, h)
	out := make([]byte, 0, len(base)+len(chunk))
	out = append(out, base[:ihdrEndOffset]...)
	out = append(out, chunk...)
	out = append(out, base[ihdrEndOffset:]...)

	return out
}

// imgAdjustQuadrantPNG builds a PNG with red, green, blue, and yellow
// quadrants so crop offsets are visible in the emitted bytes.
func imgAdjustQuadrantPNG(t *testing.T, side int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, side, side))
	half := side / 2

	for y := range side {
		for x := range side {
			switch {
			case x < half && y < half:
				img.Set(x, y, color.RGBA{R: 0xFF, A: 0xFF})
			case x >= half && y < half:
				img.Set(x, y, color.RGBA{G: 0xFF, A: 0xFF})
			case x < half:
				img.Set(x, y, color.RGBA{B: 0xFF, A: 0xFF})
			default:
				img.Set(x, y, color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF})
			}
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}

	return out.Bytes()
}
