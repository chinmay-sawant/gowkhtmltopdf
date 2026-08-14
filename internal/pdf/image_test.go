//nolint:testpackage,exhaustruct // tests reach into unexported state
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

	for posY := range 2 {
		for posX := range 4 {
			alpha := uint8(255)
			if withAlpha && posX%2 == 1 {
				alpha = 100
			}

			img.Set(posX, posY, color.RGBA{
				R: uint8(posX * 40), //nolint:gosec // test fixture, 4px bounds
				G: uint8(posY * 90), //nolint:gosec // test fixture, 4px bounds
				B: 128,
				A: alpha,
			})
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
	t.Parallel()
	data := makeJPEG(t)

	width, height, cur, err := jpegScan(data)
	if err != nil {
		t.Fatalf("jpegScan: %v", err)
	}

	if width != 8 || height != 6 {
		t.Errorf("jpegScan = %dx%d, want 8x6", width, height)
	}

	if cur != 3 {
		t.Errorf("components = %d, want 3 (YCbCr)", cur)
	}

	if !isJPEG(data) {
		t.Error("isJPEG false for JPEG data")
	}

	if _, _, _, err := jpegScan([]byte{0xFF, 0xD8, 0xFF}); err == nil {
		t.Error("expected error for truncated JPEG")
	}
}

func TestAddJPEGImage(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)

	err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
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
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)

	err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
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
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)

	cur := p.Content()
	if err := cur.AddPNGImage("I0", 0, 0, 10, 10, makePNG(t, false)); err != nil {
		t.Fatalf("body image: %v", err)
	}
	// PaintBand and body Paint both use page-local counters. Content must
	// preserve both operators instead of allowing the second I0 to replace
	// the first entry in /Resources.
	if err := cur.AddPNGImage("I0", 20, 20, 10, 10, makePNG(t, true)); err != nil {
		t.Fatalf("band image: %v", err)
	}

	out := string(writePDF(t, data))
	if !strings.Contains(out, "/I0 Do") || !strings.Contains(out, "/I0_1 Do") {
		t.Fatalf("image operators did not receive distinct resource names:\n%s", out)
	}

	if !strings.Contains(out, "/I0 3 0 R") || !strings.Contains(out, "/I0_1 5 0 R") {
		t.Fatalf("resource dictionary does not contain both image names:\n%s", out)
	}
}

func TestRepeatedPNGImageReusesXObject(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)
	cur := p.Content()
	imageData := makePNG(t, false)

	if err := cur.AddPNGImage("I0", 0, 0, 10, 10, imageData); err != nil {
		t.Fatalf("first image: %v", err)
	}

	if err := cur.AddPNGImage("I1", 20, 20, 10, 10, imageData); err != nil {
		t.Fatalf("repeated image: %v", err)
	}

	if cur.imageRefs["I0"].ref != cur.imageRefs["I1"].ref {
		t.Fatalf("repeated PNG refs = %v/%v, want one XObject", cur.imageRefs["I0"].ref, cur.imageRefs["I1"].ref)
	}

	if got := len(cur.imageDedup); got != 1 {
		t.Fatalf("dedup entries = %d, want 1", got)
	}
}

func TestAddPNGNoAlpha(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)

	err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, false))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
	if strings.Contains(out, "/SMask") {
		t.Error("unexpected SMask for opaque PNG")
	}
}

func TestAddInvalidImage(t *testing.T) {
	t.Parallel()
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

	str := string(out)

	idx := strings.Index(str, "/Subtype /Image")
	if idx < 0 {
		t.Fatal("no image object in output")
	}

	sm := strings.Index(str[idx:], "\nstream\n")
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
	t.Parallel()
	// P5-03: PNG XObjects desaturate at embed time (Rec.601 luma fold), so
	// every pixel triple is equal.
	doc := fixedDoc(t)
	doc.SetGrayscale(true)
	doc.SetCompression(false)

	p := doc.AddPage(100, 100)
	if err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, false)); err != nil {
		t.Fatal(err)
	}

	data := imageStream(t, writePDF(t, doc))
	if len(data) != 4*2*3 {
		t.Fatalf("image stream length = %d, want 24", len(data))
	}

	for posY := range 2 {
		for posX := range 4 {
			off := (posY*4 + posX) * 3

			rVal, g, b := data[off], data[off+1], data[off+2]
			if rVal != g || g != b {
				t.Errorf("pixel (%d,%d) = %d %d %d, want equal", posX, posY, rVal, g, b)
			}

			luma := uint8(0.299*float64(posX*40) + 0.587*float64(posY*90) + 0.114*128)
			if rVal != luma {
				t.Errorf("pixel (%d,%d) luma = %d, want %d", posX, posY, rVal, luma)
			}
		}
	}
}

func TestGrayscaleJPEGFold(t *testing.T) {
	t.Parallel()
	// P5-03: an RGB JPEG becomes a 1-component gray JPEG at embed time.
	data := fixedDoc(t)
	data.SetGrayscale(true)
	data.SetCompression(false)

	p := data.AddPage(100, 100)
	if err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
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
	t.Parallel()
	// The alpha soft-mask survives the grayscale fold.
	data := fixedDoc(t)
	data.SetGrayscale(true)
	data.SetCompression(false)

	p := data.AddPage(100, 100)
	if err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
	if !strings.Contains(out, "/SMask") {
		t.Error("grayscale PNG must keep its alpha soft-mask")
	}
}

func TestAddJPEGImagePDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)
	p := doc.AddPage(100, 100)

	err = p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	for _, want := range []string{
		"%PDF-1.7",
		"/Subtype /Image",
		"/Filter /DCTDecode",
		"/Width 8 /Height 6",
		"/ColorSpace /DeviceRGB",
		"/J1 Do",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 1.7 JPEG output", want)
		}
	}
}

func TestAddPNGImagePDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)
	p := doc.AddPage(100, 100)

	err = p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	for _, want := range []string{
		"%PDF-1.7",
		"/Subtype /Image",
		"/Width 4 /Height 2",
		"/ColorSpace /DeviceRGB",
		"/SMask",
		"/P1 Do",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 1.7 PNG output", want)
		}
	}
}

func TestAddPNGNoAlphaPDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)
	p := doc.AddPage(100, 100)

	err = p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, false))
	if err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	if strings.Contains(out, "/SMask") {
		t.Error("unexpected SMask for opaque PNG in PDF 1.7")
	}

	if !strings.Contains(out, "/ColorSpace /DeviceRGB") {
		t.Error("missing DeviceRGB in opaque PNG output")
	}
}

func TestRepeatedPNGImageReusesXObjectPDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)
	p := doc.AddPage(100, 100)
	cur := p.Content()
	imageData := makePNG(t, false)

	if err := cur.AddPNGImage("I0", 0, 0, 10, 10, imageData); err != nil {
		t.Fatalf("first image: %v", err)
	}

	if err := cur.AddPNGImage("I1", 20, 20, 10, 10, imageData); err != nil {
		t.Fatalf("repeated image: %v", err)
	}

	if cur.imageRefs["I0"].ref != cur.imageRefs["I1"].ref {
		t.Fatalf("repeated PNG refs = %v/%v, want one XObject in PDF 1.7", cur.imageRefs["I0"].ref, cur.imageRefs["I1"].ref)
	}
}

func TestGrayscaleJPEGFoldPDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetGrayscale(true)
	doc.SetCompression(false)

	p := doc.AddPage(100, 100)
	if err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/ColorSpace /DeviceGray") {
		t.Error("grayscale JPEG in PDF 1.7 must be embedded as /DeviceGray")
	}

	if strings.Contains(out, "/ColorSpace /DeviceRGB") {
		t.Error("grayscale JPEG in PDF 1.7 must not stay /DeviceRGB")
	}
}

func TestGrayscalePNGFoldPDF17(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetGrayscale(true)
	doc.SetCompression(false)

	p := doc.AddPage(100, 100)
	if err := p.Content().AddPNGImage("P1", 10, 10, 50, 30, makePNG(t, true)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/SMask") {
		t.Error("grayscale PNG with alpha in PDF 1.7 must retain its SMask")
	}
}

func TestValidateEmbeddedImageCapsPDF17(t *testing.T) {
	t.Parallel()

	// Ensure validation limits fail closed for oversized or bad dimensions.
	if err := validateEmbeddedImage(maxEmbeddedEncodedBytes+1, 100, 100); err == nil {
		t.Error("expected error for data exceeding maxEmbeddedEncodedBytes")
	}

	if err := validateEmbeddedImage(100, maxEmbeddedImageDimension+1, 100); err == nil {
		t.Error("expected error for width exceeding maxEmbeddedImageDimension")
	}

	if err := validateEmbeddedImage(100, 100, maxEmbeddedImageDimension+1); err == nil {
		t.Error("expected error for height exceeding maxEmbeddedImageDimension")
	}

	if err := validateEmbeddedImage(100, 0, 100); err == nil {
		t.Error("expected error for non-positive width")
	}
}
