package fixturetests

import "testing"

// TestOutputFixture30OrphansHeuristic checks the heading/body boundary and
// the final continuation marker across the three-page sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture30OrphansHeuristic(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-30-orphans-heuristic.pdf")
	fresh := freshOps(t, "fixture-30-orphans-heuristic.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	body := [3]float64{28.0 / 255, 28.0 / 255, 28.0 / 255}

	assertOpsTextRun(t, committed, 1, "Orphans and keep-with-next sample",
		28.346, 798.397, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 2, "ORPHAN-HEADING-BODY",
		28.346, 360.872, 11, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 3, "Trailing page-two prose. Marker ORPHAN-TRAIL-05.",
		28.346, 670.877, 10, "LiberationSans-Regular", body)

	assertFixtureOpsShape(t, committed, 3, 82, 0, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
