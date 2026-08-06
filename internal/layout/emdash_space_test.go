package layout

import (
	"fmt"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestEmdashLinkNoGap(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
p { text-align: justify; margin: 0; }
a { color: #0645ad; text-decoration: underline; }
`)
	src := `<html><body><p>Hollywood release—<a href="/Eli_Roth">Eli Roth</a>'s erotic thriller <i><a href="/kk">Knock Knock</a></i> (2015)—and learned her lines.</p></body></html>`
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 500, Height: 400, Sheets: []*css.Stylesheet{s}})
	if err != nil {
		t.Fatal(err)
	}
	type trun struct {
		text    string
		x, y, w float64
	}
	var runs []trun
	for _, op := range res.Ops {
		if op.Kind == OpText {
			runs = append(runs, trun{op.Text, op.X, op.Y, op.W})
			t.Logf("op %q x=%.2f w=%.2f y=%.2f", op.Text, op.X, op.W, op.Y)
		}
	}
	for i := 1; i < len(runs); i++ {
		prev, cur := runs[i-1], runs[i]
		if absF(prev.y-cur.y) > 0.5 {
			continue
		}
		gap := cur.x - (prev.x + prev.w)
		if gap > 0.6 {
			t.Errorf("gap %.2fpt between %q and %q", gap, prev.text, cur.text)
		}
	}
	var got string
	for _, r := range runs {
		got += r.text
	}
	fmt.Println(got)
	if strings.Contains(got, "Roth 's") {
		t.Error("space before possessive")
	}
}
