package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestVerticalRLInlineBlockMatchesChromeColumnWidth locks fixture-62 #119:
// writing-mode:vertical-rl shrink-to-fit is a narrow glyph column (Chrome
// ~16pt with 8pt font + padding), not the horizontal measure of "ABC 123".
func TestVerticalRLInlineBlockMatchesChromeColumnWidth(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(`<html><body style="margin:0">` +
		`<div style="writing-mode:vertical-rl;border:1px solid #336;padding:3px 4px;` +
		`height:64px;background:#ffe;font-size:8pt;display:inline-block">ABC 123</div>` +
		`</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := layout.Layout(doc, layout.Options{Width: 400, Height: 200, Background: true})
	if err != nil {
		t.Fatal(err)
	}

	yellow := findVerticalRLFill(res)
	if yellow == nil {
		t.Fatal("missing vertical-rl background fill")
	}

	assertVerticalRLColumnSize(t, yellow)
}

func findVerticalRLFill(res *layout.Result) *layout.Op {
	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != layout.OpFillRect {
			continue
		}

		if isVerticalRLFill(*paintOp) {
			return paintOp
		}
	}

	return nil
}

func isVerticalRLFill(paintOp layout.Op) bool {
	return paintOp.R > 0.95 && paintOp.G > 0.95 && paintOp.B > 0.85 && paintOp.B < 0.98 && paintOp.H > 40
}

func assertVerticalRLColumnSize(t *testing.T, yellow *layout.Op) {
	t.Helper()

	t.Logf("vertical-rl box w=%.1f h=%.1f", yellow.W, yellow.H)

	// Chrome probe: width 16.5pt. Allow a small slack for font metrics.
	if yellow.W > 24 {
		t.Fatalf("vertical-rl inline-block width = %.1f, want <= 24 "+
			"(Chrome ~16.5); still using horizontal string measure?", yellow.W)
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

	doc, err := html.Parse(`<html><body style="margin:0">` +
		`<div style="display:flex;gap:8px;align-items:flex-start">` +
		`<div style="writing-mode:horizontal-tb;border:1px solid #336;` +
		`padding:3px 4px;background:#eef;font-size:8pt">ABC 123</div>` +
		`<div style="writing-mode:vertical-rl;border:1px solid #336;` +
		`padding:3px 4px;height:64px;background:#ffe;font-size:8pt">ABC 123</div>` +
		`</div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	opts := layout.Options{
		Width: 400, Height: 200, Background: true,
	}

	res, err := layout.Layout(doc, opts)
	if err != nil {
		t.Fatal(err)
	}

	var yellow *layout.Op

	for i := range res.Ops {
		candidate := &res.Ops[i]

		if isYellowFlexFill(candidate) {
			yellow = candidate

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

func isYellowFlexFill(operation *layout.Op) bool {
	if operation.Kind != layout.OpFillRect {
		return false
	}

	if operation.R <= 0.95 || operation.G <= 0.95 {
		return false
	}

	if operation.B <= 0.85 || operation.B >= 0.98 {
		return false
	}

	return operation.H > 40
}
