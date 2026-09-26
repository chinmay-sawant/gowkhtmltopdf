package fixturetests

import "testing"

// TestOutputFixture29WPTBreakNestedFloatPrint checks the semantic label and
// the green float fragment on the three custom-sized pages.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture29WPTBreakNestedFloatPrint(t *testing.T) {
	committed := readCommittedOps(t, "fixture-29-wpt-break-nested-float-print.pdf")
	fresh := freshOps(t, "fixture-29-wpt-break-nested-float-print.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Case 29 nested float fragmentation",
		38.000, 170.427, 8, "LiberationSans", [3]float64{0, 0, 0})
	assertOpsFillRect(t, committed, 1,
		36.000, 36.000, 144.000, 144.000,
		[3]float64{22.0 / 255, 163.0 / 255, 74.0 / 255})

	assertFixtureOpsShape(t, committed, 3, 1, 0, 3, 0, [4]float64{0, 0, 360, 216})
}
