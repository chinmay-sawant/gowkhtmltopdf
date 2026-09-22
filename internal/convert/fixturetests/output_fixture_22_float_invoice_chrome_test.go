package fixturetests

import "testing"

// TestOutputFixture22FloatInvoiceChrome checks the floated letterhead, badge,
// and bordered payment note in the one-page A4 sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture22FloatInvoiceChrome(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-22-float-invoice-chrome.pdf")
	fresh := freshOps(t, "fixture-22-float-invoice-chrome.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}

	assertOpsTextRun(t, committed, 1, "ACME",
		58.901, 792.077, 10, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 1, "INV-22-0042",
		504.564, 803.130, 11, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "OPEN",
		209.956, 748.970, 8, "LiberationSans-Regular", blue)
	assertOpsFillRect(t, committed, 1,
		28.346, 777.544, 90.000, 36.000, blue)
	assertOpsTextRun(t, committed, 1, "Payment terms: Net 30. Float chrome + inline-block badge + border-box note.",
		37.346, 645.444, 10, "LiberationSans-Regular", [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 19, 40, 6, 0, [4]float64{0, 0, 595.28, 841.89})
}
