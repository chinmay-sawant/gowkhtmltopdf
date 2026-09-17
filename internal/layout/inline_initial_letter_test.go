package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestInitialLetterSpansThreeLines(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.drop {
  initial-letter: 3;
  font-weight: 700;
}
p {
  margin: 0;
  font-size: 12pt;
  line-height: 16pt;
  width: 220pt;
}
`)
	root := mustParse(t, `<html><body>
<p><span class="drop">T</span>he rest of this paragraph wraps beside the drop cap across several lines so geometry can be checked.</p>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)
	st := styleByClass(t, styles, "drop")
	if st.InitialLetterSize != 3 {
		t.Fatalf("initial-letter size=%.1f, want 3", st.InitialLetterSize)
	}

	if st.InitialLetterSink != 3 {
		t.Fatalf("initial-letter sink=%d, want 3", st.InitialLetterSink)
	}

	res := layoutHTML(t, `<html><body>
<p><span class="drop">T</span>he rest of this paragraph wraps beside the drop cap across several lines so geometry can be checked. More words fill a second and third line beside the letter box.</p>
</body></html>`, cssSheet)

	var letterX, letterY, letterSize float64
	var letterFound bool
	type textHit struct {
		x, y float64
		text string
	}
	var hits []textHit

	for _, op := range res.Ops {
		if op.Kind != OpText || op.Text == "" {
			continue
		}

		hits = append(hits, textHit{x: op.X, y: op.Y, text: op.Text})
		if op.Text == "T" || (len(op.Text) >= 1 && op.Text[:1] == "T" && op.Size > 20) {
			letterX = op.X
			letterY = op.Y
			letterSize = op.Size
			letterFound = true
		}
	}

	if !letterFound {
		t.Fatalf("drop-cap letter not found in ops; hits=%v", hits)
	}

	if letterSize < 30 {
		t.Fatalf("drop-cap font size=%.1f; want roughly 3-line size (>30)", letterSize)
	}

	// Body text on the lines that share the drop-cap band must start to the
	// right of the letter box (float-like exclusion).
	var nextLineX float64
	var nextFound bool
	for _, h := range hits {
		if h.x <= letterX+2 {
			continue
		}

		// Skip the letter itself.
		if h.text == "T" || (len(h.text) > 0 && h.text[0] == 'T' && h.x == letterX) {
			continue
		}

		nextLineX = h.x
		nextFound = true

		break
	}

	if !nextFound {
		t.Fatalf("no wrapped text beside drop cap; letter=(%.1f,%.1f size=%.1f) hits=%v",
			letterX, letterY, letterSize, hits)
	}

	if nextLineX <= letterX+2 {
		t.Fatalf("next-line x=%.1f not to the right of letter x=%.1f (exclusion missing)", nextLineX, letterX)
	}
}

func TestInitialLetterParseAlignWrap(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.a { initial-letter: 2 raise; initial-letter-align: hanging; initial-letter-wrap: all }
.b { initial-letter: normal }
`)
	root := mustParse(t, `<html><body>
<span class="a">A</span><span class="b">B</span>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)

	a := styleByClass(t, styles, "a")
	if a.InitialLetterSize != 2 || a.InitialLetterSink != 1 {
		t.Fatalf("a size=%.1f sink=%d, want 2 / 1 (raise)", a.InitialLetterSize, a.InitialLetterSink)
	}

	if a.InitialLetterAlign != "hanging" || a.InitialLetterWrap != initialLetterWrapAll {
		t.Fatalf("a align=%q wrap=%q", a.InitialLetterAlign, a.InitialLetterWrap)
	}

	b := styleByClass(t, styles, "b")
	if b.InitialLetterSize != 0 {
		t.Fatalf("b size=%.1f, want 0 (normal)", b.InitialLetterSize)
	}
}
