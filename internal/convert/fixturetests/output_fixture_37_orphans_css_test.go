package fixturetests

import "testing"

// TestOutputFixture37OrphansCSS checks the CSS keep block and the final
// forced continuation marker across the three-page sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture37OrphansCSS(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-37-orphans-css.pdf")
	fresh := freshOps(t, "fixture-37-orphans-css.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	body := [3]float64{28.0 / 255, 28.0 / 255, 28.0 / 255}

	assertOpsTextRun(t, committed, 1, "CSS orphans and widows",
		28.346, 798.397, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 2, "OW-CSS-KEEP-BLOCK",
		54.638, 775.594, 11, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 3, "Trailing page-two prose. Marker OW-CSS-TRAIL-03.",
		28.346, 709.827, 10, "LiberationSans-Regular", body)

	assertFixtureOpsShape(t, committed, 3, 49, 0, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
