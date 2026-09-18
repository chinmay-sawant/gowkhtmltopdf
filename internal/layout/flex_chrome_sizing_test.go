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
