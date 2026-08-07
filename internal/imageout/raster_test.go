package imageout

import (
	"context"
	"errors"
	"image/color"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/pdf"
)

func TestRasterPaintOrderMatchesPDFLayerPolicy(t *testing.T) {
	t.Parallel()

	ops := []layout.Op{
		{Kind: layout.OpText, ZIndex: 0, ZIndexSet: true},      //nolint:exhaustruct // intentional zero/partial fields
		{Kind: layout.OpFillRect, ZIndex: 0, ZIndexSet: true},  //nolint:exhaustruct // intentional zero/partial fields
		{Kind: layout.OpText, ZIndex: 2, ZIndexSet: true},      //nolint:exhaustruct // intentional zero/partial fields
		{Kind: layout.OpFillRect, ZIndex: -1, ZIndexSet: true}, //nolint:exhaustruct // intentional zero/partial fields
	}
	order := rasterPaintOrder(ops)
	want := []int{3, 1, 0, 2}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("paint order = %v, want %v", order, want)
		}
	}
}

func TestRenderContextHonorsCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	root, err := html.Parse(`<html><body><p>cancel me</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = RenderContext(ctx, root, RenderOptions{Width: 200}) //nolint:exhaustruct // intentional zero/partial fields
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RenderContext error = %v, want context.Canceled", err)
	}
}

func TestTTFRasterAntiAliased(t *testing.T) {
	t.Parallel()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><h1>Acme Widgets</h1><p>Invoice line item spacing.</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	img, err := Render(root, RenderOptions{Width: 400, Font: face, Background: true}) //nolint:exhaustruct // intentional zero/partial fields
	if err != nil {
		t.Fatal(err)
	}
	// Count unique colors: 5x7 bitmap had ~few solid colors; AA TTF has many.
	seen := map[color.Color]struct{}{}

	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for pixelX := b.Min.X; pixelX < b.Max.X; pixelX++ {
			seen[img.At(pixelX, py)] = struct{}{}
			if len(seen) > 40 {
				return
			}
		}
	}
	// 4x4 coverage AA yields many greys vs the old solid 5x7 bitmap (~few colors).
	if len(seen) < 10 {
		t.Errorf("unique colors = %d, want >= 10 (AA TTF text)", len(seen))
	}
}

func TestTTFAdvanceMatchesLayoutWidth(t *testing.T) {
	t.Parallel()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	sample := "Acme"

	var layoutW float64

	for _, r := range sample {
		layoutW += face.AdvanceInPoints(r, 12)
	}
	// image advances use same AdvanceInPoints * ptToPx
	imgW := layoutW * ptToPx
	if imgW < 10 || imgW > 200 {
		t.Fatalf("unexpected width scale %v", imgW)
	}
	// render single line and ensure non-white span is roughly that width
	root, err := html.Parse(`<html><body style="margin:0;padding:0"><p style="margin:0;font-size:12pt">` + sample + `</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	img, err := Render(root, RenderOptions{Width: 200, Font: face, SmartWidth: false, Background: true}) //nolint:exhaustruct // intentional zero/partial fields
	if err != nil {
		t.Fatal(err)
	}

	b := img.Bounds()
	minX, maxX := b.Max.X, b.Min.X

	for py := b.Min.Y; py < b.Max.Y; py++ {
		for pixelX := b.Min.X; pixelX < b.Max.X; pixelX++ {
			r, g, bl, _ := img.At(pixelX, py).RGBA()
			if r < 0xf000 || g < 0xf000 || bl < 0xf000 {
				if pixelX < minX {
					minX = pixelX
				}

				if pixelX > maxX {
					maxX = pixelX
				}
			}
		}
	}

	if maxX <= minX {
		t.Fatal("no ink found")
	}

	span := float64(maxX - minX)
	// allow generous tolerance for glyph bearings
	if span < imgW*0.4 || span > imgW*2.5 {
		t.Errorf("ink span %v vs advance-based %v", span, imgW)
	}
}
