//nolint:dupl // fixture contracts intentionally keep their measured assertions local
package fixturetests

import "testing"

// TestOutputFixture64Next72Props checks the masthead, the fixture needle, two
// later property rows, and the first local image placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture64Next72Props(t *testing.T) {
	committed := readCommittedOps(t, "fixture-64-next-72-props.pdf")
	fresh := freshOps(t, "fixture-64-next-72-props.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{31.0 / 255, 75.0 / 255, 153.0 / 255}
	assertOpsTextRun(t, committed, 1, "font-feature-settings to counter-set",
		202.016, 790.815, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "NEXT-72-PROPS",
		40.016, 751.668, 9.5, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 8, "aspect-ratio",
		67.266, 770.193, 8.5, "LiberationMono-Regular", [3]float64{16.0 / 255, 58.0 / 255, 122.0 / 255})
	assertOpsTextRun(t, committed, 8, "counter-set",
		67.266, 724.262, 8.5, "LiberationMono-Regular", [3]float64{16.0 / 255, 58.0 / 255, 122.0 / 255})
	assertOpsImageBox(t, committed, 1, 34.016, 776.374, 90, 27)

	assertFixtureOpsShape(t, committed, 8, 1003, 2206, 302, 6, [4]float64{0, 0, 595.28, 841.89})
}
