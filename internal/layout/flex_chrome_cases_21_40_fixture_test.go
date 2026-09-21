//nolint:lll,paralleltest,wsl // fixture assertions favor direct geometry over helper ceremony
package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func fixture21To40(t *testing.T, name string) *Result {
	t.Helper()

	return layoutChromeCase(t, readChromeCase(t, name))
}

func fixtureBox(t *testing.T, res *Result, id string) *box {
	t.Helper()

	result := findBoxByID(res.root, id)
	if result == nil {
		t.Fatalf("fixture box %q not found", id)
	}

	return result
}

func assertFixtureBox(t *testing.T, label string, got *box, want chromeRect) {
	t.Helper()

	assertChromeRect(t, label, chromeRect{x: got.x, y: got.y, w: got.w, h: got.height}, want)
}

// TestChromeFixtureCase21 checks the source fixture's two-line margin layout.
// Chromium's CSS-pixel rectangles are expressed in this repository's pt
// layout units, so the expected container is 241.5 by 85.5pt.
func TestChromeFixtureCase21RowWrapMargins(t *testing.T) {
	res := fixture21To40(t, "case-21-wpt-flow-row-wrap.html")
	container := fixtureBox(t, res, "case-21")
	items := classBoxes(res.root, "item")
	if len(items) != 4 {
		t.Fatalf("case 21 items = %d, want 4", len(items))
	}

	assertFixtureBox(t, "case 21 container", container, chromeRect{x: 0, y: 12, w: 241.5, h: 85.5})
	want := []chromeRect{
		{x: 12.75, y: 24.75, w: 96, h: 18},
		{x: 132.75, y: 24.75, w: 96, h: 18},
		{x: 12.75, y: 66.75, w: 96, h: 18},
		{x: 132.75, y: 66.75, w: 96, h: 18},
	}
	for i, item := range items {
		assertFixtureBox(t, "case 21 item", item, want[i])
	}
}

func TestChromeFixtureCase22ColumnGap(t *testing.T) {
	res := fixture21To40(t, "case-22-wpt-gap-002-ltr.html")
	container := fixtureBox(t, res, "case-22")
	items := classBoxes(res.root, "item")
	if len(items) != 3 {
		t.Fatalf("case 22 items = %d, want 3", len(items))
	}

	assertFixtureBox(t, "case 22 container", container, chromeRect{w: 150, h: 150})
	for i, item := range items {
		assertFixtureBox(t, "case 22 item", item, chromeRect{x: 0, y: float64(i) * 55, w: 150, h: 40})
	}
}

func TestChromeFixtureCase23AspectRatioFeedback(t *testing.T) {
	res := fixture21To40(t, "case-23-wpt-aspect-ratio-cross-size-002.html")
	container := fixtureBox(t, res, "case-23")
	outer := fixtureBox(t, res, "outer-23")
	inner := fixtureBox(t, res, "inner-23")
	box23 := fixtureBox(t, res, "box-23")

	assertFixtureBox(t, "case 23 container", container, chromeRect{w: 200, h: 50})
	assertFixtureBox(t, "case 23 outer", outer, chromeRect{w: 200, h: 50})
	assertFixtureBox(t, "case 23 inner", inner, chromeRect{w: 100, h: 50})
	assertFixtureBox(t, "case 23 box", box23, chromeRect{w: 100, h: 50})
}

func TestChromeFixtureCase24WritingModeMatrix(t *testing.T) {
	res := fixture21To40(t, "case-24-wpt-writing-mode-006.html")
	branchIDs := []string{
		"row-wrap", "row-wrap-reverse", "row-reverse-wrap", "row-reverse-wrap-reverse",
		"column-wrap", "column-wrap-reverse", "column-reverse-wrap", "column-reverse-wrap-reverse",
	}
	for _, id := range branchIDs {
		branch := fixtureBox(t, res, id)
		if len(branch.children) != 4 {
			t.Fatalf("case 24 branch %q children = %d, want 4", id, len(branch.children))
		}
	}

	row := fixtureBox(t, res, "row-wrap")
	first := findBoxByID(res.root, "row-wrap")
	if first == nil {
		t.Fatalf("case 24 row-wrap box is missing")
	}
	assertFixtureBox(t, "case 24 row-wrap", row, chromeRect{w: 41.5, h: 31.5})
	items := classBoxes(first, "item1")
	if len(items) != 1 {
		t.Fatalf("case 24 row-wrap item1 boxes = %d, want 1", len(items))
	}
	assertFixtureBox(t, "case 24 row-wrap item1", items[0], chromeRect{x: 0.75, y: 15.75, w: 20, h: 15})
}

func TestChromeFixtureCase25PercentageAbsoluteChild(t *testing.T) {
	res := fixture21To40(t, "case-25-wpt-flex-item-percentage-abspos.html")
	container := fixtureBox(t, res, "case-25")
	item := fixtureBox(t, res, "item-25")
	fill := fixtureBox(t, res, "fill-25")
	marker := fixtureBox(t, res, "marker-25")

	assertFixtureBox(t, "case 25 container", container, chromeRect{w: 1000, h: 100})
	assertFixtureBox(t, "case 25 item", item, chromeRect{w: 100, h: 100})
	assertFixtureBox(t, "case 25 marker", marker, chromeRect{w: 100, h: 100})
	assertFixtureBox(t, "case 25 fill", fill, chromeRect{w: 100, h: 100})
}

func TestChromeFixtureCase26MinHeightPercentage(t *testing.T) {
	res := fixture21To40(t, "case-26-wpt-definite-sizes-002.html")
	container := fixtureBox(t, res, "case-26")
	item := fixtureBox(t, res, "item-26")
	fill := fixtureBox(t, res, "fill-26")

	assertFixtureBox(t, "case 26 container", container, chromeRect{w: 1000, h: 100})
	assertFixtureBox(t, "case 26 item", item, chromeRect{x: 12, w: 100, h: 100})
	assertFixtureBox(t, "case 26 fill", fill, chromeRect{x: 12, w: 100, h: 100})
}

func TestChromeFixtureCase27ColumnPercentageHeight(t *testing.T) {
	res := fixture21To40(t, "case-27-wpt-percentage-heights-005.html")
	container := fixtureBox(t, res, "case-27")
	item := fixtureBox(t, res, "item-27")
	fill := fixtureBox(t, res, "fill-27")

	assertFixtureBox(t, "case 27 container", container, chromeRect{w: 1000, h: 100})
	assertFixtureBox(t, "case 27 item", item, chromeRect{x: 12, w: 100, h: 100})
	assertFixtureBox(t, "case 27 fill", fill, chromeRect{x: 12, w: 100, h: 100})
}

func TestChromeFixtureCase28AspectRatioMinimumWidth(t *testing.T) {
	res := fixture21To40(t, "case-28-wpt-flex-minimum-width-aspect.html")
	container := fixtureBox(t, res, "case-28")
	item := fixtureBox(t, res, "item-28")

	assertFixtureBox(t, "case 28 container", container, chromeRect{x: 12, w: 10, h: 100})
	assertFixtureBox(t, "case 28 item", item, chromeRect{x: 12, w: 100, h: 100})
}

// TestChromeFixtureCase29NestedFloatLeft pins the float's cross-axis position.
// The column flex base-size measure used to register the six-inch float in the
// live BFC, so the real build packed it beside the ghost registration and the
// float landed flush right at x=width-144pt instead of the left content edge.
func TestChromeFixtureCase29NestedFloatLeft(t *testing.T) {
	res := fixture21To40(t, "case-29-wpt-break-nested-float-print.html")
	target := fixtureBox(t, res, "target-29")

	assertFixtureBox(t, "case 29 target", target, chromeRect{x: 0, y: 0, w: 144, h: 432})
}

// TestChromeFixtureCase33SpacerFillsSurviveStrip: the description panel adds
// page ink above the case, which armed the orphan-row strip on the 150x30 flex
// spacer fills and zeroed four of them. Flex item fills are definite box
// paint, so they must survive the strip exactly like the tighten pass already
// guarantees.
func TestChromeFixtureCase33SpacerFillsSurviveStrip(t *testing.T) {
	t.Parallel()

	res := fixture21To40(t, "case-33-wpt-flex-item-compressible.html")
	if err := Paint(pdf.NewDocument(), res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	fills := 0
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpFillRect && near(op.W, 150) && near(op.H, 30) {
			fills++
		}
	}

	if fills != 5 {
		t.Fatalf("case 33 spacer fills after paint = %d, want 5", fills)
	}
}

//nolint:cyclop // the assertion covers horizontal and vertical-writing branches
func TestChromeFixtureCase30ColumnAutoMargins(t *testing.T) {
	res := fixture21To40(t, "case-30-wpt-auto-margins-column.html")
	horizontal := fixtureBox(t, res, "case-30-horizontal")
	margins := fixtureBox(t, res, "item-30-margins")
	align := fixtureBox(t, res, "item-30-align")

	assertFixtureBox(t, "case 30 horizontal", horizontal, chromeRect{w: 301.5, h: 151.5})
	horizontalCenter := horizontal.x + horizontal.w/2
	for label, item := range map[string]*box{
		"case 30 margin item": margins,
		"case 30 align item":  align,
	} {
		if item.w <= 0 || item.height <= 0 {
			t.Fatalf("%s is empty: %.2fx%.2f", label, item.w, item.height)
		}
		if math.Abs(item.x+item.w/2-horizontalCenter) > 1 {
			t.Fatalf("%s is not centered on the horizontal cross-axis: center %.2f, want %.2f", label, item.x+item.w/2, horizontalCenter)
		}
	}
	if math.Abs(align.y-(margins.y+margins.height)) > 1 {
		t.Fatalf("case 30 column items are not consecutive: align y %.2f, margin bottom %.2f", align.y, margins.y+margins.height)
	}

	vertical := fixtureBox(t, res, "case-30-vertical")
	verticalMargins := fixtureBox(t, res, "item-30-vertical-margins")
	verticalAlign := fixtureBox(t, res, "item-30-vertical-align")
	assertFixtureBox(t, "case 30 vertical", vertical, chromeRect{y: 151.5, w: 301.5, h: 151.5})
	for label, item := range map[string]*box{
		"case 30 vertical margin item": verticalMargins,
		"case 30 vertical align item":  verticalAlign,
	} {
		if item.w <= 0 || item.height <= 0 {
			t.Fatalf("%s is empty: %.2fx%.2f", label, item.w, item.height)
		}
	}
	if verticalAlign.x <= verticalMargins.x {
		t.Fatalf("case 30 vertical items did not advance along the column axis: margin x %.2f, align x %.2f", verticalMargins.x, verticalAlign.x)
	}
	// Chromium centers each item on the container's vertical cross axis: the
	// two auto margins split the free space for the margin item, and
	// align-self: center centers the align item. Their centers must therefore
	// coincide with the container center, not merely with each other.
	verticalCenterY := vertical.y + vertical.height/2
	for label, item := range map[string]*box{
		"case 30 vertical margin item": verticalMargins,
		"case 30 vertical align item":  verticalAlign,
	} {
		if center := item.y + item.height/2; math.Abs(center-verticalCenterY) > 1 {
			t.Fatalf("%s is not centered on the vertical cross-axis: center %.2f, want %.2f",
				label, center, verticalCenterY)
		}
	}
}

func TestChromeFixtureCase31ColumnReverseWrap(t *testing.T) {
	res := fixture21To40(t, "case-31-wpt-column-reverse-multiline.html")
	container := fixtureBox(t, res, "case-31")
	assertFixtureBox(t, "case 31 container", container, chromeRect{x: 0, y: 12, w: 241.5, h: 97.5})
	assertFixtureBox(t, "case 31 item one", fixtureBox(t, res, "item-one"), chromeRect{x: 12.75, y: 78.75, w: 96, h: 18})
	assertFixtureBox(t, "case 31 item two", fixtureBox(t, res, "item-two"), chromeRect{x: 12.75, y: 36.75, w: 96, h: 18})
	assertFixtureBox(t, "case 31 item three", fixtureBox(t, res, "item-three"), chromeRect{x: 132.75, y: 78.75, w: 96, h: 18})
	assertFixtureBox(t, "case 31 item four", fixtureBox(t, res, "item-four"), chromeRect{x: 132.75, y: 36.75, w: 96, h: 18})
}

func TestChromeFixtureCase32FractionalFactors(t *testing.T) {
	res := fixture21To40(t, "case-32-wpt-flex-factor-less-than-one.html")
	rowGrow := fixtureBox(t, res, "row-grow")
	rowGrowBasis := fixtureBox(t, res, "row-grow-basis")
	rowShrink := fixtureBox(t, res, "row-shrink")
	if !near(rowGrow.w, 77) || !near(rowGrowBasis.w, 77) || !near(rowShrink.w, 77) {
		t.Fatalf("case 32 container widths = %.2f/%.2f/%.2f, want 77", rowGrow.w, rowGrowBasis.w, rowShrink.w)
	}

	growHalf := classBoxes(res.root, "grow-half")
	growQuarter := classBoxes(res.root, "grow-quarter")
	if len(growHalf) != 4 || len(growQuarter) != 4 {
		t.Fatalf("case 32 grow boxes = %d/%d, want 4/4", len(growHalf), len(growQuarter))
	}
	if !near(growHalf[0].w, 37.5) || !near(growQuarter[0].w, 18.75) {
		t.Fatalf("case 32 row fractional grow widths = %.2f/%.2fpt, want 37.5/18.75pt", growHalf[0].w, growQuarter[0].w)
	}
	if !near(growHalf[1].height, 37.5) || !near(growQuarter[1].height, 18.75) {
		t.Fatalf("case 32 column fractional grow heights = %.2f/%.2fpt, want 37.5/18.75pt", growHalf[1].height, growQuarter[1].height)
	}
}

func TestChromeFixtureCase33ReplacedInputs(t *testing.T) {
	res := fixture21To40(t, "case-33-wpt-flex-item-compressible.html")
	inputs := classBoxes(res.root, "test1")
	if len(inputs) != 3 {
		t.Fatalf("case 33 test1 inputs = %d, want 3", len(inputs))
	}
	for i, input := range inputs {
		assertFixtureBox(t, "case 33 test1 input", input, chromeRect{x: 151, y: float64(i) * 37, w: 75, h: 30})
	}

	calc := classBoxes(res.root, "test2")
	wide := classBoxes(res.root, "test3")
	if len(calc) != 1 || len(wide) != 1 {
		t.Fatalf("case 33 special inputs = %d/%d, want 1/1", len(calc), len(wide))
	}
	if !near(calc[0].w, 75) || !near(wide[0].w, 105) {
		t.Fatalf("case 33 calc and expanded input widths = %.2f/%.2fpt, want 75/105pt", calc[0].w, wide[0].w)
	}
}

func TestChromeFixtureCase34MaxContentContribution(t *testing.T) {
	res := fixture21To40(t, "case-34-wpt-flex-container-max-content.html")
	row := fixtureBox(t, res, "case-34-row")
	column := fixtureBox(t, res, "case-34-column")
	if !near(row.w, 114) || !near(column.w, 92.01) {
		t.Fatalf("case 34 intrinsic widths = %.2f/%.2fpt, want 114/92.01pt", row.w, column.w)
	}
}

func TestChromeFixtureCase35BorderBoxReplacedCrossSize(t *testing.T) {
	res := fixture21To40(t, "case-35-wpt-flex-cross-size-border-box.html")
	border := fixtureBox(t, res, "case-35-border")
	content := fixtureBox(t, res, "case-35-content")
	items := classBoxes(res.root, "item")
	if len(items) != 2 {
		t.Fatalf("case 35 items = %d, want 2", len(items))
	}

	assertFixtureBox(t, "case 35 border container", border, chromeRect{w: 300, h: 150})
	assertFixtureBox(t, "case 35 content container", content, chromeRect{x: 0, y: 150, w: 316, h: 150})
	if !near(items[0].height, 0.75) {
		t.Fatalf("case 35 replaced border-box child height = %.2fpt, want 0.75pt", items[0].height)
	}
}

func TestChromeFixtureCase36MaxWidthBaseSize(t *testing.T) {
	res := fixture21To40(t, "case-36-wpt-flex-base-size-max-width.html")
	container := fixtureBox(t, res, "case-36")
	capped := findBoxByClass(t, res, "capped")
	uncapped := findBoxByClass(t, res, "uncapped")

	assertFixtureBox(t, "case 36 container", container, chromeRect{w: 225, h: 37.5})
	assertFixtureBox(t, "case 36 capped", capped, chromeRect{w: 75, h: 37.5})
	if !near(uncapped.w, 150) {
		t.Fatalf("case 36 uncapped max-width sibling = %.2fpt, want 150pt", uncapped.w)
	}
	assertFixtureBox(t, "case 36 uncapped", uncapped, chromeRect{x: 75, w: 150, h: 37.5})
}

func TestChromeFixtureCase37ReplacedRatioPrecision(t *testing.T) {
	res := fixture21To40(t, "case-37-blink-replaced-aspect-ratio-precision.html")
	container := fixtureBox(t, res, "case-37")
	glyph := fixtureBox(t, res, "svg-37")

	assertFixtureBox(t, "case 37 container", container, chromeRect{w: 37.5, h: 16.5})
	assertFixtureBox(t, "case 37 svg", glyph, chromeRect{x: 7.875, w: 21.75, h: 16.5})
}

func TestChromeFixtureCase38WrappedGapGeometry(t *testing.T) {
	res := fixture21To40(t, "case-38-blink-gap-decorations-basic.html")
	container := fixtureBox(t, res, "case-38")
	items := classBoxes(res.root, "item")
	if len(items) != 6 {
		t.Fatalf("case 38 items = %d, want 6", len(items))
	}

	assertFixtureBox(t, "case 38 container", container, chromeRect{w: 174, h: 114})
	want := []chromeRect{
		{x: 2, y: 2, w: 50, h: 50}, {x: 62, y: 2, w: 50, h: 50}, {x: 122, y: 2, w: 50, h: 50},
		{x: 2, y: 62, w: 50, h: 50}, {x: 62, y: 62, w: 50, h: 50}, {x: 122, y: 62, w: 50, h: 50},
	}
	for i, item := range items {
		assertFixtureBox(t, "case 38 item", item, want[i])
	}
}

func TestChromeFixtureCase39VerticalOverflow(t *testing.T) {
	res := fixture21To40(t, "case-39-blink-scrollbars-row-reverse-vrl.html")
	container := fixtureBox(t, res, "case-39")
	item := fixtureBox(t, res, "item-39")

	assertFixtureBox(t, "case 39 container", container, chromeRect{w: 375, h: 285})
	assertFixtureBox(t, "case 39 overflowing item", item, chromeRect{x: -1155, y: 22.5, w: 1500, h: 225})
}

func TestChromeFixtureCase40MinContentContribution(t *testing.T) {
	res := fixture21To40(t, "case-40-wpt-flex-container-min-content.html")
	container := fixtureBox(t, res, "case-40")
	items := classBoxes(res.root, "item")
	if len(items) != 2 {
		t.Fatalf("case 40 items = %d, want 2", len(items))
	}

	if !near(container.w, 110) {
		t.Fatalf("case 40 min-content width = %.2fpt, want 110pt", container.w)
	}
	if !near(items[0].x, 5) || !near(items[0].w, 40) || !near(items[1].x, 55) || !near(items[1].w, 50) {
		t.Fatalf("case 40 item geometry = %.2fx%.2f and %.2fx%.2f, want 5x40 and 55x50",
			items[0].x, items[0].w, items[1].x, items[1].w)
	}
}
