package layout

import (
	"testing"
)

// Empty <tr></tr> must not invent a bordered band above real headers.
func TestEmptyTableRowCollapsed(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 200pt; }
td, th { border: 1px solid #999; padding: 2pt; }
`)
	res := layoutHTML(t, `<html><body>
<table>
<tr></tr>
<tr><th>Year</th><th>Title</th></tr>
<tr><td>2007</td><td>Show</td></tr>
</table>
</body></html>`, cssSheet)

	var tblBox *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.kind == "table" {
			tblBox = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if tblBox == nil {
		t.Fatal("no table")
	}

	if len(tblBox.rows) != 2 {
		t.Fatalf("rows=%d, want 2 (empty tr stripped)", len(tblBox.rows))
	}

	if tblBox.headerRows != 1 {
		t.Fatalf("headerRows=%d, want 1 after stripping empty leading tr", tblBox.headerRows)
	}
	// First painted row is the Year header — no tall empty band above it.
	if len(tblBox.rows[0]) == 0 || tblBox.rows[0][0].height <= 0 {
		t.Fatal("header row missing height")
	}

	hdrH := tblBox.rows[0][0].height
	if hdrH > 28 {
		t.Fatalf("header row h=%.1f, want compact single-line (~<=28pt)", hdrH)
	}

	// Horizontal grid: top of header, mid, bottom — no extra rule for empty tr.
	var hlines []float64

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 30 {
			hlines = append(hlines, op.Y)
		}
	}
	// 2 rows → 3 horizontal bands (top + internal + bottom), each may be
	// multi-segment; unique Y count should be 3.
	uniq := map[float64]bool{}
	for _, y := range hlines {
		// quantize
		uniq[float64(int(y*10+0.5))/10] = true
	}

	if len(uniq) < 3 {
		t.Fatalf("horizontal Y bands=%d, want ≥3 (got %v)", len(uniq), hlines)
	}

	if len(uniq) > 4 {
		t.Fatalf("horizontal Y bands=%d, want ≤4 (phantom empty row?) ys=%v", len(uniq), hlines)
	}
}

// Padding-only empty cells collapse to zero row height (no ink).
func TestPaddingOnlyEmptyRowCollapsed(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; }
td, th { border: 1px solid #999; padding: 4pt; }
`)
	res := layoutHTML(t, `<html><body>
<table>
<tr><td></td><td></td></tr>
<tr><td>a</td><td>b</td></tr>
</table>
</body></html>`, cssSheet)

	var tblBox *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.kind == "table" {
			tblBox = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if tblBox == nil {
		t.Fatal("no table")
	}

	if len(tblBox.rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(tblBox.rows))
	}
	// First row cells should have been collapsed to ~0 height.
	if len(tblBox.rows[0]) > 0 && tblBox.rows[0][0].height > 0.5 {
		t.Fatalf("empty row cell h=%.2f, want ~0 (collapsed)", tblBox.rows[0][0].height)
	}

	if len(tblBox.rows[1]) == 0 || tblBox.rows[1][0].height < 8 {
		t.Fatalf("data row missing height")
	}
	// Table height should be approximately one data row, not two.
	if tblBox.height > tblBox.rows[1][0].height+4 {
		t.Fatalf("table h=%.1f >> data row h=%.1f (empty row not collapsed)", tblBox.height, tblBox.rows[1][0].height)
	}
}

// Leading all-th detection still works when a blank tr precedes the header.
func TestLeadingTHAfterEmptyRow(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
table { border-collapse: collapse; }
th, td { border: 1px solid #ccc; padding: 2pt; }
`)
	res := layoutHTML(t, `<html><body>
<table>
<tr></tr>
<tr><th>A</th><th>B</th></tr>
<tr><td>1</td><td>2</td></tr>
</table>
</body></html>`, cssSheet)

	var tblBox *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.kind == "table" {
			tblBox = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if tblBox == nil {
		t.Fatal("no table")
	}

	if tblBox.headerRows != 1 {
		t.Fatalf("headerRows=%d, want 1", tblBox.headerRows)
	}
}

// Normal data tables keep non-empty row heights and internal grid lines.
func TestNonEmptyRowsNotCollapsed(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
table { border-collapse: collapse; width: 200pt; }
td, th { border: 1px solid #999; padding: 2pt; }
`)
	res := layoutHTML(t, `<html><body>
<table>
<tr><th>Year</th><th>Title</th><th>Artist</th></tr>
<tr><td>2009</td><td>Song A</td><td>Act</td></tr>
<tr><td>2018</td><td>Song B</td><td>Act</td></tr>
</table>
</body></html>`, cssSheet)

	var hlines []float64

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 50 {
			hlines = append(hlines, op.Y)
		}
	}
	// 3 rows → ≥4 horizontal rules (top + 2 internals + bottom).
	if len(hlines) < 4 {
		t.Fatalf("horizontal grid lines=%d, want ≥4 (got Y=%v)", len(hlines), hlines)
	}
}
