//nolint:testpackage // tests exercise block-flow geometry
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
