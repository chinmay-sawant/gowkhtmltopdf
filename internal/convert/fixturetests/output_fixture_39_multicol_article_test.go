package fixturetests

import "testing"

// TestOutputFixture39MulticolArticle checks the article heading, a middle
// column continuation, and the last continuation marker.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture39MulticolArticle(t *testing.T) {
	committed := readCommittedOps(t, "fixture-39-multicol-article.pdf")
	fresh := freshOps(t, "fixture-39-multicol-article.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	body := [3]float64{28.0 / 255, 28.0 / 255, 28.0 / 255}

	assertOpsTextRun(t, committed, 1, "Multicol report article",
		28.346, 798.397, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 2, "yankee zulu MC-C22.",
		28.346, 461.694, 11, "LiberationSans-Regular", body)
	assertOpsTextRun(t, committed, 3, "yankee zulu MC-C39.",
		305.640, 634.449, 11, "LiberationSans-Regular", body)

	assertFixtureOpsShape(t, committed, 3, 215, 0, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
