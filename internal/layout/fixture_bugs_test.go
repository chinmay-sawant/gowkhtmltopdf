package layout

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestRowBackgroundShowsThroughCells(t *testing.T) {
	// tr.good { background } must paint on cells that have no own bg.
	s, err := css.Parse(`
		td { border: 1px solid #000; }
		.good { background-color: #e2f2e2; color: #1f5c2e }
		.warn { background-color: #fdf3d7; color: #7a5c00 }
	`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><table>
		<tr class="good"><td>On-time</td><td>96%</td></tr>
		<tr class="warn"><td>Turns</td><td>7.8</td></tr>
	</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 300, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var green, yellow int
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 1 {
			continue
		}
		// #e2f2e2
		if op.G > 0.9 && op.R > 0.8 && op.R < 0.95 && op.B > 0.8 {
			green++
		}
		// #fdf3d7 ≈ warm yellow
		if op.R > 0.95 && op.G > 0.9 && op.B < 0.9 {
			yellow++
		}
	}
	if green < 2 {
		t.Errorf("good-row cell fills = %d, want >= 2", green)
	}
	if yellow < 2 {
		t.Errorf("warn-row cell fills = %d, want >= 2", yellow)
	}
}

func TestRGBABackgroundCompositesLight(t *testing.T) {
	s, err := css.Parse(`.alpha { background-color: rgba(15, 58, 95, 0.15); padding: 8px }`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><div class="alpha">Alpha band</div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 200, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	// Layout still stores source rgba; paint composites. Assert layout alpha.
	found := false
	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.Alpha > 0.1 && op.Alpha < 0.3 {
			found = true
			// source rgb should be the dark blue channels, not already white
			if op.R > 0.2 || op.B < 0.3 {
				t.Errorf("unexpected source rgba fill R=%v B=%v A=%v", op.R, op.B, op.Alpha)
			}
		}
	}
	if !found {
		t.Error("expected translucent fill op from rgba(...)")
	}
}

func TestNestedTableNoMeasureLeak(t *testing.T) {
	// Nested tables must not emit ops during the outer measure pass, and
	// must keep document order (text before nested table in the same cell).
	root, err := html.Parse(`<html><body><table>
		<tr><th colspan="2">Header</th></tr>
		<tr><td>1</td><td>outer-label
			<table><tr><td>inner-a</td><td>inner-b</td></tr></table>
		</td></tr>
	</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 500, Height: 400, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var headerY, labelY, innerY float64
	var sawHeader, sawLabel, sawInner bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if strings.Contains(op.Text, "Header") {
			headerY, sawHeader = op.Y, true
		}
		if strings.Contains(op.Text, "outer-label") {
			labelY, sawLabel = op.Y, true
		}
		if strings.Contains(op.Text, "inner-a") {
			innerY, sawInner = op.Y, true
		}
	}
	if !sawHeader || !sawLabel || !sawInner {
		t.Fatalf("missing text header=%v label=%v inner=%v", sawHeader, sawLabel, sawInner)
	}
	if !(innerY > headerY) {
		t.Errorf("inner table Y=%.1f should be below header Y=%.1f", innerY, headerY)
	}
	if !(innerY > labelY) {
		t.Errorf("inner table Y=%.1f should be below outer-label Y=%.1f (document order)", innerY, labelY)
	}
	n := 0
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "inner-a") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("inner-a text ops = %d, want 1 (no double emit)", n)
	}
}

func TestBackgroundPaintsUnderText(t *testing.T) {
	// Regression: block backgrounds must be emitted before text ops so
	// yellow/blue notice boxes do not cover their labels.
	s, err := css.Parse(`.notice { background-color: #fff3d6; color: #233043; padding: 8px }`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><div class="notice">Important notice text</div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 200, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var firstFill, firstText int = -1, -1
	for i, op := range res.Ops {
		if op.Kind == OpFillRect && firstFill < 0 && op.R > 0.9 {
			firstFill = i
		}
		if op.Kind == OpText && strings.Contains(op.Text, "Important") {
			firstText = i
			break
		}
	}
	if firstFill < 0 || firstText < 0 {
		t.Fatalf("fill=%d text=%d ops=%+v", firstFill, firstText, res.Ops)
	}
	if firstFill > firstText {
		t.Errorf("background op index %d after text %d - text would be covered", firstFill, firstText)
	}
}

// TestTableCellRowHeightUsesFinalWidth guards against measuring cell height at
// max-content width (too narrow → false wraps → inflated empty row bands).
func TestTableCellRowHeightUsesFinalWidth(t *testing.T) {
	s, err := css.Parse(`
		table { width: 100%; border-collapse: collapse; }
		td { border: 1px solid #000; padding: 4px; font-size: 10pt; }
	`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body>
		<table>
			<tr><td>On-time delivery</td><td>96.4 %</td><td>above target</td></tr>
			<tr><td>First-pass yield</td><td>98.1 %</td><td>above target</td></tr>
		</table>
	</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 500, Height: 400, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var ys []float64
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "On-time") {
			ys = append(ys, op.Y)
		}
		if op.Kind == OpText && strings.Contains(op.Text, "First-pass") {
			ys = append(ys, op.Y)
		}
	}
	if len(ys) < 2 {
		t.Fatalf("need both row labels, got ys=%v", ys)
	}
	dy := ys[1] - ys[0]
	// Single-line 10pt rows with padding should be well under 30pt apart.
	// The max-content-height bug produced ~35-50pt empty bands between rows.
	if dy > 28 {
		t.Errorf("row baseline gap = %.1f pt, want <= 28 (inflated cell height from max-content measure)", dy)
	}
	if dy < 8 {
		t.Errorf("row baseline gap = %.1f pt, want >= 8 (rows collapsed?)", dy)
	}
}

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

func TestMultiImageUniqueOps(t *testing.T) {
	pngA := mustDecodeB64(t, "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAEklEQVR4nGP8z8AARIDajAoAAgwAAf8C/tH9n9kAAAAASUVORK5CYII=")
	pngB := mustDecodeB64(t, "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAEklEQVR4nGN4z8AAQTDqMSoAAgwAAZ0B/vG0cU0AAAAASUVORK5CYII=")
	root, err := html.Parse(`<html><body>
<p><img src="a.png"></p><p><img src="b.png"></p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 200, Height: 200, Background: true,
		Images: func(src string) ([]byte, error) {
			if src == "a.png" {
				return pngA, nil
			}
			if src == "b.png" {
				return pngB, nil
			}
			return nil, os.ErrNotExist
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var sizes [][2]int
	for _, op := range res.Ops {
		if op.Kind == OpImage {
			sizes = append(sizes, [2]int{op.ImgW, op.ImgH})
			if len(op.Image) < 20 {
				t.Error("empty image bytes")
			}
		}
	}
	if len(sizes) != 2 {
		t.Fatalf("images = %v, want 2", sizes)
	}
}

func mustDecodeB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
