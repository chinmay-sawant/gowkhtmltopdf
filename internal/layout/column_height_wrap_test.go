package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

//nolint:cyclop,funlen // this regression test checks several independent column invariants
func TestColumnHeightCapsColumn(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.mc {
  column-count: 2;
  column-gap: 10pt;
  column-height: 48pt;
  column-fill: auto;
  width: 220pt;
  font-size: 10pt;
}
.mc p { margin: 0 0 2pt 0; }
`)
	root := mustParse(t, `<html><body>
<div class="mc">
  <p>RowA col text one.</p>
  <p>RowA col text two.</p>
  <p>RowA col text three.</p>
  <p>RowB col text four.</p>
  <p>RowB col text five.</p>
  <p>RowB col text six.</p>
</div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)
	st := styleByClass(t, styles, "mc")

	if st.ColumnHeight < 47 || st.ColumnHeight > 49 {
		t.Fatalf("column-height stored=%.1f, want ~48", st.ColumnHeight)
	}

	res := layoutHTML(t, `<html><body>
<div class="mc">
  <p>RowA col text one.</p>
  <p>RowA col text two.</p>
  <p>RowA col text three.</p>
  <p>RowA col text threeb.</p>
  <p>RowB col text four.</p>
  <p>RowB col text five.</p>
  <p>RowB col text six.</p>
  <p>RowB col text sixb.</p>
  <p>RowC col text seven.</p>
  <p>RowC col text eight.</p>
  <p>RowC col text nine.</p>
  <p>RowC col text ten.</p>
</div>
</body></html>`, cssSheet)

	textYs := make([]float64, 0, len(res.Ops))

	for _, textOp := range res.Ops {
		if textOp.Kind != OpText || textOp.Text == "" {
			continue
		}

		textYs = append(textYs, textOp.Y)
	}

	if len(textYs) < 4 {
		t.Fatalf("expected multicol text ops, got %d", len(textYs))
	}

	minY, maxY := textYs[0], textYs[0]
	for _, textY := range textYs[1:] {
		if textY < minY {
			minY = textY
		}

		if textY > maxY {
			maxY = textY
		}
	}

	// With column-height 48pt and wrap/auto, content that exceeds one row
	// must open a second row rather than stretch a single tall column.
	span := maxY - minY
	if span < 48 {
		t.Fatalf("text Y span=%.1f; want a second row past column-height 48", span)
	}

	for _, textY := range textYs {
		band := math.Floor((textY - minY) / 48)
		top := minY + band*48

		if textY-top > 50 {
			t.Fatalf("text y=%.1f exceeds ~48pt band from %.1f", textY, top)
		}
	}
}

//nolint:cyclop,funlen // this regression test checks wrap, labels, and row span
func TestColumnWrapCreatesRow(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.mc {
  column-count: 2;
  column-gap: 8pt;
  column-height: 36pt;
  column-wrap: wrap;
  column-fill: auto;
  width: 200pt;
  font-size: 9pt;
}
.mc p { margin: 0 0 1pt 0; }
`)
	root := mustParse(t, `<html><body><div class="mc">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)
	st := styleByClass(t, styles, "mc")

	if st.ColumnWrap != columnWrapWrap {
		t.Fatalf("column-wrap=%q, want wrap", st.ColumnWrap)
	}

	res := layoutHTML(t, `<html><body>
<div class="mc">
  <p>Alpha one.</p>
  <p>Bravo two.</p>
  <p>Charlie three.</p>
  <p>Delta four.</p>
  <p>Echo five.</p>
  <p>Foxtrot six.</p>
  <p>Golf seven.</p>
  <p>Hotel eight.</p>
  <p>India nine.</p>
  <p>Juliet ten.</p>
</div>
</body></html>`, cssSheet)

	labelY := map[string]float64{}

	for _, textOp := range res.Ops {
		if textOp.Kind != OpText {
			continue
		}

		for _, key := range []string{
			"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot",
			"Golf", "Hotel", "India", "Juliet",
		} {
			if len(textOp.Text) >= len(key) && textOp.Text[:len(key)] == key {
				labelY[key] = textOp.Y
			}
		}
	}

	if len(labelY) < 6 {
		t.Fatalf("missing multicol labels: %v", labelY)
	}

	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, textY := range labelY {
		if textY < minY {
			minY = textY
		}

		if textY > maxY {
			maxY = textY
		}
	}

	if maxY-minY < 30 {
		t.Fatalf("column-wrap:wrap Y span=%.1f; want a second multicol row", maxY-minY)
	}
}

func TestColumnWrapNowrapStopsAfterOneRow(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.mc {
  column-count: 2;
  column-gap: 8pt;
  column-height: 36pt;
  column-wrap: nowrap;
  column-fill: auto;
  width: 200pt;
  font-size: 9pt;
}
.mc p { margin: 0 0 1pt 0; }
`)
	root := mustParse(t, `<html><body><div class="mc">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)
	columnStyle := styleByClass(t, styles, "mc")

	if columnStyle.ColumnWrap != columnWrapNowrap {
		t.Fatalf("column-wrap=%q, want nowrap", columnStyle.ColumnWrap)
	}

	if columnWrapCreatesRows(*columnStyle) {
		t.Fatal("nowrap must not open extra block-direction rows")
	}

	res := layoutHTML(t, `<html><body>
<div class="mc">
  <p>Alpha one.</p>
  <p>Bravo two.</p>
  <p>Charlie three.</p>
  <p>Delta four.</p>
  <p>Echo five.</p>
  <p>Foxtrot six.</p>
  <p>Golf seven.</p>
  <p>Hotel eight.</p>
  <p>India nine.</p>
  <p>Juliet ten.</p>
</div>
</body></html>`, cssSheet)

	minY, maxY, n := columnWrapLabelYSpan(t, res)
	if n < 2 {
		t.Fatalf("nowrap produced %d labels", n)
	}

	if maxY-minY > 40 {
		t.Fatalf("column-wrap:nowrap Y span=%.1f; want one ~36pt row", maxY-minY)
	}
}

func TestColumnWrapAutoCreatesRowWhenHeightSet(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.mc {
  column-count: 2;
  column-gap: 8pt;
  column-height: 36pt;
  column-wrap: auto;
  column-fill: auto;
  width: 200pt;
  font-size: 9pt;
}
.mc p { margin: 0 0 1pt 0; }
`)
	root := mustParse(t, `<html><body><div class="mc">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)
	columnStyle := styleByClass(t, styles, "mc")

	if columnStyle.ColumnWrap != columnWrapAuto {
		t.Fatalf("column-wrap=%q, want auto", columnStyle.ColumnWrap)
	}

	if !columnWrapCreatesRows(*columnStyle) {
		t.Fatal("auto with column-height must wrap to extra rows")
	}

	res := layoutHTML(t, `<html><body>
<div class="mc">
  <p>Alpha one.</p>
  <p>Bravo two.</p>
  <p>Charlie three.</p>
  <p>Delta four.</p>
  <p>Echo five.</p>
  <p>Foxtrot six.</p>
  <p>Golf seven.</p>
  <p>Hotel eight.</p>
  <p>India nine.</p>
  <p>Juliet ten.</p>
</div>
</body></html>`, cssSheet)

	minY, maxY, n := columnWrapLabelYSpan(t, res)
	if n < 6 {
		t.Fatalf("auto produced %d labels", n)
	}

	if maxY-minY < 30 {
		t.Fatalf("column-wrap:auto Y span=%.1f; want a second multicol row", maxY-minY)
	}
}

func columnWrapLabelYSpan(t *testing.T, res *Result) (float64, float64, int) {
	t.Helper()

	labelY := map[string]float64{}

	for _, textOp := range res.Ops {
		if textOp.Kind != OpText {
			continue
		}

		for _, key := range []string{
			"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot",
			"Golf", "Hotel", "India", "Juliet",
		} {
			if len(textOp.Text) >= len(key) && textOp.Text[:len(key)] == key {
				labelY[key] = textOp.Y
			}
		}
	}

	if len(labelY) == 0 {
		return 0, 0, 0
	}

	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, textY := range labelY {
		if textY < minY {
			minY = textY
		}

		if textY > maxY {
			maxY = textY
		}
	}

	return minY, maxY, len(labelY)
}
