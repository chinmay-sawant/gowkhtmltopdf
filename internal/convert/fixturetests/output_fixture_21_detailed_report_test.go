package fixturetests

import "testing"

// TestOutputFixture21DetailedReport checks the report letterhead, KPI banner,
// and the final requirements section across the four-page A4 sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture21DetailedReport(t *testing.T) {

	committed := readCommittedOps(t, "fixture-21-detailed-report.pdf")
	fresh := freshOps(t, "fixture-21-detailed-report.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	darkBlue := [3]float64{15.0 / 255, 58.0 / 255, 95.0 / 255}

	assertOpsTextRun(t, committed, 1, "Nordwind Industries GmbH",
		28.346, 795.750, 17, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 3, "4. Commercial summary (invoice extract)",
		28.346, 787.343, 15, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 4, "5. Platform requirements status",
		28.346, 787.343, 15, "LiberationSans-Bold", blue)
	assertOpsFillRect(t, committed, 1,
		28.346, 712.444, 538.587, 45.900, darkBlue)
	assertOpsTextRun(t, committed, 4, "REQ-07",
		47.846, 538.900, 9.5, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 4, "6. Risks, mitigations, and outlook",
		28.346, 374.743, 15, "LiberationSans-Bold", blue)

	assertFixtureOpsShape(t, committed, 4, 267, 411, 118, 0, [4]float64{0, 0, 595.28, 841.89})
}
