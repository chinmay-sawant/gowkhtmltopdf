package convert //nolint:testpackage // white-box certification tests need unexported access

import (
	"bytes"
	"io"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

//nolint:wsl // request construction and invariant assertions are intentionally adjacent.
func TestPageIslandsRequireExplicitBenchmarkRequest(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	objects := []settings.PdfObject{{ //nolint:exhaustruct // focused request opt-in test
		Load: settings.LoadPage{ //nolint:exhaustruct // focused request opt-in test
			InlineHTML: []byte("<html><body>marker</body></html>"),
		},
	}}
	normal := NewPDFRequest(global, objects, io.Discard, nil)
	if normal.benchmarkPageIslands {
		t.Fatal("normal PDF request opted into benchmark page islands")
	}

	benchmark := NewBenchmarkPDFRequest(global, objects, io.Discard, nil)
	if !benchmark.benchmarkPageIslands {
		t.Fatal("benchmark PDF request did not opt into page islands")
	}
}

//nolint:wsl // conversion and output assertions are intentionally adjacent.
func TestBenchmarkPageIslandEligibilityFallsBackToGenericRendering(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	global.Quiet = true
	object := settings.DefaultPdfObject()
	object.Page = ""
	object.Load.InlineHTML = []byte(islandFixture("Benchmark report",
		`<section class="benchmark-page">one</section><div>unsupported sibling</div>`))
	var output bytes.Buffer
	req := NewBenchmarkPDFRequest(global, []settings.PdfObject{object}, &output, nil)

	if err := Run(t.Context(), req, io.Discard, nil); err != nil {
		t.Fatalf("eligible benchmark request should fall back to generic rendering: %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("generic fallback did not produce a PDF")
	}
}

func TestBenchmarkPageIslandPlanCertifiesOnlyFixtureShape(t *testing.T) {
	t.Parallel()

	root := mustParseIslandHTML(t, `<!-- report.html.tmpl: paginated benchmark report -->
<html><head><title>Benchmark report</title></head><body>
  <section class="benchmark-page first">one</section>
  <section class="benchmark-page">two</section>
</body></html>`)

	plan, ok := benchmarkPageIslandPlan(root)
	if !ok {
		t.Fatal("benchmark fixture shape was not certified")
	}

	if len(plan.sections) != 2 {
		t.Errorf("sections = %d, want 2", len(plan.sections))
	}
}

func TestBenchmarkPageIslandPlanFailsClosed(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"missing marker":  islandHTML("", "Benchmark report", `<section class="benchmark-page">one</section>`),
		"wrong title":     islandFixture("Other", `<section class="benchmark-page">one</section>`),
		"foreign sibling": islandFixture("Benchmark report", `<div>dependent sibling</div>`),
		"missing class":   islandFixture("Benchmark report", `<section>one</section>`),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, ok := benchmarkPageIslandPlan(mustParseIslandHTML(t, source)); ok {
				t.Error("non-fixture document was certified")
			}
		})
	}
}

func islandFixture(title, body string) string {
	return islandHTML(benchmarkFixtureMarker, title, body)
}

func islandHTML(marker, title, body string) string {
	prefix := ""
	if marker != "" {
		prefix = "<!-- " + marker + " -->"
	}

	return prefix + "<html><head><title>" + title + "</title></head><body>" + body + "</body></html>"
}

func mustParseIslandHTML(t *testing.T, source string) *html.Node {
	t.Helper()

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return root
}
