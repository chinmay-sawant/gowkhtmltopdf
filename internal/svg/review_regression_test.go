//nolint:testpackage // tests exercise the bounded SVG probe seam.
package svg

import (
	"bytes"
	"errors"
	"image/png"
	"testing"
)

func TestRasterizeReportsLogicalSizeForSupersampledPNG(t *testing.T) {
	t.Parallel()

	src := []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20" viewBox="0 0 40 20">` +
			`<rect width="40" height="20" fill="red"/></svg>`,
	)
	pngBytes, width, height, err := Rasterize(src, 256)

	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}

	if width != 40 || height != 20 {
		t.Fatalf("logical size = %dx%d, want 40x20", width, height)
	}

	if got := img.Bounds().Size(); got.X != 160 || got.Y != 80 {
		t.Fatalf("encoded pixel size = %v, want 160x80 supersampled pixels", got)
	}
}

func TestRasterizeRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	data := make([]byte, maxSVGBytes+1)
	data[0] = '<'

	pngBytes, width, height, err := Rasterize(data, 64)
	if !errors.Is(err, errSVGTooLarge) {
		t.Fatalf("Rasterize error = %v, want errSVGTooLarge", err)
	}

	if pngBytes != nil || width != 0 || height != 0 {
		t.Fatalf("oversized input returned png=%v size=%dx%d", pngBytes != nil, width, height)
	}
}

func TestLooksLikeSVGInspectsOnlyBoundedPrefix(t *testing.T) {
	t.Parallel()

	data := append(make([]byte, maxSVGProbeBytes), []byte("<svg")...)
	if looksLikeSVG(data) {
		t.Fatal("looksLikeSVG detected a tag outside its bounded prefix")
	}
}
