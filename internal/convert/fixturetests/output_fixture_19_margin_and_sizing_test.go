package fixturetests

import "testing"

// TestOutputFixture19MarginAndSizing checks the fixed-size box, its fill, and
// the explicit border-box outline.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title "Sizing & Box Model Specimen" #1a3d6d at (28.346, 799.343), 15pt bold
//	fixed box fill #dbe7f3 at (28.346, 729.044), 194.000x59.000
//	border-box rule #1a3d6d, 2.25pt, (28.346, 579.044) to (193.346, 579.044)
//	1 page, A4, 24 text runs, 42 strokes, 4 fills, no images
func TestOutputFixture19MarginAndSizing(t *testing.T) {
	committed := readCommittedOps(t, "fixture-19-margin-and-sizing.pdf")
	fresh := freshOps(t, "fixture-19-margin-and-sizing.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	fixedFill := [3]float64{219.0 / 255, 231.0 / 255, 243.0 / 255}

	assertOpsTextRun(t, committed, 1, "Sizing & Box Model Specimen",
		28.346, 799.343, 15, "LiberationSans-Bold", headingBlue)
	assertOpsFillRect(t, committed, 1,
		28.346, 729.044, 194.000, 59.000, fixedFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 579.044, 193.346, 579.044, headingBlue, 2.25)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 24 {
		t.Errorf("text runs = %d, want 24", len(committed.Texts))
	}

	if len(committed.Strokes) != 42 {
		t.Errorf("strokes = %d, want 42", len(committed.Strokes))
	}

	if len(committed.Fills) != 4 {
		t.Errorf("fills = %d, want 4", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
