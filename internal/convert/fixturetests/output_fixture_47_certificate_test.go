package fixturetests

import "testing"

// TestOutputFixture47Certificate checks the institute heading, certificate
// title, issue metadata, and the full-page background image.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture47Certificate(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-47-certificate.pdf")
	fresh := freshOps(t, "fixture-47-certificate.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "NORTHSTAR LEARNING INSTITUTE",
		155.504, 749.045, 16.18, "LiberationSans-Bold", [3]float64{138.0 / 255, 96.0 / 255, 26.0 / 255})
	assertOpsTextRun(t, committed, 1, "Certificate of Completion",
		86.154, 636.206, 34.16, "LiberationSans-Bold", [3]float64{36.0 / 255, 52.0 / 255, 71.0 / 255})
	assertOpsTextRun(t, committed, 1, "Amara Okafor",
		208.125, 546.480, 28.76, "LiberationSans-Regular", [3]float64{36.0 / 255, 52.0 / 255, 71.0 / 255})
	assertOpsImageBox(t, committed, 1, -22.576, -67.842, 657.565, 931.206)

	assertFixtureOpsShape(t, committed, 1, 16, 12, 0, 1, [4]float64{0, 0, 595.28, 841.89})
}
