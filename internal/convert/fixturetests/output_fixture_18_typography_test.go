package fixturetests

import "testing"

// TestOutputFixture18Typography checks heading sizing, inline italic text,
// small text, and the blockquote border.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	h1 "Heading level 1 - document title" #1a3d6d at (28.346, 794.253), 18pt bold
//	italic run "italic via em" #222222 at (209.494, 650.677), 10pt italic
//	small run "small text" #222222 at (144.543, 636.177), 8.33pt regular
//	blockquote border #1a3d6d, 2.25pt, (29.471, 546.713) to (29.471, 573.463)
//	1 page, A4, 35 low-level text runs, 5 strokes, no fills, no images
func TestOutputFixture18Typography(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-18-typography.pdf")
	fresh := freshOps(t, "fixture-18-typography.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}

	assertOpsTextRun(t, committed, 1, "Heading level 1 - document title",
		28.346, 794.253, 18, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "italic via em",
		209.494, 650.677, 10, "LiberationSans-Italic", bodyColor)
	assertOpsTextRun(t, committed, 1, "small text",
		144.543, 636.177, 8.33, "LiberationSans-Regular", bodyColor)
	assertOpsStrokeSegment(t, committed, 1,
		29.471, 573.463, 29.471, 546.713, headingBlue, 2.25)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 35 {
		t.Errorf("text runs = %d, want 35", len(committed.Texts))
	}

	if len(committed.Strokes) != 5 {
		t.Errorf("strokes = %d, want 5", len(committed.Strokes))
	}

	if len(committed.Fills) != 0 {
		t.Errorf("fills = %d, want 0", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
