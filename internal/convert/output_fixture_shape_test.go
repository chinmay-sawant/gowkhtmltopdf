package convert

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func assertFixtureOpsShape(
	t *testing.T,
	ops *pdf.PageOps,
	pages, texts, strokes, fills, images int,
	mediaBox [4]float64,
) {
	t.Helper()

	if ops.Pages != pages || len(ops.MediaBoxes) != pages {
		t.Errorf("pages = %d, mediaboxes = %d, want %d pages and mediaboxes", ops.Pages, len(ops.MediaBoxes), pages)
	}

	for index, box := range ops.MediaBoxes {
		if !opsBoxClose(box, mediaBox) {
			t.Errorf("mediabox[%d] = %v, want %v", index, box, mediaBox)
		}
	}

	if len(ops.Texts) != texts {
		t.Errorf("text runs = %d, want %d", len(ops.Texts), texts)
	}

	if len(ops.Strokes) != strokes {
		t.Errorf("strokes = %d, want %d", len(ops.Strokes), strokes)
	}

	if len(ops.Fills) != fills {
		t.Errorf("fills = %d, want %d", len(ops.Fills), fills)
	}

	if len(ops.Images) != images {
		t.Errorf("images = %d, want %d", len(ops.Images), images)
	}
}
