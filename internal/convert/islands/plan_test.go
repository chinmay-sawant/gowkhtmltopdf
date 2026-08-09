package islands_test

import (
	"testing"

	"gowkhtmltopdf/internal/convert/islands"
	"gowkhtmltopdf/internal/html"
)

func TestReleaseBenchmarkBodyChildrenKeepsCertifiedSectionsUsable(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<!-- report.html.tmpl: paginated benchmark report -->
<html><head><title>Benchmark report</title></head><body>
  <section class="benchmark-page first">one</section>
  <section class="benchmark-page">two</section>
</body></html>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	plan, ok := islands.BenchmarkPlan(root)
	if !ok {
		t.Fatal("benchmark fixture shape was not certified")
	}

	islands.ReleaseBenchmarkBodyChildren(root)

	body := root.FirstChild("html").FirstChild("body")
	if len(body.Children) != 0 {
		t.Fatalf("body children = %d, want 0 after release", len(body.Children))
	}

	if got := plan.Sections[0].TextContent(); got != "one" {
		t.Fatalf("first released section text = %q, want one", got)
	}

	if got := islands.Root(root, plan.Sections[1]).TextContent(); got != "two" {
		t.Fatalf("virtual second section text = %q, want two", got)
	}
}
