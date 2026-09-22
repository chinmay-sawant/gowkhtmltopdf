package fixturetests

import "testing"

// TestOutputFixture55LanternCooperativeReport checks the report masthead, the
// final-page route-sheet callout, and the final page counter.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture55LanternCooperativeReport(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-55-lantern-cooperative-report.pdf")
	fresh := freshOps(t, "fixture-55-lantern-cooperative-report.html")

	assertPageOpsMatch(t, committed, fresh)

	green := [3]float64{23.0 / 255, 63.0 / 255, 69.0 / 255}
	muted := [3]float64{135.0 / 255, 146.0 / 255, 141.0 / 255}

	assertOpsTextRun(t, committed, 1, "NORTHLINE COOPERATIVE",
		73.107, 784.557, 11, "LiberationSans-Bold", green)
	assertOpsTextRun(t, committed, 3, "PUBLISH THE NEXT ROUTE SHEET",
		77.607, 469.313, 10, "LiberationSans-Bold", green)
	assertOpsTextRun(t, committed, 3, "03 / 03",
		524.363, 28.349, 7.5, "LiberationSans-Regular", muted)

	assertFixtureOpsShape(t, committed, 3, 118, 164, 39, 0, [4]float64{0, 0, 595.28, 841.89})
}
