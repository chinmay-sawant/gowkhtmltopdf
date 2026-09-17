package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// programiz-cpp-16: a flex row of two code cards split at a page boundary.
// Both pre cards straddle the boundary and must move whole to the next page
// with their borders hugging their own code and both column tops aligned.
// Before the fix the avoid-inside lift moved the first pre's op range plus the
// sibling's interior ops, leaving the sibling's top border behind: the right
// pre border sat 51.6pt above the left one and 63pt above its own code.
func TestFlexRowPreBordersStayWithCodeAfterBreak(t *testing.T) {
	t.Parallel()

	res := layoutFlexRowPreCards(t)
	leftCard, rightCard := collectPreCardsOnPage(t, res, 1)
	assertPreCardBordersHugCode(t, "left", leftCard)
	assertPreCardBordersHugCode(t, "right", rightCard)

	if diff := leftCard.borderTop - rightCard.borderTop; diff > 1 || diff < -1 {
		t.Fatalf("pre card tops differ by %.1fpt (left %.1f, right %.1f), want the "+
			"two flex columns top-aligned", diff, leftCard.borderTop, rightCard.borderTop)
	}
}

type preCardGeom struct {
	borderTop, borderBottom float64
	firstCode, lastCode     float64
}

func newPreCardGeom() *preCardGeom {
	return &preCardGeom{borderTop: -1, borderBottom: -1, firstCode: -1, lastCode: -1}
}

func layoutFlexRowPreCards(t *testing.T) *Result {
	t.Helper()

	cssSheet := sheet(t, `
body { margin: 0; font-family: monospace; font-size: 12pt; line-height: 1.2 }
.filler { height: 745pt }
.seg { display: flex; flex-direction: row; align-items: start; padding: 30pt 12pt }
.item { display: flex; flex-direction: column; flex-grow: 1; flex-basis: 0; align-items: start; padding: 0 30pt }
h4 { margin: 10pt 0 }
pre { margin: 0; border: 1pt solid #999; page-break-inside: avoid; white-space: pre-wrap }
`)

	var left, right strings.Builder

	left.WriteString("lineL0")

	for lineIdx := 1; lineIdx < 16; lineIdx++ {
		fmt.Fprintf(&left, "\nlineL%d", lineIdx)
	}

	right.WriteString("lineR0")

	for lineIdx := 1; lineIdx < 14; lineIdx++ {
		fmt.Fprintf(&right, "\nlineR%d", lineIdx)
	}

	res := layoutHTML(t,
		`<html><body><div class="filler">filler</div><div class="seg">`+
			`<div class="item"><h4>main.cpp</h4><pre>`+left.String()+`</pre></div>`+
			`<div class="item"><h4>main.py</h4><pre>`+right.String()+`</pre></div>`+
			`</div></body></html>`, cssSheet)

	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	return res
}

// collectPreCardsOnPage gathers left/right pre border and code geometry on
// pageIdx. Horizontal borders are the only non-full-width OpLine entries
// (x about 42 and 280); code lines start with lineL/lineR.
func collectPreCardsOnPage(t *testing.T, res *Result, pageIdx int) (*preCardGeom, *preCardGeom) {
	t.Helper()

	leftCard := newPreCardGeom()
	rightCard := newPreCardGeom()
	borderX := -1.0

	for opIdx, paintOp := range res.Ops {
		if pageOfIdx(t, res, opIdx) != pageIdx {
			continue
		}

		if recordPreCardBorder(paintOp, leftCard, rightCard, &borderX) {
			continue
		}

		recordPreCardCode(paintOp, leftCard, rightCard)
	}

	return leftCard, rightCard
}

func recordPreCardBorder(paintOp Op, leftCard, rightCard *preCardGeom, borderX *float64) bool {
	if paintOp.Kind != OpLine || paintOp.W <= 0 || paintOp.H != 0 || paintOp.W >= 300 {
		return false
	}

	if *borderX < 0 {
		*borderX = paintOp.X
	}

	target := leftCard
	if math.Abs(paintOp.X-*borderX) > 1 {
		target = rightCard
	}

	if target.borderTop < 0 || paintOp.Y < target.borderTop {
		target.borderTop = paintOp.Y
	}

	if paintOp.Y > target.borderBottom {
		target.borderBottom = paintOp.Y
	}

	return true
}

func recordPreCardCode(paintOp Op, leftCard, rightCard *preCardGeom) {
	if paintOp.Kind != OpText {
		return
	}

	var target *preCardGeom

	switch {
	case strings.HasPrefix(paintOp.Text, "lineL"):
		target = leftCard
	case strings.HasPrefix(paintOp.Text, "lineR"):
		target = rightCard
	default:
		return
	}

	if target.firstCode < 0 || paintOp.Y < target.firstCode {
		target.firstCode = paintOp.Y
	}

	if paintOp.Y > target.lastCode {
		target.lastCode = paintOp.Y
	}
}

func assertPreCardBordersHugCode(t *testing.T, name string, preCard *preCardGeom) {
	t.Helper()

	if preCard.borderTop < 0 || preCard.firstCode < 0 {
		t.Fatalf("%s pre: borderTop=%.1f firstCode=%.1f, want both found",
			name, preCard.borderTop, preCard.firstCode)
	}

	if gap := preCard.firstCode - preCard.borderTop; gap < 0 || gap > 20 {
		t.Fatalf("%s pre border top %.1f sits %.1fpt above its first code line "+
			"%.1f; the border must hug its own code", name, preCard.borderTop, gap,
			preCard.firstCode)
	}

	if preCard.borderBottom < preCard.lastCode {
		t.Fatalf("%s pre border bottom %.1f cuts through its last code line "+
			"%.1f", name, preCard.borderBottom, preCard.lastCode)
	}
}
