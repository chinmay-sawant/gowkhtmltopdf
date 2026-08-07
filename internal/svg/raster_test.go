package svg_test

import (
	"testing"

	"gowkhtmltopdf/internal/svg"
)

// Tests exercise the canvas-only Rasterize path (tdewolff/canvas).

func TestRasterizeRect(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20" viewBox="0 0 40 20">
  <rect x="0" y="0" width="40" height="20" fill="#0645ad"/>
</svg>`)

	png, width, height, err := svg.Rasterize(src, 256)
	if err != nil {
		t.Fatal(err)
	}

	if width != 40 || height != 20 {
		t.Fatalf("size %dx%d, want 40x20", width, height)
	}

	if len(png) < 50 || png[0] != 0x89 {
		t.Fatalf("expected PNG bytes, got %d", len(png))
	}
}

func TestRasterizePath(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg viewBox="0 0 10 10"><path d="M0 0 L10 0 L10 10 Z" fill="#000"/></svg>`)

	png, width, height, err := svg.Rasterize(src, 128)
	if err != nil {
		t.Fatal(err)
	}

	if width < 1 || height < 1 || len(png) < 40 {
		t.Fatalf("bad raster %dx%d len=%d", width, height, len(png))
	}
}

func TestRasterizeNotSVG(t *testing.T) {
	t.Parallel()

	png, width, height, err := svg.Rasterize([]byte("not an svg"), 64)
	if err == nil {
		t.Fatal("expected error for non-SVG input")
	}

	if png != nil || width != 0 || height != 0 {
		t.Fatalf("on failure want empty result, got png=%v %dx%d", png != nil, width, height)
	}
}

func TestRasterizeBrokenSVG(t *testing.T) {
	t.Parallel()
	// Malformed path can panic inside tdewolff/canvas; Rasterize must recover
	// and return a clean error (no second rasterizer / no shell fallback).
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M this is garbage"/></svg>`)

	png, width, height, err := svg.Rasterize(src, 64)
	if err == nil {
		// If canvas ever starts tolerating this, still require a real image.
		if width < 1 || height < 1 || len(png) < 1 {
			t.Fatalf("success with empty image %dx%d len=%d", width, height, len(png))
		}

		return
	}

	if png != nil || width != 0 || height != 0 {
		t.Fatalf("on error want empty, got png=%v %dx%d err=%v", png != nil, width, height, err)
	}
}
