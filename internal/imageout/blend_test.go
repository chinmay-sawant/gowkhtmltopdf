package imageout

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
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
