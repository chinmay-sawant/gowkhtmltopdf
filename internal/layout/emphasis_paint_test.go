package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
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


func TestTextDecorationSkipInkGapsDescenders(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 16pt; }
.auto { text-decoration: underline; text-decoration-skip-ink: auto; }
.none { text-decoration: underline; text-decoration-skip-ink: none; }
`)
	root, err := html.Parse(`<html><body>` +
		`<span class="auto">paying</span> <span class="none">paying</span>` +
		`</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []Op
	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 0.2 {
			lines = append(lines, op)
			t.Logf("line x=%.1f w=%.1f", op.X, op.W)
		}
	}
	if len(lines) < 2 {
		t.Fatalf("want ≥2 underline segments (gapped auto + solid none), got %d", len(lines))
	}
	// auto on "paying" gaps at p and y → several short segments; none → one long stroke
	var long int
	for _, op := range lines {
		if op.W > 30 {
			long++
		}
	}
	if long < 1 {
		t.Fatalf("expected a long continuous underline for skip-ink:none, lines=%d", len(lines))
	}
	if len(lines) < 3 {
		t.Fatalf("expected auto to split underline around descenders, got %d segments", len(lines))
	}
}

