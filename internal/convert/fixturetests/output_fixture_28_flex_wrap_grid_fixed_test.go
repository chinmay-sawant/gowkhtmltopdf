package fixturetests

import "testing"

// TestOutputFixture28FlexWrapGridFixed checks the first-page heading and the
// second-page body while the fixed header remains present on both pages.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture28FlexWrapGridFixed(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-28-flex-wrap-grid-fixed.pdf")
	fresh := freshOps(t, "fixture-28-flex-wrap-grid-fixed.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Wrap + grid",
		40.346, 769.210, 20, "LiberationSans-Bold", black)
	assertOpsTextRun(t, committed, 2, "Page two body ",
		40.346, 794.077, 10, "LiberationSans-Regular", black)
	assertOpsTextRun(t, committed, 2, "FIXED",
		485.934, 798.970, 8, "LiberationSans-Regular", [3]float64{1, 1, 1})

	assertFixtureOpsShape(t, committed, 2, 14, 8, 10, 0, [4]float64{0, 0, 595.28, 841.89})
}
