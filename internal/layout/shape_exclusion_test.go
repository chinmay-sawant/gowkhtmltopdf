package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// A left float with shape-outside:circle(50%) must shorten mid-height lines
// more than near-top lines (contour wrap, not a full rectangle).
//
//nolint:cyclop,funlen // shape regression checks parsing, layout, and contour geometry
func TestShapeOutsideCircleShortensLines(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; line-height: 12pt; }
.box { overflow: hidden; width: 240pt; }
.f {
  float: left;
  width: 80pt;
  height: 80pt;
  margin: 0;
  background: #9cf;
  shape-outside: circle(50%);
}
`)
	words := strings.Repeat("word ", 60)

	root, err := html.Parse(`<html><body>
<div class="box"><div class="f">F</div><p style="margin:0">` + words + `</p></div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 300, Height: 400, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	texts := make([]Op, 0, len(res.Ops))

	for _, textOp := range res.Ops {
		if textOp.Kind == OpText && strings.Contains(textOp.Text, "word") {
			texts = append(texts, textOp)
		}
	}

	if len(texts) < 4 {
		t.Fatalf("expected multi-line wrap, got %d text ops", len(texts))
	}

	// Group by Y; pick a near-top line and a mid-float line.
	first := texts[0]

	var mid *Op

	for i := range texts {
		op := &texts[i]
		// Mid-float is ~40pt below the first line start (float is 80pt tall).
		if op.Y > first.Y+30 && op.Y < first.Y+55 {
			mid = op

			break
		}
	}

	if mid == nil {
		t.Fatalf("no mid-float line found; first Y=%.1f, lines=%d", first.Y, len(texts))
	}

	// Circle mid-chord is wider than the top chord, so mid line starts further right.
	if mid.X <= first.X+2 {
		t.Fatalf("mid line x=%.1f should exceed near-top x=%.1f (circle contour)", mid.X, first.X)
	}

	// Mid line should still be inside the float's rectangular right edge (~80pt).
	if mid.X > 85 {
		t.Fatalf("mid line x=%.1f exceeds float width; shape resolution looks wrong", mid.X)
	}
}

// shape-margin expands the exclusion so mid-line text starts further right
// than the same circle without a margin.
//
//nolint:cyclop,funlen // shape-margin regression compares two full layout passes
func TestShapeMarginExpandsExclusion(t *testing.T) {
	t.Parallel()

	layoutWords := func(shapeCSS string) float64 {
		t.Helper()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; line-height: 12pt; }
.box { overflow: hidden; width: 240pt; }
.f {
  float: left;
  width: 80pt;
  height: 80pt;
  margin: 0;
  background: #9cf;
  `+shapeCSS+`
}
`)
		words := strings.Repeat("word ", 60)
		root, err := html.Parse(`<html><body>
<div class="box"><div class="f">F</div><p style="margin:0">` + words + `</p></div>
</body></html>`)

		if err != nil {
			t.Fatal(err)
		}

		res, err := Layout(root, Options{
			Width: 300, Height: 400, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		var midX float64

		var firstY float64

		seen := false

		for _, textOp := range res.Ops {
			if textOp.Kind != OpText || !strings.Contains(textOp.Text, "word") {
				continue
			}

			if !seen {
				firstY = textOp.Y
				seen = true

				continue
			}

			if textOp.Y > firstY+30 && textOp.Y < firstY+55 {
				midX = textOp.X

				break
			}
		}

		if midX == 0 {
			t.Fatal("missing mid-float text line")
		}

		return midX
	}

	base := layoutWords("shape-outside: circle(50%);")
	grown := layoutWords("shape-outside: circle(50%); shape-margin: 10pt;")

	if grown <= base+2 {
		t.Fatalf("shape-margin mid x=%.1f should exceed base mid x=%.1f", grown, base)
	}
}

func TestShapeOutsideApplyParsesBasicShapes(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.a { shape-outside: circle(50% at 50% 50%); shape-margin: 8px; }
.b { shape-outside: ellipse(40% 30%); }
.c { shape-outside: inset(10px 20px); }
.d { shape-outside: polygon(0 0, 100% 0, 100% 100%); }
`)
	root := mustParse(t, `<html><body>
<div class="a">a</div>
<div class="b">b</div>
<div class="c">c</div>
<div class="d">d</div>
</body></html>`)
	styles := resolveStylesWith(root, Options{
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print", Width: testViewport, Height: 800,
	}, nil)

	aStyle := styleRecordsByClass(t, root, styles, "a")[0]
	bStyle := styleRecordsByClass(t, root, styles, "b")[0]
	cStyle := styleRecordsByClass(t, root, styles, "c")[0]
	dStyle := styleRecordsByClass(t, root, styles, "d")[0]

	if aStyle.ShapeOutside != "circle(50% at 50% 50%)" {
		t.Fatalf("a shape-outside=%q", aStyle.ShapeOutside)
	}

	if aStyle.ShapeMargin <= 0 || aStyle.ShapeMarginPercent >= 0 {
		t.Fatalf("a shape-margin pt=%v pct=%v", aStyle.ShapeMargin, aStyle.ShapeMarginPercent)
	}

	if bStyle.ShapeOutside != "ellipse(40% 30%)" {
		t.Fatalf("b shape-outside=%q", bStyle.ShapeOutside)
	}

	if cStyle.ShapeOutside != "inset(10px 20px)" {
		t.Fatalf("c shape-outside=%q", cStyle.ShapeOutside)
	}

	if dStyle.ShapeOutside != shapeOutsideNone {
		t.Fatalf("polygon must stay unsupported (initial), got %q", dStyle.ShapeOutside)
	}
}
