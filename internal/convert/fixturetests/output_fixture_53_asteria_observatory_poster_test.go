package fixturetests

import "testing"

// TestOutputFixture53AsteriaObservatoryPoster checks the poster title, archive
// year, and the local illustration placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture53AsteriaObservatoryPoster(t *testing.T) {
	committed := readCommittedOps(t, "fixture-53-asteria-observatory-poster.pdf")
	fresh := freshOps(t, "fixture-53-asteria-observatory-poster.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Asteria Sky Archive",
		65.197, 183.862, 32, "LiberationSans-Bold", [3]float64{251.0 / 255, 244.0 / 255, 230.0 / 255})
	assertOpsTextRun(t, committed, 1, "2026",
		508.480, 57.530, 8, "LiberationMono-Regular", [3]float64{183.0 / 255, 200.0 / 255, 202.0 / 255})
	assertOpsImageBox(t, committed, 1, 28.346, 261.281, 538.587, 552.262)

	assertFixtureOpsShape(t, committed, 1, 7, 1, 3, 1, [4]float64{0, 0, 595.28, 841.89})
}
