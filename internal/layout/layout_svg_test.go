//nolint:all // inline SVG paint probes
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestInlineSVGFillPaints(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">
  <rect x="2" y="2" width="60" height="24" style="fill:#0a7;stroke:#333;"/>
</svg>
</body></html>`, sheet(t, `body { margin: 0; }`))

	var images int
	for _, op := range res.Ops {
		if op.Kind == OpImage && len(op.Image) > 0 {
			images++
		}
	}
	if images == 0 {
		t.Fatal("inline <svg> with fill produced no OpImage")
	}
}

func TestInlineSVGFillOpacityPaints(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">
  <rect x="2" y="2" width="60" height="24" style="fill-opacity:0.5;fill:#cde;stroke:#333;"/>
</svg>
</body></html>`, sheet(t, `body{margin:0}`))

	var images int
	for _, op := range res.Ops {
		if op.Kind == OpImage && len(op.Image) > 0 {
			images++
		}
	}
	if images == 0 {
		t.Fatal("inline <svg> with fill-opacity produced no OpImage")
	}
}

func TestSerializeInlineSVGBakesFill(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<svg width="10" height="10"><rect style="fill:#0a7;fill-opacity:0.5"/></svg>
</body></html>`)
	sheets := []*css.Stylesheet{sheet(t, `body{margin:0}`)}
	styles := resolveStyles(root, sheets, "print", 500, 800)
	eng := &engine{styles: styles, scale: 1} //nolint:exhaustruct
	var svgNode *html.Node
	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})
	if svgNode == nil {
		t.Fatal("svg missing")
	}
	out := string(eng.serializeInlineSVG(svgNode))
	if !strings.Contains(out, `fill-opacity="0.5"`) {
		t.Fatalf("missing fill-opacity in %s", out)
	}
	if !strings.Contains(out, `fill="`) {
		t.Fatalf("missing fill in %s", out)
	}
}
