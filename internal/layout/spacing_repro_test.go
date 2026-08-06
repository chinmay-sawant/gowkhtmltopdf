package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestJustifyDoesNotGapBeforePunctuation(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
p { text-align: justify; margin: 0; }
a { color: blue; text-decoration: underline; }
sup.reference { font-size: 8pt; }
`)
	src := `<html><body><p style="width:420pt">` +
		`born 30 April 1988)<sup class="reference"><a href="#c1"><span class="cite-bracket">[</span>1<span class="cite-bracket">]</span></a></sup> is a Cuban-born actress<sup class="reference"><a href="#c2"><span class="cite-bracket">[</span>2<span class="cite-bracket">]</span></a></sup> holding Cuban,<sup class="reference"><a href="#c2b"><span class="cite-bracket">[</span>2<span class="cite-bracket">]</span></a></sup> Spanish, and American citizenship.` +
		` including an <a href="/Academy_Award">Academy Award</a>, an <a href="/Actor">Actor Award</a>, a <a href="/BAFTA">BAFTA</a>, and two <a href="/Golden_Globe">Golden Globes</a>.` +
		` After moving to <a href="/LA">Los Angeles</a>, de Armas had English-speaking roles.` +
		` She starred opposite <a href="/r">Keanu Reeves</a> in her first Hollywood release—<a href="/er">Eli Roth</a>'s erotic thriller.` +
		`</p></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 420, Height: 800, Sheets: []*css.Stylesheet{s}, Background: true})
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
		}
	}
	var got string
	for _, r := range runs {
		got += r.text
	}
	t.Log("TEXT:", got)

	for i := 1; i < len(runs); i++ {
		prev, cur := runs[i-1], runs[i]
		if absF(prev.y-cur.y) > 0.5 {
			continue
		}
		gap := cur.x - (prev.x + prev.w)
		if gap <= 0.5 {
			continue
		}
		if strings.HasPrefix(cur.text, ",") || strings.HasPrefix(cur.text, ".") ||
			strings.HasPrefix(cur.text, "[") || strings.HasPrefix(cur.text, "]") ||
			strings.HasPrefix(cur.text, "'") || strings.HasPrefix(cur.text, "\u2019") {
			t.Errorf("justify gap %.2fpt before %q after %q", gap, cur.text, prev.text)
		}
		if strings.HasSuffix(prev.text, ")") && strings.HasPrefix(cur.text, "[") {
			t.Errorf("justify gap %.2fpt between %q and %q", gap, prev.text, cur.text)
		}
		if strings.HasSuffix(prev.text, "\u2014") || strings.HasSuffix(prev.text, "\u2013") || strings.HasSuffix(prev.text, "—") {
			// emdash may be end of prev; next shouldn't have large gap if glued in source
			if gap > 0.5 && !strings.HasPrefix(cur.text, " ") {
				t.Logf("note gap %.2f after dash %q before %q", gap, prev.text, cur.text)
			}
		}
	}
	if strings.Contains(got, "Roth 's") || strings.Contains(got, "Roth  's") {
		t.Errorf("space before possessive: %q", got)
	}
	if !strings.Contains(got, "Roth's") && !strings.Contains(got, "Roth\u2019s") {
		// may be split across ops; check geometrically
		ok := false
		for i := 1; i < len(runs); i++ {
			if strings.HasSuffix(runs[i-1].text, "Roth") && strings.HasPrefix(runs[i].text, "'") {
				gap := runs[i].x - (runs[i-1].x + runs[i-1].w)
				if gap <= 0.5 {
					ok = true
				} else {
					t.Errorf("geometric space before possessive: gap=%.2f", gap)
				}
			}
		}
		if !ok {
			t.Logf("Roth's ops not found as expected; text=%q", got)
		}
	}
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestCiteDoesNotSplitAcrossLines(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
p { text-align: justify; margin: 0; }
sup.reference { font-size: 8pt; white-space: nowrap; }
`)
	src := `<html><body><p>` +
		`She often did not know what she was saying.<sup class="reference"><a href="#c"><span class="cite-bracket">[</span>37<span class="cite-bracket">]</span></a></sup> She spent four months learning English afterward with more words here to continue.` +
		`</p></body></html>`
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 200, Height: 800, Sheets: []*css.Stylesheet{s}, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var citeOps []Op
	for _, op := range res.Ops {
		if op.Kind == OpText && (strings.Contains(op.Text, "[") || strings.Contains(op.Text, "]") ||
			op.Text == "37" || strings.Contains(op.Text, "37")) {
			citeOps = append(citeOps, op)
		}
	}
	if len(citeOps) == 0 {
		var got string
		for _, op := range res.Ops {
			if op.Kind == OpText {
				got += op.Text
			}
		}
		if !strings.Contains(got, "[37]") {
			t.Fatalf("missing cite in %q", got)
		}
		return
	}
	y0 := citeOps[0].Y
	for _, op := range citeOps {
		if absF(op.Y-y0) > 0.5 {
			t.Fatalf("cite split across lines: %#v", citeOps)
		}
	}
}
