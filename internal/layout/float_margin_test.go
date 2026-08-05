package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestFloatMarginBoxExclusion: in-flow text must clear a float's horizontal
// margin (margin box), not sit flush against the border box — wiki infobox
// uses margin-left:1em on float:right.
func TestFloatMarginBoxExclusion(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
.box {
  float: right;
  width: 100pt;
  margin-left: 24pt;
  background: #ccc;
  border: 1pt solid #000;
}
p { margin: 0; }
`)
	htmlSrc := `<html><body>
<div class="box">FRAME</div>
<p>Lead paragraph words that wrap beside the floated frame and must leave the margin gap clear.</p>
</body></html>`
	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}
	const pageW = 400.0
	res, err := Layout(root, Options{
		Width: pageW, Height: 400, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var frameX float64 = pageW
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "FRAME") {
			if op.X < frameX {
				frameX = op.X
			}
		}
	}
	if frameX > pageW-40 {
		t.Fatal("float frame text not found on the right")
	}
	// Exclusion edge = frame border-box left − margin-left (24pt).
	limit := frameX - 24
	for _, op := range res.Ops {
		if op.Kind != OpText || strings.Contains(op.Text, "FRAME") {
			continue
		}
		right := op.X + op.W
		if right > limit+1.5 {
			t.Fatalf("in-flow text %q ends at x=%.1f, enters float margin (limit %.1f, frame %.1f)",
				op.Text, right, limit, frameX)
		}
	}
}
