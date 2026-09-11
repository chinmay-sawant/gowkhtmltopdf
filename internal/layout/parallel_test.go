package layout

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const parallelReportCSS = `
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

func parallelReportHTML(sections int) string {
	var src strings.Builder

	src.WriteString(`<html><body>`)

	for page := 1; page <= sections; page++ {
		className := "benchmark-page"
		if page == 1 {
			className += " first"
		}

		fmt.Fprintf(&src, `<section class="%s"><h1>Benchmark report - page %d</h1>`+
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

	return src.String()
}

func parallelTestRoot(t *testing.T, markup string) (*html.Node, []*css.Stylesheet) {
	t.Helper()

	root, err := html.Parse(markup)
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(parallelReportCSS)
	if err != nil {
		t.Fatal(err)
	}

	return root, []*css.Stylesheet{sheet}
}

const (
	parallelPageW = 595.28
	parallelPageH = 841.89
)

func parallelTestOptions(sheets []*css.Stylesheet) Options {
	const mmPt = 72.0 / 25.4

	return Options{
		Width:      parallelPageW - 2*10*mmPt,
		Height:     parallelPageH - 2*10*mmPt,
		Sheets:     sheets,
		Media:      "print",
		Background: true,
	}
}

func parallelTestPaintOptions() PaintOptions {
	const mmPt = 72.0 / 25.4

	return PaintOptions{
		PageWidth:  parallelPageW,
		PageHeight: parallelPageH,
		MarginTop:  10 * mmPt, MarginBottom: 10 * mmPt,
		MarginLeft: 10 * mmPt, MarginRight: 10 * mmPt,
	}
}

// parallelPDFDate normalizes the PDF timestamp for byte comparisons.
var parallelPDFDate = regexp.MustCompile(`D:[0-9]{14}Z`)

// parallelPaintBytes paints a result into a PDF and returns date-normalized
// bytes and the page count.
func parallelPaintBytes(t *testing.T, res *Result) ([]byte, int) {
	t.Helper()

	doc := pdf.NewDocument()

	if err := Paint(doc, res, parallelTestPaintOptions()); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	return parallelPDFDate.ReplaceAll(buf.Bytes(), []byte("D:STAMPZ")), doc.PageCount()
}

// requireParallelOpsEqual compares two display lists op for op.
//
//nolint:cyclop // field-by-field differential probe
func requireParallelOpsEqual(t *testing.T, label string, serial, parallel *Result) {
	t.Helper()

	if len(serial.Ops) != len(parallel.Ops) {
		t.Fatalf("%s: op count serial=%d parallel=%d", label, len(serial.Ops), len(parallel.Ops))
	}

	for idx := range serial.Ops {
		serialOp, parallelOp := &serial.Ops[idx], &parallel.Ops[idx]

		if serialOp.Kind != parallelOp.Kind || serialOp.ID != parallelOp.ID {
			t.Fatalf("%s: op %d kind/id serial=(%d,%d) parallel=(%d,%d)",
				label, idx, serialOp.Kind, serialOp.ID, parallelOp.Kind, parallelOp.ID)
		}

		if serialOp.X != parallelOp.X || serialOp.Y != parallelOp.Y ||
			serialOp.W != parallelOp.W || serialOp.H != parallelOp.H {
			t.Fatalf("%s: op %d geometry serial=(%.4f,%.4f,%.4f,%.4f) parallel=(%.4f,%.4f,%.4f,%.4f)",
				label, idx, serialOp.X, serialOp.Y, serialOp.W, serialOp.H, parallelOp.X, parallelOp.Y, parallelOp.W, parallelOp.H)
		}

		if serialOp.Text != parallelOp.Text || serialOp.Font != parallelOp.Font || serialOp.Size != parallelOp.Size {
			t.Fatalf("%s: op %d text/font/size serial=(%q,%p,%.3f) parallel=(%q,%p,%.3f)",
				label, idx, serialOp.Text, serialOp.Font, serialOp.Size, parallelOp.Text, parallelOp.Font, parallelOp.Size)
		}

		if serialOp.R != parallelOp.R || serialOp.G != parallelOp.G ||
			serialOp.B != parallelOp.B || serialOp.Alpha != parallelOp.Alpha ||
			serialOp.Width != parallelOp.Width {
			t.Fatalf("%s: op %d paint serial=(%.4f,%.4f,%.4f,%.4f,%.4f) parallel=(%.4f,%.4f,%.4f,%.4f,%.4f)",
				label, idx, serialOp.R, serialOp.G, serialOp.B, serialOp.Alpha, serialOp.Width,
				parallelOp.R, parallelOp.G, parallelOp.B, parallelOp.Alpha, parallelOp.Width)
		}
	}
}

// requireParallelBoxesEqual compares box geometry and op ranges for every node
// that appears in both results.
func requireParallelBoxesEqual(t *testing.T, label string, serial, parallel *Result) {
	t.Helper()

	serialByNode := make(map[*html.Node]*box)

	for _, boxNode := range serial.boxes {
		if boxNode.node != nil {
			serialByNode[boxNode.node] = boxNode
		}
	}

	compared := 0

	for _, boxNode := range parallel.boxes {
		if boxNode.node == nil {
			continue
		}

		want, ok := serialByNode[boxNode.node]
		if !ok {
			t.Fatalf("%s: parallel box node %p not in serial boxes", label, boxNode.node)
		}

		compared++

		if want.y != boxNode.y || want.opStart != boxNode.opStart || want.opEnd != boxNode.opEnd {
			t.Fatalf("%s: node %s box serial=(y%.4f,%d..%d) parallel=(y%.4f,%d..%d)",
				label, boxNode.node.Name, want.y, want.opStart, want.opEnd,
				boxNode.y, boxNode.opStart, boxNode.opEnd)
		}
	}

	if compared == 0 {
		t.Fatalf("%s: no boxes compared", label)
	}
}

// TestParallelTwoSectionEquivalence is the P0 differential: two
// benchmark-shaped sections must produce an identical display list and
// byte-identical PDF at every worker count.
func TestParallelTwoSectionEquivalence(t *testing.T) {
	t.Parallel()

	root, sheets := parallelTestRoot(t, parallelReportHTML(2))
	opts := parallelTestOptions(sheets)

	// One parse feeds both arms so node pointers are comparable. Compare the
	// un-paginated display lists first: Paint mutates op Y in place.
	serial, err := LayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatal(err)
	}

	parallels := make([]*Result, 0, 3)

	for _, workers := range []int{1, 2, 8} {
		parallel, err := ParallelLayout(t.Context(), root, opts, workers)
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}

		requireParallelOpsEqual(t, fmt.Sprintf("workers=%d", workers), serial, parallel)
		requireParallelBoxesEqual(t, fmt.Sprintf("workers=%d", workers), serial, parallel)

		parallels = append(parallels, parallel)
	}

	serialBytes, serialPages := parallelPaintBytes(t, serial)

	if serialPages != 2 {
		t.Fatalf("serial pages=%d want 2", serialPages)
	}

	for idx, parallel := range parallels {
		parallelBytes, parallelPages := parallelPaintBytes(t, parallel)
		if parallelPages != serialPages {
			t.Fatalf("parallel[%d] pages=%d want %d", idx, parallelPages, serialPages)
		}

		if !bytes.Equal(serialBytes, parallelBytes) {
			t.Fatalf("parallel[%d] PDF bytes differ: %d vs %d", idx, len(serialBytes), len(parallelBytes))
		}
	}
}

// TestParallelDetectorRejectsCoupling pins that the detector falls back when a
// candidate cannot be built in isolation.
func TestParallelDetectorRejectsCoupling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		markup string
	}{
		{
			name: "body margin",
			markup: `<html><body style="margin: 4pt">` +
				`<section class="benchmark-page first">a</section>` +
				`<section class="benchmark-page">b</section></body></html>`,
		},
		{
			name: "floating child",
			markup: `<html><body><section class="benchmark-page first">` +
				`<div style="float: left">a</div></section>` +
				`<section class="benchmark-page">b</section></body></html>`,
		},
		{
			name: "absolute child",
			markup: `<html><body><section class="benchmark-page first">` +
				`<div style="position: absolute">a</div></section>` +
				`<section class="benchmark-page">b</section></body></html>`,
		},
		{
			name: "counter",
			markup: `<html><body><section class="benchmark-page first" ` +
				`style="counter-reset: n">a</section>` +
				`<section class="benchmark-page">b</section></body></html>`,
		},
		{
			name: "transform on section",
			markup: `<html><body><section class="benchmark-page first" ` +
				`style="transform: translate(1px,2px)">a</section>` +
				`<section class="benchmark-page">b</section></body></html>`,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, sheets := parallelTestRoot(t, testCase.markup)
			opts := parallelTestOptions(sheets)

			styles, containers, err := resolveStylesForLayoutContext(t.Context(), root, opts)
			if err != nil {
				t.Fatal(err)
			}

			if plan := detectParallelSections(root, styles, opts, containers); plan != nil {
				t.Fatalf("detector certified a coupled document (%s)", testCase.name)
			}
		})
	}
}
