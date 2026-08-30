//nolint:testpackage,wsl,cyclop // table-layout:fixed column share proof
package layout

import (
	"testing"
)

// TestTableLayoutFixed proves table-layout:fixed is consumed: a definite
// table width plus a narrow first-row cell uses equal-share columns instead
// of auto max-content (layout_tables.go sizeTableColumns fixed flag).
func TestTableLayoutFixed(t *testing.T) {
	t.Parallel()

	const (
		tableWidth = 300.0
		long       = "supercalifragilisticexpialidocious"
	)

	fixedSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { table-layout: fixed; width: 300pt; border-collapse: collapse; border-spacing: 0; }
td { padding: 0; }
`)
	autoSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { table-layout: auto; width: 300pt; border-collapse: collapse; border-spacing: 0; }
td { padding: 0; }
`)

	src := `<html><body>
<table>
<tr><td>a</td><td>` + long + `</td></tr>
</table>
</body></html>`

	fixed := layoutHTML(t, src, fixedSheet)
	auto := layoutHTML(t, src, autoSheet)

	fixedTable := findNamedBox(fixed.root, "table")
	autoTable := findNamedBox(auto.root, "table")
	if fixedTable == nil || autoTable == nil {
		t.Fatal("missing table box")
	}

	if fixedTable.w < tableWidth-2 || fixedTable.w > tableWidth+2 {
		t.Fatalf("fixed table width=%.2f, want specified %.0fpt", fixedTable.w, tableWidth)
	}

	if len(fixedTable.rows) == 0 || len(fixedTable.rows[0]) < 2 {
		t.Fatalf("fixed table cells=%v", cellSizes(fixedTable))
	}

	if len(autoTable.rows) == 0 || len(autoTable.rows[0]) < 2 {
		t.Fatalf("auto table cells=%v", cellSizes(autoTable))
	}

	fixedFirst := fixedTable.rows[0][0].w
	autoFirst := autoTable.rows[0][0].w
	half := tableWidth / 2
	if fixedFirst < half-30 || fixedFirst > half+30 {
		t.Fatalf("fixed first-col width=%.2f, want ~%.0fpt equal share of specified width",
			fixedFirst, half)
	}

	if autoFirst >= fixedFirst-10 {
		t.Fatalf("auto first-col width=%.2f should be below fixed=%.2f (narrow first cell)",
			autoFirst, fixedFirst)
	}
}

func cellSizes(tableBox *box) []float64 {
	if tableBox == nil || len(tableBox.rows) == 0 {
		return nil
	}

	out := make([]float64, 0, len(tableBox.rows[0]))
	for _, cell := range tableBox.rows[0] {
		out = append(out, cell.w)
	}

	return out
}
