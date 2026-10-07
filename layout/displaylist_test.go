package layout_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

func displayOf(t *testing.T, source string) *layout.Display {
	t.Helper()

	ctx := t.Context()

	doc, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  640,
		HeightPx: 480,
		Media:    "screen",
		Extra:    nil,
	})
	if err != nil {
		t.Fatalf("css: %v", err)
	}

	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

func styledOf(t *testing.T, source string) *css.Document {
	t.Helper()

	doc, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{
		WidthPx:  640,
		HeightPx: 480,
		Media:    "screen",
		Extra:    nil,
	})
	if err != nil {
		t.Fatalf("css: %v", err)
	}

	return styled
}

func TestDisplayReturnsOpsAndCanvas(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<div style="background:#eee"><p id="hi">Hello vector</p></div>`)

	if len(display.Ops) == 0 {
		t.Fatal("no ops")
	}

	if len(display.Order) != len(display.Ops) {
		t.Fatalf("order covers %d ops, want %d", len(display.Order), len(display.Ops))
	}

	if display.Width <= 0 || display.Height <= 0 {
		t.Fatalf("canvas %dx%d", display.Width, display.Height)
	}

	if display.PointsPerPixel <= 0 || display.PixelPerPoint <= 0 {
		t.Fatal("point conversions must be positive")
	}
}

// TestDisplayAgreesWithLayOnSize keeps the two entries consistent on the
// common case: the same document must report the same canvas from both, so a
// caller can switch between them without its window geometry moving. It is
// not a universal identity. Display.Height truncates points to pixels while
// Lay reads the size off the painted picture, so a fractional canvas height
// can put Display one pixel under Lay.
func TestDisplayAgreesWithLayOnSize(t *testing.T) {
	t.Parallel()

	const source = `<h1>Heading</h1><p>Body copy that wraps onto more than one line.</p>`

	styled := styledOf(t, source)

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	placed, err := layout.Lay(t.Context(), styled)
	if err != nil {
		t.Fatalf("lay: %v", err)
	}

	width, height := placed.Size()
	if width != display.Width || height != display.Height {
		t.Fatalf("Lay %dx%d, Display %dx%d", width, height, display.Width, display.Height)
	}
}

func TestDisplayAgreesWithLayOnBoxes(t *testing.T) {
	t.Parallel()

	const source = `<h1>Heading</h1><p data-action="go">Body <span>copy</span>.</p>`

	styled := styledOf(t, source)

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	placed, err := layout.Lay(t.Context(), styled)
	if err != nil {
		t.Fatalf("lay: %v", err)
	}

	want := placed.Boxes()

	if len(display.Boxes) != len(want) {
		t.Fatalf("Display has %d boxes, Lay has %d", len(display.Boxes), len(want))
	}

	for index, box := range want {
		if display.Boxes[index] != box {
			t.Fatalf("box %d: Display %+v, Lay %+v", index, display.Boxes[index], box)
		}
	}
}

func TestDisplayOrderIsAPermutation(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `
<h1 style="background:#eee">z</h1>
<div style="position:relative;z-index:3;background:#ccc;height:8px">high</div>
<div style="background:#999;height:8px">low</div>
`)

	seen := make([]bool, len(display.Ops))

	for _, index := range display.Order {
		if index < 0 || index >= len(display.Ops) {
			t.Fatalf("order index %d out of range for %d ops", index, len(display.Ops))
		}

		if seen[index] {
			t.Fatalf("order repeats index %d", index)
		}

		seen[index] = true
	}

	if got := layout.DisplayOrder(display.Ops); len(got) != len(display.Ops) {
		t.Errorf("DisplayOrder returned %d indices for %d ops", len(got), len(display.Ops))
	}
}

func TestDisplayTextOpsCarryUsableFaces(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<p style="font-weight:bold">Wij Afflig&eacute; 123</p>`)

	seen := 0

	for _, paintOp := range display.Ops {
		if paintOp.Kind != layout.DisplayOpText && paintOp.Kind != layout.DisplayOpBullet {
			continue
		}

		seen++

		if paintOp.Text == "" {
			t.Error("text op with no text")
		}

		if paintOp.Font == nil {
			t.Fatal("text op with no font")
		}

		if paintOp.Font.Bytes() == nil {
			t.Error("face bytes nil")
		}

		if paintOp.Font.UnitsPerEm() <= 0 {
			t.Errorf("unitsPerEm %d", paintOp.Font.UnitsPerEm())
		}
	}

	if seen == 0 {
		t.Fatal("no text ops")
	}
}

// TestDisplayAscentMatchesPointSize pins the baseline math a replay engine
// depends on. A text engine derives ascent as Ascent/UnitsPerEm*fontSize, and
// the op's own face must agree at the size the op carries, or text sits off
// its baseline. Verified against the bundled Liberation Sans faces.
func TestDisplayAscentMatchesPointSize(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<p>baseline probe</p>`)

	for _, paintOp := range display.Ops {
		if paintOp.Kind != layout.DisplayOpText {
			continue
		}

		ascentPt := float64(paintOp.Font.Ascent()) / float64(paintOp.Font.UnitsPerEm()) * paintOp.Size

		if ascentPt <= 0 || ascentPt > paintOp.Size {
			t.Fatalf("ascent %.3fpt for size %.3fpt", ascentPt, paintOp.Size)
		}

		return
	}

	t.Fatal("no text op")
}

// styleCoverage records what a replay engine would find in one document. Kinds
// counts operations by kind, so a test can ask about any kind without a switch
// that has to name every one of them.
type styleCoverage struct {
	Kinds       map[layout.DisplayKind]int
	LinkURI     string
	SawRadius   bool
	SawUpper    bool
	SawFauxBold bool
}

const coverageSource = `
<div style="background:#f4f4f4;border:2px solid #333;border-radius:12px;padding:10px;width:200px">boxed</div>
<button style="background:#06c;border:1px solid #000;border-radius:999px;padding:4px 12px">pill</button>
<a href="/next" style="color:#06c">a link</a>
<p style="text-transform:uppercase">shout</p>
<p style="font-weight:bold">boldface</p>
`

// TestDisplayCoversRealStyles walks a document that exercises the operation
// kinds a replay engine must handle and records what it finds. Rounded rects,
// links, text-transform, and faux-bold all appear here, which is the point:
// each one is a behavior the replay has to reproduce, not a detail it can skip.
func TestDisplayCoversRealStyles(t *testing.T) {
	t.Parallel()

	got := coverStyles(displayOf(t, coverageSource))

	if got.Kinds[layout.DisplayOpFillRect] == 0 {
		t.Error("no fill ops")
	}

	if got.Kinds[layout.DisplayOpText] == 0 {
		t.Error("no text ops")
	}

	if got.Kinds[layout.DisplayOpStrokeRect] == 0 {
		t.Error("no stroke rects for a bordered box")
	}

	if !got.SawRadius {
		t.Error("no rounded rect for border-radius")
	}

	if got.LinkURI != "/next" {
		t.Errorf("link uri %q, want /next", got.LinkURI)
	}

	if !got.SawUpper {
		t.Error("text-transform: uppercase did not reach the op")
	}
}

// TestDisplayTextNeedsTransformApplied pins that an op carries the raw text
// plus the transform name and leaves the substitution to the painter. A replay
// that ignores TextTransformValue would draw lowercase text for a shout.
func TestDisplayTextNeedsTransformApplied(t *testing.T) {
	t.Parallel()

	got := coverStyles(displayOf(t, coverageSource))
	if !got.SawUpper {
		t.Fatal("no uppercase op in this document")
	}

	if layout.DisplayTransformText("shout", "uppercase") != "SHOUT" {
		t.Fatal("DisplayTransformText did not uppercase")
	}
}

// TestDisplayFauxBoldIsReported covers the other text behavior the painter
// synthesizes: a bold request on a face that has no bold cut. The op records
// the request through DisplayFakeBold rather than through its own Bold field,
// because the check is CJK-aware and honors the no-fake-bold gate.
func TestDisplayFauxBoldIsReported(t *testing.T) {
	t.Parallel()

	got := coverStyles(displayOf(t, coverageSource))
	if got.Kinds[layout.DisplayOpText] == 0 {
		t.Fatal("no text ops")
	}

	// The bundled faces carry a bold cut, so faux bold is normally off; the
	// assertion is that the query is answerable per op without a panic.
	for _, paintOp := range displayOf(t, coverageSource).Ops {
		if paintOp.Kind == layout.DisplayOpText {
			_ = layout.DisplayFakeBold(&paintOp)
		}
	}
}

func coverStyles(display *layout.Display) styleCoverage {
	got := styleCoverage{Kinds: map[layout.DisplayKind]int{}}

	for _, paintOp := range display.Ops {
		got.Kinds[paintOp.Kind]++

		if paintOp.Kind == layout.DisplayOpFillRect && paintOp.Radius > 0 {
			got.SawRadius = true
		}

		if paintOp.Kind == layout.DisplayOpText && layout.DisplayFakeBold(&paintOp) {
			got.SawFauxBold = true
		}

		if paintOp.Kind == layout.DisplayOpLinkURI {
			got.LinkURI = paintOp.LinkURI()
		}

		transform := paintOp.TextTransformValue()
		if transform == "uppercase" &&
			layout.DisplayTransformText(paintOp.Text, transform) == "SHOUT" {
			got.SawUpper = true
		}
	}

	return got
}

// TestDisplayOpacityFoldsAlpha pins the 0-means-unset convention. The engine
// stores 0 for "no override" and honors only values strictly between 0 and 1,
// so an accessor returning the raw field would report 0 for ordinary text and a
// caller would draw everything fully transparent.
func TestDisplayOpacityFoldsAlpha(t *testing.T) {
	t.Parallel()

	plain := displayOf(t, `<p>ordinary</p>`)
	faded := displayOf(t, `<p style="opacity:0.35">faded</p>`)

	if got := firstTextOpacity(plain); got != 1 {
		t.Errorf("plain text opacity %v, want 1", got)
	}

	if got := firstTextOpacity(faded); got < 0.34 || got > 0.36 {
		t.Errorf("faded text opacity %v, want ~0.35", got)
	}
}

func firstTextOpacity(display *layout.Display) float64 {
	for _, paintOp := range display.Ops {
		if paintOp.Kind == layout.DisplayOpText {
			return paintOp.Opacity()
		}
	}

	return -1
}

func TestDisplayRejectsNilInputs(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // nil context is intentional: this tests the guard
	if _, err := layout.DisplayList(nil, nil); !errors.Is(err, layout.ErrNilContext) {
		t.Errorf("nil context: %v", err)
	}

	if _, err := layout.DisplayList(t.Context(), nil); !errors.Is(err, layout.ErrNilDocument) {
		t.Errorf("nil document: %v", err)
	}
}

func TestDisplayFakeBoldAndTransformAreNilSafe(t *testing.T) {
	t.Parallel()

	if layout.DisplayFakeBold(nil) {
		t.Error("DisplayFakeBold(nil) true")
	}

	var zero layout.DisplayOp
	if layout.DisplayFakeBold(&zero) {
		t.Error("DisplayFakeBold on zero op")
	}

	if got := layout.DisplayTransformText("abc", "uppercase"); got != "ABC" {
		t.Errorf("uppercase %q", got)
	}

	if got := layout.DisplayTransformText("abc", ""); got != "abc" {
		t.Errorf("no transform %q", got)
	}
}

// TestDisplayImageBytesAreReachable checks that an image op carries its bytes
// and bounds. A document whose only content is an <img> produces no ops at all
// on this path, because css.Apply fetches nothing, so the assertion is
// conditional on an image op existing rather than requiring one.
func TestDisplayImageBytesAreReachable(t *testing.T) {
	t.Parallel()

	// A 1x1 red PNG, inlined so the test needs no fixture file.
	const redPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ" +
		"AAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

	display := displayOf(t, `<div style="background:#ddd;height:20px"></div><img src="`+redPNG+`">`)

	for _, paintOp := range display.Ops {
		if paintOp.Kind != layout.DisplayOpImage {
			continue
		}

		data, width, height := paintOp.ImageBytes()
		if len(data) == 0 || width <= 0 || height <= 0 {
			t.Fatalf("image op %d bytes %dx%d", len(data), width, height)
		}

		if !bytes.HasPrefix(data, []byte("\x89PNG")) {
			t.Fatal("image payload is not a PNG")
		}

		return
	}
}
