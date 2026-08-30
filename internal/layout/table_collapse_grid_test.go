//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"testing"
)

// border-collapse grids must stroke internal row boundaries (not only the
// outer perimeter). Missing internal rules show up as "broken" wiki tables.
func TestCollapsedTableInternalBorders(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 200pt; }
td, th { border: 1px solid #999; padding: 2pt; }
`)
	res := layoutHTML(t, `<html><body>
<table>
<tr><th>Year</th><th>Title</th><th>Artist</th></tr>
<tr><td>2009</td><td>Song A</td><td>Act</td></tr>
<tr><td>2018</td><td>Song B</td><td>Act</td></tr>
<tr><td>2020</td><td>Song C</td><td>Act</td></tr>
</table>
</body></html>`, cssSheet)

	var hlines []float64

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 50 {
			hlines = append(hlines, op.Y)
		}
	}
	// 4 rows → 5 horizontal rules (top + 3 internals + bottom).
	if len(hlines) < 5 {
		t.Fatalf("horizontal grid lines=%d, want ≥5 (got Y=%v)", len(hlines), hlines)
	}
}

// border-collapse alone must not create visible borders. A collapsed table
// with no table or cell border declarations has no grid to paint.
func TestCollapsedTableWithoutBordersDoesNotPaintGrid(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 200pt; }
`)
	res := layoutHTML(t, `<html><body>
<table><tr><th>Item</th><th>Qty</th></tr>
<tr><td>Widget A</td><td>2</td></tr></table>
</body></html>`, cssSheet)

	for _, op := range res.Ops {
		if op.Kind == OpLine {
			t.Fatalf("collapsed table without borders painted line: %#v", op)
		}
	}
}

// Collapsed grids must honor the sides actually declared by the cells. A
// bottom-only dotted border must not become a full solid-looking grid.
func TestCollapsedTableUsesDeclaredBorderSides(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 240pt; }
td { border: none; border-bottom: 1px dotted #bbb; padding: 2pt; }
`)
	res := layoutHTML(t, `<html><body>
<table><tr><td>A</td><td>1</td></tr>
<tr><td>B</td><td>2</td></tr>
<tr><td>C</td><td>3</td></tr></table>
</body></html>`, cssSheet)

	horizontal, vertical := 0, 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine {
			continue
		}

		if paintOp.H == 0 && paintOp.W > 0 {
			horizontal++
		}

		if paintOp.W == 0 && paintOp.H > 0 {
			vertical++
		}
	}

	if horizontal == 0 {
		t.Fatal("bottom border did not produce horizontal lines")
	}

	if vertical != 0 {
		t.Fatalf("bottom-only border produced %d vertical lines", vertical)
	}
}

func TestCollapsedTableBorderConflictWiderWins(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 200pt; }
.thin { border-right: 1pt solid #000; }
.thick { border-left: 4pt solid #000; }
`)
	res := layoutHTML(t, `<html><body>
<table><tr><td class="thin">A</td><td class="thick">B</td></tr></table>
</body></html>`, cssSheet)

	var vlines []Op

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.W == 0 && op.H > 0 {
			vlines = append(vlines, op)
		}
	}

	if len(vlines) == 0 {
		t.Fatal("no vertical border lines painted")
	}

	foundThick := false

	for _, line := range vlines {
		if near(line.Width, 4) {
			foundThick = true

			break
		}
	}

	if !foundThick {
		t.Fatalf("expected 4pt winning border line in vertical borders, got: %+v", vlines)
	}
}

//nolint:lll // test CSS strings and assertion messages
func TestTableRowVisibilityCollapseReducesHeight(t *testing.T) {
	t.Parallel()

	sheetNormal := sheet(t,
		`body { margin: 0; font-size: 10pt; } table { width: 200pt; } td { padding: 10pt; }`)
	sheetHidden := sheet(t,
		`body { margin: 0; font-size: 10pt; } table { width: 200pt; } td { padding: 10pt; } tr.target { visibility: hidden; }`)
	sheetCollapse := sheet(t,
		`body { margin: 0; font-size: 10pt; } table { width: 200pt; } td { padding: 10pt; } tr.target { visibility: collapse; }`)

	htmlTable := `<html><body><table>
<tr class="target"><td>Row 1 Content</td></tr>
<tr><td>Row 2 Content</td></tr>
</table></body></html>`

	resNormal := layoutHTML(t, htmlTable, sheetNormal)
	resHidden := layoutHTML(t, htmlTable, sheetHidden)
	resCollapse := layoutHTML(t, htmlTable, sheetCollapse)

	tableNormal := findBox(t, resNormal, "table")
	tableHidden := findBox(t, resHidden, "table")
	tableCollapse := findBox(t, resCollapse, "table")

	// visibility: hidden keeps layout size
	if !near(tableNormal.height, tableHidden.height) {
		t.Fatalf("visibility: hidden changed table height: normal=%.1f, hidden=%.1f",
			tableNormal.height, tableHidden.height)
	}

	// visibility: collapse removes row from layout and reduces table height
	if tableCollapse.height >= tableNormal.height {
		t.Fatalf("visibility: collapse should reduce table height: normal=%.1f, collapse=%.1f",
			tableNormal.height, tableCollapse.height)
	}
}
