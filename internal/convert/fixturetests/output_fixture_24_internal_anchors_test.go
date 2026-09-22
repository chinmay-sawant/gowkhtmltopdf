package fixturetests

import "testing"

// TestOutputFixture24InternalAnchors checks both the source heading and the
// destination heading reached by the internal fragment links.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture24InternalAnchors(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-24-internal-anchors.pdf")
	fresh := freshOps(t, "fixture-24-internal-anchors.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	assertOpsTextRun(t, committed, 1, "Internal link report",
		34.346, 778.443, 18, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 2, "Appendix",
		34.346, 784.228, 16.5, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 2, "Secondary destination used by the second fragment link.",
		34.346, 695.550, 11, "LiberationSans-Regular", dark)

	assertFixtureOpsShape(t, committed, 2, 10, 2, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
