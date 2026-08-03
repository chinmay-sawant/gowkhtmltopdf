package layout

import (
	"os"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestTableCellBackgroundHeight(t *testing.T) {
	s, err := css.Parse(`th { background-color: #1a3d6d; color: #fff } td { background-color: #f2f6fa }`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><table>
		<tr><th>H1</th><th>H2</th></tr>
		<tr><td>a</td><td>b</td></tr>
	</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 400, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var thFills, tdFills int
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 1 {
			continue
		}
		// dark blue header
		if op.R < 0.2 && op.B > 0.3 {
			thFills++
		}
		// light zebra-ish
		if op.R > 0.9 && op.B > 0.9 {
			tdFills++
		}
	}
	if thFills < 2 {
		t.Errorf("th background fills with height>=1 = %d, want >= 2", thFills)
	}
	if tdFills < 1 {
		t.Errorf("td background fills with height>=1 = %d, want >= 1", tdFills)
	}
}

func TestPrePreservesNewlines(t *testing.T) {
	root, err := html.Parse("<html><body><pre>alpha\n  beta\ngamma</pre></body></html>")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 400, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	var ys []float64
	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, op.Text)
			ys = append(ys, op.Y)
		}
	}
	if len(texts) < 3 {
		t.Fatalf("pre lines = %v, want >= 3 separate lines", texts)
	}
	if texts[0] != "alpha" || !strings.HasPrefix(texts[1], "  beta") || texts[2] != "gamma" {
		t.Errorf("pre segments = %q", texts)
	}
	if !(ys[1] > ys[0] && ys[2] > ys[1]) {
		t.Errorf("pre lines should stack downward: %v", ys)
	}
}

func TestMarginAutoCenters(t *testing.T) {
	s, err := css.Parse(`.rule { width: 100pt; margin: 10pt auto; border-top: 4px solid #000 }`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><div class="rule"></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	const vp = 300.0
	res, err := Layout(root, Options{Width: vp, Height: 400, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	// body UA margin 8px = 6pt; content width = 300-12 = 288; rule width 100 centered
	// origin inside body content: (288-100)/2 = 94; absolute x = 6+94 = 100
	var line *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpLine || (op.Kind == OpFillRect && op.H > 0 && op.H < 5 && op.W > 50) {
			line = op
			break
		}
		if op.Kind == OpStrokeRect && op.W > 50 {
			line = op
			break
		}
	}
	// border may paint as four edges; find widest horizontal stroke near top of box
	var best *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpLine && op.W >= 90 {
			best = op
			break
		}
		if op.Kind == OpFillRect && op.W >= 90 && op.H <= 5 {
			best = op
			break
		}
	}
	if best == nil && line == nil {
		// dump for diagnosis
		for _, op := range res.Ops {
			t.Logf("op kind=%v x=%.1f w=%.1f h=%.1f", op.Kind, op.X, op.W, op.H)
		}
		t.Fatal("no centered rule geometry found")
	}
	op := best
	if op == nil {
		op = line
	}
	// expect roughly centered in viewport
	mid := op.X + op.W/2
	if mid < vp*0.35 || mid > vp*0.65 {
		t.Errorf("rule center x=%.1f (op x=%.1f w=%.1f), want near viewport center %.1f", mid, op.X, op.W, vp/2)
	}
}

func TestFixture16HeaderBG(t *testing.T) {
	b, err := os.ReadFile("../../testdata/golden/fixture-16-invoice-with-css.html")
	if err != nil {
		t.Skip(err)
	}
	root, err := html.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}
	// Layout does not auto-collect <style>; mirror convert.collectSheets lightly.
	var sheets []*css.Stylesheet
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Name == "style" {
			var sb strings.Builder
			for _, c := range n.Children {
				if c.Type == html.TextNode {
					sb.WriteString(c.Text)
				}
			}
			if s, err := css.Parse(sb.String()); err == nil {
				sheets = append(sheets, s)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	res, err := Layout(root, Options{Width: 595, Height: 842, Sheets: sheets, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, op := range res.Ops {
		// #1a3d6d ≈ 0.102, 0.239, 0.427
		if op.Kind == OpFillRect && op.H >= 8 &&
			op.R > 0.08 && op.R < 0.15 && op.B > 0.35 && op.B < 0.5 {
			n++
		}
	}
	if n < 4 {
		t.Errorf("dark header/table fills with real height = %d, want >= 4", n)
	}
}
