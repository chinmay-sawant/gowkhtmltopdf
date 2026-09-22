package convert

import "testing"

// TestOutputFixture17CoverAndContent checks the centered cover and the content
// heading after the forced page break.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	cover title page 1, "Annual Report 2024" #1a3d6d at (194.338, 672.717), 22pt bold
//	content title page 2, "1. Letter to the shareholders" #1a3d6d at (28.346, 787.677), 16pt bold
//	cover rule #1a3d6d, 3pt, (191.423, 630.244) to (403.857, 630.244)
//	2 pages, A4, 21 text runs, 3 strokes, no fills, no images
func TestOutputFixture17CoverAndContent(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-17-cover-and-content.pdf")
	fresh := freshOps(t, "fixture-17-cover-and-content.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}

	assertOpsTextRun(t, committed, 1, "Annual Report 2024",
		194.338, 672.717, 22, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 2, "1. Letter to the shareholders",
		28.346, 787.677, 16, "LiberationSans-Bold", headingBlue)
	assertOpsStrokeSegment(t, committed, 1,
		191.423, 630.244, 403.857, 630.244, headingBlue, 3)

	if committed.Pages != 2 || len(committed.MediaBoxes) != 2 {
		t.Errorf("pages = %d, mediaboxes = %d, want 2 pages and 2 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 21 {
		t.Errorf("text runs = %d, want 21", len(committed.Texts))
	}

	if len(committed.Strokes) != 3 {
		t.Errorf("strokes = %d, want 3", len(committed.Strokes))
	}

	if len(committed.Fills) != 0 {
		t.Errorf("fills = %d, want 0", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
