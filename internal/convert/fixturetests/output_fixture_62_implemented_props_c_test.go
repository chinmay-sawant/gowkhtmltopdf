//nolint:dupl // fixture contracts intentionally keep their measured assertions local
package fixturetests

import "testing"

// TestOutputFixture62ImplementedPropsC checks the masthead, the fixture
// needle, two later property effects, and the first local image placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture62ImplementedPropsC(t *testing.T) {

	committed := readCommittedOps(t, "fixture-62-implemented-props-c.pdf")
	fresh := freshOps(t, "fixture-62-implemented-props-c.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{31.0 / 255, 75.0 / 255, 153.0 / 255}
	assertOpsTextRun(t, committed, 1, "Implemented CSS audit C",
		202.016, 790.815, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "IMPLEMENTED-PROPS-C",
		40.016, 751.668, 9.5, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 2, "object-view-box",
		67.266, 726.475, 8.5, "LiberationMono-Regular", [3]float64{16.0 / 255, 58.0 / 255, 122.0 / 255})
	assertOpsTextRun(t, committed, 8, "front",
		433.265, 331.943, 9.5, "LiberationSans-Regular", [3]float64{26.0 / 255, 31.0 / 255, 36.0 / 255})
	assertOpsImageBox(t, committed, 1, 34.016, 776.374, 90, 27)

	assertFixtureOpsShape(t, committed, 8, 890, 4478, 500, 7, [4]float64{0, 0, 595.28, 841.89})
}
