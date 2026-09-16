package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// learncpp.com header branding at the 718px CSS viewport (Phase 9). The live
// markup is a float chain: #branding{float:left} > .identity{float:left} with
// a 32px logo plus #site-text{float:left;margin-left:1em} holding the
// uppercase title and the tagline. Before the fix the 32px logo sized the
// whole #branding float, #site-text received ~13pt, and the tagline wrapped
// one fragment per line.
const learnCppHeaderCSS = `
body { margin: 0; font-size: 14px; }
#branding { float: left; }
#branding .identity { float: left; }
#branding .identity img { width: 32px; height: 32px; display: block; }
#branding #site-text { float: left; margin-left: 1em; }
#branding #site-title { float: left; font-size: 120%; font-weight: 700; margin: 0; }
#branding #site-title a { display: block; text-transform: uppercase; text-decoration: none; }
#branding #site-description { float: left; clear: left; font-size: .9em; }
`

const learnCppHeaderHTML = `<html><body>
<header id="masthead" class="cryout">
<div id="branding">
  <div class="identity"><a href="/"><img src="logo.png"></a></div>
  <div id="site-text">
    <h1 id="site-title"><span><a href="/">Learn C++</a></span></h1>
    <span id="site-description">Skill up with our free tutorials</span>
  </div>
</div>
</header>
</body></html>`

// onePixelPNG returns a 1x1 PNG accepted by the layout image resolver.
func onePixelPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func layoutLearnCppHeader(t *testing.T, extraCSS string) *Result {
	t.Helper()

	root, err := html.Parse(learnCppHeaderHTML)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 538.5, Height: 400, Media: "print", Background: true,
		Sheets: []*css.Stylesheet{sheet(t, learnCppHeaderCSS+extraCSS)},
		Images: func(string) ([]byte, error) { return onePixelPNG(), nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	return res
}

// boxByID returns the first box whose element carries the id.
func boxByID(t *testing.T, res *Result, nodeID string) *box {
	t.Helper()

	var found *box

	var walk func(current *box)
	walk = func(current *box) {
		if found != nil {
			return
		}

		if current.node != nil && current.node.Type == html.ElementNode &&
			current.node.Attribute("id") == nodeID {
			found = current

			return
		}

		for _, c := range current.children {
			walk(c)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("no box with id %q in box tree", nodeID)
	}

	return found
}

// Phase 9.4: #site-text{position:relative;top:50%} inside a definite-height
// header shifts by half the containing block height, then translateY(-50%)
// centers the box. Before the deferred percentage pass the top was ignored and
// the transform pulled the title above the header top (clipped in screen
// media; print sets transform:none).
func TestLearnCppRelativePercentTopCentersTitle(t *testing.T) {
	t.Parallel()

	const baseCSS = `
body { margin: 0; font-size: 14px; }
#masthead { height: 200pt; }
#site-text { position: relative; float: left; margin-left: 1em; }
#site-title { margin: 0; font-size: 20pt; }
`

	const htmlSrc = `<html><body><header id="masthead"><div id="site-text">` +
		`<h1 id="site-title">Learn C++</h1></div></header></body></html>`

	control := layoutHTML(t, htmlSrc, sheet(t, baseCSS))
	pct := layoutHTML(t, htmlSrc, sheet(t, baseCSS+`
#site-text { top: 50%; transform: translateY(-50%); }
`))

	masthead := boxByID(t, pct, "masthead")
	siteText := boxByID(t, pct, "site-text")
	ctrlText := boxByID(t, control, "site-text")

	if got := siteText.y - ctrlText.y; !near(got, 100) {
		t.Fatalf("relative top:50%% shift = %.2fpt, want 100pt (half of the 200pt header)", got)
	}

	paintedTop := siteText.y + siteText.style.TranslateYPercent/oneHundred*siteText.height
	wantTop := masthead.y + (masthead.height-siteText.height)/2

	if !near(paintedTop, wantTop) {
		t.Fatalf("painted top after translateY(-50%%) = %.2fpt, want %.2fpt (centered)", paintedTop, wantTop)
	}

	if paintedTop < masthead.y {
		t.Fatalf("title painted top %.2fpt above the header top %.2fpt (clipped)", paintedTop, masthead.y)
	}
}

// taglineOps collects the text ops painted below the title baseline, with the
// tagline line count and painted width.
func taglineOps(t *testing.T, res *Result) ([]*Op, int, float64) {
	t.Helper()

	titleY := titleBaselineY(t, res)

	ops := make([]*Op, 0, len(res.Ops))

	minX, maxRight := 0.0, 0.0
	bands := map[int]bool{}

	for i := range res.Ops {
		textOp := &res.Ops[i]
		if textOp.Kind != OpText || textOp.Y <= titleY {
			continue
		}

		if len(ops) == 0 {
			minX = textOp.X
		}

		if right := textOp.X + textOp.W; right > maxRight {
			maxRight = right
		}

		ops = append(ops, textOp)
		bands[int(textOp.Y*2+0.5)] = true
	}

	if len(ops) == 0 {
		t.Fatal("missing tagline text ops")
	}

	return ops, len(bands), maxRight - minX
}

// titleBaselineY returns the Y of the title text op that contains "Learn".
func titleBaselineY(t *testing.T, res *Result) float64 {
	t.Helper()

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText && strings.Contains(res.Ops[i].Text, "Learn") {
			return res.Ops[i].Y
		}
	}

	t.Fatal("missing title text op")

	return 0
}

// The 32px logo float must not cap the tagline float: #site-text keeps its
// text max-content and the tagline paints on one line.
func TestLearnCppBrandingTextFloatNotCappedByLogo(t *testing.T) {
	t.Parallel()

	res := layoutLearnCppHeader(t, "")
	ops, lines, width := taglineOps(t, res)

	t.Logf("tagline lines=%d painted width=%.1fpt x=%.1f", lines, width, ops[0].X)

	if lines != 1 {
		t.Fatalf("tagline painted on %d lines (width %.1f), want 1; #site-text capped by the logo image", lines, width)
	}

	// One line at the .9em tagline font is ~120pt; the collapse capped the
	// box at ~13pt.
	if width < 100 {
		t.Fatalf("tagline painted width %.1fpt, want >100pt", width)
	}

	// The logo float itself stays image-driven: 32px = 24pt used width.
	var logo *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpImage {
			logo = &res.Ops[i]

			break
		}
	}

	if logo == nil {
		t.Fatal("missing logo image op")
	}

	if !near(logo.W, 24) {
		t.Fatalf("logo used width %.1fpt, want 24pt (image-driven float sizing)", logo.W)
	}
}

// In-flow images keep the image-driven float policy (wiki thumbs): a float
// sized by a descendant <img> must not letterbox to the caption max-content.
func TestLearnCppInFlowImageFloatKeepsImageWidth(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.thumb { float: left; border: 1px solid #ccc; }
.thumb img { display: block; width: 120px; height: 80px; }
.thumb .cap { font-size: 8pt; }
`)

	root, err := html.Parse(`<html><body>
<div class="thumb"><img src="t.png">
<div class="cap">A caption long enough that its unwrapped max-content would be far wider than the image box</div>
</div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 400, Height: 300, Sheets: []*css.Stylesheet{cssSheet},
		Images: func(string) ([]byte, error) { return onePixelPNG(), nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	img, captionRight := imageAndCaptionRight(res)

	if img == nil {
		t.Fatal("missing image op")
	}

	// 120px = 90pt plus 1px borders.
	if img.W < 89 || img.W > 92 {
		t.Fatalf("image width %.1fpt, want ~90pt", img.W)
	}

	// Caption wraps inside the image float, not at its unwrapped max-content.
	if captionRight > img.X+img.W+2 {
		t.Fatalf("caption right edge %.1f past image right edge %.1f; float letterboxed to caption max-content",
			captionRight, img.X+img.W)
	}
}

// imageAndCaptionRight returns the float's image op and the right edge of its
// caption text.
func imageAndCaptionRight(res *Result) (*Op, float64) {
	var img *Op

	captionRight := 0.0

	for i := range res.Ops {
		paintedOp := &res.Ops[i]

		switch {
		case paintedOp.Kind == OpImage:
			img = paintedOp
		case paintedOp.Kind == OpText && strings.Contains(paintedOp.Text, "caption"):
			if right := paintedOp.X + paintedOp.W; right > captionRight {
				captionRight = right
			}
		}
	}

	return img, captionRight
}

// CSS 2.1 16.3.1: text-decoration does not propagate into floats. The print
// rule #masthead.cryout #site-text{text-decoration:underline} must not stroke
// the tagline; a same-sheet non-floated control still receives the stroke.
func TestLearnCppFloatsSkipInheritedTextDecoration(t *testing.T) {
	t.Parallel()

	const underlineRule = `
#masthead.cryout #site-text { text-decoration: underline; }
`

	floatRes := layoutLearnCppHeader(t, underlineRule)

	floatOps, floatLines, _ := taglineOps(t, floatRes)
	if floatLines != 1 {
		t.Fatalf("tagline lines=%d, want 1 before counting decoration strokes", floatLines)
	}

	if got := taglineStrokes(floatRes, floatOps); got != 0 {
		t.Fatalf("tagline float painted %d underline strokes, want 0 (CSS 2.1 16.3.1)", got)
	}

	// A/B: the same tagline as an in-flow span inherits and paints the
	// strokes, proving the probe detects decoration.
	inFlowRes := layoutLearnCppHeader(t, underlineRule+
		"#branding #site-description { float: none; }\n")

	inFlowOps, _, _ := taglineOps(t, inFlowRes)

	if got := taglineStrokes(inFlowRes, inFlowOps); got == 0 {
		t.Fatal("in-flow control painted 0 underline strokes; A/B probe cannot see decoration")
	}
}

// taglineStrokes counts underline-length line ops in the tagline's band.
func taglineStrokes(res *Result, tagline []*Op) int {
	strokes := 0

	for i := range res.Ops {
		strokeOp := &res.Ops[i]
		if strokeOp.Kind != OpLine || strokeOp.H != 0 || strokeOp.W < 2 {
			continue
		}

		for _, text := range tagline {
			if strokeOp.Y >= text.Y-1 && strokeOp.Y <= text.Y+6 &&
				strokeOp.X+strokeOp.W >= text.X && strokeOp.X <= text.X+text.W {
				strokes++

				break
			}
		}
	}

	return strokes
}

// A text-transform must not separate measured and painted advances: fallback
// face runs and emergency chunks whose glyphs widen under uppercase
// ("learn" -> "LEARN") advance by their painted width, or the next fragment
// paints on top of them. The live page split "Learn C++" into a run whose
// measurement used the lowercase advance and overlapped "C++" by 6.1pt.
func TestLearnCppRunAdvanceMatchesUppercasePaint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		html string
		css  string
		box  float64 // content right edge to respect, 0 = unchecked
	}{
		{
			// The umbrella is missing from the primary face, forcing the
			// per-face fallback split (the live page's Open Sans / '+' shape).
			name: "fallback face run split",
			html: `<div class="t">learn ` + "\u2602" + ` cpp</div>`,
			css:  `.t { text-transform: uppercase; font-size: 12pt; font-weight: 700; }`,
		},
		{
			// Emergency mid-token chunks in a box narrower than the painted
			// uppercase token.
			name: "emergency chunk split",
			html: `<div class="t">learn cpp</div>`,
			css:  `.t { text-transform: uppercase; font-size: 12pt; font-weight: 700; word-break: break-all; width: 22pt; }`,
			box:  22,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, err := html.Parse(`<html><body>` + testCase.html + `</body></html>`)
			if err != nil {
				t.Fatal(err)
			}

			res, err := Layout(root, Options{
				Width: 300, Height: 200,
				Sheets: []*css.Stylesheet{sheet(t, "body { margin: 0; }\n"+testCase.css)},
			})
			if err != nil {
				t.Fatal(err)
			}

			assertNoSameLineOverlap(t, res)

			if testCase.box > 0 {
				for i := range res.Ops {
					textOp := &res.Ops[i]
					if textOp.Kind == OpText && textOp.X+textOp.W > testCase.box+0.5 {
						t.Fatalf("run %q right edge %.2f past the %.0fpt box; chunk width used the untransformed advance",
							textOp.Text, textOp.X+textOp.W, testCase.box)
					}
				}
			}
		})
	}
}

// assertNoSameLineOverlap checks that consecutive text ops sharing a baseline
// do not overlap their painted advances.
func assertNoSameLineOverlap(t *testing.T, res *Result) {
	t.Helper()

	var prev *Op

	for i := range res.Ops {
		textOp := &res.Ops[i]
		if textOp.Kind != OpText {
			continue
		}

		if prev != nil && textOp.Y == prev.Y {
			if textOp.X < prev.X+prev.W-0.5 {
				t.Fatalf("run %q at x=%.2f overlaps previous run %q "+
					"(x=%.2f w=%.2f painted); measured advance used the untransformed width",
					textOp.Text, textOp.X, prev.Text, prev.X, prev.W)
			}
		}

		prev = textOp
	}

	if prev == nil {
		t.Fatal("no text ops painted")
	}
}
