package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Phase-12 probe: a print row whose number badge is snapped apart from its own
// chrome and title. The site CSS for .lessontable-row / .lessontable-row-number
// is inlined from learncpp.com custom-css 18062.css (the Appendix D shape).

const lessonRowCSS = `
.lessontable { border-radius:8px; padding:0px 11px 9px 11px; max-width:800px; margin-bottom:12px }
.lessontable-header-chapter {
  color:#fff; font-size:1.2em; float:right; background-color:#6daaf3;
  padding:0px 8px 0px 8px
}
.lessontable-header-title { color:#3B4C5A; font-size:1.4em; font-weight:700; padding-top:4px }
.lessontable-list { margin-top:4px }
.lessontable-row {
  margin:3px 0px 3px 0px; background-color:#EDF0F9; border-radius:6px;
  display:flex; align-items:center
}
.lessontable-row:nth-child(odd) { background:#f4f6fd }
.lessontable-row-number {
  background-color:#6daaf3; color:#FFFFFF; font-size:1.1em; font-weight:400;
  border-radius:4px; padding:0px; margin:3px 10px 3px 4px; width:46px;
  min-width:46px; text-align:center; line-height:20px
}
.lessontable-row-title a { font-size:1.2em; color:#3990D8 }
`

// lessonTableHTML builds filler paragraphs plus one Appendix D style table.
// The row labels are unique so a probe can find them by text.
func lessonTableHTML(filler, rows int) string {
	var builder strings.Builder

	builder.WriteString(`<html><body>`)

	for i := range filler {
		fmt.Fprintf(&builder, `<p>Filler paragraph %d with enough words to take one full line of text in the flow.</p>`, i)
	}

	builder.WriteString(`<div class=lessontable><div class=lessontable-header>` +
		`<div class=lessontable-header-chapter>Appendix D</div>` +
		`<div class=lessontable-header-title>Deprecated Articles</div></div><div class=lessontable-list>`)

	for r := range rows {
		lesson := r + 1
		title := "Short title"

		if r%3 == 1 {
			title = "A considerably longer lesson title that will require a second line when it wraps inside the row"
		}

		fmt.Fprintf(&builder, `<div class=lessontable-row><div class=lessontable-row-number>25.%d</div>`+
			`<div class=lessontable-row-title><a href=https://www.learncpp.com/cpp-tutorial/lesson/>%s 25.%d</a></div></div>`,
			lesson, title, lesson)
	}

	builder.WriteString(`</div></div></body></html>`)

	return builder.String()
}

// layoutPrintRows lays out and paints the fixture at the A4 / 10mm geometry
// the learncpp print run uses (content height 785.2pt).
func layoutPrintRows(t *testing.T, filler, rows int) *Result {
	t.Helper()

	root := mustParse(t, lessonTableHTML(filler, rows))
	s := sheet(t, lessonRowCSS)

	res, err := Layout(root, Options{
		Width: 538.5, Height: 841.9 - 2*28.35, Sheets: []*css.Stylesheet{s},
		Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	source := pdf.NewDocument()

	if err := Paint(source, res, PaintOptions{
		PageWidth: 595.4, PageHeight: 841.9,
		MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35,
	}); err != nil {
		t.Fatalf("Paint: %v", err)
	}

	return res
}

// findTextOp returns the first text op whose run equals text.
func findTextOp(res *Result, text string) (int, Op, bool) {
	for idx, op := range res.Ops {
		if op.Kind == OpText && op.Text == text {
			return idx, op, true
		}
	}

	return -1, Op{}, false
}

// TestLearnCppDeprecatedRowNumberStaysWithRow is the Phase-12 red probe: the
// last row's number must travel with its own pill background and title.
//
// Before the fix (filler=24, rows=4): the title snapped to the next page with
// the pill chrome (number Y=769.10 on page 0 in the gap; pill Y=797.14 and
// title Y=809.63 on page 1), so the number painted in the blank band above its
// own row and the pill was an empty blue rect. After the fix the number op
// lands at Y=809.22 inside the pill band [797.14, 812.14] on the title's page.
func TestLearnCppDeprecatedRowNumberStaysWithRow(t *testing.T) {
	t.Parallel()

	const contentH = 841.9 - 2*28.35

	res := layoutPrintRows(t, 24, 4)

	numIdx, num, found := findTextOp(res, "25.4")
	if !found {
		t.Fatal("missing number op 25.4")
	}

	titleIdx, title, found := findTextOp(res, "Short title 25.4")
	if !found {
		t.Fatal("missing title op Short title 25.4")
	}

	pill, foundPill := findNumberPill(res, title)
	if !foundPill {
		t.Fatal("no blue number pill under the title")
	}

	numPage := int(num.Y / contentH)
	titlePage := int(title.Y / contentH)

	if numPage != titlePage {
		t.Fatalf("number op=%d Y=%.2f on page %d, title op=%d Y=%.2f on page %d",
			numIdx, num.Y, numPage, titleIdx, title.Y, titlePage)
	}

	if num.Y < pill.Y-layoutEpsilon || num.Y > pill.Y+pill.H+layoutEpsilon {
		t.Fatalf("number op=%d Y=%.2f outside its pill band [%.2f,%.2f] (empty pill, number in the page gap)",
			numIdx, num.Y, pill.Y, pill.Y+pill.H)
	}
}

// findNumberPill returns the blue pill fill whose band holds title, preferring
// the lowest such fill on the page.
func findNumberPill(res *Result, title Op) (Op, bool) {
	var pill Op

	foundPill := false

	for _, paintedOp := range res.Ops {
		if !isPillBlue(paintedOp) {
			continue
		}

		if paintedOp.Y > title.Y+layoutEpsilon || paintedOp.Y+paintedOp.H < title.Y-layoutEpsilon {
			continue
		}

		if !foundPill || paintedOp.Y > pill.Y {
			pill = paintedOp
			foundPill = true
		}
	}

	return pill, foundPill
}

// isPillBlue reports whether the op is the #6daaf3 number-pill fill.
func isPillBlue(paintedOp Op) bool {
	return paintedOp.Kind == OpFillRect && near(paintedOp.R, 0.427) &&
		near(paintedOp.G, 0.667) && near(paintedOp.B, 0.953)
}
