package fixturetests

import "testing"

// TestOutputFixture13PreCodeBlock checks monospace pre content, inline code,
// and the shaded bordered code blocks.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title "Configuration Reference" #1a3d6d at (28.346, 799.343), 15pt bold
//	pre line "  tls:" #222222 at (36.846, 706.813), 8.5pt monospace
//	inline code "pre" #7a2c1e at (267.370, 780.577), 10pt monospace
//	first pre fill #f4f4f2 at (28.346, 558.638), 538.587x197.600
//	first pre border #d8d8d4, 0.75pt, (28.346, 756.238) to (566.934, 756.238)
//	1 page, A4, 39 text runs, 12 strokes, 3 fills, no images
func TestOutputFixture13PreCodeBlock(t *testing.T) {

	committed := readCommittedOps(t, "fixture-13-pre-code-block.pdf")
	fresh := freshOps(t, "fixture-13-pre-code-block.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	codeColor := [3]float64{122.0 / 255, 44.0 / 255, 30.0 / 255}
	preFill := [3]float64{244.0 / 255, 244.0 / 255, 242.0 / 255}
	preBorder := [3]float64{216.0 / 255, 216.0 / 255, 212.0 / 255}

	assertOpsTextRun(t, committed, 1, "Configuration Reference",
		28.346, 799.343, 15, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "  tls:",
		36.846, 706.813, 8.5, "LiberationMono-Regular", bodyColor)
	assertOpsTextRun(t, committed, 1, "pre",
		267.370, 780.577, 10, "LiberationMono-Regular", codeColor)
	assertOpsFillRect(t, committed, 1,
		28.346, 558.638, 538.587, 197.600, preFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 756.238, 566.934, 756.238, preBorder, 0.75)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 39 {
		t.Errorf("text runs = %d, want 39", len(committed.Texts))
	}

	if len(committed.Strokes) != 12 {
		t.Errorf("strokes = %d, want 12", len(committed.Strokes))
	}

	if len(committed.Fills) != 3 {
		t.Errorf("fills = %d, want 3", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
