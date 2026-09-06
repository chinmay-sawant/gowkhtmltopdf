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
<td><div class="mc">Multi-column sample text repeated. Multi-column sample text repeated. `+
		`Multi-column sample text repeated. Multi-column sample text repeated.</div></td>
</tr><tr><td>next row</td></tr></table></body></html>`, cssSheet)

	multicolBox := findMulticolBoxByCount(res.root, 2)

	if multicolBox == nil {
		t.Fatal("no multicol box")
	}

	if !hasMulticolBottomBorder(res.Ops, multicolBox) {
		bot := multicolBox.y + multicolBox.height

		t.Fatalf("missing multicol bottom border at y=%.1f (box y=%.1f h=%.1f)",
			bot, multicolBox.y, multicolBox.height)
	}
}

// findMulticolBoxByCount returns the first box using the given column count.
func findMulticolBoxByCount(root *box, count int) *box {
	var found *box

	var walk func(current *box)

	walk = func(current *box) {
		if current == nil {
			return
		}

		if current.node != nil && current.style != nil && current.style.ColumnCount == count {
			found = current
		}

		for _, child := range current.children {
			walk(child)
		}
	}

	walk(root)

	return found
}

// hasMulticolBottomBorder reports whether a horizontal line spans the bottom
// edge of the multicol box.
func hasMulticolBottomBorder(ops []Op, multicolBox *box) bool {
	bot := multicolBox.y + multicolBox.height

	for _, paintOp := range ops {
		if paintOp.Kind != OpLine || paintOp.H >= 0.5 || paintOp.W <= multicolBox.w*0.5 {
			continue
		}

		if math.Abs(paintOp.Y-bot) < 1.0 {
			return true
		}
	}

	return false
}
