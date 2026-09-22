package convert

import "testing"

// TestOutputFixture06ExternalLink checks link text styling and the surrounding
// table geometry. URI annotation presence remains covered by golden tests.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title  "Partner Handbook" #0f3a5f at (28.346, 795.750), 17pt bold
//	link   "example.com/partners/register" #0000ee at (137.853, 720.377), 10pt regular
//	section "Resources" #0f3a5f at (28.346, 679.783), 12pt bold
//	footer #8a94a0 at (28.346, 483.770), 8pt regular
//	header #e8eef5 at (28.346, 648.944), 218.996x20.600
//	rule   #c9d4de, 1pt, (28.346, 669.544) to (247.342, 669.544)
//	1 page, A4, 23 text runs, 34 strokes, 2 fills, no images
func TestOutputFixture06ExternalLink(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-06-external-link.pdf")
	fresh := freshOps(t, "fixture-06-external-link.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{15.0 / 255, 58.0 / 255, 95.0 / 255}
	linkBlue := [3]float64{0, 0, 238.0 / 255}
	muted := [3]float64{138.0 / 255, 148.0 / 255, 160.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	border := [3]float64{201.0 / 255, 212.0 / 255, 222.0 / 255}

	assertOpsTextRun(t, committed, 1, "Partner Handbook",
		28.346, 795.750, 17, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "example.com/partners/register",
		137.853, 720.377, 10, "LiberationSans-Regular", linkBlue)
	assertOpsTextRun(t, committed, 1, "Resources",
		28.346, 679.783, 12, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1,
		"External URIs render as PDF link annotations; the engine never fetches the link targets.",
		28.346, 483.770, 8, "LiberationSans-Regular", muted)

	assertOpsFillRect(t, committed, 1, 28.346, 648.944, 218.996, 20.600, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 669.544, 247.342, 669.544, border, 1)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1", committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 23 {
		t.Errorf("text runs = %d, want 23", len(committed.Texts))
	}

	if len(committed.Strokes) != 34 {
		t.Errorf("strokes = %d, want 34", len(committed.Strokes))
	}

	if len(committed.Fills) != 2 {
		t.Errorf("fills = %d, want 2", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
