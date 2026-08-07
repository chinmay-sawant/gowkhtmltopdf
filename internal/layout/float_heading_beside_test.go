package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Blocks after a tall float must pack beside it (Chrome), not clear below
// unless clear is set. Non-BFC parents must not grow around floats — that
// used to push the next <section> below the entire floated infobox.
func TestBlockHeadingPacksBesideTallFloat(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
.box { float: right; width: 100pt; display: table; border: 1px solid #000; }
.box td { padding: 4pt; }
p { margin: 0 0 6pt 0; }
.hd { display: flow-root; border-bottom: 2px solid #000; margin: 0.25em 0; font-size: 14pt; font-weight: bold; }
.hd h2 { display: inline; margin: 0; padding: 0; border: 0; font: inherit; }
`)
	res := layoutHTML(t, `<html><body>
<section class="lead">
<table class="box"><tr><td>PHOTO<br/>2<br/>3<br/>4<br/>5<br/>6<br/>7<br/>8<br/>9<br/>10<br/>11<br/>12<br/>Born<br/>Occ<br/>Spouse</td></tr></table>
<p>Lead one wraps beside the float with enough words to fill a couple of lines here.</p>
<p>Lead two ends with Ballerina.</p>
</section>
<section class="next">
<div class="hd"><h2>Early life</h2></div>
<p>Born in Havana text that should also sit beside the float if room remains.</p>
</section>
</body></html>`, s)

	var earlyY, leadEnd, floatBottom float64
	earlyY = -1

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		switch {
		case op.Text == "Early life":
			earlyY = op.Y
		case strings.Contains(op.Text, "Ballerina"):
			if op.Y > leadEnd {
				leadEnd = op.Y
			}
		case strings.Contains(op.Text, "Spouse") || strings.Contains(op.Text, "PHOTO"):
			if op.Y > floatBottom {
				floatBottom = op.Y
			}
		}
	}

	if earlyY < 0 {
		t.Fatal("Early life text not found")
	}

	t.Logf("leadEnd=%.1f earlyY=%.1f floatBottom=%.1f", leadEnd, earlyY, floatBottom)

	if floatBottom > leadEnd+20 && earlyY > floatBottom-5 {
		t.Fatalf("heading cleared below float: earlyY=%.1f floatBottom=%.1f leadEnd=%.1f", earlyY, floatBottom, leadEnd)
	}

	if earlyY > leadEnd+40 {
		t.Fatalf("too much gap before heading: earlyY=%.1f leadEnd=%.1f", earlyY, leadEnd)
	}

	// flow-root heading establishes a BFC — border must stop before the float.
	var borderRight float64

	var hasBorder bool

	for _, op := range res.Ops {
		if op.Kind != OpLine || op.H >= 1 || op.W < 10 {
			continue
		}

		if earlyY > 0 && op.Y > earlyY-2 && op.Y < earlyY+24 {
			right := op.X + op.W
			if !hasBorder || right > borderRight {
				borderRight = right
				hasBorder = true
			}
		}
	}

	if !hasBorder {
		t.Fatal("expected border-bottom line near Early life heading")
	}
	// Float is 100pt on a 500pt viewport → left edge at 400. Border may meet
	// that edge but must not extend into the float band.
	if earlyY < floatBottom-5 && borderRight > 402 {
		t.Fatalf("heading border under float: right=%.1f (floatBottom=%.1f earlyY=%.1f)",
			borderRight, floatBottom, earlyY)
	}
}

func TestHeadingBFCShortensBesideFloat(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
.box { float: right; width: 100pt; height: 200pt; background: #ccc; }
.hd { display: flow-root; border-bottom: 2px solid #000; margin: 0.25em 0; font-size: 14pt; }
`)

	root, err := html.Parse(`<html><body>
<div class="box">FLOAT</div>
<div class="hd">Early life</div>
<p>Body text after heading that wraps beside or under the float.</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 500, 800)

	var hd *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Attribute("class") == "hd" {
			hd = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if hd == nil {
		t.Fatal("no hd")
	}

	st := styles[hd]
	t.Logf("hd display=%q overflow=%q establishesBFC=%v", st.Display, st.Overflow, establishesBFC(st))

	if !establishesBFC(st) {
		t.Fatal("flow-root must establish BFC")
	}

	res, err := Layout(root, Options{Width: 500, Height: 800, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}

	var maxBorderRight float64

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H < 1 && op.W > 50 {
			t.Logf("hline x=%.1f w=%.1f y=%.1f", op.X, op.W, op.Y)

			if r := op.X + op.W; r > maxBorderRight {
				maxBorderRight = r
			}
		}
	}
	// Float occupies [400,500]; border of BFC heading must end ≤ 400.
	if maxBorderRight > 405 {
		t.Fatalf("BFC heading border right=%.1f extends under 100pt right float", maxBorderRight)
	}
}
