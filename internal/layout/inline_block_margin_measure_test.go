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

	texts := marginDemoTextOps(res.Ops)
	if len(texts) == 0 {
		t.Fatal("missing margin demo text")
	}

	assertMarginDemoSingleLine(t, texts)

	pink, green := marginDemoFills(res.Ops)
	if pink == nil || green == nil {
		t.Fatalf("missing pink/green fills (pink=%v green=%v)", pink != nil, green != nil)
	}

	gap := green.X - pink.X
	if gap < 8 || gap > 14 {
		t.Fatalf("left margin gap=%.1fpt, want ~9pt (12px)", gap)
	}
}

// marginDemoTextOps returns the text ops that paint the margin demo label.
func marginDemoTextOps(ops []Op) []Op {
	var texts []Op

	for _, item := range ops {
		if item.Kind == OpText && (strings.Contains(item.Text, "all-sides") || strings.Contains(item.Text, "12px")) {
			texts = append(texts, item)
		}
	}

	return texts
}

// assertMarginDemoSingleLine fails when the demo label wrapped to a new line.
func assertMarginDemoSingleLine(t *testing.T, texts []Op) {
	t.Helper()

	firstY := texts[0].Y

	for _, item := range texts[1:] {
		if item.Y > firstY+2 {
			t.Fatalf("text wrapped to y=%.1f (first y=%.1f); shrink-to-fit undersized nested margin box", item.Y, firstY)
		}
	}
}

// marginDemoFills returns the largest pink outer fill and the green inner fill.
func marginDemoFills(ops []Op) (*Op, *Op) {
	var pink, green *Op

	for idx := range ops {
		cand := &ops[idx]
		if cand.Kind != OpFillRect {
			continue
		}

		red, grn, blu := cand.R, cand.G, cand.B

		if isPinkFill(red, grn, blu) {
			if pink == nil || cand.W*cand.H > pink.W*pink.H {
				pink = cand
			}
		}

		if isGreenFill(red, grn, blu) {
			if green == nil || cand.W < 200 {
				green = cand
			}
		}
	}

	return pink, green
}

// isPinkFill reports the #fce8e8 outer background fill.
func isPinkFill(red, grn, blu float64) bool {
	return red > 0.95 && grn > 0.85 && grn < 0.95 && blu > 0.85
}

// isGreenFill reports the #dfd inner background fill.
func isGreenFill(red, grn, blu float64) bool {
	return red > 0.8 && red < 0.92 && grn > 0.95 && blu > 0.8
}
