package convert

import "testing"

// TestOutputFixture34GridAreasDense checks named-area placement and dense
// auto-flow headings in the one-page sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture34GridAreasDense(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-34-grid-areas-dense.pdf")
	fresh := freshOps(t, "fixture-34-grid-areas-dense.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	label := [3]float64{51.0 / 255, 51.0 / 255, 51.0 / 255}

	assertOpsTextRun(t, committed, 1, "Grid areas + dense auto-flow",
		38.346, 790.290, 14, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 1, "Named template areas",
		38.346, 766.330, 11, "LiberationSans-Bold", label)
	assertOpsTextRun(t, committed, 1, "Dense auto-flow",
		38.346, 557.930, 11, "LiberationSans-Bold", label)
	assertOpsFillRect(t, committed, 1,
		38.346, 647.944, 280.000, 100.000,
		[3]float64{227.0 / 255, 242.0 / 255, 253.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 21, 52, 13, 0, [4]float64{0, 0, 595.28, 841.89})
}
