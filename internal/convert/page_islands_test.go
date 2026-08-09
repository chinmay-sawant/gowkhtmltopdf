package convert //nolint:testpackage // white-box certification tests need unexported access

import (
	"testing"

	"gowkhtmltopdf/internal/html"
)

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
