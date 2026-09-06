//nolint:all // targeted intrinsic-width and direction probes
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestInlineBlockNestedTextMaxContent(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.outer { display: inline-block; border: 1pt solid #333; background: #fce8e8; }
.inner { margin-inline: 14pt; background: #dfd; border: 1pt dashed #080; padding: 4pt; }
`)
	root := mustParse(t, `<html><body>
<div class="outer"><div class="inner">inline 14px</div></div>
</body></html>`)
	res, err := Layout(root, Options{
		Width: 500, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	outer := findElementByClass(root, "outer")
	var outerBox *box
	var walkBoxes func(*box)
	walkBoxes = func(boxNode *box) {
		if boxNode == nil {
			return
		}
		if boxNode.node == outer {
			outerBox = boxNode
		}
		for _, child := range boxNode.children {
			walkBoxes(child)
		}
	}
	walkBoxes(res.root)
	if outerBox == nil {
		t.Fatal("missing outer box")
	}
	t.Logf("outer.w=%.2f", outerBox.w)
	if outerBox.w < 80 {
		t.Fatalf("outer.w=%.2f too narrow for margin-inline child", outerBox.w)
	}
	joined := ""
	for _, op := range res.Ops {
		if op.Kind == OpText {
			joined += op.Text
		}
	}
	if !strings.Contains(joined, "inline") || !strings.Contains(joined, "14px") {
		t.Fatalf("text=%q", joined)
	}
}

func TestDirectionRTLKeepsLatinOrder(t *testing.T) {
	t.Parallel()
	res := layoutHTML(t, `<html><body>
<div style="direction:rtl;width:200pt;font-size:14pt;">ABC 123</div>
</body></html>`, sheet(t, `body{margin:0}`))
	var texts []string
	var startX []float64
	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, op.Text)
			startX = append(startX, op.X)
		}
	}
	joined := strings.Join(texts, "")
	if strings.Contains(joined, "123ABC") || strings.HasPrefix(joined, "123") {
		t.Fatalf("rtl latin should keep logical order, got %v", texts)
	}
	if len(startX) == 0 || startX[0] < 100 {
		t.Fatalf("expected right-aligned start x>=100, got %v", startX)
	}
}
