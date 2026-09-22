package fixturetests

import "testing"

// TestOutputFixture20ImageGrid checks the table heading, a caption, and all
// four intrinsic-size image placements.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title "Product color palette - swatch grid" #1a3d6d at (28.346, 799.343), 15pt bold
//	palette heading "Approved palette" #0f3a5f at (258.842, 773.550), 9.5pt bold
//	closing line "Every swatch above is a data: URI PNG decoded by the loader;
//	none of them fetches from the network. The grid cell width is controlled by
//	the" #555555 at (28.346, 560.197), 8.5pt regular
//	images page 1: (147.925, 717.644) 24x24, (411.219, 711.644) 36x36,
//	(135.925, 611.944) 48x48, and (393.219, 617.944) 72x36
//	1 page, A4, 8 text runs, 16 strokes, 1 fill, 4 images
func TestOutputFixture20ImageGrid(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-20-image-grid.pdf")
	fresh := freshOps(t, "fixture-20-image-grid.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	paletteBlue := [3]float64{15.0 / 255, 58.0 / 255, 95.0 / 255}
	captionColor := [3]float64{85.0 / 255, 85.0 / 255, 85.0 / 255}

	assertOpsTextRun(t, committed, 1, "Product color palette - swatch grid",
		28.346, 799.343, 15, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Approved palette",
		258.842, 773.550, 9.5, "LiberationSans-Bold", paletteBlue)
	assertOpsTextRun(t, committed, 1,
		"Every swatch above is a data: URI PNG decoded by the loader; "+
			"none of them fetches from the network. The grid cell width is controlled by the",
		28.346, 560.197, 8.5, "LiberationSans-Regular", captionColor)

	assertOpsImageBox(t, committed, 1, 147.925, 717.644, 24, 24)
	assertOpsImageBox(t, committed, 1, 411.219, 711.644, 36, 36)
	assertOpsImageBox(t, committed, 1, 135.925, 611.944, 48, 48)
	assertOpsImageBox(t, committed, 1, 393.219, 617.944, 72, 36)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 8 {
		t.Errorf("text runs = %d, want 8", len(committed.Texts))
	}

	if len(committed.Strokes) != 16 {
		t.Errorf("strokes = %d, want 16", len(committed.Strokes))
	}

	if len(committed.Fills) != 1 {
		t.Errorf("fills = %d, want 1", len(committed.Fills))
	}

	if len(committed.Images) != 4 {
		t.Errorf("images = %d, want 4", len(committed.Images))
	}
}
