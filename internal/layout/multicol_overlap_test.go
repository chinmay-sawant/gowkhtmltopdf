//nolint:testpackage // exercises unexported helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func isMulticolSample(s string) bool {
	switch s {
	case "Multi-column", "sample", "text", "repeated.", "repeated", "Multi", "column":
		return true
	default:
		// Band cuts leave mid-line continuations such as "sample text repeated."
		// at the top of column 2 (Chrome keeps them; see fixture-61 #30/#31).
		return strings.HasPrefix(s, "Multi-column") ||
			strings.Contains(s, "sample") ||
			strings.Contains(s, "repeated")
	}
}

// Fixture-61 effect pattern in a table: definite-height multicol must not spill
// text into the following row (contain), and the two columns must top-align
// without stacking glyphs on the same baseline in the same x band.
func TestMulticolAnonInTableNoOverlap(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
@page { size: A4; margin: 12mm; }
body { margin: 0; font-size: 9.5pt; }
table.audit { width: 100%; border-collapse: collapse; table-layout: fixed; }
td { border: 1px solid #b8c0cc; vertical-align: top; padding: 6px 7px; font-size: 8.5pt; }
td.desc { width: 38%; }
td.effect { width: 34%; }
.mc { column-count:2; border:1px solid #888; padding:4px; font-size:8pt; height:48px; }
.cols { columns:2; border:1px solid #888; padding:4px; font-size:8pt; height:48px; }
.contain { contain:inherit; border:1px solid #6a6; padding:4px; background:#f7fff7; }
`)
	res := layoutHTML(t, `<html><body>
<table class="audit">
<tr>
  <td class="desc">With column-count: 2 the columns should be 2.</td>
  <td class="effect"><div class="mc">Multi-column sample text repeated. Multi-column sample text repeated. Multi-column sample text repeated.</div></td>
</tr>
<tr>
  <td class="desc">With columns: 2 the columns should be 2.</td>
  <td class="effect"><div class="cols">Multi-column sample text repeated. Multi-column sample text repeated. Multi-column sample text repeated.</div></td>
</tr>
<tr>
  <td class="desc">With contain: inherit the effect should match inherit.</td>
  <td class="effect"><div class="contain"><code>contain</code> applied</div></td>
</tr>
</table>
</body></html>`, cssSheet)

	type textOp struct {
		x, y float64
		s    string
	}
	var sample, contain []textOp
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Text == "" {
			continue
		}
		tx := textOp{op.X, op.Y, op.Text}
		if isMulticolSample(op.Text) {
			sample = append(sample, tx)
		}
		if op.Text == "contain" || op.Text == "applied" {
			contain = append(contain, tx)
		}
	}
	if len(sample) < 4 {
		t.Fatalf("expected multicol sample text, got %d", len(sample))
	}
	if len(contain) == 0 {
		t.Fatal("expected contain label text")
	}

	containTop := contain[0].y
	for _, c := range contain[1:] {
		if c.y < containTop {
			containTop = c.y
		}
	}

	spill := 0
	for _, tx := range sample {
		// Sample glyphs must stay above the contain row (definite-height clip).
		if tx.y >= containTop-0.5 {
			spill++
			t.Logf("sample %q y=%.1f >= contain top %.1f", tx.s, tx.y, containTop)
		}
	}

	// Same baseline + nearly same x with different words = stacked paint.
	stack := 0
	for i := range sample {
		for j := i + 1; j < len(sample); j++ {
			a, b := sample[i], sample[j]
			if math.Abs(a.y-b.y) < 1.2 && math.Abs(a.x-b.x) < 2 && a.s != b.s {
				stack++
				t.Logf("stacked %q and %q at (%.1f,%.1f)", a.s, b.s, a.x, a.y)
			}
		}
	}

	// Topmost glyphs in the first multicol box (lowest Y pair across columns)
	// must share a baseline. Aligning mid-band orphans to the box top is what
	// pulled fixture-61 #23/#24/#32 through the border.
	topY := sample[0].y
	for _, tx := range sample[1:] {
		if tx.y < topY {
			topY = tx.y
		}
	}
	leftY, rightY := math.Inf(1), math.Inf(1)
	minX := sample[0].x
	for _, tx := range sample[1:] {
		if tx.x < minX {
			minX = tx.x
		}
	}
	for _, tx := range sample {
		if tx.y > topY+30 {
			continue
		}
		if math.Abs(tx.x-minX) < 20 {
			if tx.y < leftY {
				leftY = tx.y
			}
			continue
		}
		if tx.y < rightY {
			rightY = tx.y
		}
	}
	misalign := 0
	if !math.IsInf(leftY, 1) && !math.IsInf(rightY, 1) && math.Abs(leftY-rightY) > 1.5 {
		misalign = 1
		t.Logf("column tops misaligned: left y=%.1f right y=%.1f", leftY, rightY)
	}

	if spill > 0 || stack > 0 || misalign > 0 {
		t.Fatalf("multicol spill_into_contain=%d stacked_glyphs=%d col_misalign=%d", spill, stack, misalign)
	}
}

func TestColumnFillAloneIsNotMulticol(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.cf { column-fill: auto; width: 180pt; height: 48pt; font-size: 8pt; border: 1pt solid #888; }
`)
	styles := resolveStyles(mustParse(t, `<html><body><div class="cf">x</div></body></html>`),
		[]*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	st := styleByClass(t, styles, "cf")
	if isMulticol(*st) {
		t.Fatalf("column-fill:auto alone must not establish multicol (count=%d width=%.1f)", st.ColumnCount, st.ColumnWidth)
	}
}
