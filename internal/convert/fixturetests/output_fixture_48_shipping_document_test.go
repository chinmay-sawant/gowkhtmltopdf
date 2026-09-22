package fixturetests

import "testing"

// TestOutputFixture48ShippingDocument checks the document heading and the
// final delivery instruction in the one-page shipping sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture48ShippingDocument(t *testing.T) {

	committed := readCommittedOps(t, "fixture-48-shipping-document.pdf")
	fresh := freshOps(t, "fixture-48-shipping-document.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Packing List and Shipping Document",
		28.346, 796.503, 18, "LiberationSans-Bold", [3]float64{18.0 / 255, 79.0 / 255, 98.0 / 255})
	assertOpsTextRun(t, committed, 1, "signing delivery.",
		28.346, 528.273, 9, "LiberationSans-Regular", [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255})
	assertOpsStrokeSegment(t, committed, 1,
		29.471, 768.769, 565.809, 768.769,
		[3]float64{18.0 / 255, 79.0 / 255, 98.0 / 255}, 2.25)

	assertFixtureOpsShape(t, committed, 1, 37, 359, 5, 0, [4]float64{0, 0, 595.28, 841.89})
}
