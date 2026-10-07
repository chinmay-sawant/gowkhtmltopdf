package layout

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// decodeNRGBA decodes PNG bytes into an NRGBA copy so tests can sample pixels
// with stable coordinates.
func decodeNRGBA(t *testing.T, data []byte) *image.NRGBA {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), img, bounds.Min, draw.Src)

	return out
}

func TestConicGradientRasterPixels(t *testing.T) {
	t.Parallel()

	pngData, width, height, rendered := renderGradientPNG(
		"conic-gradient(from 0deg, red, blue)", 64, 64, [3]float64{},
	)
	if !rendered || width != 64 || height != 64 {
		t.Fatalf("conic render ok=%v size=%dx%d", rendered, width, height)
	}

	img := decodeNRGBA(t, pngData)
	// 0deg points up and angles grow clockwise: top is red, left is past 270deg.
	top := img.NRGBAAt(32, 1)
	left := img.NRGBAAt(1, 32)

	if top.R < 180 || top.B > 90 {
		t.Errorf("top pixel = %+v, want red near from angle", top)
	}

	if left.B < 180 || left.R > 90 {
		t.Errorf("left pixel = %+v, want blue at 270deg clockwise", left)
	}

	rotated, _, _, rotatedOK := renderGradientPNG(
		"conic-gradient(from 90deg, red, blue)", 64, 64, [3]float64{},
	)
	if !rotatedOK {
		t.Fatal("from 90deg conic render failed")
	}

	if got := decodeNRGBA(t, rotated).NRGBAAt(32, 1); got.B < 180 {
		t.Errorf("from 90deg top pixel = %+v, want blue", got)
	}
}

func TestRepeatingConicGradientWraps(t *testing.T) {
	t.Parallel()

	pngData, _, _, repeatOK := renderGradientPNG(
		"repeating-conic-gradient(from 0deg, red 0%, blue 25%)", 64, 64, [3]float64{},
	)
	if !repeatOK {
		t.Fatal("repeating conic render failed")
	}

	// 197deg is past the 25% stop; repeating must wrap back to red. A
	// non-repeating gradient would clamp at blue.
	img := decodeNRGBA(t, pngData)
	pixel := img.NRGBAAt(22, 62)

	if pixel.R < 150 || pixel.B > 100 {
		t.Errorf("wrapped pixel = %+v, want red after period wrap", pixel)
	}
}

func TestParseClipPathShapes(t *testing.T) {
	t.Parallel()

	assertClipPathInsetShape(t)
	assertClipPathCircleShape(t)
	assertClipPathEllipseShape(t)
	assertClipPathPolygonShape(t)
	assertClipPathInvalidValues(t)
}

// assertClipPathInsetShape pins inset() parsing, resolved lengths, and
// containment.
func assertClipPathInsetShape(t *testing.T) {
	t.Helper()

	inset, insetOK := parseClipPathShape("inset(10px 20px 30px 40px)", 12)
	if !insetOK || inset.kind != clipPathInset {
		t.Fatalf("inset parse ok=%v kind=%v", insetOK, inset.kind)
	}

	if !near(inset.top.value, 7.5) || !near(inset.right.value, 15) ||
		!near(inset.bottom.value, 22.5) || !near(inset.left.value, 30) {
		t.Errorf("inset lengths = %v %v %v %v", inset.top, inset.right, inset.bottom, inset.left)
	}

	if !inset.contains(50, 50, 100, 100) || inset.contains(5, 5, 100, 100) {
		t.Error("inset containment wrong")
	}
}

// assertClipPathCircleShape pins circle() parsing and containment.
func assertClipPathCircleShape(t *testing.T) {
	t.Helper()

	circle, circleOK := parseClipPathShape("circle(30% at 50% 50%)", 12)
	if !circleOK || circle.kind != clipPathCircle {
		t.Fatalf("circle parse ok=%v kind=%v", circleOK, circle.kind)
	}

	if !circle.contains(50, 50, 100, 100) || circle.contains(95, 50, 100, 100) {
		t.Error("circle containment wrong")
	}
}

// assertClipPathEllipseShape pins ellipse() parsing and containment.
func assertClipPathEllipseShape(t *testing.T) {
	t.Helper()

	ellipse, ellipseOK := parseClipPathShape("ellipse(25% 10% at 50% 50%)", 12)
	if !ellipseOK || ellipse.kind != clipPathEllipse {
		t.Fatalf("ellipse parse ok=%v kind=%v", ellipseOK, ellipse.kind)
	}

	if !ellipse.contains(60, 55, 100, 100) || ellipse.contains(80, 50, 100, 100) {
		t.Error("ellipse containment wrong")
	}
}

// assertClipPathPolygonShape pins polygon() parsing, containment, and the
// evenodd fill rule.
func assertClipPathPolygonShape(t *testing.T) {
	t.Helper()

	poly, polyOK := parseClipPathShape("polygon(50% 0%, 100% 100%, 0% 100%)", 12)
	if !polyOK || poly.kind != clipPathPolygon {
		t.Fatalf("polygon parse ok=%v kind=%v", polyOK, poly.kind)
	}

	if !poly.contains(50, 80, 100, 100) || poly.contains(5, 5, 100, 100) {
		t.Error("polygon containment wrong")
	}

	evenOdd, evenOddOK := parseClipPathShape("polygon(evenodd, 50% 0%, 100% 100%, 0% 100%)", 12)
	if !evenOddOK || evenOdd.fillRule != clipPathEvenOdd || !evenOdd.contains(50, 80, 100, 100) {
		t.Error("polygon evenodd parse wrong")
	}
}

// assertClipPathInvalidValues pins that unsupported and malformed values
// parse to no shape.
func assertClipPathInvalidValues(t *testing.T) {
	t.Helper()

	for _, invalid := range []string{
		"url(#c)", "path('M0 0')", "shape(from 0 0, line to 10 10)",
		"inset(10px", "polygon(0 0, 10 10)", "circle(10px 20px)", "",
	} {
		if _, invalidOK := parseClipPathShape(invalid, 12); invalidOK {
			t.Errorf("parseClipPathShape(%q) accepted invalid/unsupported value", invalid)
		}
	}
}

func TestBackgroundImageClipPathMasksPixels(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
.box {
  width: 100pt;
  height: 100pt;
  background-image: conic-gradient(from 0deg, #ff0000, #0000ff);
  clip-path: circle(30% at 50% 50%);
}`)
	if err != nil {
		t.Fatal(err)
	}

	root := mustParse(t, `<html><body><div class="box"></div></body></html>`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 400, Background: true, Media: "print",
		Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	var imageOp *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpImage && res.Ops[i].IsBackground {
			imageOp = &res.Ops[i]

			break
		}
	}

	if imageOp == nil {
		t.Fatal("no background image op emitted")
	}

	img := decodeNRGBA(t, imageOp.Image)

	if got := img.NRGBAAt(50, 50); got.A != 255 {
		t.Errorf("center alpha = %d, want 255", got.A)
	}

	if got := img.NRGBAAt(1, 1); got.A != 0 {
		t.Errorf("corner alpha = %d, want 0 (masked outside circle)", got.A)
	}
}

func TestImageClipPathInsetAndInvalidFallback(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
img.clipped { clip-path: inset(25%); }
img.plain { clip-path: url(#missing); }`)
	if err != nil {
		t.Fatal(err)
	}

	source := tinyPNG(40, 40)
	provider := func(string) ([]byte, error) { return source, nil }

	root := mustParse(t, `<html><body><img class="clipped" src="x.png"></body></html>`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 400, Background: true, Media: "print",
		Sheets: []*css.Stylesheet{sheet}, Images: provider,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("clipped img ops = %d, want 1", len(imgs))
	}

	img := decodeNRGBA(t, imgs[0].Image)

	if got := img.NRGBAAt(20, 20); got.A != 255 {
		t.Errorf("center alpha = %d, want 255", got.A)
	}

	if got := img.NRGBAAt(2, 2); got.A != 0 {
		t.Errorf("corner alpha = %d, want 0 (masked by inset)", got.A)
	}

	plainRoot := mustParse(t, `<html><body><img class="plain" src="x.png"></body></html>`)

	plainRes, err := Layout(plainRoot, Options{
		Width: testViewport, Height: 400, Background: true, Media: "print",
		Sheets: []*css.Stylesheet{sheet}, Images: provider,
	})
	if err != nil {
		t.Fatalf("Layout plain: %v", err)
	}

	plainImgs := opsOfKind(plainRes, OpImage)
	if len(plainImgs) != 1 {
		t.Fatalf("plain img ops = %d, want 1", len(plainImgs))
	}

	if !bytes.Equal(plainImgs[0].Image, source) {
		t.Error("unsupported clip-path must leave image bytes unchanged")
	}
}
