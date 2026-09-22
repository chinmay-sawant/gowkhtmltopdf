package convert

import "testing"

// TestOutputFixture57VanguardTelemetryAudit checks the masthead, the full
// implemented-property needle, and the final probe gallery heading.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture57VanguardTelemetryAudit(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-57-vanguard-telemetry-audit.pdf")
	fresh := freshOps(t, "fixture-57-vanguard-telemetry-audit.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "VANGUARD-IV ARCHITECTURE",
		168.596, 795.876, 8.33, "LiberationSans", [3]float64{9.0 / 255, 168.0 / 255, 225.0 / 255})
	assertOpsTextRun(t, committed, 3, "NEEDLE-MARK: VANGUARD-CSS-356-IMPLEMENTED",
		34.016, 732.980, 7.84, "LiberationSans-Regular", [3]float64{84.0 / 255, 99.0 / 255, 120.0 / 255})
	assertOpsTextRun(t, committed, 9, "Implemented CSS probe gallery 6/6 (356 properties)",
		34.016, 795.089, 11.77, "LiberationSans", [3]float64{25.0 / 255, 44.0 / 255, 87.0 / 255})

	assertFixtureOpsShape(t, committed, 9, 924, 3198, 949, 413, [4]float64{0, 0, 595.28, 841.89})
}
