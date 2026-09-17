package layout

import (
	"strings"
	"testing"
)

// audit2 flex intrinsic image percentages. Programiz prints `img{width:100%}`;
// during intrinsic sizing a percentage image width has no definite containing
// block, so it must fall back to the width attribute / intrinsic size. The old
// viewport fallback measured the brand image as wide as the page, which
// inflated the brand flex item to the whole row and squeezed the flex:1 search
// field to zero width (its placeholder then painted at the row's right edge).
// Assertions are painted geometry: the image op width and the placeholder x.

func TestFlexItemImagePercentWidthIsIndefiniteInMeasure(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 14px }
.row { display: flex; align-items: center; position: relative }
img { width: 100% }
.brand a { display: inline-block; width: 28px; height: 28px; margin-right: 8px }
.field { flex: 1 1 auto }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="row" style="width:400px">`+
			`<div class="brand"><a href="/"><img src="logo.png" width="28" height="28" alt="Logo"></a></div>`+
			`<form class="field"><input type="text" placeholder="Search tutorials" autocomplete="off"></form>`+
			`</div></body></html>`,
		tinyPNG(28, 28), "logo.png", cssSheet)

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("image ops = %d, want 1", len(imgs))
	}
	// 28 CSS px = 21pt. The viewport fallback would have measured 500pt.
	if imgs[0].W > 40 {
		t.Fatalf("brand image width = %.2fpt, want about 21pt (28px attribute); "+
			"percentage width resolved against the viewport during intrinsic sizing",
			imgs[0].W)
	}

	found := false

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || !strings.Contains(paintOp.Text, "Search") {
			continue
		}

		found = true

		if paintOp.X > 60 {
			t.Fatalf("search placeholder x = %.2fpt, want near the brand (about 36pt); "+
				"the oversized brand item pushed the field to the row edge", paintOp.X)
		}
	}

	if !found {
		t.Fatal("no search placeholder text op")
	}
}

// Control: an explicit CSS width still wins over the attribute and stays
// independent of the containing block.
func TestFlexItemImageExplicitWidthInMeasure(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 14px }
.row { display: flex; align-items: center }
.brand img { width: 40px }
.field { flex: 1 1 auto }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="row" style="width:400px">`+
			`<div class="brand"><img src="logo.png" width="28" height="28"></div>`+
			`<form class="field"><input type="text" placeholder="Search tutorials"></form>`+
			`</div></body></html>`,
		tinyPNG(28, 28), "logo.png", cssSheet)

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("image ops = %d, want 1", len(imgs))
	}

	if !near(imgs[0].W, 30) {
		t.Fatalf("image width = %.2fpt, want 30pt (40px CSS)", imgs[0].W)
	}
}
