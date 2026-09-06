//nolint:testpackage,exhaustruct // border-image display-list coverage
package layout

import (
	"bytes"
	"image/png"
	"testing"
)

func TestBorderImageStretchUsesSlicedBorderOps(t *testing.T) {
	t.Parallel()

	eng := newBorderImageStretchEngine()
	sty := newBorderImageStretchStyle()

	ops := eng.appendBorderImage(nil, sty, 0, 0, 100, 80)

	if len(ops) != 8 {
		t.Fatalf("border-image stretch ops = %d, want 8 sliced border ops", len(ops))
	}

	assertBorderImageSlices(t, ops)
	assertBorderImageOuterBounds(t, ops)
}

func newBorderImageStretchEngine() *engine {
	return &engine{ //nolint:exhaustruct // image resolver only
		opts: Options{ //nolint:exhaustruct // image provider only
			Images: func(string) ([]byte, error) {
				return tinyPNG(4, 4), nil
			},
		},
		scale: 1,
	}
}

func newBorderImageStretchStyle() ResolvedStyle {
	return ResolvedStyle{ //nolint:exhaustruct // border-image fields under test
		BorderImageSource: "url(border.png)",
		BorderImageSlice:  "1",
		BorderImageRepeat: "stretch",
		BorderTop:         border{Width: 6, PaintWidth: 4.5},
		BorderRight:       border{Width: 6, PaintWidth: 4.5},
		BorderBottom:      border{Width: 6, PaintWidth: 4.5},
		BorderLeft:        border{Width: 6, PaintWidth: 4.5},
		BorderImageOutset: "2",
	}
}

func assertBorderImageSlices(t *testing.T, ops []Op) {
	t.Helper()

	sourceSizes := collectBorderImageSourceSizes(t, ops)

	if sourceSizes[[2]int{1, 1}] != 4 || sourceSizes[[2]int{2, 1}] != 2 ||
		sourceSizes[[2]int{1, 2}] != 2 {
		t.Fatalf("border-image source slices = %v, want four corners, two horizontal edges, two vertical edges",
			sourceSizes)
	}
}

func collectBorderImageSourceSizes(t *testing.T, ops []Op) map[[2]int]int {
	t.Helper()

	sourceSizes := map[[2]int]int{}

	for _, paintOp := range ops {
		if paintOp.Kind != OpImage || !paintOp.IsBackground {
			t.Fatalf("border-image op = %+v, want background image", paintOp)
		}

		cfg, err := png.DecodeConfig(bytes.NewReader(paintOp.Image))

		if err != nil {
			t.Fatalf("decode border-image slice: %v", err)
		}

		sourceSizes[[2]int{cfg.Width, cfg.Height}]++
	}

	return sourceSizes
}

func assertBorderImageOuterBounds(t *testing.T, ops []Op) {
	t.Helper()

	minX, minY, maxX, maxY := borderImageOuterBoundsOf(ops)

	if minX != -9 || minY != -9 || maxX != 109 || maxY != 89 {
		t.Fatalf("border-image outer bounds = (%.1f,%.1f)-(%.1f,%.1f), want (-9,-9)-(109,89)",
			minX, minY, maxX, maxY)
	}
}

func borderImageOuterBoundsOf(ops []Op) (float64, float64, float64, float64) {
	minX, minY := ops[0].X, ops[0].Y
	maxX, maxY := ops[0].X+ops[0].W, ops[0].Y+ops[0].H

	for _, paintOp := range ops[1:] {
		if paintOp.X < minX {
			minX = paintOp.X
		}

		if paintOp.Y < minY {
			minY = paintOp.Y
		}

		if paintOp.X+paintOp.W > maxX {
			maxX = paintOp.X + paintOp.W
		}

		if paintOp.Y+paintOp.H > maxY {
			maxY = paintOp.Y + paintOp.H
		}
	}

	return minX, minY, maxX, maxY
}

func TestBorderImageShorthandParsesAllSections(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyBorderImageProps(&style, "border-image", "url(logo.png) 10 / 6px / 2 stretch") {
		t.Fatal("border-image shorthand was not recognized")
	}

	if style.BorderImageSource == "" || style.BorderImageSlice != "10" ||
		style.BorderImageWidth != "6px" || style.BorderImageOutset != "2" ||
		style.BorderImageRepeat != "stretch" {
		t.Fatalf("border-image shorthand = source=%q slice=%q width=%q outset=%q repeat=%q",
			style.BorderImageSource, style.BorderImageSlice, style.BorderImageWidth,
			style.BorderImageOutset, style.BorderImageRepeat)
	}
}

func TestBorderImageUsesPaintWidthForContent(t *testing.T) {
	t.Parallel()

	html := `<html><body style="font-size:9.5pt;">` + "\n" +
		`<div style="box-sizing:border-box;border:6px solid transparent;` +
		`border-image-source:url(border.png);border-image-slice:1;padding:8px;width:90px;">img border</div>` + "\n" +
		`</body></html>`

	res := layoutHTML(t, html)

	texts := opsOfKind(res, OpText)
	if len(texts) != 1 || texts[0].Text != "img border" {
		t.Fatalf("border-image content text = %+v, want one unwrapped run", texts)
	}

	boxNode := findNamedBox(res.root, "div")
	if boxNode == nil || boxNode.height <= 30 {
		t.Fatalf("border-image auto height = %v, want bottom image border included", boxNode)
	}
}
