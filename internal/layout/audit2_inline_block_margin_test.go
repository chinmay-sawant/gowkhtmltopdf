package layout

import (
	"strings"
	"testing"
)

// learn-cpp-org-13: the Start Exercise button (Bootstrap .btn is
// display:inline-block) carries mt-2, and its 8px top margin must move the
// button down. Painted geometry: the button text sits one margin lower than
// the same button without margin-top, and the following line shifts too.

func TestInlineBlockTopMarginMovesContent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 16px }
p { margin: 0 }
.ib { display: inline-block; margin-top: 45pt }
`)

	res := layoutHTML(t,
		`<html><body><p>Intro.</p><span class="ib">Button</span></body></html>`,
		cssSheet)

	bodyBaseline := -1.0
	btnBaseline := -1.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(paintOp.Text, "Intro"):
			bodyBaseline = paintOp.Y
		case strings.Contains(paintOp.Text, "Button"):
			btnBaseline = paintOp.Y
		}
	}

	if bodyBaseline < 0 || btnBaseline < 0 {
		t.Fatalf("missing ops: bodyBaseline=%.1f btnBaseline=%.1f", bodyBaseline, btnBaseline)
	}

	// The inline-block's margin box top sits 45pt above its content, so its
	// baseline lands at least 45pt below the shared line's ascent. Without
	// the margin both baselines coincide.
	if btnBaseline < bodyBaseline+40 {
		t.Fatalf("inline-block baseline %.1f, want >= %.1f (Intro baseline + 45pt "+
			"margin-top); the inline-block vertical margin was dropped",
			btnBaseline, bodyBaseline+40)
	}
}
