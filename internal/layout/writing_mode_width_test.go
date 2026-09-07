package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestVerticalRLInlineBlockMatchesChromeColumnWidth locks fixture-62 #119:
// writing-mode:vertical-rl shrink-to-fit is a narrow glyph column (Chrome
// ~16pt with 8pt font + padding), not the horizontal measure of "ABC 123".
func TestVerticalRLInlineBlockMatchesChromeColumnWidth(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(`<html><body style="margin:0">
<div style="writing-mode:vertical-rl;border:1px solid #336;padding:3px 4px;height:64px;background:#ffe;font-size:8pt;display:inline-block">ABC 123</div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{Width: 400, Height: 200, Background: true})
	if err != nil {
		t.Fatal(err)
	}

	var yellow *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind != OpFillRect {
			continue
		}
		// #ffe
		if op.R > 0.95 && op.G > 0.95 && op.B > 0.85 && op.B < 0.98 && op.H > 40 {
			yellow = op
			break
		}
	}
	if yellow == nil {
		t.Fatal("missing vertical-rl background fill")
	}

	t.Logf("vertical-rl box w=%.1f h=%.1f", yellow.W, yellow.H)

	// Chrome probe: width 16.5pt. Allow a small slack for font metrics.
	if yellow.W > 24 {
		t.Fatalf("vertical-rl inline-block width = %.1f, want <= 24 (Chrome ~16.5); still using horizontal string measure?", yellow.W)
	}
	if yellow.W < 10 {
		t.Fatalf("vertical-rl inline-block width = %.1f, want >= 10 (too collapsed)", yellow.W)
	}
	if yellow.H < 50 || yellow.H > 70 {
		t.Fatalf("vertical-rl inline-block height = %.1f, want ~64px content box", yellow.H)
	}
}

// TestVerticalRLFlexItemColumnWidth covers the fixture-62 Effect flex row.
func TestVerticalRLFlexItemColumnWidth(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(`<html><body style="margin:0">
<div style="display:flex;gap:8px;align-items:flex-start">
<div style="writing-mode:horizontal-tb;border:1px solid #336;padding:3px 4px;background:#eef;font-size:8pt">ABC 123</div>
<div style="writing-mode:vertical-rl;border:1px solid #336;padding:3px 4px;height:64px;background:#ffe;font-size:8pt">ABC 123</div>
</div>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{Width: 400, Height: 200, Background: true})
	if err != nil {
		t.Fatal(err)
	}

	var yellow *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpFillRect && op.R > 0.95 && op.G > 0.95 && op.B > 0.85 && op.B < 0.98 && op.H > 40 {
			yellow = op
			break
		}
	}
	if yellow == nil {
		t.Fatal("missing vertical-rl flex item fill")
	}
	if yellow.W > 24 {
		t.Fatalf("vertical-rl flex item width = %.1f, want <= 24 (Chrome ~16.5)", yellow.W)
	}
}
