package pdf

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func makePNG(t *testing.T, withAlpha bool) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 2))

	for y := range 2 {
		for x := range 4 {
			a := uint8(255)
			if withAlpha && x%2 == 1 {
				a = 100
			}

			img.Set(x, y, color.RGBA{R: uint8(x * 40), G: uint8(y * 90), B: 128, A: a})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func makeJPEG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for i := range img.Pix {
		img.Pix[i] = byte(i * 3)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestJPEGScan(t *testing.T) {
	data := makeJPEG(t)

	w, h, c, err := jpegScan(data)
	if err != nil {
		t.Fatalf("jpegScan: %v", err)
	}

	if w != 8 || h != 6 {
		t.Errorf("jpegScan = %dx%d, want 8x6", w, h)
	}

	if c != 3 {
		t.Errorf("components = %d, want 3 (YCbCr)", c)
	}

	if !isJPEG(data) {
		t.Error("isJPEG false for JPEG data")
	}

	if _, _, _, err := jpegScan([]byte{0xFF, 0xD8, 0xFF}); err == nil {
		t.Error("expected error for truncated JPEG")
	}
}

func TestAddJPEGImage(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(100, 100)

	err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, d))
	for _, want := range []string{
		"/Subtype /Image",
		"/Filter /DCTDecode",
		"/Width 8 /Height 6",
		"/ColorSpace /DeviceRGB",
		"/J1 Do",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestAddPNGImage(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(100, 100)

	err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, d))
	for _, want := range []string{
		"/Subtype /Image",
		"/Width 4 /Height 2",
		"/ColorSpace /DeviceRGB",
		"/SMask",
		"/P1 Do",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestImageResourceNamesDoNotCollideAcrossBands(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(100, 100)

	c := p.Content()
	if err := c.AddPNGImage("I0", 0, 0, 10, 10, makePNG(t, false)); err != nil {
		t.Fatalf("body image: %v", err)
	}
	// PaintBand and body Paint both use page-local counters. Content must
	// preserve both operators instead of allowing the second I0 to replace
	// the first entry in /Resources.
	if err := c.AddPNGImage("I0", 20, 20, 10, 10, makePNG(t, true)); err != nil {
		t.Fatalf("band image: %v", err)
	}

	out := string(writePDF(t, d))
	if !strings.Contains(out, "/I0 Do") || !strings.Contains(out, "/I0_1 Do") {
		t.Fatalf("image operators did not receive distinct resource names:\n%s", out)
	}

	if !strings.Contains(out, "/I0 3 0 R") || !strings.Contains(out, "/I0_1 5 0 R") {
		t.Fatalf("resource dictionary does not contain both image names:\n%s", out)
	}
}

func TestAddPNGNoAlpha(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(100, 100)

	err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, false))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, d))
	if strings.Contains(out, "/SMask") {
		t.Error("unexpected SMask for opaque PNG")
	}
}

func TestAddInvalidImage(t *testing.T) {
	d := fixedDoc(t)

	p := d.AddPage(100, 100)
	if err := p.Content().AddJPEGImage("J1", 0, 0, 10, 10, []byte("not an image")); err == nil {
		t.Error("expected error for non-JPEG data")
	}

	if err := p.Content().AddPNGImage("P1", 0, 0, 10, 10, []byte("not a png")); err == nil {
		t.Error("expected error for non-PNG data")
	}
}

// imageStream returns the raw bytes of the first /Subtype /Image stream.
func imageStream(t *testing.T, out []byte) []byte {
	t.Helper()

	s := string(out)

	idx := strings.Index(s, "/Subtype /Image")
	if idx < 0 {
		t.Fatal("no image object in output")
	}

	sm := strings.Index(s[idx:], "\nstream\n")
	if sm < 0 {
		t.Fatal("no stream marker after image dict")
	}

	data := out[idx+sm+len("\nstream\n"):]

	end := bytes.Index(data, []byte("\nendstream"))
	if end < 0 {
		t.Fatal("no endstream after image stream")
	}

	return data[:end]
}

func TestGrayscalePNGFold(t *testing.T) {
	// P5-03: PNG XObjects desaturate at embed time (Rec.601 luma fold), so
	// every pixel triple is equal.
	d := fixedDoc(t)
	d.SetGrayscale(true)
	d.SetCompression(false)

	p := d.AddPage(100, 100)
	if err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, false)); err != nil {
		t.Fatal(err)
	}

	data := imageStream(t, writePDF(t, d))
	if len(data) != 4*2*3 {
		t.Fatalf("image stream length = %d, want 24", len(data))
	}

	for y := range 2 {
		for x := range 4 {
			off := (y*4 + x) * 3

			r, g, b := data[off], data[off+1], data[off+2]
			if r != g || g != b {
				t.Errorf("pixel (%d,%d) = %d %d %d, want equal", x, y, r, g, b)
			}

			luma := uint8(0.299*float64(x*40) + 0.587*float64(y*90) + 0.114*128)
			if r != luma {
				t.Errorf("pixel (%d,%d) luma = %d, want %d", x, y, r, luma)
			}
		}
	}
}

func TestGrayscaleJPEGFold(t *testing.T) {
	// P5-03: an RGB JPEG becomes a 1-component gray JPEG at embed time.
	d := fixedDoc(t)
	d.SetGrayscale(true)
	d.SetCompression(false)

	p := d.AddPage(100, 100)
	if err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, d))
	if !strings.Contains(out, "/ColorSpace /DeviceGray") {
		t.Error("grayscale JPEG must be embedded as /DeviceGray")
	}

	if strings.Contains(out, "/ColorSpace /DeviceRGB") {
		t.Error("grayscale JPEG must not stay /DeviceRGB")
	}

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(imageStream(t, []byte(out))))
	if err != nil {
		t.Fatalf("embedded gray JPEG does not decode: %v", err)
	}

	if cfg.ColorModel != color.GrayModel {
		t.Errorf("embedded JPEG color model = %v, want gray", cfg.ColorModel)
	}
}

func TestGrayscalePNGAlphaKept(t *testing.T) {
	// The alpha soft-mask survives the grayscale fold.
	d := fixedDoc(t)
	d.SetGrayscale(true)
	d.SetCompression(false)

	p := d.AddPage(100, 100)
	if err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, d))
	if !strings.Contains(out, "/SMask") {
		t.Error("grayscale PNG must keep its alpha soft-mask")
	}
}
