//nolint:testpackage // exercises unexported helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const multicolSampleWord = "Multi-column"

// multicolSampleText is the three-sentence fixture body shared by both
// multicol demo cells below.
const multicolSampleText = multicolSampleWord + " sample text repeated. " +
	multicolSampleWord + " sample text repeated. " +
	multicolSampleWord + " sample text repeated."

func isMulticolSample(text string) bool {
	switch text {
	case multicolSampleWord, "sample", "text", "repeated.", "repeated", "Multi", "column":
		return true
	default:
		// Band cuts leave mid-line continuations such as "sample text repeated."
		// at the top of column 2 (Chrome keeps them; see fixture-61 #30/#31).
		return strings.HasPrefix(text, multicolSampleWord) ||
			strings.Contains(text, "sample") ||
			strings.Contains(text, "repeated")
	}
}

type multicolTextOp struct {
	x, y float64
	text string
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
  <td class="effect"><div class="mc">`+multicolSampleText+`</div></td>
</tr>
<tr>
  <td class="desc">With columns: 2 the columns should be 2.</td>
  <td class="effect"><div class="cols">`+multicolSampleText+`</div></td>
</tr>
<tr>
  <td class="desc">With contain: inherit the effect should match inherit.</td>
  <td class="effect"><div class="contain"><code>contain</code> applied</div></td>
</tr>
</table>
</body></html>`, cssSheet)

	sample, contain := collectMulticolSampleOps(res.Ops)

	if len(sample) < 4 {
		t.Fatalf("expected multicol sample text, got %d", len(sample))
	}

	if len(contain) == 0 {
		t.Fatal("expected contain label text")
	}

	containTop := multicolContainTop(contain)
	spill := countMulticolSpill(t, sample, containTop)
	stack := countMulticolStacked(t, sample)
	misalign := multicolTopMisalign(t, sample)

	if spill > 0 || stack > 0 || misalign > 0 {
		t.Fatalf("multicol spill_into_contain=%d stacked_glyphs=%d col_misalign=%d", spill, stack, misalign)
	}
}

// collectMulticolSampleOps splits paint ops into multicol sample words and
// contain-row labels.
func collectMulticolSampleOps(ops []Op) ([]multicolTextOp, []multicolTextOp) {
	var sample, contain []multicolTextOp

	for _, paintOp := range ops {
		if paintOp.Kind != OpText || paintOp.Text == "" {
			continue
		}

		entry := multicolTextOp{paintOp.X, paintOp.Y, paintOp.Text}

		if isMulticolSample(paintOp.Text) {
			sample = append(sample, entry)
		}

		if paintOp.Text == "contain" || paintOp.Text == "applied" {
			contain = append(contain, entry)
		}
	}

	return sample, contain
}

// multicolContainTop returns the smallest Y among contain-row labels.
func multicolContainTop(contain []multicolTextOp) float64 {
	containTop := contain[0].y

	for _, entry := range contain[1:] {
		if entry.y < containTop {
			containTop = entry.y
		}
	}

	return containTop
}

// countMulticolSpill counts sample glyphs at or below the contain row
// (definite-height clip must keep them above).
func countMulticolSpill(t *testing.T, sample []multicolTextOp, containTop float64) int {
	t.Helper()

	spill := 0

	for _, entry := range sample {
		// Sample glyphs must stay above the contain row (definite-height clip).
		if entry.y >= containTop-0.5 {
			spill++

			t.Logf("sample %q y=%.1f >= contain top %.1f", entry.text, entry.y, containTop)
		}
	}

	return spill
}

// countMulticolStacked counts glyph pairs sharing a baseline and x band with
// different words (stacked paint).
func countMulticolStacked(t *testing.T, sample []multicolTextOp) int {
	t.Helper()

	stack := 0

	for i := range sample {
		for j := i + 1; j < len(sample); j++ {
			first, second := sample[i], sample[j]

			if math.Abs(first.y-second.y) < 1.2 && math.Abs(first.x-second.x) < 2 && first.text != second.text {
				stack++

				t.Logf("stacked %q and %q at (%.1f,%.1f)", first.text, second.text, first.x, first.y)
			}
		}
	}

	return stack
}

// multicolTopMisalign checks that the topmost glyphs of both columns share a
// baseline. Aligning mid-band orphans to the box top is what pulled
// fixture-61 #23/#24/#32 through the border.
func multicolTopMisalign(t *testing.T, sample []multicolTextOp) int {
	t.Helper()

	topY := sample[0].y

	for _, entry := range sample[1:] {
		if entry.y < topY {
			topY = entry.y
		}
	}

	minX := sample[0].x

	for _, entry := range sample[1:] {
		if entry.x < minX {
			minX = entry.x
		}
	}

	leftY, rightY := multicolColumnTops(sample, topY, minX)

	if !math.IsInf(leftY, 1) && !math.IsInf(rightY, 1) && math.Abs(leftY-rightY) > 1.5 {
		t.Logf("column tops misaligned: left y=%.1f right y=%.1f", leftY, rightY)

		return 1
	}

	return 0
}

// multicolColumnTops returns the topmost Y in the left column band and in the
// remaining (right) band, ignoring glyphs below the top band.
func multicolColumnTops(sample []multicolTextOp, topY, minX float64) (float64, float64) {
	leftY, rightY := math.Inf(1), math.Inf(1)

	for _, entry := range sample {
		if entry.y > topY+30 {
			continue
		}

		if math.Abs(entry.x-minX) < 20 {
			if entry.y < leftY {
				leftY = entry.y
			}

			continue
		}

		if entry.y < rightY {
			rightY = entry.y
		}
	}

	return leftY, rightY
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
