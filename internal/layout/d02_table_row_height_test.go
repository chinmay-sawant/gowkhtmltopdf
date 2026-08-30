//nolint:testpackage // probes table row box heights for fixture-56 d02-table
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestD02SurfaceTableRowHeights(t *testing.T) {
	t.Parallel()
	htmlSrc := `<!DOCTYPE html><html><body>
<section class="d02">
<table class="d02-table">
<thead><tr><th>Surface</th><th>Contract</th></tr></thead>
<tbody>
<tr><td><code>Converter</code></td><td>PDF, multi-object, <code>OnPhase</code>/<code>OnProgress</code> (final 100); no <code>Now</code> hook</td></tr>
<tr><td><code>ImageConverter</code></td><td>PNG/JPEG, single most-recent page, lazy zero-value init, no phase callbacks</td></tr>
<tr><td><code>RunPDF</code> / <code>RunImage</code></td><td>typed path - <code>Now</code> injectable for reproducible metadata</td></tr>
</tbody>
</table>
</section>
</body></html>`
	cssSrc := `
:root { --ink:#20262f; --paper:#f6f2e9; --line:#d8d1c2; --mono: ui-monospace, monospace; }
code { font-family: var(--mono); font-size: 0.92em; background: #efe9dc; border-radius: 4px; padding: 0.05em 0.3em; }
section.d02 .d02-table { width: 100%; border-collapse: collapse; margin: 0; font-size: 10.5px; }
section.d02 .d02-table th, section.d02 .d02-table td {
  border: 1px solid var(--line); padding: 5px 8px; text-align: left; vertical-align: top;
}
section.d02 .d02-table th { background: var(--ink); color: var(--paper); font-family: var(--mono); font-size: 9.5px; }
section.d02 .d02-table code { font-size: 10px; white-space: nowrap; }
`
	assertSurfaceContractGaps(t, layoutHTML(t, htmlSrc, sheet(t, cssSrc)), "minimal")
}

func TestD02SurfaceTableRowHeightsFullFixture(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(filepath.Join(root, "fixture-56-architecture-diagram.html"))
	if err != nil {
		t.Fatal(err)
	}
	cssBytes, err := os.ReadFile(filepath.Join(root, "fixture-56-architecture-diagram.css"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(string(htmlBytes))
	if err != nil {
		t.Fatal(err)
	}
	cssSheet, err := css.Parse(string(cssBytes))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(doc, Options{
		Width: 527.2, Height: 20000, Sheets: []*css.Stylesheet{cssSheet},
		Background: true, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertSurfaceContractGaps(t, res, "full-fixture-tall")
}

func assertSurfaceContractGaps(t *testing.T, res *Result, label string) {
	t.Helper()

	// Anchor on the Surface/Contract header pair, then take the next Converter /
	// ImageConverter / RunPDF labels within a short vertical window.
	var surfaceY float64
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "Surface" {
			surfaceY = op.Y
			break
		}
	}
	if surfaceY == 0 {
		t.Fatalf("%s: Surface header not found", label)
	}

	var convY, imgY, runY float64
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if op.Y < surfaceY-1 || op.Y > surfaceY+120 {
			continue
		}
		switch {
		case op.Text == "Converter":
			convY = op.Y
		case op.Text == "ImageConverter" || strings.HasPrefix(op.Text, "ImageConverter"):
			imgY = op.Y
		case op.Text == "RunPDF" || strings.HasPrefix(op.Text, "RunPDF"):
			runY = op.Y
		}
	}
	if convY == 0 || imgY == 0 || runY == 0 {
		t.Fatalf("%s: missing labels near Surface(y=%.2f) conv=%v img=%v run=%v",
			label, surfaceY, convY, imgY, runY)
	}
	gap1 := imgY - convY
	gap2 := runY - imgY
	t.Logf("%s gaps Converter->Image=%.2f Image->RunPDF=%.2f (surface=%.2f conv=%.2f img=%.2f run=%.2f)",
		label, gap1, gap2, surfaceY, convY, imgY, runY)
	if gap2 > gap1*1.6 {
		t.Fatalf("%s: ImageConverter row inflated: gap1=%.2f gap2=%.2f", label, gap1, gap2)
	}
	if gap1 < 14 {
		t.Fatalf("%s: Converter/ImageConverter crush: gap1=%.2f", label, gap1)
	}
}

func TestD02SurfaceTableRowHeightsAfterPagination(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(filepath.Join(root, "fixture-56-architecture-diagram.html"))
	if err != nil {
		t.Fatal(err)
	}
	cssBytes, err := os.ReadFile(filepath.Join(root, "fixture-56-architecture-diagram.css"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(string(htmlBytes))
	if err != nil {
		t.Fatal(err)
	}
	cssSheet, err := css.Parse(string(cssBytes))
	if err != nil {
		t.Fatal(err)
	}

	const (
		pageW    = 595.28
		pageH    = 841.89
		marginMM = 12.0
	)
	margin := marginMM * 72 / 25.4
	contentW := pageW - 2*margin
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{
		Width: contentW, Height: contentH, Sheets: []*css.Stylesheet{cssSheet},
		Background: true, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Run the same pagination fixpoint Paint uses (without writing a PDF).
	paginateOps(res, contentH)

	assertSurfaceContractGaps(t, res, "after-paginate")
}
