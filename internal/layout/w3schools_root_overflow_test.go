package layout

import (
	"strings"
	"testing"
)

// w3schools-5: root-element overflow is a viewport property, not a box clip.
// CSS propagates html{overflow-x:hidden} (w3.css and main.v1.0.3.css set it,
// the latter with overflow-y:scroll) to the viewport: content that flows onto
// later pages must still paint. The engine applied the root padding box as a
// clip over the flat display list, so the footer tail that lands below the
// first viewport disappeared from the text layer entirely.
func TestRootOverflowDoesNotClipFooter(t *testing.T) {
	t.Parallel()

	for _, overflowRule := range []string{
		"html { overflow-x: hidden }",
		"html { overflow-x: hidden; overflow-y: scroll }",
	} {
		t.Run(overflowRule, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, overflowRule+`
			body { margin: 0; font-family: sans-serif; font-size: 12pt }
			.col { float: left; width: 100% }
			.b { height: 380pt; background: #eef; margin-bottom: 10pt }
			#wrappercontainer { width: 100%; height: 100pt; background: #f00; position: relative; z-index: 2 }
			#footer { background: #282A35; color: #fff }
			.links { overflow: hidden }
			.fl { float: left; padding: 20pt }
			`)

			src := `<html><body>` +
				`<div class="col"><div class="b">BLOCK ONE</div>` +
				`<div class="b">BLOCK TWO</div><div class="b">BLOCK THREE</div></div>` +
				`<div id="wrappercontainer"><div id="footer"><div class="links">` +
				`<div class="fl">PLUS</div><div class="fl">SPACES</div>` +
				`<div class="fl">FOR TEACHERS</div></div>` +
				`<h2>Top Tutorials</h2><p>FOOTER SENTINEL TEXT</p></div></div>` +
				`</body></html>`

			res := layoutHTML(t, src, cssSheet)

			joined := ""

			for _, op := range res.Ops {
				if op.Kind == OpText {
					joined += op.Text + " "
				}
			}

			for _, word := range []string{"Tutorials", "SENTINEL"} {
				if !strings.Contains(joined, word) {
					t.Fatalf("footer word %q missing from the display list with %s; text layer = %q",
						word, overflowRule, joined)
				}
			}
		})
	}
}
