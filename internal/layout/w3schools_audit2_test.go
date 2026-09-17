package layout

import (
	"fmt"
	"strings"
	"testing"
)

// w3schools-2: a white-space:nowrap strip of inline-block links must keep the
// links on one row. CSS suppresses every soft wrap opportunity inside a nowrap
// context, including the boundaries between atomic inlines. gowk wrapped the
// 46-link language strip into a five-row grid whose dark background then
// covered the page content on every page.
func TestNowrapInlineBlockStripStaysOnOneRow(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 15px }
#nav {
	position: fixed; top: 30pt; left: 0; width: 100%;
	white-space: nowrap; overflow: auto; background: #282A35;
	font-size: 0;
}
#nav a { display: inline-block; padding: 5pt 15pt; font-size: 15px; color: #fff }
`)

	var src strings.Builder

	src.WriteString(`<html><body><div id="nav">`)

	for i := range 40 {
		fmt.Fprintf(&src, `<a href="#x%d">Link %d</a>`, i, i)
	}

	src.WriteString(`</div><p>Sentinel content under the strip.</p></body></html>`)

	res := layoutHTML(t, src.String(), cssSheet)

	nav := boxByID(t, res, "nav")
	if nav.height > 40 {
		t.Fatalf("fixed nowrap strip height = %.2fpt, want under 40pt (one row)", nav.height)
	}

	rows := map[float64]bool{}

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Link") {
			rows[op.Y] = true
		}
	}

	if len(rows) != 1 {
		t.Fatalf("nowrap strip link text painted on %d rows, want 1 (CSS nowrap suppresses "+
			"breaks between inline-blocks too)", len(rows))
	}
}

// w3schools-1: after a page-tall float column, the footer chain lands below the
// floats with whole words, and the footer's in-flow background does not paint
// over the float. The site's #wrappercontainer/#footerwrapper/#spacemyfooter
// chain plus floated link columns is the shape: the inner overflow:hidden link
// bar is a BFC root that must clear below the floats (CSS2.1 9.5), the h2 and
// paragraph lines clear per line, and the float paints as its own group after
// in-flow block chrome (Appendix E).
func TestFooterClearsPageTallFloats(t *testing.T) {
	t.Parallel()

	res := layoutFooterClearsFloatsFixture(t)
	col := findBoxByClass(t, res, "col")
	colBottom := col.y + col.height

	assertFooterWordsIntactBelowFloat(t, res, colBottom)
	assertFooterBackgroundPaintsBeforeFloat(t, res)
}

func layoutFooterClearsFloatsFixture(t *testing.T) *Result {
	t.Helper()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 12pt }
.col { float: left; width: 100% }
.b { height: 380pt; background: #eef; margin-bottom: 10pt }
#footer { background: #282A35; color: #fff }
.links { overflow: hidden }
.fl { float: left; padding: 20pt }
`)

	src := `<html><body>` +
		`<div class="col"><div class="b">BLOCK ONE</div><div class="b">BLOCK TWO</div></div>` +
		`<div id="footer"><div class="links">` +
		`<div class="fl">PLUS</div><div class="fl">SPACES</div><div class="fl">FOR TEACHERS</div></div>` +
		`<h2>Top Tutorials</h2></div>` +
		`</body></html>`

	return layoutHTML(t, src, cssSheet)
}

func assertFooterWordsIntactBelowFloat(t *testing.T, res *Result, colBottom float64) {
	t.Helper()

	joined := joinPaintedText(res)
	for _, word := range []string{"Tutorials", "SPACES", "TEACHERS"} {
		if !strings.Contains(joined, word) {
			t.Fatalf("footer word %q missing or cut mid-word; footer text layer = %q", word, joined)
		}
	}

	assertFooterTextClearsFloat(t, res, colBottom)
}

func joinPaintedText(res *Result) string {
	joined := ""

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText {
			joined += paintOp.Text + " "
		}
	}

	return joined
}

func assertFooterTextClearsFloat(t *testing.T, res *Result, colBottom float64) {
	t.Helper()

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || !isFooterProbeText(paintOp.Text) {
			continue
		}

		if paintOp.Y < colBottom-0.5 {
			t.Fatalf("footer text %q at y=%.1f sits over the float column ending at %.1f",
				paintOp.Text, paintOp.Y, colBottom)
		}
	}
}

func isFooterProbeText(text string) bool {
	return strings.Contains(text, "Tutorials") ||
		strings.Contains(text, "SPACES") ||
		strings.Contains(text, "TEACHERS")
}

// assertFooterBackgroundPaintsBeforeFloat checks Appendix E paint order: the
// float group paints after in-flow block chrome, so the dark footer background
// must precede the light float fills.
func assertFooterBackgroundPaintsBeforeFloat(t *testing.T, res *Result) {
	t.Helper()

	darkPos := firstPaintOrderMatch(res, isDarkFooterFill)
	lightPos := firstPaintOrderMatch(res, isLightFloatFill)

	if darkPos == -1 || lightPos == -1 {
		t.Fatalf("missing fills for the paint-order check: dark=%d light=%d", darkPos, lightPos)
	}

	if darkPos > lightPos {
		t.Fatalf("footer background paints at order %d after the float fill at order %d; "+
			"the float group must cover it", darkPos, lightPos)
	}
}

func firstPaintOrderMatch(res *Result, match func(Op) bool) int {
	for pos, idx := range PaintOrder(res.Ops) {
		if match(res.Ops[idx]) {
			return pos
		}
	}

	return -1
}

func isDarkFooterFill(paintOp Op) bool {
	return paintOp.Kind == OpFillRect && paintOp.R < 0.2 && paintOp.G < 0.2 && paintOp.B < 0.3
}

func isLightFloatFill(paintOp Op) bool {
	return paintOp.Kind == OpFillRect &&
		paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B > 0.9 && paintOp.H > 100
}
