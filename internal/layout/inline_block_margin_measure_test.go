//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Inline-block shrink-to-fit around a nested margin/padding/border box must
// size to one line of content (fixture-61 margin demos). Omitting the child's
// horizontal padding/border undersizes the outer box and forces a wrap.
func TestInlineBlockNestedMarginBoxOneLine(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; font-family: sans-serif; }
.outer { background: #fce8e8; border: 1px solid #333; display: inline-block; }
.inner { margin: 12px; background: #dfd; border: 1px dashed #080; padding: 4px; }
`)
	root, err := html.Parse(`<html><body>
<div class="outer"><div class="inner">all-sides 12px</div></div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 120, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var texts []Op
	for _, op := range res.Ops {
		if op.Kind == OpText && (strings.Contains(op.Text, "all-sides") || strings.Contains(op.Text, "12px")) {
			texts = append(texts, op)
		}
	}
	if len(texts) == 0 {
		t.Fatal("missing margin demo text")
	}
	y0 := texts[0].Y
	for _, op := range texts[1:] {
		if op.Y > y0+2 {
			t.Fatalf("text wrapped to y=%.1f (first y=%.1f); shrink-to-fit undersized nested margin box", op.Y, y0)
		}
	}

	// Pink outer fill must be wider than green inner by ~2*12px margin (+ borders).
	var pink, green *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind != OpFillRect {
			continue
		}
		r, g, b := op.R, op.G, op.B
		if r > 0.95 && g > 0.85 && g < 0.95 && b > 0.85 {
			if pink == nil || op.W*op.H > pink.W*pink.H {
				pink = op
			}
		}
		if r > 0.8 && r < 0.92 && g > 0.95 && b > 0.8 {
			if green == nil || op.W < 200 {
				green = op
			}
		}
	}
	if pink == nil || green == nil {
		t.Fatalf("missing pink/green fills (pink=%v green=%v)", pink != nil, green != nil)
	}
	ml := green.X - pink.X
	if ml < 8 || ml > 14 {
		t.Fatalf("left margin gap=%.1fpt, want ~9pt (12px)", ml)
	}
}
