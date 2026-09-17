package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

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

	var ys []float64
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Text == "" {
			continue
		}

		ys = append(ys, op.Y)
	}

	if len(ys) < 4 {
		t.Fatalf("expected multicol text ops, got %d", len(ys))
	}

	minY, maxY := ys[0], ys[0]
	for _, y := range ys[1:] {
		if y < minY {
			minY = y
		}

		if y > maxY {
			maxY = y
		}
	}

	// With column-height 48pt and wrap/auto, content that exceeds one row
	// must open a second row rather than stretch a single tall column.
	span := maxY - minY
	if span < 48 {
		t.Fatalf("text Y span=%.1f; want a second row past column-height 48", span)
	}

	for _, y := range ys {
		band := math.Floor((y - minY) / 48)
		top := minY + band*48
		if y-top > 50 {
			t.Fatalf("text y=%.1f exceeds ~48pt band from %.1f", y, top)
		}
	}
}

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

	ys := map[string]float64{}
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		for _, key := range []string{
			"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot",
			"Golf", "Hotel", "India", "Juliet",
		} {
			if len(op.Text) >= len(key) && op.Text[:len(key)] == key {
				ys[key] = op.Y
			}
		}
	}

	if len(ys) < 6 {
		t.Fatalf("missing multicol labels: %v", ys)
	}

	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, y := range ys {
		if y < minY {
			minY = y
		}

		if y > maxY {
			maxY = y
		}
	}

	if maxY-minY < 30 {
		t.Fatalf("column-wrap:wrap Y span=%.1f; want a second multicol row", maxY-minY)
	}
}
