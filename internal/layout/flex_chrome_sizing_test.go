package layout

import "testing"

func flexChromeItemWidths(t *testing.T, res *Result, wantCount int) []float64 {
	t.Helper()

	items := classBoxes(res.root, "item")
	if len(items) != wantCount {
		t.Fatalf("flex item boxes = %d, want %d", len(items), wantCount)
	}

	widths := make([]float64, len(items))
	for i, item := range items {
		widths[i] = item.w
	}

	return widths
}

func assertFlexChromeWidths(t *testing.T, got, want []float64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("flex item width count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if !near(got[i], want[i]) {
			t.Errorf("flex item[%d] width = %.2f, want %.2f", i, got[i], want[i])
		}
	}
}

// Chromium's flex-algorithm fixture uses a zero basis and distributes the
// positive free space by the grow factors 1, 2, and 3.
func TestChromeFlexBasisAndGrow(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.case { display:flex; width:600pt; height:20pt }
.item { flex-basis:0; min-width:0; height:20pt }
.one { flex-grow:1 }
.two { flex-grow:2 }
.three { flex-grow:3 }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item one"></div>
  <div class="item two"></div>
  <div class="item three"></div>
</div>
</body></html>`, cssSheet)

	assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 3), []float64{100, 200, 300})
}

// Chromium's negative-flexing case uses the scaled shrink factors from the
// flex bases: 1*300 and 3*300, leaving widths 250 and 150 beside a fixed 200.
func TestChromeFlexShrinkUsesScaledFactors(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.case { display:flex; width:600pt; height:20pt }
.item { min-width:0; height:20pt }
.first { flex:1 1 300pt }
.second { flex:2 3 300pt }
.fixed { flex:0 0 200pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item first"></div>
  <div class="item second"></div>
  <div class="item fixed"></div>
</div>
</body></html>`, cssSheet)

	assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 3), []float64{250, 150, 200})
}

// Chromium freezes the max-width item, then redistributes the remaining
// positive free space across the unfrozen items.
func TestChromeFlexMinMaxFreezesAndRedistributes(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.case { display:flex; width:600pt; height:20pt }
.item { flex:1 1 200pt; min-width:0; height:20pt }
.frozen { max-width:100pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item frozen"></div>
  <div class="item"></div>
  <div class="item"></div>
</div>
</body></html>`, cssSheet)

	assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 3), []float64{100, 250, 250})
}

// The Chromium fractional-factor fixture keeps the unused 25pt when the
// grow-factor sum is below one: .5 gets 50pt and .25 gets 25pt.
func TestChromeFlexFractionalGrowFactorsLeaveUnusedSpace(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.case { display:flex; width:100pt; height:20pt }
.item { flex-basis:0; min-width:0; height:20pt }
.half { flex-grow:.5 }
.quarter { flex-grow:.25 }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item half"></div>
  <div class="item quarter"></div>
</div>
</body></html>`, cssSheet)

	assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 2), []float64{50, 25})
}

// TestChromeFlexPercentageBasisIndefiniteColumn covers case
// wpt-flex-basis-011 (layout-unit), source
// third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-basis-011.html.
// Expected: a 100% flex-basis in a nested column resolves against the
// column's stretched used height. CSS Flexbox 9.4 step 5 makes a stretched
// flex item's used cross size definite for its contents, so the column's
// content-sized height (two 22pt items = 44pt) becomes the percentage basis:
// each item's content resolves to 44pt, 46pt with its borders, and the two
// non-shrinking items overflow the column. The definite branch keeps a 100pt
// column at 100pt while its two 102pt, non-shrinking items overflow it.
//
//nolint:cyclop,funlen // indefinite and definite branches share one source fixture
func TestChromeFlexPercentageBasisIndefiniteColumn(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.flexbox { display: flex }
.column { flex-direction: column }
.item { flex: 1 0 100%; border: 1pt solid blue; font-size: 10pt; line-height: 20pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="flexbox">
  <div class="flexbox column">
    <div class="item item-a"><div>AAA</div></div>
    <div class="item item-b"><div>BBB</div></div>
  </div>
</div>
</body></html>`, cssSheet)

	column := findBoxByClass(t, res, "column")
	itemA := findBoxByClass(t, res, "item-a")
	itemB := findBoxByClass(t, res, "item-b")

	const (
		columnHeight = 44.0 // two content-sized 22pt items
		itemHeight   = 46.0 // 100% of the 44pt stretched column plus 1pt borders
	)

	if !near(itemA.height, itemHeight) || !near(itemB.height, itemHeight) {
		t.Fatalf("item heights = %.2f/%.2f, want %.2f each (100%% of the stretched column)",
			itemA.height, itemB.height, itemHeight)
	}

	if !near(column.height, columnHeight) {
		t.Fatalf("column height = %.2f, want %.2f", column.height, columnHeight)
	}

	if !near(itemA.y, column.y) || !near(itemB.y, itemA.y+itemHeight) {
		t.Fatalf("item y positions = %.2f/%.2f, want %.2f/%.2f",
			itemA.y, itemB.y, column.y, itemA.y+itemHeight)
	}

	t.Run("definite-column", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html><body>
<div class="flexbox column definite">
  <div class="item item-a"><div>AAA</div></div>
  <div class="item item-b"><div>BBB</div></div>
</div>
</body></html>`, sheet(t, `
body { margin: 0 }
.flexbox { display: flex }
.column { flex-direction: column; height: 100pt }
.item { flex: 1 0 100%; border: 1pt solid blue; font-size: 10pt; line-height: 20pt }
`))

		column := findBoxByClass(t, res, "definite")
		itemA := findBoxByClass(t, res, "item-a")
		itemB := findBoxByClass(t, res, "item-b")

		const itemHeight = 102.0 // 100pt basis plus 1pt border on each side

		if !near(itemA.height, itemHeight) || !near(itemB.height, itemHeight) {
			t.Fatalf("definite item heights = %.2f/%.2f, want %.2f each",
				itemA.height, itemB.height, itemHeight)
		}

		if !near(column.height, 100) {
			t.Fatalf("definite column height = %.2f, want 100", column.height)
		}

		if !near(itemA.y, column.y) || !near(itemB.y, itemA.y+itemHeight) {
			t.Fatalf("definite item y positions = %.2f/%.2f, want %.2f/%.2f",
				itemA.y, itemB.y, column.y, column.y+itemHeight)
		}
	})
}

// TestChromeFlexFactorLessThanOneRowAndColumn covers case
// wpt-flex-factor-less-than-one (layout-unit), source
// third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-factor-less-than-one.html.
// Expected: fractional grow and shrink factors distribute space proportionally
// in both axes. Grow factors .5 and .25 sum below one, so the space they may
// consume is first multiplied by that sum: 40pt of free space over 30pt bases
// gives 50/40, and a 100pt column over zero bases gives 50/25. Shrink factors
// .5 and .25 over two 200pt bases in a 100pt container leave 50/125 because
// the deficit is multiplied by the .75 factor sum before the scaled shrink
// split. TestChromeFlexFractionalGrowFactorsLeaveUnusedSpace already covers
// row grow with zero bases, so this test adds the basis-bearing row case plus
// the column and shrink axes.
//
//nolint:funlen // four fractional-factor axes share one fixture family
func TestChromeFlexFactorLessThanOneRowAndColumn(t *testing.T) {
	t.Parallel()

	t.Run("row-grow-with-basis", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; width: 100pt; height: 20pt }
.item { flex-basis: 30pt; min-width: 0; height: 20pt }
.half { flex-grow: .5 }
.quarter { flex-grow: .25 }
`)
		res := layoutHTML(t, `<html><body>
<div class="case"><div class="item half"></div><div class="item quarter"></div></div>
</body></html>`, cssSheet)

		assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 2), []float64{50, 40})
	})

	t.Run("column-grow", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; flex-direction: column; width: 20pt; height: 100pt }
.item { flex-basis: 0; min-height: 0 }
.half { flex-grow: .5 }
.quarter { flex-grow: .25 }
`)
		res := layoutHTML(t, `<html><body>
<div class="case"><div class="item half"></div><div class="item quarter"></div></div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		items := classBoxes(res.root, "item")

		if len(items) != 2 {
			t.Fatalf("column item boxes = %d, want 2", len(items))
		}

		if !near(items[0].height, 50) || !near(items[1].height, 25) {
			t.Fatalf("column grow heights = %.2f/%.2f, want 50/25", items[0].height, items[1].height)
		}

		if !near(items[0].y, container.y) || !near(items[1].y, container.y+50) {
			t.Fatalf("column grow y = %.2f/%.2f, want %.2f/%.2f",
				items[0].y, items[1].y, container.y, container.y+50)
		}
	})

	t.Run("row-shrink", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; width: 100pt; height: 20pt }
.item { width: 200pt; min-width: 0; height: 20pt }
.half { flex-shrink: .5 }
.quarter { flex-shrink: .25 }
`)
		res := layoutHTML(t, `<html><body>
<div class="case"><div class="item half"></div><div class="item quarter"></div></div>
</body></html>`, cssSheet)

		assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 2), []float64{50, 125})
	})

	t.Run("column-shrink", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; flex-direction: column; width: 20pt; height: 100pt }
.item { height: 200pt; min-height: 0 }
.half { flex-shrink: .5 }
.quarter { flex-shrink: .25 }
`)
		res := layoutHTML(t, `<html><body>
<div class="case"><div class="item half"></div><div class="item quarter"></div></div>
</body></html>`, cssSheet)

		items := classBoxes(res.root, "item")
		if len(items) != 2 {
			t.Fatalf("column item boxes = %d, want 2", len(items))
		}

		if !near(items[0].height, 50) || !near(items[1].height, 125) {
			t.Fatalf("column shrink heights = %.2f/%.2f, want 50/125", items[0].height, items[1].height)
		}
	})
}

// TestChromeFlexBaseSizeIgnoresMaxWidth covers case
// wpt-flex-base-size-max-width (layout-unit), source
// third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-base-size-ignores-max-width.html.
// Expected: the flex base size uses the 300pt content before max-width clamps
// the item, then the frozen item redistributes remaining space. Both bases are
// 300pt in a 300pt container, so the first shrink pass gives each item 150pt;
// the capped item clamps to its 100pt max-width, and the unfrozen item
// re-shrinks from its 300pt base by the 100pt remaining deficit to 200pt. The
// final 100pt green box is exactly half the 200pt blue box.
func TestChromeFlexBaseSizeIgnoresMaxWidth(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.flex { display: flex; width: 300pt }
.item { min-width: 0; height: 50pt }
.capped { max-width: 100pt }
.content { width: 300pt; height: 50pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="flex">
  <div class="item capped"><div class="content"></div></div>
  <div class="item uncapped"><div class="content"></div></div>
</div>
</body></html>`, cssSheet)

	assertFlexChromeWidths(t, flexChromeItemWidths(t, res, 2), []float64{100, 200})

	flex := findBoxByClass(t, res, "flex")
	capped := findBoxByClass(t, res, "capped")
	uncapped := findBoxByClass(t, res, "uncapped")

	if !near(capped.x, flex.x) || !near(uncapped.x, flex.x+100) {
		t.Fatalf("item x positions = %.2f/%.2f, want %.2f/%.2f",
			capped.x, uncapped.x, flex.x, flex.x+100)
	}
}
