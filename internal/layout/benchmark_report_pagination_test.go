//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"fmt"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// TestBenchmarkReportRowsStayAligned exercises the same five forced sections
// as the benchmark report. The fifth section used to snap row 12's text during
// provisional pagination while leaving its collapsed-table chrome behind.
func TestBenchmarkReportRowsStayAligned(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	const style = `
body { color: #172033; font-family: sans-serif; font-size: 9pt; margin: 0; }
.benchmark-page { page-break-before: always; page-break-inside: avoid; padding: 2mm 0; }
.benchmark-page.first { page-break-before: auto; }
h1 { color: #174a7c; font-size: 16pt; margin: 0 0 3mm; }
p { margin: 0 0 3mm; }
table { border-collapse: collapse; width: 100%; }
tr { page-break-inside: avoid; break-inside: avoid; }
th, td { border: 1px solid #a8b5c5; padding: 1.5mm 2mm; white-space: nowrap; }
th { background: #e6eef7; text-align: left; }
td.amount { text-align: right; }
`

	var src strings.Builder

	src.WriteString(`<html><head><style>` + style + `</style></head><body>`)

	for page := 1; page <= 5; page++ {
		className := "benchmark-page"
		if page == 1 {
			className += " first"
		}

		fmt.Fprintf(&src, `<section class="%s"><h1>Benchmark report — page %d</h1>`+
			`<p>Representative invoice and operations data for the full HTML-to-PDF pipeline.</p>`+
			`<table><thead><tr><th>Line</th><th>SKU</th><th>Description</th>`+
			`<th>Quantity</th><th>Amount</th></tr></thead><tbody>`, className, page)

		for row := 1; row <= 20; row++ {
			fmt.Fprintf(&src, `<tr><td>%d</td><td>SKU-%03d-%03d</td>`+
				`<td>Platform operations and support service %d</td><td>%d</td><td class="amount">%d.%02d</td></tr>`,
				row, page, row, row, (row+page-1)%7+1, page*row, (page+row-1)%100)
		}

		src.WriteString(`</tbody></table></section>`)
	}

	src.WriteString(`</body></html>`)

	root, err := html.Parse(src.String())
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(style)
	if err != nil {
		t.Fatal(err)
	}

	const mm = 72.0 / 25.4

	pageW, pageH := 595.28, 841.89
	margin := 10 * mm
	contentH := pageH - 2*margin

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: pageW - 2*margin, Height: contentH,
		Sheets: []*css.Stylesheet{sheet}, Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	if got := doc.PageCount(); got != 5 {
		t.Fatalf("page count = %d, want 5", got)
	}

	var tables []*box

	var walk func(*box)
	walk = func(b *box) {
		if b.kind == "table" {
			tables = append(tables, b)
		}

		for _, child := range b.children {
			walk(child)
		}
	}
	walk(res.root)

	if len(tables) != 5 {
		t.Fatalf("tables = %d, want 5", len(tables))
	}

	row := tables[4].rows[12] // thead row plus body rows 1..20
	rowTop, rowBottom := row[0].y, row[0].y+row[0].height

	for _, cell := range row[1:] {
		if cell.y < rowTop {
			rowTop = cell.y
		}

		if cell.y+cell.height > rowBottom {
			rowBottom = cell.y + cell.height
		}
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || !strings.Contains(paintOp.Text, "SKU-005-012") {
			continue
		}

		if paintOp.Y < rowTop || paintOp.Y >= rowBottom {
			t.Fatalf("row 12 text y=%.3f outside row box [%.3f, %.3f]", paintOp.Y, rowTop, rowBottom)
		}

		return
	}

	t.Fatal("row 12 text operation not found")
}
