//nolint:dupl // fixture contracts intentionally keep their measured assertions local
package convert

import "testing"

// TestOutputFixture15BulletedRequirements checks the requirement identifiers,
// nested bullet content, and the acceptance table header.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title "Requirements - Ordering API v2" #1a3d6d at (28.346, 797.843), 15pt bold
//	italic label "Idempotency:" #222222 at (92.856, 616.327), 10pt italic
//	table header "Criteria" #222222 at (33.846, 443.777), 10pt bold
//	table header fill #e8eef5 at (28.346, 436.244), 181.601x22.000
//	table border #b9c6d2, 1pt, (28.346, 458.244) to (209.947, 458.244)
//	1 page, A4, 49 text runs, 27 strokes, 2 fills, no images
func TestOutputFixture15BulletedRequirements(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-15-bulleted-requirements.pdf")
	fresh := freshOps(t, "fixture-15-bulleted-requirements.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	border := [3]float64{185.0 / 255, 198.0 / 255, 210.0 / 255}

	assertOpsTextRun(t, committed, 1, "Requirements - Ordering API v2",
		28.346, 797.843, 15, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Idempotency:",
		92.856, 616.327, 10, "LiberationSans-Italic", bodyColor)
	assertOpsTextRun(t, committed, 1, "Criteria",
		33.846, 443.777, 10, "LiberationSans-Bold", bodyColor)
	assertOpsFillRect(t, committed, 1,
		28.346, 436.244, 181.601, 22.000, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 458.244, 209.947, 458.244, border, 1)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 49 {
		t.Errorf("text runs = %d, want 49", len(committed.Texts))
	}

	if len(committed.Strokes) != 27 {
		t.Errorf("strokes = %d, want 27", len(committed.Strokes))
	}

	if len(committed.Fills) != 2 {
		t.Errorf("fills = %d, want 2", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
