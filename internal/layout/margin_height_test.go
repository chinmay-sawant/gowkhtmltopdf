package layout

import (
	"strings"
	"testing"
)

func TestFinalChildMarginContributesInsidePaddedParent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="parent"><p>one</p></div><div>two</div></body></html>`, sheet(t, `
body { margin: 0; font-size: 10pt }
.parent { padding-bottom: 10pt; background: #eee }
p { margin: 0 0 8pt }
`))

	var oneY, twoY float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "one") {
			oneY = paintOp.Y
		}

		if strings.Contains(paintOp.Text, "two") {
			twoY = paintOp.Y
		}
	}

	if twoY-oneY < 25 {
		t.Fatalf("next block y=%.2f is too close to first text y=%.2f", twoY, oneY)
	}
}

// LCO-13: a last-child bottom margin must escape a borderless parent and
// collapse with the following sibling's top margin (CSS 2.1 8.3.1). Padded
// parents still absorb the margin (TestFinalChildMarginContributesInsidePaddedParent).
func TestLastChildMarginEscapesBorderlessParent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt }
.wrap { margin: 0; background: #eee }
ul { margin: 0 0 40pt; padding: 0 0 0 20pt; list-style: none }
.next { margin: 0 }
`)

	res := layoutHTML(t,
		`<html><body><div class="wrap"><ul><li>ITEM</li></ul></div><div class="next">AFTER</div></body></html>`,
		cssSheet)

	var itemY, afterY float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(paintOp.Text, "ITEM"):
			itemY = paintOp.Y
		case strings.Contains(paintOp.Text, "AFTER"):
			afterY = paintOp.Y
		}
	}

	if itemY == 0 || afterY == 0 {
		t.Fatalf("missing text ops: ITEM y=%.2f AFTER y=%.2f", itemY, afterY)
	}

	gap := afterY - itemY
	// 40pt ul margin-bottom must show up between the lines (plus ~one line box).
	if gap < 45 {
		t.Fatalf("AFTER-ITEM gap = %.2fpt, want >= 45pt; ul margin-bottom must escape the borderless wrap", gap)
	}
}
