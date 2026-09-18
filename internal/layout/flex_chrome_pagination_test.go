package layout

import (
	"fmt"
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// flexWrapFixture builds the six 50x50 item wrap case with a spacer of
// spacerPt above it, using the Chrome Flex wrap CSS. It returns the laid-out
// (not yet painted) result.
func flexWrapFixture(t *testing.T, spacerPt string) *Result {
	t.Helper()

	styleSheet := sheet(t, fmt.Sprintf(`
html, body { margin: 0; padding: 0 }
.spacer { height: %s }
.case { display: flex; flex-wrap: wrap; column-gap: 10pt; row-gap: 10pt; width: 170pt; border: 2pt dotted #608ba8 }
.item { width: 50pt; height: 50pt; background: rgba(96, 139, 168, 0.2); font-size: 8pt }
`, spacerPt))
	res := layoutHTML(t, `<html><body><div class="spacer"></div><div class="case">`+
		`<div class="item">One</div><div class="item">Two</div><div class="item">Three</div>`+
		`<div class="item">Four</div><div class="item">Five</div><div class="item">Six</div>`+
		`</div></body></html>`, styleSheet)

	if items := classBoxes(res.root, "item"); len(items) != 6 {
		t.Fatalf("item boxes = %d, want 6", len(items))
	}

	return res
}

// flexRowGroups assigns each item its pre-paint flex line index. Call it
// before Paint: pagination moves item boxes, so post-paint Y values can no
// longer identify the original lines.
func flexRowGroups(t *testing.T, items []*box) []int {
	t.Helper()

	rowOf := make([]int, len(items))
	rowTops := make([]float64, 0, 2)

	for itemIdx, itemBox := range items {
		rowOf[itemIdx] = -1

		for rowIdx, top := range rowTops {
			if near(itemBox.y, top) {
				rowOf[itemIdx] = rowIdx

				break
			}
		}

		if rowOf[itemIdx] == -1 {
			rowOf[itemIdx] = len(rowTops)
			rowTops = append(rowTops, itemBox.y)
		}
	}

	if len(rowTops) != 2 {
		t.Fatalf("flex line count = %d, want 2 (items at y=%v)", len(rowTops), itemYs(items))
	}

	return rowOf
}

// assertFlexRowFillGeometry checks each item's own fill against its box and
// requires same-line items to share one fill Y. rowOf is the pre-paint line
// assignment from flexRowGroups.
func assertFlexRowFillGeometry(t *testing.T, res *Result, items []*box, rowOf []int) {
	t.Helper()

	fillY := make([]float64, len(items))

	for itemIdx, itemBox := range items {
		fill, ok := chromeItemFill(res, itemBox)
		if !ok {
			t.Errorf("item %d (x=%.1f y=%.1f): no matching OpFillRect", itemIdx, itemBox.x, itemBox.y)

			continue
		}

		fillY[itemIdx] = fill.Y

		if !near(fill.Y, itemBox.y) {
			t.Errorf("item %d fill Y=%.2f, box y=%.2f (drift %.2f)", itemIdx, fill.Y, itemBox.y, fill.Y-itemBox.y)
		}

		if !near(fill.H, itemBox.height) {
			t.Errorf("item %d fill H=%.2f, box height=%.2f", itemIdx, fill.H, itemBox.height)
		}
	}

	for itemIdx := range items {
		for otherIdx := itemIdx + 1; otherIdx < len(items); otherIdx++ {
			if rowOf[itemIdx] != rowOf[otherIdx] {
				continue
			}

			if math.Abs(fillY[itemIdx]-fillY[otherIdx]) > 0.01 {
				t.Errorf("same-line items %d,%d fills at Y=%.2f,%.2f (staircase %.2f)",
					itemIdx, otherIdx, fillY[itemIdx], fillY[otherIdx], fillY[otherIdx]-fillY[itemIdx])
			}
		}
	}
}

// TestChromeFlexSnapLineChromeFindsRow pins the collection step: the snapped
// "Four" text op must resolve three flex-line boxes and their three 50pt
// fills before any shift. The row shares one top Y, so the line band lookup
// has to include the snapped item's own box, not just its ancestors.
func TestChromeFlexSnapLineChromeFindsRow(t *testing.T) {
	t.Parallel()

	res := flexWrapFixture(t, "770pt")

	if assertSnapLineFindsRow(t, res) {
		return
	}

	t.Fatal("Four text op not found")
}

// assertSnapLineFindsRow resolves the snapped "Four" text and checks the line
// collection. It reports whether the text op was found.
func assertSnapLineFindsRow(t *testing.T, res *Result) bool {
	t.Helper()

	for opIdx, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Text != "Four" {
			continue
		}

		refs, boxes := snapLineChrome(res, opIdx, paintOp.Y)

		assertSnapLineBoxes(t, boxes, refs)
		assertSnapLineRefs(t, res, refs, boxes)

		return true
	}

	return false
}

func assertSnapLineBoxes(t *testing.T, boxes []lineBox, refs []lineChromeRef) {
	t.Helper()

	if len(boxes) != 3 || len(refs) != 3 {
		t.Fatalf("line collection = %d boxes, %d refs, want 3 boxes and 3 fills", len(boxes), len(refs))
	}

	for _, line := range boxes {
		if !near(line.box.y, boxes[0].box.y) {
			t.Fatalf("line boxes at y=%.2f and %.2f, want one line band", boxes[0].box.y, line.box.y)
		}

		// The old rowChromeBandCandidate cap rejected fills over 40pt;
		// these items are 50pt, so ownership, not a raised cap, must be
		// what collects them.
		if line.box.height <= 40 {
			t.Fatalf("line box height=%.2f, want over 40pt to prove the height cap is not the fix", line.box.height)
		}
	}
}

func assertSnapLineRefs(t *testing.T, res *Result, refs []lineChromeRef, boxes []lineBox) {
	t.Helper()

	for _, ref := range refs {
		refOp := res.Ops[ref.index]
		if refOp.Kind != OpFillRect || !near(refOp.Y, boxes[0].box.y) || !near(refOp.H, boxes[0].box.height) {
			t.Fatalf("ref op = kind %v y=%.2f h=%.2f, want the item fill at the line top", refOp.Kind, refOp.Y, refOp.H)
		}
	}
}

// TestChromeFlexWrapRowChromeCrossesWithText pins the wrapped-row pagination defect:
// a wrapped flex row whose second line straddles a page boundary must move
// its item backgrounds with its texts. Before the fix, snapCrossingTextOps
// moved the text (and its own path chrome) but left the 50pt sibling fills
// behind; the orphans/widows fixpoint then re-snapped each item separately,
// producing a 10pt staircase of fills while all texts landed on one Y.
func TestChromeFlexWrapRowChromeCrossesWithText(t *testing.T) {
	t.Parallel()

	// Portrait geometry: the 770pt spacer leaves row 2 with its text ink
	// crossing the 842pt boundary, so the snap path carries the line.
	res := flexWrapFixture(t, "770pt")
	items := classBoxes(res.root, "item")
	rowOf := flexRowGroups(t, items)

	if err := Paint(pdf.NewDocument(), res, PaintOptions{PageWidth: 400, PageHeight: 842}); err != nil {
		t.Fatal(err)
	}

	assertFlexRowFillGeometry(t, res, items, rowOf)
	logWrapDiagnostics(t, res)
}

// TestChromeFlexFixtureGeometryRowChromeCrosses reproduces the wrapped-row geometry at
// its real geometry: A4 landscape with 12mm (34.02pt) margins, content height
// 527.24pt. Row 2 starts ~23pt above the page 3/4 boundary, so its fills
// cross while its text ink stays on page 3. Every same-line fill must still
// land on the item box Y without a split.
func TestChromeFlexFixtureGeometryRowChromeCrosses(t *testing.T) {
	t.Parallel()

	// Spacer 4*527.24 - 23.08 - 62 puts row 2 at 2085.88: fills cross the
	// 2108.96 boundary, text baselines sit at 2093.45 (inside page 3).
	res := flexWrapFixture(t, "2023.88pt")
	items := classBoxes(res.root, "item")
	rowOf := flexRowGroups(t, items)

	margin := 34.02
	opts := PaintOptions{
		PageWidth:    841.89,
		PageHeight:   595.28,
		MarginTop:    margin,
		MarginBottom: margin,
		MarginLeft:   margin,
		MarginRight:  margin,
	}

	if err := Paint(pdf.NewDocument(), res, opts); err != nil {
		t.Fatal(err)
	}

	assertFlexRowFillGeometry(t, res, items, rowOf)
	logWrapDiagnostics(t, res)
}

// flexSectionsPaginationFixture builds the Chrome Flex sections 11 and 12
// markup behind a spacer tall enough to cross page boundaries. It returns the
// laid-out (not yet painted) result at the fixture's section geometry.
func flexSectionsPaginationFixture(t *testing.T, spacerPt string) *Result {
	t.Helper()

	styleSheet := sheet(t, fmt.Sprintf(`
html, body { margin: 0; padding: 0; font: 11pt/1.4 sans-serif }
.spacer-page { height: %s }
.ex { page-break-inside: avoid; overflow: hidden; border: 1px solid #bbb; padding: 8pt; margin: 0 0 12pt }
.ex h2 { font-size: 12pt; margin: 0 0 2pt }
.ex .note { color: #666; font-size: 9pt; margin: 0 0 8pt }
.flexbox { display: flex; width: 300pt; height: 40pt }
.spacer { flex: 0 0 200pt; background: #e6e6e6 }
.test1 { width: 100%%; border: 0; padding: 0; margin: 0; background: #9ec5fe }
.flex { display: flex; width: 400pt; border: 10pt solid #222; margin: 0 0 8pt }
.border-box { height: 200pt; box-sizing: border-box }
.content-box { height: 180pt }
.child { width: 100pt; background: #b7e4c7 }
.tail { height: 20pt; background: #0aa }
`, spacerPt))

	return layoutHTML(t, `<html><body><div class="spacer-page"></div>`+
		`<section class="ex"><h2>11. compressible</h2><p class="note">note eleven</p>`+
		`<div class="flexbox"><div class="spacer"></div><input class="test1" type="text"></div></section>`+
		`<section class="ex"><h2>12. border-box</h2><p class="note">note twelve</p>`+
		`<div class="flex border-box"><div class="child"></div></div>`+
		`<div class="flex content-box"><div class="child"></div></div></section>`+
		`<section class="ex"><h2>13. trailing frame</h2><p class="note">note thirteen</p>`+
		`<div class="tail"></div></section>`+
		`</body></html>`, styleSheet)
}

// pagedOpSet marks every op index that Paint assigned to a page.
func pagedOpSet(res *Result) map[int]bool {
	paged := make(map[int]bool)

	for _, idxs := range res.Pages {
		for _, opIdx := range idxs {
			paged[opIdx] = true
		}
	}

	return paged
}

// pagedFillExtent returns the widest painted fill in boxNode's op range and
// the total height of its painted fragments, so a fill split at a page
// boundary still reads as one 200x40 box.
func pagedFillExtent(res *Result, paged map[int]bool, boxNode *box) (float64, float64) {
	maxW, totalH := 0.0, 0.0

	for opIdx := boxNode.opStart; opIdx <= boxNode.opEnd && opIdx < len(res.Ops) && opIdx >= 0; opIdx++ {
		if !paged[opIdx] {
			continue
		}

		paintOp := res.Ops[opIdx]
		if paintOp.Kind != OpFillRect {
			continue
		}

		totalH += paintOp.H

		if paintOp.W > maxW {
			maxW = paintOp.W
		}
	}

	return maxW, totalH
}

// TestChromeFlexSectionsSurviveOverflowClip pins the Chrome Flex sections 11
// and 12 defect: the flexbox item fills, the stretched child fills and the
// 10pt section borders are deactivated before Paint, so the PDF pages show
// bare text where the fixture's box model should paint. Each fill must land
// on a page.
func TestChromeFlexSectionsSurviveOverflowClip(t *testing.T) {
	t.Parallel()

	res := flexSectionsPaginationFixture(t, "600pt")

	opts := PaintOptions{
		PageWidth:    841.89,
		PageHeight:   595.28,
		MarginTop:    34.02,
		MarginBottom: 34.02,
		MarginLeft:   34.02,
		MarginRight:  34.02,
	}

	if err := Paint(pdf.NewDocument(), res, opts); err != nil {
		t.Fatal(err)
	}

	paged := pagedOpSet(res)

	assertPagedBoxFill(t, res, paged, "flex spacer", "spacer", 200, 40)
	assertPagedBoxFill(t, res, paged, "input", "test1", 100, 40)
	assertPagedBoxFill(t, res, paged, "child 0 and 1", "child", 100, 180)
	assertPagedFlexBorders(t, res, paged)
}

func assertPagedBoxFill(t *testing.T, res *Result, paged map[int]bool, label, className string, wantW, wantH float64) {
	t.Helper()

	boxes := classBoxes(res.root, className)
	if len(boxes) == 0 {
		t.Fatalf("%s: no %q boxes", label, className)
	}

	for boxIdx, boxNode := range boxes {
		width, height := pagedFillExtent(res, paged, boxNode)

		if !near(width, wantW) || !near(height, wantH) {
			t.Errorf("%s %d: painted fill %.1fx%.1f, want %.0fx%.0f (box range [%d,%d])",
				label, boxIdx, width, height, wantW, wantH, boxNode.opStart, boxNode.opEnd)
		}
	}
}

func assertPagedFlexBorders(t *testing.T, res *Result, paged map[int]bool) {
	t.Helper()

	strokes := 0

	for _, boxNode := range classBoxes(res.root, "flex") {
		for opIdx := boxNode.opStart; opIdx <= boxNode.opEnd && opIdx < len(res.Ops) && opIdx >= 0; opIdx++ {
			paintOp := res.Ops[opIdx]
			if paged[opIdx] && (paintOp.Kind == OpStrokeRect || paintOp.Kind == OpLine) && near(paintOp.Width, 10) {
				strokes++
			}
		}
	}

	if strokes == 0 {
		t.Errorf("painted 10pt flex container borders = 0, want at least one")
	}
}

// TestChromeFlexTextOnlyLineSnapsTogether covers a wrapped flex line whose
// items have no chrome at all: the snap sweep still has to move every line
// box with the snapped text, not just the snapped item's own box.
func TestChromeFlexTextOnlyLineSnapsTogether(t *testing.T) {
	t.Parallel()

	styleSheet := sheet(t, `
html, body { margin: 0; padding: 0 }
.spacer { height: 770pt }
.case { display: flex; flex-wrap: wrap; column-gap: 10pt; row-gap: 10pt; width: 170pt }
.item { width: 50pt; height: 50pt; font-size: 8pt }
`)
	res := layoutHTML(t, `<html><body><div class="spacer"></div><div class="case">`+
		`<div class="item">One</div><div class="item">Two</div><div class="item">Three</div>`+
		`<div class="item">Four</div><div class="item">Five</div><div class="item">Six</div>`+
		`</div></body></html>`, styleSheet)
	items := classBoxes(res.root, "item")
	rowOf := flexRowGroups(t, items)

	if err := Paint(pdf.NewDocument(), res, PaintOptions{PageWidth: 400, PageHeight: 842}); err != nil {
		t.Fatal(err)
	}

	assertSameLineBoxesTogether(t, items, rowOf)
	assertLineTextsInsideItems(t, res, items)
}

// assertSameLineBoxesTogether requires same-line item boxes to share one top Y
// after the snap sweep.
func assertSameLineBoxesTogether(t *testing.T, items []*box, rowOf []int) {
	t.Helper()

	for itemIdx := range items {
		for otherIdx := itemIdx + 1; otherIdx < len(items); otherIdx++ {
			if rowOf[itemIdx] == rowOf[otherIdx] && !near(items[itemIdx].y, items[otherIdx].y) {
				t.Errorf("same-line items %d,%d boxes at y=%.2f,%.2f",
					itemIdx, otherIdx, items[itemIdx].y, items[otherIdx].y)
			}
		}
	}
}

// assertLineTextsInsideItems requires every wrapped-row text op to sit inside
// one of the item boxes.
func assertLineTextsInsideItems(t *testing.T, res *Result, items []*box) {
	t.Helper()

	for _, paintOp := range res.Ops {
		if !isWrappedRowText(paintOp) {
			continue
		}

		if !textInsideAnyItem(paintOp, items) {
			t.Errorf("text %q at y=%.2f sits outside its item box", paintOp.Text, paintOp.Y)
		}
	}
}

// isWrappedRowText reports whether paintOp is one of the wrapped row texts.
func isWrappedRowText(paintOp Op) bool {
	if paintOp.Kind != OpText {
		return false
	}

	return paintOp.Text == "Four" || paintOp.Text == "Five" || paintOp.Text == "Six"
}

// textInsideAnyItem reports whether the text op's ink sits inside one item box.
func textInsideAnyItem(paintOp Op, items []*box) bool {
	for _, itemBox := range items {
		if near(itemBox.x, paintOp.X) && paintOp.Y > itemBox.y && paintOp.Y < itemBox.y+itemBox.height {
			return true
		}
	}

	return false
}

// logWrapDiagnostics prints item fill and text positions so a failing run
// shows the staircase in absolute canvas coordinates.
func logWrapDiagnostics(t *testing.T, res *Result) {
	t.Helper()

	for fillIdx, fill := range itemFills(res) {
		t.Logf("DIAG fill[%d] x=%.2f y=%.2f w=%.2f h=%.2f", fillIdx, fill.X, fill.Y, fill.W, fill.H)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText {
			t.Logf("DIAG text %q x=%.2f y=%.2f", paintOp.Text, paintOp.X, paintOp.Y)
		}
	}
}

// itemYs reports each item's y for diagnostics.
func itemYs(items []*box) []float64 {
	out := make([]float64, len(items))

	for itemIdx, itemBox := range items {
		out[itemIdx] = itemBox.y
	}

	return out
}

// chromeItemFill picks the item's own background fill: an OpFillRect at the
// item's x with the item's width, choosing the fragment nearest the box's top
// so items in different rows (which share x) cannot be confused. A tie
// prefers the taller fragment, so an intact fill beats a page fragment.
func chromeItemFill(res *Result, itemBox *box) (Op, bool) {
	var best Op

	found := false

	for _, candidate := range res.Ops {
		if candidate.Kind != OpFillRect || !near(candidate.X, itemBox.x) ||
			!near(candidate.W, itemBox.w) || candidate.H <= 0 {
			continue
		}

		if !found {
			best = candidate
			found = true

			continue
		}

		bestDist := math.Abs(best.Y - itemBox.y)
		opDist := math.Abs(candidate.Y - itemBox.y)

		if opDist < bestDist-layoutCoordEpsilon || (math.Abs(opDist-bestDist) <= layoutCoordEpsilon && candidate.H > best.H) {
			best = candidate
		}
	}

	return best, found
}

func itemFills(res *Result) []Op {
	var out []Op

	for _, fill := range res.Ops {
		if fill.Kind == OpFillRect && fill.W > 45 && fill.W < 55 && fill.H > 10 && fill.H <= 51 {
			out = append(out, fill)
		}
	}

	return out
}
