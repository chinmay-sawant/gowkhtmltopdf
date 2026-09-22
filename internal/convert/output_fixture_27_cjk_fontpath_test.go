package convert

import "testing"

// TestOutputFixture27CJKFontpath checks the CJK heading and the Latin fallback
// run resolved from the configured font paths.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture27CJKFontpath(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-27-cjk-fontpath.pdf")
	fresh := freshOps(t, "fixture-27-cjk-fontpath.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{17.0 / 255, 17.0 / 255, 17.0 / 255}
	assertOpsTextRun(t, committed, 1, "Hello",
		160.346, 743.327, 14, "NotoSansKR-HangulSubset", dark)
	assertOpsTextRun(t, committed, 1, "mixed",
		198.395, 743.327, 14, "NotoSansKR-HangulSubset", dark)

	assertFixtureOpsShape(t, committed, 1, 24, 0, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
