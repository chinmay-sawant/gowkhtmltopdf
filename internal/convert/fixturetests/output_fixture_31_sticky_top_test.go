package fixturetests

import "testing"

// TestOutputFixture31StickyTop checks the sticky bar in its authored fragment
// and the non-repeated trailing content on page two.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture31StickyTop(t *testing.T) {

	committed := readCommittedOps(t, "fixture-31-sticky-top.pdf")
	fresh := freshOps(t, "fixture-31-sticky-top.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Sticky print scrollport",
		44.346, 782.397, 16, "LiberationSans-Bold", black)
	assertOpsTextRun(t, committed, 1, "Section header (sticky top:0)",
		55.346, 746.930, 11, "LiberationSans-Bold", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 2, "After the section: sticky must not replicate here like position:fixed.",
		53.346, 525.080, 11, "LiberationSans-Regular", black)

	assertFixtureOpsShape(t, committed, 2, 108, 332, 41, 0, [4]float64{0, 0, 595.28, 841.89})
}
