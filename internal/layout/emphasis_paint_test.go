package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestTextEmphasisPaintsDots(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body style="font-size:14pt;">` +
		`<span style="text-emphasis:filled dot;text-emphasis-color:#c00;">ABCDE</span>` +
		`</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	var smallFills int
	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.W > 0 && op.W < 12 && op.H > 0 && op.H < 12 {
			smallFills++
			t.Logf("dot fill %.2fx%.2f rgb=(%.2f,%.2f,%.2f) at (%.1f,%.1f)",
				op.W, op.H, op.R, op.G, op.B, op.X, op.Y)
		}
	}
	if smallFills < 3 {
		t.Fatalf("expected ≥3 emphasis dots for ABCDE, got %d small fills (total ops=%d)",
			smallFills, len(res.Ops))
	}
}
