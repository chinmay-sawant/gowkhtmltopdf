package fixturetests

import "testing"

// TestOutputFixture43ComplexDossier checks the first-page readiness text, the
// fifth-page release decision, and a dossier image placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture43ComplexDossier(t *testing.T) {
	committed := readCommittedOps(t, "fixture-43-complex-dossier.pdf")
	fresh := freshOps(t, "fixture-43-complex-dossier.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Launch candidate",
		55.850, 620.132, 7.5, "LiberationSans-Bold", [3]float64{33.0 / 255, 100.0 / 255, 63.0 / 255})
	assertOpsTextRun(t, committed, 5, "Controlled rollout authorization",
		36.850, 734.828, 14, "LiberationSans-Bold", [3]float64{18.0 / 255, 59.0 / 255, 93.0 / 255})
	assertOpsImageBox(t, committed, 1, 412.430, 641.675, 132, 87.057)

	assertFixtureOpsShape(t, committed, 5, 372, 361, 118, 6, [4]float64{0, 0, 595.28, 841.89})
}
