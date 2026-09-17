package layout

import (
	"strings"
	"testing"
)

// programiz-cpp-1: the header row's CTA wrapper carries
// position:absolute; right:0; width:auto. An absolutely positioned flex child
// is out of flow (CSS Flexbox 4.1), so it must not take a flex item slot or
// displace the search field. Before the fix gowk placed the blue band in flow
// at the row's left edge and pushed the field to x=415.
func TestAbsposFlexItemDoesNotDisplaceSiblings(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 14px }
.row { display: flex; flex-direction: row; align-items: center; height: 60px }
.brand { width: 40px; height: 20px }
.cta { background: #0556f3; color: #fff; border: 1px solid #0556f3; padding: 8px 4px }
.closed { position: absolute; right: 0; width: auto; justify-content: flex-end }
.field { flex: 1 1 auto; height: 20px }
`)

	res := layoutHTML(t,
		`<html><body><div class="row">`+
			`<div class="brand">M</div>`+
			`<div class="cta closed" id="cta">Programiz PRO</div>`+
			`<div class="field" id="field">Search tutorials</div>`+
			`</div></body></html>`,
		cssSheet)

	fieldX, ctaRight, ctaX := -1.0, -1.0, -1.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(paintOp.Text, "Search"):
			fieldX = paintOp.X
		case strings.Contains(paintOp.Text, "Programiz"):
			ctaX = paintOp.X
			ctaRight = paintOp.X + paintOp.W
		}
	}

	if fieldX < 0 {
		t.Fatal("no search-field text op")
	}

	// The brand is 40px (30pt) wide; the field follows it immediately. A CTA
	// in flow would push the field past the CTA's painted width (about 369pt
	// in the audit), so anything past the brand means flow participation.
	if fieldX > 31 {
		t.Fatalf("search field left edge = %.2fpt, want about 30 (right after the "+
			"brand); the absolute CTA took flow space", fieldX)
	}

	if ctaX < 0 {
		t.Fatal("no CTA text op")
	}

	// right:0 pins the CTA's right edge to the row's right edge (500pt).
	if ctaRight < 490 {
		t.Fatalf("CTA text right edge = %.2fpt, want about 500 (right:0)", ctaRight)
	}
}
