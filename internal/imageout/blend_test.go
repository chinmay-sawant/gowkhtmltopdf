package imageout

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestPaintBlendedFillUsesMultiply(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	atlas := newGlyphAtlas()
	cache := newRasterImageCache()
	paint(img, &layout.Op{
		Kind: layout.OpFillRect,
		W:    1,
		H:    1,
		R:    0.5,
		G:    0.5,
		B:    0.5,
	}, 1, atlas, cache)

	multiply := layout.Op{Kind: layout.OpFillRect, W: 1, H: 1, R: 1}
	multiply.SetBlendMode("multiply")
	paint(img, &multiply, 1, atlas, cache)

	got := img.NRGBAAt(0, 0)
	if got.R != 127 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Fatalf("pixel = %#v, want red 127 over opaque black channels", got)
	}
}

// paintBlendedFullCanvas is the pre-IMPROV-08 reference: the blend scratch
// spans the whole destination canvas. The parity test uses it as the oracle
// for the op-bounded scratch.
func paintBlendedFullCanvas(dst *image.NRGBA, paintOp *layout.Op, pxPerPt float64) {
	source := image.NewNRGBA(dst.Bounds())
	opCopy := paintOp.Clone()
	opCopy.SetBlendMode("")
	paint(source, &opCopy, pxPerPt, newGlyphAtlas(), newRasterImageCache())
	compositeBlend(dst, source, paintOp.BlendMode)
}

// fillBlendBackdrop writes a deterministic translucent pattern so blending
// runs against a non-trivial backdrop instead of zeros.
func fillBlendBackdrop(img *image.NRGBA) {
	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetNRGBA(x, row, color.NRGBA{
				R: uint8((x*7 + row*3) % 256),   //nolint:gosec // modulo 256 fits uint8
				G: uint8((x*5 + row*11) % 256),  //nolint:gosec // modulo 256 fits uint8
				B: uint8((x*13 + row*17) % 256), //nolint:gosec // modulo 256 fits uint8
				A: uint8(160 + (x+2*row)%96),    //nolint:gosec // modulo 96 stays below 96
			})
		}
	}
}

// TestPaintOpBoundsStaysConservative covers the geometry contract the bounded
// blend relies on: a small op gets a scratch smaller than the canvas, text
// expands to its font box, a rotation swaps the extent, and an off-canvas op
// intersects to empty.
func TestPaintOpBoundsStaysConservative(t *testing.T) {
	t.Parallel()

	canvas := image.Rect(0, 0, 200, 150)

	fill := layout.Op{Kind: layout.OpFillRect, X: 50, Y: 40, W: 10, H: 10}
	bounds := paintOpBounds(&fill, 1)

	if bounds != image.Rect(48, 38, 62, 52) {
		t.Fatalf("fill bounds = %v, want the 2pt margin box %v", bounds, image.Rect(48, 38, 62, 52))
	}

	if clipped := bounds.Intersect(canvas); clipped.Dx() >= canvas.Dx() {
		t.Fatalf("small op scratch %v spans the whole canvas %v", clipped, canvas)
	}

	text := layout.Op{Kind: layout.OpText, X: 20, Y: 60, Size: 10, Text: "abcd"}
	if textBounds := paintOpBounds(&text, 1); textBounds != image.Rect(18, 48, 62, 67) {
		t.Fatalf("text bounds = %v, want the expanded font box %v", textBounds, image.Rect(18, 48, 62, 67))
	}

	rotated := layout.Op{Kind: layout.OpFillRect, X: 50, Y: 40, W: 20, H: 6}
	rotated.SetXform(layout.RotateDeg(90))
	rotatedBounds := paintOpBounds(&rotated, 1)

	if rotatedBounds != image.Rect(-48, 48, -38, 72) {
		t.Fatalf("rotated bounds = %v, want the swapped extent %v", rotatedBounds, image.Rect(-48, 48, -38, 72))
	}

	far := layout.Op{Kind: layout.OpFillRect, X: 500, Y: 500, W: 5, H: 5}
	if bounds := paintOpBounds(&far, 1).Intersect(canvas); !bounds.Empty() {
		t.Fatalf("off-canvas op bounds = %v, want empty", bounds)
	}
}

// blendParityCase is one synthetic op fed through both blend paths.
type blendParityCase struct {
	name string
	op   layout.Op
}

// blendParityCases covers the op shapes the bounded scratch has to reproduce:
// interior fills, blobs straddling the canvas edges, a fully off-canvas op,
// strokes, text, and a rotation.
func blendParityCases() []blendParityCase {
	rotated := layout.Op{
		Kind: layout.OpFillRect, X: 24, Y: 14, W: 18, H: 10,
		R: 0.4, G: 0.7, B: 0.1,
	}
	rotated.SetXform(layout.RotateDeg(30))

	return []blendParityCase{
		{
			name: "fill interior",
			op: blendOp(layout.Op{
				Kind: layout.OpFillRect, X: 12, Y: 10, W: 20, H: 14,
				R: 0.9, G: 0.3, B: 0.2, Alpha: 0.6,
			}, "multiply"),
		},
		{
			name: "blob at the left and top edges",
			op: blendOp(layout.Op{
				Kind: layout.OpFillRect, X: -17, Y: -9, W: 30, H: 24,
				R: 0.2, G: 0.8, B: 0.4,
			}, "screen"),
		},
		{
			name: "blob at the right and bottom edges",
			op: blendOp(layout.Op{
				Kind: layout.OpFillRect, X: 55, Y: 34, W: 30, H: 18,
				R: 0.7, G: 0.1, B: 0.9, Alpha: 0.45,
			}, "difference"),
		},
		{
			name: "fully outside the canvas",
			op:   blendOp(layout.Op{Kind: layout.OpFillRect, X: 200, Y: 200, W: 10, H: 10, R: 1}, "darken"),
		},
		{
			name: "stroke rect",
			op: blendOp(layout.Op{
				Kind: layout.OpStrokeRect, X: 8, Y: 6, W: 24, H: 18, Width: 3,
				R: 1, G: 1, B: 0.2, Alpha: 0.7,
			}, "multiply"),
		},
		{
			name: "text",
			op:   blendOp(layout.Op{Kind: layout.OpText, X: 6, Y: 34, Text: "Blend", Size: 12, R: 1}, "darken"),
		},
		{
			name: "rotated fill",
			op:   blendOp(rotated, "difference"),
		},
	}
}

func blendOp(op layout.Op, mode string) layout.Op {
	op.SetBlendMode(mode)

	return op
}

// TestElementGroupBlendsOncePerGroup proves element-group semantics: two
// overlapping opaque children inside one multiplied group composite with each
// other first, then multiply once against the backdrop. Per-op blending
// multiplies the second child into the already multiplied first child and
// produces black in the overlap.
func TestElementGroupBlendsOncePerGroup(t *testing.T) {
	t.Parallel()

	group := &layout.BlendGroup{ID: 1, Mode: "multiply", Isolate: true}

	red := layout.Op{Kind: layout.OpFillRect, X: 10, Y: 10, W: 50, H: 50, R: 1}
	red.SetBlendGroup(group)

	blue := layout.Op{Kind: layout.OpFillRect, X: 40, Y: 40, W: 50, H: 50, B: 1}
	blue.SetBlendGroup(group)

	grouped := whiteRasterCanvas(100, 100)
	if err := paintWithElementGroups(
		t.Context(), grouped, []layout.Op{red, blue}, 0, 1,
		newGlyphAtlas(), newRasterImageCache(),
	); err != nil {
		t.Fatal(err)
	}

	if got := grouped.NRGBAAt(45, 45); got != (color.NRGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Fatalf("group overlap = %#v, want opaque blue (children composite before the backdrop)", got)
	}

	if got := grouped.NRGBAAt(20, 20); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("group red area = %#v, want opaque red", got)
	}

	if got := grouped.NRGBAAt(80, 80); got != (color.NRGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Fatalf("group blue area = %#v, want opaque blue", got)
	}

	// Per-op reference: multiply blue against the already multiplied red.
	perOp := whiteRasterCanvas(100, 100)
	atlas := newGlyphAtlas()
	cache := newRasterImageCache()
	perOpOps := []layout.Op{
		blendOp(layout.Op{Kind: layout.OpFillRect, X: 10, Y: 10, W: 50, H: 50, R: 1}, "multiply"),
		blendOp(layout.Op{Kind: layout.OpFillRect, X: 40, Y: 40, W: 50, H: 50, B: 1}, "multiply"),
	}

	for i := range perOpOps {
		paint(perOp, &perOpOps[i], 1, atlas, cache)
	}

	if got := perOp.NRGBAAt(45, 45); got != (color.NRGBA{A: 255}) {
		t.Fatalf("per-op overlap = %#v, want black (the pre-group approximation)", got)
	}

	if grouped.NRGBAAt(45, 45) == perOp.NRGBAAt(45, 45) {
		t.Fatal("grouped and per-op rasterization agree in the overlap; grouping had no effect")
	}
}

// TestIsolatedGroupResetsBackdrop proves an isolated group starts on a
// transparent backdrop: a multiply child inside it cannot see the red page
// content beneath the group. Outside isolation the same child multiplies
// against red and darkens.
func TestIsolatedGroupResetsBackdrop(t *testing.T) {
	t.Parallel()

	group := &layout.BlendGroup{ID: 1, Mode: "", Isolate: true}
	child := layout.Op{Kind: layout.OpFillRect, X: 20, Y: 20, W: 60, H: 60, R: 0.5, G: 0.5, B: 0.5}
	child.SetBlendGroup(group)
	child.SetBlendMode("multiply")

	grouped := redRasterCanvas(100, 100)
	if err := paintWithElementGroups(
		t.Context(), grouped, []layout.Op{child}, 0, 1,
		newGlyphAtlas(), newRasterImageCache(),
	); err != nil {
		t.Fatal(err)
	}

	if got := grouped.NRGBAAt(50, 50); got != (color.NRGBA{R: 127, G: 127, B: 127, A: 255}) {
		t.Fatalf("isolated group pixel = %#v, want neutral gray (multiply against transparent)", got)
	}
}

// TestRasterizeSiblingOutsideBlendGroupKeepsPixels pins fixture-62 cell 16 on
// the raster path: the plain flex sibling sorts between the blended item's
// backgrounds and its text in paint order. An innermost-frame scratch absorbed
// that sibling into the multiply group, so its translucent blue multiplied
// against the orange parent instead of alpha-compositing over it.
//
//nolint:cyclop // full-canvas pixel census and setup assertions
func TestRasterizeSiblingOutsideBlendGroupKeepsPixels(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body style="margin:0;background:#f80;padding:6px;` +
		`display:flex;gap:8px;font-size:7pt;color:#fff">` +
		`<div style="mix-blend-mode:multiply;background:#08f;padding:3px 7px">multiply</div>` +
		`<div style="background:rgba(0,136,255,0.55);padding:3px 7px">normal</div>` +
		`</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	res, err := layout.Layout(root, layout.Options{
		Width: 200, Height: 80, Font: font, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !hasElementGroups(res.Ops) {
		t.Fatal("layout produced no element-group ops for mix-blend-mode")
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	multiplySeen, plainSeen := false, false

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pixel := img.NRGBAAt(x, y)

			// #08f multiply #f80 = rgb(0,72,0).
			if pixel.R < 8 && pixel.G >= 66 && pixel.G <= 78 && pixel.B < 8 {
				multiplySeen = true
			}

			// rgba(0,136,255,0.55) over #f80 = rgb(115,136,140).
			if pixel.R >= 110 && pixel.R <= 120 && pixel.G >= 132 && pixel.G <= 140 &&
				pixel.B >= 136 && pixel.B <= 144 {
				plainSeen = true
			}
		}
	}

	if !multiplySeen {
		t.Fatal("multiply chip pixel rgb(0,72,0) not found")
	}

	if !plainSeen {
		t.Fatal("plain sibling pixel rgb(115,136,140) not found; it was multiplied into the group or flattened")
	}
}

func whiteRasterCanvas(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillNRGBAOpaque(img, img.Bounds(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	return img
}

func redRasterCanvas(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillNRGBAOpaque(img, img.Bounds(), color.NRGBA{R: 255, A: 255})

	return img
}

// TestTilePaintElementGroup drives the strip/tile paint loop, which shares
// the group buffering with the full-canvas raster path.
func TestTilePaintElementGroup(t *testing.T) {
	t.Parallel()

	group := &layout.BlendGroup{ID: 1, Mode: "multiply", Isolate: true}

	red := layout.Op{Kind: layout.OpFillRect, X: 10, Y: 10, W: 50, H: 50, R: 1}
	red.SetBlendGroup(group)

	blue := layout.Op{Kind: layout.OpFillRect, X: 40, Y: 40, W: 50, H: 50, B: 1}
	blue.SetBlendGroup(group)

	img := whiteRasterCanvas(100, 100)
	if err := paintDisplayList(
		t.Context(), img, []layout.Op{red, blue}, 1, newGlyphAtlas(), newRasterImageCache(),
	); err != nil {
		t.Fatal(err)
	}

	if got := img.NRGBAAt(45, 45); got != (color.NRGBA{B: 255, A: 255}) {
		t.Fatalf("tile group overlap = %#v, want opaque blue", got)
	}
}

// TestRasterizeElementGroupFromLayout drives the layout -> raster path
// instead of hand-built ops: the mix-blend-mode group from HTML must route
// through group buffering, so the child overlap stays blue rather than the
// per-op multiply result (black).
//
//nolint:cyclop,varnamelen // full-canvas pixel census uses x/y loop names
func TestRasterizeElementGroupFromLayout(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body style="margin:0;background:#fff">` +
		`<div style="mix-blend-mode:multiply;width:90px;height:80px;position:relative">` +
		`<div style="position:absolute;left:0;top:0;width:60px;height:60px;background:#f00"></div>` +
		`<div style="position:absolute;left:30px;top:30px;width:60px;height:60px;background:#00f"></div>` +
		`</div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	res, err := layout.Layout(root, layout.Options{
		Width: 100, Height: 100, Font: font, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !hasElementGroups(res.Ops) {
		t.Fatal("layout produced no element-group ops for mix-blend-mode")
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	foundBlue := false

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pixel := img.NRGBAAt(x, y)
			if pixel == (color.NRGBA{B: 255, A: 255}) {
				foundBlue = true
			}

			if pixel == (color.NRGBA{A: 255}) {
				t.Fatalf("pixel (%d,%d) is black; per-op blending leaked into the group", x, y)
			}
		}
	}

	if !foundBlue {
		t.Fatal("rendered group has no blue child region")
	}
}

// TestPaintBlendedMatchesFullCanvasScratch compares the op-bounded scratch
// against the full-canvas reference for synthetic ops, including blobs that
// straddle the canvas edges. Pixel parity is the whole point: the bounded
// scratch must never clip a pixel the full canvas would have blended.
func TestPaintBlendedMatchesFullCanvasScratch(t *testing.T) {
	t.Parallel()

	canvas := image.Rect(0, 0, 64, 48)

	for _, testCase := range blendParityCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			want := image.NewNRGBA(canvas)
			fillBlendBackdrop(want)
			paintBlendedFullCanvas(want, &testCase.op, 1)

			got := image.NewNRGBA(canvas)
			fillBlendBackdrop(got)
			paintBlended(got, &testCase.op, 1, newGlyphAtlas(), newRasterImageCache())

			if !bytes.Equal(got.Pix, want.Pix) {
				t.Fatalf("bounded blend differs from the full-canvas reference for %s", testCase.name)
			}
		})
	}
}
