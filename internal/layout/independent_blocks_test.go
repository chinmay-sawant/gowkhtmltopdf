package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestIndependentBlocksReportShape(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<section class="page">one</section>
		<section class="page">two</section>
		<section class="page">three</section>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.page { display: block; page-break-inside: avoid; }
		.page + .page { page-break-before: always; }
	`)}, "print", testViewport, 800)

	nodes, ok := IndependentBlocks(root, styles, 700)
	if !ok {
		t.Fatal("report-shaped sections should match")
	}

	if len(nodes) != 3 {
		t.Fatalf("candidates = %d, want 3", len(nodes))
	}

	for i, node := range nodes {
		if node == nil || node.Name != htmlSection {
			t.Fatalf("candidate %d name = %v", i, node)
		}
	}
}

func TestIndependentBlocksSpanningTable(t *testing.T) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(`<html><body><table>`)

	for range 80 {
		body.WriteString(`<tr><td>row</td></tr>`)
	}

	body.WriteString(`</table></body></html>`)

	root := mustParse(t, body.String())
	styles := resolveStyles(root, nil, "print", testViewport, 400)

	if _, ok := IndependentBlocks(root, styles, 400); ok {
		t.Fatal("tall table without page-break-inside avoid must fail closed")
	}
}

func TestIndependentBlocksFlexBody(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<section>one</section>
		<section>two</section>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		body { display: flex; }
		section { display: block; page-break-inside: avoid; page-break-before: always; }
	`)}, "print", testViewport, 800)

	if _, ok := IndependentBlocks(root, styles, 700); ok {
		t.Fatal("flex body must fail closed")
	}
}

func TestCopyPaintMetadata(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
		<p id="top">hello</p>
		<p style="page-break-before:always"><a href="#top">back</a></p>
	</body></html>`)

	meta := CopyPaintMetadata(res, 800)
	if len(meta.Locations) == 0 {
		t.Fatal("expected locations")
	}

	if !meta.HasIDs {
		t.Fatal("expected HasIDs")
	}

	if !meta.HasFragmentLinks {
		t.Fatal("expected HasFragmentLinks")
	}
}

func TestWorkspaceReusesOps(t *testing.T) {
	t.Parallel()

	workspace := &Workspace{}
	root := mustParse(t, `<html><body><p>one</p></body></html>`)
	opts := Options{Width: 400, Height: 400}

	first, err := WithWorkspace(t.Context(), root, opts, workspace)
	if err != nil {
		t.Fatal(err)
	}

	if len(first.Ops) == 0 {
		t.Fatal("expected ops")
	}

	capBefore := cap(first.Ops)
	_ = CopyPaintMetadata(first, 400)
	workspace.Release(first)

	second, err := WithWorkspace(t.Context(), root, opts, workspace)
	if err != nil {
		t.Fatal(err)
	}

	if cap(second.Ops) < capBefore {
		t.Fatalf("workspace ops cap shrank: %d -> %d", capBefore, cap(second.Ops))
	}
}

func TestIndependentBlocksUsesStylesNotClassName(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="benchmark-page">a</div>
		<div class="benchmark-page">b</div>
	</body></html>`)
	styles := resolveStyles(root, nil, "print", testViewport, 800)

	if _, ok := IndependentBlocks(root, styles, 700); ok {
		t.Fatal("class name alone must not match")
	}
}

func TestSectionChromeHashHoles(t *testing.T) {
	t.Parallel()

	a := mustParse(t, `<html><body><section><h1>Invoice 1</h1><p>SKU-1</p></section></body></html>`)
	b := mustParse(t, `<html><body><section><h1>Invoice 2</h1><p>SKU-2</p></section></body></html>`)
	stylesA := resolveStyles(a, nil, "print", testViewport, 800)
	stylesB := resolveStyles(b, nil, "print", testViewport, 800)

	secA := a.FirstChild("html").FirstChild("body").FirstChild("section")
	secB := b.FirstChild("html").FirstChild("body").FirstChild("section")

	if SectionChromeHash(secA, stylesA) == 0 {
		t.Fatal("expected non-zero hash")
	}

	if SectionChromeHash(secA, stylesA) == SectionChromeHash(secB, stylesB) {
		t.Fatal("different style pointers should not share a hash")
	}

	same := mustParse(t, `<html><body><section><h1>x</h1></section><section><h1>y</h1></section></body></html>`)
	styles := resolveStyles(same, nil, "print", testViewport, 800)
	body := same.FirstChild("html").FirstChild("body")

	var secs []*html.Node

	for _, child := range body.Children {
		if child.Type == html.ElementNode && child.Name == htmlSection {
			secs = append(secs, child)
		}
	}

	if len(secs) != 2 {
		t.Fatalf("sections = %d", len(secs))
	}

	if SectionChromeHash(secs[0], styles) != SectionChromeHash(secs[1], styles) {
		t.Fatal("same tree shape and interned styles should share a hash")
	}
}
