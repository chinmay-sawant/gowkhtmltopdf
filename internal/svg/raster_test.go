package svg

import "testing"

func TestRasterizeRect(t *testing.T) {
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20" viewBox="0 0 40 20">
  <rect x="0" y="0" width="40" height="20" fill="#0645ad"/>
</svg>`)
	png, w, h, err := Rasterize(src, 256)
	if err != nil {
		t.Fatal(err)
	}
	if w != 40 || h != 20 {
		t.Fatalf("size %dx%d, want 40x20", w, h)
	}
	if len(png) < 50 || png[0] != 0x89 {
		t.Fatalf("expected PNG bytes, got %d", len(png))
	}
}

func TestRasterizePath(t *testing.T) {
	src := []byte(`<svg viewBox="0 0 10 10"><path d="M0 0 L10 0 L10 10 Z" fill="#000"/></svg>`)
	png, w, h, err := Rasterize(src, 128)
	if err != nil {
		t.Fatal(err)
	}
	if w < 1 || h < 1 || len(png) < 40 {
		t.Fatalf("bad raster %dx%d len=%d", w, h, len(png))
	}
}
