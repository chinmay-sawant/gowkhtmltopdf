package fixturetests

import "testing"

// TestOutputFixture58UnsupportedWorklistAudit checks the first and last
// unsupported-property gallery headings across the nine-page sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture58UnsupportedWorklistAudit(t *testing.T) {
	committed := readCommittedOps(t, "fixture-58-unsupported-worklist-audit.pdf")
	fresh := freshOps(t, "fixture-58-unsupported-worklist-audit.html")

	assertPageOpsMatch(t, committed, fresh)

	gallery := [3]float64{25.0 / 255, 44.0 / 255, 87.0 / 255}
	assertOpsTextRun(t, committed, 2, "Unsupported CSS probe gallery 1/8 (462 properties)",
		34.016, 775.602, 11.77, "LiberationSans", gallery)
	assertOpsTextRun(t, committed, 9, "Unsupported CSS probe gallery 8/8 (462 properties)",
		34.016, 795.089, 11.77, "LiberationSans", gallery)

	assertFixtureOpsShape(t, committed, 9, 995, 3728, 969, 216, [4]float64{0, 0, 595.28, 841.89})
}
