//nolint:testpackage // white-box tests need raster internals (paint order, ptToPx)
package imageout

import (
	"context"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
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

func TestRoundedBorderTopOverlayKeepsAccentThroughCorners(t *testing.T) {
	t.Parallel()

	const (
		boxX   = 12.0
		boxY   = 12.0
		boxW   = 80.0
		boxH   = 32.0
		radius = 8.0
	)

	res := &layout.Result{ //nolint:exhaustruct // synthetic raster result
		Width:  110,
		Height: 70,
		Ops: []layout.Op{
			{ //nolint:exhaustruct // synthetic rounded stroke
				Kind: layout.OpStrokeRect, X: boxX, Y: boxY, W: boxW, H: boxH,
				R: 0.7, G: 0.8, B: 0.85, Width: 1.5, Radius: radius,
			},
			{ //nolint:exhaustruct // synthetic accent line
				Kind: layout.OpLine, X: boxX + radius, Y: boxY, W: boxW - 2*radius,
				R: 0.1, G: 0.7, B: 0.4, Width: 4,
			},
		},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false)
	if err != nil {
		t.Fatal(err)
	}

	// The accent must cover pixels on the top-left arc. A straight overlay
	// leaves that arc in the base border color and produces a visible cap gap.
	accentPixels := 0
	startX, startY := boxX*ptToPx, boxY*ptToPx
	endX, endY := (boxX+radius)*ptToPx, (boxY+radius)*ptToPx

	for y := int(endY); y >= int(startY); y-- {
		for x := int(startX); x < int(endX); x++ {
			pixel := img.NRGBAAt(x, y)
			if pixel.G > pixel.R+30 && pixel.G > pixel.B+10 {
				accentPixels++
			}
		}
	}

	if accentPixels == 0 {
		t.Fatal("rounded top overlay did not paint the accent through the top-left corner")
	}
}

func TestRoundedTopStrokeLeavesOutsideCornerUnpainted(t *testing.T) {
	t.Parallel()

	const (
		boxX   = 12.0
		boxY   = 12.0
		boxW   = 80.0
		boxH   = 32.0
		radius = 8.0
	)

	res := &layout.Result{ //nolint:exhaustruct // synthetic rounded raster result
		Width:  110,
		Height: 70,
		Ops: []layout.Op{{ //nolint:exhaustruct // focused rounded-border operation
			Kind: layout.OpStrokeRect, X: boxX, Y: boxY, W: boxW, H: boxH,
			R: 0.1, G: 0.7, B: 0.4, Width: 4, Radius: radius,
			StrokeMask: layout.StrokeMaskTop,
		}},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false)
	if err != nil {
		t.Fatal(err)
	}

	corner := img.NRGBAAt(int(boxX*ptToPx)-2, int(boxY*ptToPx)-2)
	if int(corner.G) > int(corner.R)+20 {
		t.Fatalf("rounded top stroke painted the outside corner: pixel=%+v", corner)
	}
}

//nolint:cyclop // fixture setup and display-list probe intentionally stay together
func TestFixture56UsesRoundedTopOverlayGeometry(t *testing.T) {
	t.Parallel()

	base := filepath.Join("..", "..", "testdata", "golden")

	htmlData, err := os.ReadFile(filepath.Join(base, "fixture-56-architecture-diagram.html"))
	if err != nil {
		t.Fatal(err)
	}

	cssData, err := os.ReadFile(filepath.Join(base, "fixture-56-architecture-diagram.css"))
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(htmlData))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(string(cssData))
	if err != nil {
		t.Fatal(err)
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	res, err := layout.Layout(root, layout.Options{ //nolint:exhaustruct // fixture image-mode geometry
		Width: 1024 * cssPxToPt, Height: 1024 * cssPxToPt,
		Font: font, Sheets: []*css.Stylesheet{sheet}, Media: "screen", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	overlays := 0

	for _, op := range res.Ops {
		if op.Kind == layout.OpStrokeRect && op.StrokeMask == layout.StrokeMaskTop && op.Radius > 0 {
			overlays++
		}
	}

	if overlays == 0 {
		t.Fatal("fixture 56 emitted no rounded top StrokeMask overlays")
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

	img, err := Render(root, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 400, Font: face, Background: true,
	})
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

// inkSpan returns the min/max X of painted (non-white) pixels.
func inkSpan(img image.Image) (int, int) {
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

	return minX, maxX
}

// renderForInk renders root and returns its ink span.
func renderForInk(t *testing.T, root *html.Node, opts RenderOptions) (int, int) {
	t.Helper()

	img, err := Render(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	return inkSpan(img)
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
	htmlSrc := `<html><body style="margin:0;padding:0"><p style="margin:0;font-size:12pt">` +
		sample + `</p></body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	minX, maxX := renderForInk(t, root, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 200, Font: face, SmartWidth: false, Background: true,
	})
	if maxX <= minX {
		t.Fatal("no ink found")
	}

	span := float64(maxX - minX)
	// allow generous tolerance for glyph bearings
	if span < imgW*0.4 || span > imgW*2.5 {
		t.Errorf("ink span %v vs advance-based %v", span, imgW)
	}
}

//nolint:funlen,wsl,exhaustruct // subtest table for raster op policies
func TestRasterOpPolicyParity(t *testing.T) {
	t.Parallel()

	// 1. TextTransform uppercase
	t.Run("TextTransform", func(t *testing.T) {
		t.Parallel()

		res := &layout.Result{
			Width: 100, Height: 50,
			Ops: []layout.Op{
				{Kind: layout.OpText, X: 10, Y: 20, Text: "abc", Size: 12, TextTransform: "uppercase"},
			},
		}

		img, err := rasterizeContext(t.Context(), res, 50, true)
		if err != nil {
			t.Fatal(err)
		}

		minX, maxX := inkSpan(img)
		if maxX <= minX {
			t.Fatal("no text ink rendered for uppercase transform")
		}
	})

	// 2. Opacity 0.5
	t.Run("Opacity", func(t *testing.T) {
		t.Parallel()

		resOpaque := &layout.Result{
			Width: 50, Height: 50,
			Ops: []layout.Op{
				{Kind: layout.OpFillRect, X: 10, Y: 10, W: 30, H: 30, R: 1, G: 0, B: 0, Alpha: 1},
			},
		}
		imgOpaque, err := rasterizeContext(t.Context(), resOpaque, 50, false)
		if err != nil {
			t.Fatal(err)
		}

		resTrans := &layout.Result{
			Width: 50, Height: 50,
			Ops: []layout.Op{
				{Kind: layout.OpFillRect, X: 10, Y: 10, W: 30, H: 30, R: 1, G: 0, B: 0, Alpha: 1, PaintOpacity: 0.5},
			},
		}
		imgTrans, err := rasterizeContext(t.Context(), resTrans, 50, false)
		if err != nil {
			t.Fatal(err)
		}

		colOpaque := imgOpaque.NRGBAAt(20, 20)
		colTrans := imgTrans.NRGBAAt(20, 20)

		if colTrans.G <= colOpaque.G {
			t.Fatalf("expected translucent blend over white, got %v vs opaque %v", colTrans, colOpaque)
		}
	})

	// 3. Transform rotate(45deg)
	t.Run("TransformRotate45", func(t *testing.T) {
		t.Parallel()

		resRot := &layout.Result{
			Width: 100, Height: 100,
			Ops: []layout.Op{
				{
					Kind: layout.OpFillRect, X: 40, Y: 40, W: 20, H: 20, R: 1, G: 0, B: 0, Alpha: 1,
					Xform: layout.RotateDeg(45), XformSet: true,
				},
			},
		}
		imgRot, err := rasterizeContext(t.Context(), resRot, 100, false)
		if err != nil {
			t.Fatal(err)
		}

		minX, maxX := inkSpan(imgRot)
		if maxX <= minX {
			t.Fatal("no ink found for rotate(45deg)")
		}
	})

	// 4. WritingMode vertical-rl
	t.Run("WritingModeVertical", func(t *testing.T) {
		t.Parallel()

		resVert := &layout.Result{
			Width: 100, Height: 100,
			Ops: []layout.Op{
				{Kind: layout.OpText, X: 50, Y: 20, Text: "Vertical", Size: 12, RotateDeg: -90},
			},
		}
		imgVert, err := rasterizeContext(t.Context(), resVert, 100, false)
		if err != nil {
			t.Fatal(err)
		}

		minX, maxX := inkSpan(imgVert)
		if maxX <= minX {
			t.Fatal("no ink found for vertical text")
		}
	})
}
