package layout

import (
	"strings"
	"testing"
)

// programiz-cpp-14: a column-flex card whose height is smaller than its
// content. The paragraph wraps to two lines; the flex shrink pass used a
// one-line content height (laid out at an indefinite width) as its floor and
// squeezed the paragraph box under its own last line, so the following
// "Learn more" button painted over the text. The floor for an auto-height
// non-replaced item is the height at its used width. Assertions are painted
// geometry: the paragraph's last text bottom vs the button fill top.

func TestFlexColumnParagraphNotSqueezedUnderButton(t *testing.T) { //nolint:cyclop // paragraph/button geometry census
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 16px; line-height: 24px }
.card { display: flex; flex-direction: column; width: 319pt }
.col { display: flex; flex-direction: column; height: 81pt }
h3 { margin: 0 0 8px; font-size: 20px; line-height: 24px }
p { margin: 0 0 16px }
.btn { display: flex; background: #0556f3; color: #fff; padding: 12px 16px }
`)

	res := layoutHTML(t,
		`<html><body><div class="card"><div class="col">`+
			`<h3>Getting Started with C++</h3>`+
			`<p>Learn how you can install and use C++ on your own computer.</p>`+
			`<a class="btn" href="#">Learn more</a>`+
			`</div></div></body></html>`, cssSheet)

	paraBottom := -1.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && (strings.Contains(paintOp.Text, "computer") ||
			strings.Contains(paintOp.Text, "Learn how")) {
			if bottom := paintOp.Y + paintOp.InkDescent; bottom > paraBottom {
				paraBottom = bottom
			}
		}
	}

	if paraBottom < 0 {
		t.Fatal("no paragraph text ops")
	}

	btnTop := -1.0
	btnBlue := func(paintOp Op) bool {
		return paintOp.Kind == OpFillRect && paintOp.B > 0.8 && paintOp.R < 0.1 && paintOp.G < 0.5
	}

	for _, paintOp := range res.Ops {
		if btnBlue(paintOp) && (btnTop < 0 || paintOp.Y < btnTop) {
			btnTop = paintOp.Y
		}
	}

	if btnTop < 0 {
		t.Fatal("no button fill op")
	}

	if btnTop < paraBottom {
		t.Fatalf("button fill top %.2f paints over the paragraph bottom %.2f "+
			"(overlap %.2fpt); the paragraph must keep its wrapped height",
			btnTop, paraBottom, paraBottom-btnTop)
	}
}
