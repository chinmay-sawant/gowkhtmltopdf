package svg_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/svg"
)

func TestRasterizeWikiWordmark(t *testing.T) {
	t.Parallel()
	// Optional host cache of live Wikipedia logo SVGs (see samples recipe).
	// Prefer the 25th-anniversary icon/wordmark/tagline used by enwiki Vector 2022.
	for _, path := range []string{
		"/tmp/enwiki-25.svg",
		"/tmp/wikipedia-wordmark-en-25.svg",
		"/tmp/wikipedia-tagline-en-25.svg",
		"/tmp/wiki-wordmark.svg",
		"/tmp/enwiki.svg",
		"/tmp/wiki-tagline.svg",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		pngBytes, width, height, err := svg.Rasterize(data, 512)
		if err != nil {
			t.Errorf("%s: %v", path, err)

			continue
		}

		img, err := png.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			t.Errorf("%s decode: %v", path, err)

			continue
		}

		nonZero := 0

		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a > 0 {
					nonZero++
				}
			}
		}

		t.Logf("%s %dx%d nonzero=%d", path, width, height, nonZero)

		if nonZero < 50 {
			t.Errorf("%s: mostly empty raster (nonzero=%d)", path, nonZero)
		}
	}
}

func TestRasterizeArcPath(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
<path fill="#0e65c0" d="M10 50 a40 40 0 1 1 80 0 a40 40 0 1 1 -80 0"/>
</svg>`)

	pngBytes, width, height, err := svg.Rasterize(src, 128)
	if err != nil {
		t.Fatal(err)
	}

	img, _ := png.Decode(bytes.NewReader(pngBytes))
	nonZero := 0

	data := img.Bounds()
	for y := data.Min.Y; y < data.Max.Y; y++ {
		for x := data.Min.X; x < data.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				nonZero++
			}
		}
	}

	t.Logf("arc %dx%d nonzero=%d", width, height, nonZero)

	if nonZero < 100 {
		t.Fatalf("arc path produced empty image (nonzero=%d)", nonZero)
	}
}

// ensure image import used when decoding wordmark in other tests.
var _ = image.RGBA{} //nolint:exhaustruct // intentional zero/partial fields
