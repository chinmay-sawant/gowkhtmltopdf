package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// learncpp-3: a rounded lesson card (background + row pills) that splits
// across pages must keep its rows inside the card fill on both pages, with no
// reserved blank band above the first continuation row. Two spacings are
// covered: 665pt splits after row 4 with a 48pt blank band (a row underline
// crossed the boundary, snapped to the next page, then the orphans
// keep-together pass shifted it again from a stale box top), and 660pt splits
// after row 4 with the tail rows below the card fill bottom.
const lessonCardSplitCSS = `
body { margin: 0; font-family: sans-serif; font-size: 12pt }
.lessontable {
  box-shadow: 0 2px 12px 0px rgba(44,85,126,0.2); border-radius:8px;
  padding:0px 11px 9px 11px; max-width:800px; margin-bottom:12px
}
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

// lessonCardSplitHTML places a card with a header and rows below a fixed
// spacer. The spacer selects where the card meets the page boundary at the
// A4 / 10mm geometry the learncpp print run uses.
func lessonCardSplitHTML(spacer string, rows int) string {
	var builder strings.Builder

	builder.WriteString(`<html><body><div style="height:` + spacer + `"></div>`)
	builder.WriteString(`<div class=lessontable><div class=lessontable-header>` +
		`<div class=lessontable-header-chapter>Chapter 25</div>` +
		`<div class=lessontable-header-title>Virtual Functions</div></div><div class=lessontable-list>`)

	for rowNum := range rows {
		fmt.Fprintf(&builder,
			`<div class=lessontable-row><div class=lessontable-row-number>25.%d</div>`+
				`<div class=lessontable-row-title>`+
				`<a href=https://www.learncpp.com/cpp-tutorial/lesson/>`+
				`Short title 25.%d</a></div></div>`,
			rowNum+1, rowNum+1)
	}

	builder.WriteString(`</div></div></body></html>`)

	return builder.String()
}

// layoutSplitCard lays out and paints the split-card fixture at the A4 / 10mm
// geometry the learncpp print run uses (content height 785.2pt).
func layoutSplitCard(t *testing.T, spacer string, rows int) *Result {
	t.Helper()

	const contentH = 841.9 - 2*28.35

	root := mustParse(t, lessonCardSplitHTML(spacer, rows))
	styleSheet := sheet(t, lessonCardSplitCSS)

	res, err := Layout(root, Options{
		Width: 538.5, Height: contentH, Sheets: []*css.Stylesheet{styleSheet},
		Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: 595.4, PageHeight: 841.9,
		MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35,
	}); err != nil {
		t.Fatalf("Paint: %v", err)
	}

	return res
}

func isLessonRowBand(paintOp Op) bool {
	if paintOp.Kind != OpFillRect {
		return false
	}

	even := near(paintOp.R, 0.9294) && near(paintOp.G, 0.9412) && near(paintOp.B, 0.9765)
	odd := near(paintOp.R, 0.9569) && near(paintOp.G, 0.9647) && near(paintOp.B, 0.9922)

	return even || odd
}

// isLessonCardFill matches the box-shadow core layer, the card face behind the
// rows. Blur steps share the color at a fraction of the alpha.
func isLessonCardFill(paintOp Op) bool {
	return paintOp.Kind == OpFillRect &&
		near(paintOp.R, 0.1725) && near(paintOp.G, 0.3333) && near(paintOp.B, 0.4941) &&
		paintOp.Alpha > 0.15
}

func TestSplitCardRowsStayInsideCardFill(t *testing.T) {
	t.Parallel()

	const (
		contentH  = 841.9 - 2*28.35
		rowBandH  = 17.28
		rowMargin = 2.25
		bandSlack = 0.6
		wantRows  = 8
	)

	cases := []struct {
		name   string
		spacer string
	}{
		{"blank band after the split", "665pt"},
		{"tail rows past the fill", "660pt"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertSplitCard(t, layoutSplitCard(t, testCase.spacer, wantRows), contentH, rowBandH, rowMargin, bandSlack, wantRows)
		})
	}
}

// assertSplitCard checks the continuation geometry of a card that split after
// four rows: (a) the first continuation row starts within one row height of
// the card fill top, (b) inter-row gaps stay at the row margin, and (c) every
// row band center sits inside a card fill on its page.
func assertSplitCard(
	t *testing.T, res *Result, contentH, rowBandH, rowMargin, bandSlack float64, wantRows int,
) {
	t.Helper()

	rowBands, cardFills := collectSplitCardOps(res)
	if len(rowBands) != wantRows {
		t.Fatalf("row bands = %d, want %d", len(rowBands), wantRows)
	}

	assertSplitCardPageCounts(t, rowBands, contentH, wantRows)
	assertContinuationNoBlankBand(t, rowBands, cardFills, contentH, rowBandH, bandSlack)
	assertSplitCardRowGaps(t, rowBands, contentH, rowMargin, bandSlack)
	assertRowsInsideCardFills(t, rowBands, cardFills, contentH)
}

func collectSplitCardOps(res *Result) ([]Op, []Op) {
	var rowBands, cardFills []Op

	for _, paintOp := range res.Ops {
		if isLessonRowBand(paintOp) {
			rowBands = append(rowBands, paintOp)
		}

		if isLessonCardFill(paintOp) {
			cardFills = append(cardFills, paintOp)
		}
	}

	return rowBands, cardFills
}

func assertSplitCardPageCounts(t *testing.T, rowBands []Op, contentH float64, wantRows int) {
	t.Helper()

	pageOneTop := contentH
	pageTwoTop := 2 * contentH
	pageOneRows, pageTwoRows := 0, 0

	for _, band := range rowBands {
		switch {
		case band.Y < pageOneTop:
			pageOneRows++
		case band.Y < pageTwoTop:
			pageTwoRows++
		}
	}

	if pageOneRows != 4 || pageTwoRows != wantRows-4 {
		t.Fatalf("rows split as %d/%d, want 4/%d (band Ys %v)",
			pageOneRows, pageTwoRows, wantRows-4, rowBandYs(rowBands))
	}
}

func assertContinuationNoBlankBand(
	t *testing.T, rowBands, cardFills []Op, contentH, rowBandH, bandSlack float64,
) {
	t.Helper()

	pageOneTop := contentH
	pageTwoTop := 2 * contentH
	pageTwoFillTop := firstOpYInRange(cardFills, pageOneTop-0.5, pageTwoTop)
	firstPageTwoRow := firstOpYInRange(rowBands, pageOneTop-0.5, pageTwoTop)

	if math.IsInf(pageTwoFillTop, 1) || math.IsInf(firstPageTwoRow, 1) {
		t.Fatalf("missing continuation card fill or row (fill=%v rows=%v)",
			pageTwoFillTop, rowBandYs(rowBands))
	}

	if firstPageTwoRow-pageTwoFillTop > rowBandH+bandSlack {
		t.Fatalf("first continuation row at %.2f sits %.2f below the card fill top %.2f; "+
			"want under one row height (%.2f)",
			firstPageTwoRow, firstPageTwoRow-pageTwoFillTop, pageTwoFillTop, rowBandH)
	}
}

func firstOpYInRange(ops []Op, minY, maxY float64) float64 {
	best := math.Inf(1)

	for _, paintOp := range ops {
		if paintOp.Y >= minY && paintOp.Y < maxY && paintOp.Y < best {
			best = paintOp.Y
		}
	}

	return best
}

func assertSplitCardRowGaps(t *testing.T, rowBands []Op, contentH, rowMargin, bandSlack float64) {
	t.Helper()

	for _, pageTop := range []float64{0, contentH} {
		lastBottom := -1.0

		for _, band := range rowBands {
			if band.Y < pageTop-0.5 || band.Y >= pageTop+contentH {
				continue
			}

			if lastBottom >= 0 && band.Y-lastBottom > rowMargin+bandSlack {
				t.Fatalf("row gap %.2f at y=%.2f exceeds the row margin %.2f",
					band.Y-lastBottom, band.Y, rowMargin)
			}

			lastBottom = band.Y + band.H
		}
	}
}

func assertRowsInsideCardFills(t *testing.T, rowBands, cardFills []Op, contentH float64) {
	t.Helper()

	for _, band := range rowBands {
		center := band.Y + band.H/2
		bandPage := int((center + layoutEpsilon) / contentH)
		inside := false

		for _, fill := range cardFills {
			fillPage := int((fill.Y + layoutEpsilon) / contentH)
			if fillPage == bandPage && center >= fill.Y-0.5 && center <= fill.Y+fill.H+0.5 {
				inside = true

				break
			}
		}

		if !inside {
			t.Fatalf("row band center %.2f (page %d) outside every card fill; bands=%v fills=%v",
				center, bandPage, rowBandYs(rowBands), fillYs(cardFills))
		}
	}
}

func rowBandYs(bands []Op) []float64 {
	out := make([]float64, 0, len(bands))
	for _, band := range bands {
		out = append(out, band.Y)
	}

	return out
}

func fillYs(fills []Op) []float64 {
	out := make([]float64, 0, len(fills))
	for _, fill := range fills {
		out = append(out, fill.Y, fill.Y+fill.H)
	}

	return out
}
