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
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
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

func TestJPEGSizeAndColorSpace(t *testing.T) {
	data := makeJPEG(t)
	w, h, err := jpegSize(data)
	if err != nil {
		t.Fatalf("jpegSize: %v", err)
	}
	if w != 8 || h != 6 {
		t.Errorf("jpegSize = %dx%d, want 8x6", w, h)
	}
	c, err := jpegColorSpace(data)
	if err != nil {
		t.Fatalf("jpegColorSpace: %v", err)
	}
	if c != 3 {
		t.Errorf("components = %d, want 3 (YCbCr)", c)
	}
	if !isJPEG(data) {
		t.Error("isJPEG false for JPEG data")
	}
	if _, _, err := jpegSize([]byte{0xFF, 0xD8, 0xFF}); err == nil {
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
