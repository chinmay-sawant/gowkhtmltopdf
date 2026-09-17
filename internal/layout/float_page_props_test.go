package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// float-offset length nudges the float down the block axis on placeFloat.
func TestFloatOffsetNudge(t *testing.T) {
	t.Parallel()

	layoutFloatY := func(offsetCSS string) float64 {
		t.Helper()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.box { overflow: auto; width: 220pt; border: 1px dashed #888; }
.f {
  float: left;
  width: 36pt;
  height: 28pt;
  background: #9cf;
  margin-right: 6pt;
  `+offsetCSS+`
}
`)
		root, err := html.Parse(`<html><body>
<div class="box"><div class="f">F</div><span>following text wraps beside the float.</span></div>
</body></html>`)
		if err != nil {
			t.Fatal(err)
		}

		res, err := Layout(root, Options{
			Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Background fill of the float is the first painted box hint; use text "F".
		for _, op := range res.Ops {
			if op.Kind == OpText && strings.TrimSpace(op.Text) == "F" {
				return op.Y
			}
		}

		t.Fatal("missing float text F")

		return 0
	}

	base := layoutFloatY("")
	nudged := layoutFloatY("float-offset: 12pt;")

	if nudged < base+10 || nudged > base+14 {
		t.Fatalf("float-offset Y=%.1f, want ~base+12 (base=%.1f)", nudged, base)
	}
}

// float-reference:inline documents the current BFC: the float still packs
// beside following inline content (no page/column relocation).
func TestFloatReferenceInline(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.box { overflow: auto; width: 220pt; border: 1px dashed #888; }
.f {
  float: left;
  width: 36pt;
  height: 28pt;
  background: #9cf;
  margin-right: 6pt;
  float-reference: inline;
}
`)
	root, err := html.Parse(`<html><body>
<div class="box"><div class="f">F</div><span>following text wraps beside the float.</span></div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStylesWith(root, Options{
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print", Width: 300, Height: 200,
	}, nil)
	floatStyle := styleRecordsByClass(t, root, styles, "f")[0]

	if floatStyle.FloatReference != floatRefInline {
		t.Fatalf("FloatReference=%q, want inline", floatStyle.FloatReference)
	}

	var floatX, followX float64
	var sawF, sawFollow bool

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		if strings.TrimSpace(op.Text) == "F" {
			floatX = op.X
			sawF = true
		}

		if strings.Contains(op.Text, "following") {
			followX = op.X
			sawFollow = true
		}
	}

	if !sawF || !sawFollow {
		t.Fatal("missing float or following text")
	}

	if followX < floatX+20 {
		t.Fatalf("following text x=%.1f should sit beside float (float x=%.1f)", followX, floatX)
	}
}

// float-reference:page uses the page content box, so a nested 50% float is
// wider (or further left) than the same float with float-reference:inline.
func TestFloatReferencePageVsInline(t *testing.T) {
	t.Parallel()

	layoutRef := func(ref string) (x, w float64, stored string) {
		t.Helper()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.box { display: flow-root; width: 100pt; margin-left: 80pt; border: 1px dashed #888; }
.f {
  float: left;
  width: 50%;
  height: 24pt;
  background: #9cf;
  float-reference: `+ref+`;
}
`)
		root, err := html.Parse(`<html><body>
<div class="box"><div class="f">F</div><span>beside</span></div>
</body></html>`)
		if err != nil {
			t.Fatal(err)
		}

		res, err := Layout(root, Options{
			Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		styles := resolveStylesWith(root, Options{
			Sheets: []*css.Stylesheet{cssSheet}, Media: "print", Width: 300, Height: 200,
		}, nil)
		stored = styleRecordsByClass(t, root, styles, "f")[0].FloatReference

		var textX float64
		var boxW float64
		var sawF bool

		for _, op := range res.Ops {
			if op.Kind == OpText && strings.TrimSpace(op.Text) == "F" {
				textX = op.X
				sawF = true
			}

			if op.Kind == OpFillRect && op.H > 20 && op.H < 28 {
				if op.W > boxW {
					boxW = op.W
				}
			}
		}

		if !sawF {
			t.Fatalf("missing float text F for float-reference:%s", ref)
		}

		return textX, boxW, stored
	}

	inX, inW, inRef := layoutRef("inline")
	pgX, pgW, pgRef := layoutRef("page")

	if inRef != floatRefInline {
		t.Fatalf("inline FloatReference=%q", inRef)
	}

	if pgRef != floatRefPage {
		t.Fatalf("page FloatReference=%q", pgRef)
	}

	widthDiffers := pgW > inW+20
	xDiffers := pgX < inX-20
	if !widthDiffers && !xDiffers {
		t.Fatalf("page vs inline: page x=%.1f w=%.1f, inline x=%.1f w=%.1f; want page left of BFC or wider page CB",
			pgX, pgW, inX, inW)
	}
}

func TestFloatPagePropsApply(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.a { float-offset: 12px; float-reference: page; }
.b { float-reference: column; }
.c { float-defer: 1; }
`)
	root := mustParse(t, `<html><body>
<div class="a">a</div><div class="b">b</div><div class="c">c</div>
</body></html>`)
	styles := resolveStylesWith(root, Options{
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print", Width: testViewport, Height: 800,
	}, nil)

	a := styleRecordsByClass(t, root, styles, "a")[0]
	if a.FloatOffset <= 0 || a.FloatOffsetPercent >= 0 {
		t.Fatalf("a float-offset pt=%v pct=%v", a.FloatOffset, a.FloatOffsetPercent)
	}

	if a.FloatReference != floatRefPage {
		t.Fatalf("a float-reference=%q", a.FloatReference)
	}

	b := styleRecordsByClass(t, root, styles, "b")[0]
	if b.FloatReference != floatRefColumn {
		t.Fatalf("b float-reference=%q", b.FloatReference)
	}

	// float-defer has no apply arm: stays at zero-value / unset.
	c := styleRecordsByClass(t, root, styles, "c")[0]
	_ = c
}
