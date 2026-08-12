package islands_test

import (
	"testing"

	"gowkhtmltopdf/internal/convert/islands"
	"gowkhtmltopdf/internal/html"
)

//nolint:wsl // source and virtual-tree invariants are checked together.
func TestRootClonesSectionWithOwnedParentPointers(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<!-- report.html.tmpl: paginated benchmark report -->
<html><head><title>Benchmark report</title></head><body>
  <section class="benchmark-page first"><div><span>one</span></div></section>
  <section class="benchmark-page"><div><span>two</span></div></section>
</body></html>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	plan, ok := islands.BenchmarkPlan(root)
	if !ok {
		t.Fatal("benchmark fixture shape was not certified")
	}

	virtual := islands.Root(root, plan.Sections[1])
	if got := virtual.TextContent(); got != "two" {
		t.Fatalf("virtual section text = %q, want two", got)
	}

	virtual.Walk(func(node *html.Node) {
		for _, child := range node.Children {
			if child.Parent != node {
				t.Errorf("child %q parent = %p, want owner %p", child.Name, child.Parent, node)
			}
		}
	})

	if got := plan.Sections[1].TextContent(); got != "two" {
		t.Fatalf("source section text = %q, want two", got)
	}
	if sourceChild := plan.Sections[1].Children[0]; sourceChild.Parent != plan.Sections[1] {
		t.Fatalf("source child parent changed to %p, want original section %p", sourceChild.Parent, plan.Sections[1])
	}
}
