package convert //nolint:testpackage // white-box tests need unexported access

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tenPageTableReportHTML builds a ~10-page invoice report: ten sections, each
// with a table of 40 realistic line-item rows and a repeated <thead>, using
// page-break-before sections (same pattern as golden fixture-03). The first
// section carries no break so the document starts on page 1 without a blank
// page; white-space: nowrap keeps one line item per row. All content is
// generated in-test; nothing is committed.
func tenPageTableReportHTML() string {
	var buf strings.Builder

	buf.WriteString(`<!DOCTYPE html><html><head><style>
body { font-family: sans-serif; font-size: 9pt; color: #111; }
h2 { font-size: 12pt; color: #1a3d6d; }
table { width: 100%; border-collapse: collapse; }
th, td { border: 1px solid #bbb; padding: 2px 4px; white-space: nowrap; }
thead th { background-color: #e8eef5; }
.section { page-break-before: always; }
.num { text-align: right; }
</style></head><body>`)

	const sections, items = 10, 40
	for str := 1; str <= sections; str++ {
		cls := "section"
		if str == 1 {
			cls = "first"
		}

		fmt.Fprintf(&buf, `<div class="%s"><h2>Invoice %d - line items</h2>`, cls, str)
		buf.WriteString(
			`<table><thead><tr><th>#</th><th>Item</th><th>SKU</th><th class="num">Qty</th><th class="num">Unit</th><th class="num">Total</th></tr></thead><tbody>`, //nolint:lll // generated HTML row
		)

		for itemIdx := 1; itemIdx <= items; itemIdx++ {
			qty := (itemIdx*3)%7 + 1
			unit := 12.5*float64(itemIdx%9+1) + float64(itemIdx%100)/100.0
			fmt.Fprintf(
				&buf,
				`<tr><td>%d</td><td>Line item %d - consulting service %s</td><td>SKU-%04d</td><td class="num">%d</td><td class="num">%.2f</td><td class="num">%.2f</td></tr>`, //nolint:lll // generated HTML row
				itemIdx, itemIdx, descriptionWord(itemIdx), itemIdx, qty, unit, unit*float64(qty),
			)
		}

		buf.WriteString(`</tbody></table></div>`)
	}

	buf.WriteString(`</body></html>`)

	return buf.String()
}

// descriptionWord yields a short realistic descriptor for line-item text.
func descriptionWord(itemIdx int) string {
	words := []string{
		"setup", "deployment", "maintenance", "migration", "support",
		"training", "review", "integration", "consulting",
	}

	return words[itemIdx%len(words)]
}

// TestTenPageTableReportPerformance is the Phase 9.3 perf gate: a ~10-page
// invoice table report (10 sections x 40 line-item rows, repeated <thead>,
// page-break-before sections) converted through the FULL RunPDF pipeline
// (load -> parse -> style -> layout -> paginate -> paint -> assemble -> write).
// It runs twice (cold, then warm) and asserts a generous per-run budget so CI
// stays stable. Skipped in -short mode.
//
// Command:  go test ./internal/convert -run TestTenPageTableReportPerformance -v
// Machine:  go1.26.4 linux/amd64, Linux x86_64, 13th Gen Intel(R) Core(TM)
//
//	i7-13700HX (24 threads), 2026-08-03
//
// Measured (three runs; output bytes deterministic):
//
//	cold run:  120.3ms | 149.1ms | 149.1ms   (mean ~140ms)
//	warm run:  203.2ms | 131.8ms | 131.8ms   (mean ~156ms)
//	PDF bytes: 96341, 10 pages (identical across runs)
func TestTenPageTableReportPerformance(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("perf budget test skipped in -short mode")
	}
	// Budget per run: 5s. Generous on purpose: the gate catches order-of-
	// magnitude regressions (e.g. an accidental O(n^2) in layout), not noise.
	const budget = 5 * time.Second

	cmd, _ := newCommand(t, tenPageTableReportHTML(), filepath.Join(t.TempDir(), "out.pdf"))

	var sizes []int64

	for run := 1; run <= 2; run++ {
		start := time.Now()

		if err := RunPDF(cmd, io.Discard); err != nil {
			t.Fatalf("run %d: RunPDF: %v", run, err)
		}

		dur := time.Since(start)

		data, err := os.ReadFile(cmd.Output)
		if err != nil {
			t.Fatalf("run %d: read output: %v", run, err)
		}

		sizes = append(sizes, int64(len(data)))
		t.Logf("run %d (cold=%v): full pipeline %v, %d bytes, %d pages",
			run, run == 1, dur, len(data), pageCount(data))

		if dur >= budget {
			t.Errorf("run %d took %v, want < %v", run, dur, budget)
		}
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if n := pageCount(data); n < 10 {
		t.Errorf("pages = %d, want >= 10", n)
	}

	t.Logf("pdf byte sizes: cold=%d warm=%d", sizes[0], sizes[1])
}
