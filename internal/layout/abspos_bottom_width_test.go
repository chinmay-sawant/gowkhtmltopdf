//nolint:all
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestAbsposBottomLeftShrinkWidth(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><body style="font-size:9.5pt">
<div style="position:relative;height:40px;border:1px dashed #888;width:200px;">
  <div class="pill" style="position:absolute;bottom:4px;left:4px;background:#fd8;padding:2px 6px;">pos</div>
</div>
</body></html>`)
	res, err := Layout(root, Options{
		Width: 500, Height: 200, Media: "print", Zoom: 0.8,
		Sheets: []*css.Stylesheet{sheet(t, `body{margin:0}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	pill := findElementByClass(root, "pill")
	var pillBox *box
	var walk func(*box)
	walk = func(b *box) {
		if b == nil {
			return
		}
		if b.node == pill {
			pillBox = b
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
	t.Logf("pill.w=%.2f", pillBox.w)
	// Must shrink-wrap (not fill 200px). Font metrics may exceed Chrome slightly.
	if pillBox.w > 50 || pillBox.w < 12 {
		t.Fatalf("pill.w=%.2f, want shrink-to-fit (not full CB)", pillBox.w)
	}
}
