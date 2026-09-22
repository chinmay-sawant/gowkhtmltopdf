package fixturetests

import "testing"

// TestOutputFixture59ApexDigitalLanding checks the first-page hero, the
// pricing section, the final navigation item, and the hero image placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture59ApexDigitalLanding(t *testing.T) {

	committed := readCommittedOps(t, "fixture-59-apex-digital-landing.pdf")
	fresh := freshOps(t, "fixture-59-apex-digital-landing.html")

	assertPageOpsMatch(t, committed, fresh)

	ink := [3]float64{15.0 / 255, 23.0 / 255, 42.0 / 255}
	assertOpsTextRun(t, committed, 1, "Design and Build",
		71.433, 654.093, 30, "LiberationSans-Bold", ink)
	assertOpsTextRun(t, committed, 7, "Transparent Pricing",
		168.902, 769.365, 27, "LiberationSans-Bold", ink)
	assertOpsTextRun(t, committed, 9, "Home",
		162.611, 744.751, 12, "LiberationSans-Regular", [3]float64{148.0 / 255, 163.0 / 255, 184.0 / 255})
	assertOpsImageBox(t, committed, 1, 28.346, 36.044, 538.587, 717.700)

	assertFixtureOpsShape(t, committed, 9, 66, 57, 135, 9, [4]float64{0, 0, 595.28, 841.89})
}
