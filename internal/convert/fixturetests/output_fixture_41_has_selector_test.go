package fixturetests

import "testing"

// TestOutputFixture41HasSelector checks the article border and the warning
// row selected by the relational selector.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture41HasSelector(t *testing.T) {
	committed := readCommittedOps(t, "fixture-41-has-selector.pdf")
	fresh := freshOps(t, "fixture-41-has-selector.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Article with a footnote should show a left border.",
		64.346, 766.183, 12, "LiberationSans-Regular", black)
	assertOpsTextRun(t, committed, 1, "warning cell highlights row",
		246.608, 508.903, 12, "LiberationSans-Regular", black)
	assertOpsStrokeSegment(t, committed, 1,
		54.346, 787.544, 54.346, 738.744,
		[3]float64{34.0 / 255, 68.0 / 255, 102.0 / 255}, 4)

	assertFixtureOpsShape(t, committed, 1, 13, 23, 4, 0, [4]float64{0, 0, 595.28, 841.89})
}
