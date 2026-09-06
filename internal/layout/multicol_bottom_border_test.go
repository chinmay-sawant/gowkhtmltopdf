//nolint:testpackage // exercises unexported helpers
package layout

import (
	"math"
	"testing"
)

// Definite-height multicol in a table must keep a live bottom border line after
// anonymous band clipping (fixture-61 page-2 column demos).
func TestMulticolAnonKeepsBottomBorder(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
@page { size: A4; margin: 12mm; }
body { margin: 0; }
table { width: 100%; border-collapse: collapse; table-layout: fixed; }
td { border: 1px solid #b8c0cc; vertical-align: top; padding: 6px; }
.mc { column-count:2; border:1px solid #888; padding:4px; font-size:8pt; height:48px; overflow:hidden; }
`)
	res := layoutHTML(t, `<html><body><table><tr>
<td><div class="mc">Multi-column sample text repeated. Multi-column sample text repeated. Multi-column sample text repeated. Multi-column sample text repeated.</div></td>
</tr><tr><td>next row</td></tr></table></body></html>`, cssSheet)

	var mc *box
	var walk func(*box)
	walk = func(b *box) {
		if b == nil {
			return
		}
		if b.node != nil && b.style != nil && b.style.ColumnCount == 2 {
			mc = b
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
	if mc == nil {
		t.Fatal("no multicol box")
	}
	bot := mc.y + mc.height
	hasBot := false
	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H < 0.5 && op.W > mc.w*0.5 && math.Abs(op.Y-bot) < 1.0 {
			hasBot = true
			break
		}
	}
	if !hasBot {
		t.Fatalf("missing multicol bottom border at y=%.1f (box y=%.1f h=%.1f)", bot, mc.y, mc.height)
	}
}
