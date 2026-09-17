package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Pagination relocation consistency and fragment background invariants.
//
// The real-site audit (plans/0.2.7/real-sites/*/evidence/2026-09-16-audit2)
// found four defects where pagination moved a text line without its
// decoration, or painted a background fragment whose content had moved on:
//
//   - gobyexample-7: a snapped link line moved but its OpLine underline and
//     OpLinkURI rect stayed at the natural slot.
//   - tutorialspoint-cpp-9: an OpBullet stayed one line above the relocated
//     item text.
//   - tutorialspoint-cpp-10 / cplusplus-tutorial-10: a split list background
//     fragment closed one line short of its last text line.
//   - programiz-cpp-15: an empty .faq-section fragment painted a 133pt band
//     at the page bottom while all its content moved to the next page.
//
// These tests pin the invariants at the A4 / 10mm-margin geometry the drills
// use (content width 538.58pt, content height 785.2pt).

const (
	relocationPageW = 595.28
	relocationPageH = 841.9
	relocationPad   = 28.35
)

// layoutPaintA4 lays out and paints one fixture at the audit geometry and
// returns the settled display list.
func layoutPaintA4(t *testing.T, src, cssSrc string) *Result {
	t.Helper()

	root := mustParse(t, src)
	styleSheet := sheet(t, cssSrc)
	contentH := relocationPageH - 2*relocationPad

	res, err := Layout(root, Options{
		Width:      relocationPageW - 2*relocationPad,
		Height:     contentH,
		Sheets:     []*css.Stylesheet{styleSheet},
		Background: true,
		Media:      "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth:    relocationPageW,
		PageHeight:   relocationPageH,
		MarginTop:    relocationPad,
		MarginBottom: relocationPad,
		MarginLeft:   relocationPad,
		MarginRight:  relocationPad,
	}); err != nil {
		t.Fatalf("Paint: %v", err)
	}

	return res
}

// gobyListCSS mirrors gobyexample's Meyer reset plus the site body/list rules
// used by the audit's links-ul probe.
const gobyListCSS = `
html, body, div, span, ul, li, p, a {
  margin:0; padding:0; border:0; font-size:100%; font:inherit; vertical-align:baseline;
}
ol, ul { list-style: none; }
body { font-family: 'Georgia', serif; font-size: 16px; line-height: 20px; }
div#intro {
  width: 420px; min-width: 420px; max-width: 420px;
  margin-left:auto; margin-right:auto; margin-bottom:120px;
}
div#intro ul { padding-top: 20px; }
a, a:visited { color: #261a3b; }
`

// gobyListHTML builds the audit's linked list with n items at the probe
// geometry.
func gobyListHTML(itemCount int) string {
	var builder strings.Builder

	builder.WriteString(`<html><head><meta charset="utf-8"><style>` + gobyListCSS + `</style></head>`)
	builder.WriteString(`<body><div id="intro"><ul>`)

	for i := 1; i <= itemCount; i++ {
		fmt.Fprintf(&builder, `<li><a href="https://example.com/i%02d">Item %02d</a></li>`, i, i)
	}

	builder.WriteString(`</ul></div></body></html>`)

	return builder.String()
}

// TestSnapRelocatesUnderlineAndHitboxWithText is gobyexample-7: the last list
// line before a page break snaps to the next page, and its underline stroke
// and link hitbox must share the move. Before the fix the text moved +19.34pt
// while the underline and the annotation rect stayed at the natural slot.
func TestSnapRelocatesUnderlineAndHitboxWithText(t *testing.T) {
	t.Parallel()

	res := layoutPaintA4(t, gobyListHTML(52), gobyListCSS)
	assertUnderlineTravelsWithText(t, res)
	assertLinkHitboxesCoverText(t, res)
}

// assertUnderlineTravelsWithText checks every linked text run keeps a stable
// underline gap (closest OpLine sharing its left edge and at least its width).
func assertUnderlineTravelsWithText(t *testing.T, res *Result) {
	t.Helper()

	canonicalGap := math.Inf(1)
	checked := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		underY, found := closestUnderlineY(res.Ops, paintOp)
		if !found {
			t.Fatalf("link text %q at y=%.2f has no underline run", paintOp.Text, paintOp.Y)
		}

		gap := underY - paintOp.Y
		if math.IsInf(canonicalGap, 1) {
			canonicalGap = gap
		}

		if math.Abs(gap-canonicalGap) > 0.05 {
			t.Fatalf("underline of %q sits %.3fpt from its baseline, want %.3fpt: "+
				"the underline did not travel with the snapped line", paintOp.Text, gap, canonicalGap)
		}

		checked++
	}

	if checked == 0 {
		t.Fatal("no linked text lines found; fixture must cross a page boundary")
	}
}

func closestUnderlineY(ops []Op, textOp Op) (float64, bool) {
	underY, found := math.Inf(1), false

	for _, under := range ops {
		if under.Kind != OpLine || math.Abs(under.X-textOp.X) > 0.6 || under.W < textOp.W-0.6 {
			continue
		}

		if d := math.Abs(under.Y - textOp.Y); d < math.Abs(underY-textOp.Y) {
			underY = under.Y
			found = true
		}
	}

	return underY, found
}

// assertLinkHitboxesCoverText checks every linked text baseline still sits
// inside an OpLinkURI rect that shares its left edge.
func assertLinkHitboxesCoverText(t *testing.T, res *Result) {
	t.Helper()

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		if !linkHitboxCoversBaseline(res.Ops, paintOp) {
			t.Fatalf("link text %q baseline y=%.2f is not covered by any link rect: "+
				"the hitbox did not travel with the snapped line", paintOp.Text, paintOp.Y)
		}
	}
}

func linkHitboxCoversBaseline(ops []Op, textOp Op) bool {
	for _, link := range ops {
		if link.Kind != OpLinkURI || math.Abs(link.X-textOp.X) > 0.6 {
			continue
		}

		if textOp.Y >= link.Y-0.5 && textOp.Y <= link.Y+link.H+0.5 {
			return true
		}
	}

	return false
}

// tutorialListCSS mirrors the audit's orphan probe (tutorialspoint-cpp
// list/table pages): 15px Liberation Sans text, a gray disc list, and 24px
// filler lines that walk the list over the page boundary.
const tutorialListCSS = `
@page { size: A4; margin: 28.35pt; }
body { margin: 0; font-family: "Liberation Sans", sans-serif; font-size: 15px; }
p.filler { margin: 0; line-height: 24px; }
ul.list { list-style-type: disc; margin: 0.5rem 0; padding-left: 16px; background: #d6d6d6; }
ul.list li { font-size: 15px; line-height: 24px; font-family: inherit; }
`

// tutorialListHTML places fillers filler lines before a 20-item disc list.
func tutorialListHTML(fillers int) string {
	var builder strings.Builder

	builder.WriteString(`<html><head><meta charset="utf-8"><style>` + tutorialListCSS + `</style></head><body>`)

	for i := range fillers {
		fmt.Fprintf(&builder, `<p class="filler">filler line %d</p>`, i)
	}

	builder.WriteString(`<ul class="list">`)
	builder.WriteString(`<li>C++ &lt;fstream&gt;</li><li>C++ &lt;iomanip&gt;</li><li>C++ &lt;ios&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;iosfwd&gt;</li><li>C++ &lt;iostream&gt;</li><li>C++ &lt;istream&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;ostream&gt;</li><li>C++ &lt;sstream&gt;</li><li>C++ &lt;streambuf&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;atomic&gt;</li><li>C++ &lt;complex&gt;</li><li>C++ &lt;exception&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;functional&gt;</li><li>C++ &lt;limits&gt;</li><li>C++ &lt;locale&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;memory&gt;</li><li>C++ &lt;new&gt;</li><li>C++ &lt;numeric&gt;</li>`)
	builder.WriteString(`<li>C++ &lt;regex&gt;</li><li>C++ &lt;stdexcept&gt;</li>`)
	builder.WriteString(`</ul></body></html>`)

	return builder.String()
}

// TestSnapKeepsListMarkerWithItsLine is tutorialspoint-cpp-9: a bullet whose
// item text snaps to the next page must travel with the text instead of
// painting one line above it.
func TestSnapKeepsListMarkerWithItsLine(t *testing.T) { //nolint:cyclop // marker/text baseline pairing census
	t.Parallel()

	res := layoutPaintA4(t, tutorialListHTML(23), tutorialListCSS)

	markers, lines := 0, 0

	for _, bullet := range res.Ops {
		if bullet.Kind != OpBullet || strings.TrimSpace(bullet.Text) == "" {
			continue
		}

		markers++

		partner := false

		for _, paintOp := range res.Ops {
			if paintOp.Kind != OpText || paintOp.X <= bullet.X {
				continue
			}

			if math.Abs(paintOp.Y-bullet.Y) <= 0.5 {
				partner = true

				break
			}
		}

		if !partner {
			t.Fatalf("marker %q at y=%.2f has no text run on its baseline: "+
				"it was left behind by the snap", bullet.Text, bullet.Y)
		}
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "C++") {
			lines++
		}
	}

	if markers == 0 || lines == 0 {
		t.Fatalf("fixture did not split a disc list (markers=%d lines=%d)", markers, lines)
	}
}

// TestSplitListFragmentFillCoversLastLine is tutorialspoint-cpp-10 /
// cplusplus-tutorial-10: a split list background must cover the glyph ink of
// every line inside its fragment, including the last line on a continuation
// page. Before the fix the fragment height was finalized before the last line
// was placed, and the gray box closed mid-glyph.
func TestSplitListFragmentFillCoversLastLine(t *testing.T) { //nolint:cyclop // fragment fill vs last-line ink census
	t.Parallel()

	res := layoutPaintA4(t, tutorialListHTML(24), tutorialListCSS)

	contentH := relocationPageH - 2*relocationPad

	fragments, coveredLines := 0, 0

	for _, fill := range res.Ops {
		if fill.Kind != OpFillRect || !near(fill.R, 0.8392) || !near(fill.G, 0.8392) || !near(fill.B, 0.8392) {
			continue
		}

		fragments++

		fillPage := int((fill.Y + layoutEpsilon) / contentH)

		for _, paintOp := range res.Ops {
			if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
				continue
			}

			if paintOp.X < fill.X-0.6 || paintOp.X > fill.X+fill.W+0.6 || paintOp.Y < fill.Y-0.6 {
				continue
			}

			if int((paintOp.Y+layoutEpsilon)/contentH) != fillPage {
				continue
			}

			inkBot := paintOp.Y + opVisibleInkHeight(paintOp)
			if inkBot > fill.Y+fill.H+0.5 {
				t.Fatalf("list background fragment [%.2f, %.2f] does not cover text %q (ink bottom %.2f): "+
					"the fragment shut before its last line",
					fill.Y, fill.Y+fill.H, paintOp.Text, inkBot)
			}

			coveredLines++
		}
	}

	if fragments == 0 || coveredLines == 0 {
		t.Fatalf("fixture did not split a background list (fragments=%d lines=%d)", fragments, coveredLines)
	}
}

// faqSectionCSS is the programiz .faq-section shape: a padded,
// background-colored section whose first line does not fit under the page
// fold, so the content starts on the next page.
const faqSectionCSSPrefix = `
@page { size: A4; margin: 28.35pt; }
body { margin: 0; font-family: sans-serif; font-size: 12pt; line-height: 18pt; }
p { margin: 0; orphans: 3; widows: 3; }
.faq { background: #fafafa; padding: 45pt 0; }
`

// faqSectionFixture places a padded section after a spacer of fillerPT so the
// first paragraph line either lands past the fold or snaps to the next page.
func faqSectionFixture(fillerPT float64, text string) (string, string) {
	css := fmt.Sprintf(".filler { height: %gpt }", fillerPT) + faqSectionCSSPrefix
	html := `<html><head><meta charset="utf-8"><style>` + css + `</style></head>` +
		`<body><div class="filler">filler</div><div class="faq"><p>` + text + `</p>` +
		`</div></body></html>`

	return html, css
}

// TestEmptySectionFragmentPaintsNoBackground is programiz-cpp-15: when a
// section's content moves wholly to the next page, the stale leading slice of
// its background must not paint an empty band at the foot of the page. The
// first case overflows the fold; the second has its first line snap to the
// next page, leaving the section's leading slice without ink.
func TestEmptySectionFragmentPaintsNoBackground(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fillerPT float64
		text     string
	}{
		{
			name:     "content past the fold",
			fillerPT: 745,
			text:     "faq line one. faq line two. faq line three. faq line four. faq line five. faq line six.",
		},
		{
			name:     "first line snaps away",
			fillerPT: 700,
			text:     strings.Repeat("alpha beta gamma delta epsilon zeta eta theta iota kappa. ", 40),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			html, css := faqSectionFixture(testCase.fillerPT, testCase.text)
			assertNoEmptySectionFragment(t, layoutPaintA4(t, html, css))
		})
	}
}

// assertNoEmptySectionFragment checks that every section background fragment
// contains ink and that one covers the relocated paragraph baseline.
func assertNoEmptySectionFragment(t *testing.T, res *Result) {
	t.Helper()

	faqText := firstNonFillerText(t, res)
	assertSectionFragmentsHaveInk(t, res)
	assertSectionFragmentCoversText(t, res, faqText)
}

func firstNonFillerText(t *testing.T, res *Result) Op {
	t.Helper()

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		if strings.Contains(paintOp.Text, "filler") {
			continue
		}

		return paintOp
	}

	t.Fatal("section paragraph did not paint")

	return Op{}
}

func assertSectionFragmentsHaveInk(t *testing.T, res *Result) {
	t.Helper()

	for fillIdx, fill := range res.Ops {
		if !isFAQSectionFill(fill) {
			continue
		}

		fillPage := pageOfIdx(t, res, fillIdx)
		if !sectionFragmentHasInk(t, res, fill, fillPage) {
			t.Fatalf("section background fragment H=%.2f on page %d paints with no ink: "+
				"an empty fragment must not paint", fill.H, fillPage+1)
		}
	}
}

func isFAQSectionFill(fill Op) bool {
	return fill.Kind == OpFillRect && near(fill.R, 0.9804) && fill.H > 1
}

func sectionFragmentHasInk(t *testing.T, res *Result, fill Op, fillPage int) bool {
	t.Helper()

	for inkIdx, paintOp := range res.Ops {
		switch paintOp.Kind { //nolint:exhaustive // only content ink validates a fragment
		case OpText, OpBullet, OpImage:
		default:
			continue
		}

		if pageOfIdx(t, res, inkIdx) != fillPage {
			continue
		}

		if paintOp.X < fill.X-0.6 || paintOp.X > fill.X+fill.W+0.6 {
			continue
		}

		if paintOp.Y < fill.Y-0.6 || paintOp.Y > fill.Y+fill.H+0.6 {
			continue
		}

		return true
	}

	return false
}

func assertSectionFragmentCoversText(t *testing.T, res *Result, faqText Op) {
	t.Helper()

	for _, fill := range res.Ops {
		if !isFAQSectionFill(fill) {
			continue
		}

		if fill.Y-0.6 <= faqText.Y && faqText.Y <= fill.Y+fill.H+0.6 {
			return
		}
	}

	t.Fatalf("no section background fragment covers the relocated text baseline y=%.2f", faqText.Y)
}
