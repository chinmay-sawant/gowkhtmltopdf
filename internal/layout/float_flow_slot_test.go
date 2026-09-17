package layout

import (
	"strings"
	"testing"
)

// w3schools-3: a normal-flow footer chain (no BFC root anywhere in the chain)
// follows float columns that cover the content width and run past the page
// bottom. CSS 2.1 section 9.5 shortens the line boxes beside the float; when
// the remaining slot cannot hold a line's content, the line moves down until
// it fits. Before the fix the h2 and paragraph wrapped mid-word inside the
// narrow slot ("Top / Tutori / als", "SENTIN / EL TEXT") instead of clearing
// below the floats.
func TestNormalFlowFooterClearsNarrowFloatSlot(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 12pt }
.w3-col { float: left }
.b { height: 380pt; background: #eef; margin-bottom: 10pt }
#wrappercontainer { }
#footerwrapper { background-color: #282A35; color: white }
#spacemyfooter { padding: 20pt 40pt }
`)

	src := `<html><body>` +
		`<div class="w3-col" style="width:83%">` +
		`<div class="b">BLOCK ONE</div><div class="b">BLOCK TWO</div><div class="b">BLOCK THREE</div></div>` +
		`<div class="w3-col" style="width:17%">` +
		`<div class="b" style="background:#fee">SIDE A</div>` +
		`<div class="b" style="background:#fee">SIDE B</div>` +
		`<div class="b" style="background:#fee">SIDE C</div></div>` +
		`<div id="wrappercontainer"><div id="footerwrapper"><div id="spacemyfooter">` +
		`<h2>Top Tutorials</h2><p>FOOTER SENTINEL TEXT</p>` +
		`</div></div></div></body></html>`

	res := layoutHTML(t, src, cssSheet)

	joined := ""

	for _, op := range res.Ops {
		if op.Kind == OpText {
			joined += op.Text + " "
		}
	}

	for _, phrase := range []string{"Top Tutorials", "FOOTER SENTINEL TEXT"} {
		if !strings.Contains(joined, phrase) {
			t.Fatalf("footer phrase %q broken mid-word; text layer = %q", phrase, joined)
		}
	}
}
