package convert

import "testing"

// TestOutputFixture12Lists checks nested list indentation and the ordered-list
// marker emitted by the current list renderer.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title "Project Kickoff Checklist - Merger Integration" #1a3d6d at (28.346, 799.343), 15pt bold
//	nested item "IT landscape inventory (owners, SLAs, sunset dates)." #222222 at (70.346, 714.663), 9.5pt regular
//	ordered marker "1." #222222 at (37.507, 601.502), 10pt regular
//	1 page, A4, 50 text runs, no strokes, no fills, no images
func TestOutputFixture12Lists(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-12-lists.pdf")
	fresh := freshOps(t, "fixture-12-lists.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}

	assertOpsTextRun(t, committed, 1, "Project Kickoff Checklist - Merger Integration",
		28.346, 799.343, 15, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1,
		"IT landscape inventory (owners, SLAs, sunset dates).",
		70.346, 714.663, 9.5, "LiberationSans-Regular", bodyColor)
	assertOpsTextRun(t, committed, 1, "1.",
		37.507, 601.502, 10, "LiberationSans-Regular", bodyColor)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 50 {
		t.Errorf("text runs = %d, want 50", len(committed.Texts))
	}

	if len(committed.Strokes) != 0 {
		t.Errorf("strokes = %d, want 0", len(committed.Strokes))
	}

	if len(committed.Fills) != 0 {
		t.Errorf("fills = %d, want 0", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
