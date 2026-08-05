package svg

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"
)

func TestRasterizeWikiWordmark(t *testing.T) {
	for _, f := range []string{"/tmp/wiki-wordmark.svg", "/tmp/enwiki.svg", "/tmp/wiki-tagline.svg"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Skip(err)
		}
		pngb, w, h, err := Rasterize(b, 512)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		img, err := png.Decode(bytes.NewReader(pngb))
		if err != nil {
			t.Errorf("%s decode: %v", f, err)
			continue
		}
		nz := 0
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a > 0 {
					nz++
				}
			}
		}
		t.Logf("%s %dx%d nonzero=%d", f, w, h, nz)
		if nz < 50 {
			t.Errorf("%s: mostly empty raster (nonzero=%d)", f, nz)
		}
	}
}

func TestRasterizeArcPath(t *testing.T) {
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
<path fill="#0e65c0" d="M10 50 a40 40 0 1 1 80 0 a40 40 0 1 1 -80 0"/>
</svg>`)
	pngb, w, h, err := Rasterize(src, 128)
	if err != nil {
		t.Fatal(err)
	}
	img, _ := png.Decode(bytes.NewReader(pngb))
	nz := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				nz++
			}
		}
	}
	t.Logf("arc %dx%d nonzero=%d", w, h, nz)
	if nz < 100 {
		t.Fatalf("arc path produced empty image (nonzero=%d)", nz)
	}
}

// ensure image import used when decoding wordmark in other tests
var _ = image.RGBA{}
