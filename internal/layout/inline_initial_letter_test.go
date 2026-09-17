package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestInitialLetterSpansThreeLines(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.drop {
  initial-letter: 3;
  initial-letter-wrap: all;
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

type initialLetterHit struct {
	x, y, size float64
	text       string
}

const dropCapBody = `he rest of this paragraph wraps beside the drop cap across several lines so geometry can be checked. More words fill a second and third line beside the letter box so wrap keywords can diverge.`

func layoutDropCap(t *testing.T, extraCSS string) *Result {
	t.Helper()

	cssSheet := sheet(t, `
.drop {
  initial-letter: 3;
  font-weight: 700;
`+extraCSS+`
}
p {
  margin: 0;
  font-size: 12pt;
  line-height: 16pt;
  width: 220pt;
}
`)

	return layoutHTML(t, `<html><body>
<p><span class="drop">T</span>`+dropCapBody+`</p>
</body></html>`, cssSheet)
}

func collectDropCapHits(t *testing.T, res *Result) (letter initialLetterHit, body []initialLetterHit) {
	t.Helper()

	var found bool
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Text == "" || strings.TrimSpace(op.Text) == "" {
			continue
		}

		hit := initialLetterHit{x: op.X, y: op.Y, size: op.Size, text: op.Text}
		isLetter := op.Text == "T" || (strings.HasPrefix(op.Text, "T") && op.Size > 20)
		if isLetter && !found {
			letter = hit
			found = true

			continue
		}

		body = append(body, hit)
	}

	if !found {
		t.Fatalf("drop-cap letter T not found")
	}

	if len(body) == 0 {
		t.Fatalf("no body text beside drop cap; letter=(%.1f,%.1f size=%.1f)",
			letter.x, letter.y, letter.size)
	}

	return letter, body
}

func TestInitialLetterAlignHangingShiftsY(t *testing.T) {
	t.Parallel()

	alpha := layoutDropCap(t, "initial-letter-align: alphabetic;\n  initial-letter-wrap: none;")
	hang := layoutDropCap(t, "initial-letter-align: hanging;\n  initial-letter-wrap: none;")
	lead := layoutDropCap(t, "initial-letter-align: leading;\n  initial-letter-wrap: none;")

	alphaLetter, _ := collectDropCapHits(t, alpha)
	hangLetter, _ := collectDropCapHits(t, hang)
	leadLetter, _ := collectDropCapHits(t, lead)

	if hangLetter.y == alphaLetter.y {
		t.Fatalf("hanging letter Y=%.2f equal to alphabetic Y=%.2f; want a distinct hanging metric",
			hangLetter.y, alphaLetter.y)
	}

	if leadLetter.y == alphaLetter.y {
		t.Fatalf("leading letter Y=%.2f equal to alphabetic Y=%.2f; want a distinct leading metric",
			leadLetter.y, alphaLetter.y)
	}

	if hangLetter.y == leadLetter.y {
		t.Fatalf("hanging and leading share Y=%.2f; metrics must differ", hangLetter.y)
	}

	// Hanging raises the letter (smaller Y); leading sits lower than alphabetic.
	if hangLetter.y >= alphaLetter.y {
		t.Fatalf("hanging Y=%.2f should be above alphabetic Y=%.2f", hangLetter.y, alphaLetter.y)
	}

	if leadLetter.y <= alphaLetter.y {
		t.Fatalf("leading Y=%.2f should be below alphabetic Y=%.2f", leadLetter.y, alphaLetter.y)
	}
}

func TestInitialLetterWrapNoneDoesNotExclude(t *testing.T) {
	t.Parallel()

	noneRes := layoutDropCap(t, "initial-letter-wrap: none;")
	firstRes := layoutDropCap(t, "initial-letter-wrap: first;")
	allRes := layoutDropCap(t, "initial-letter-wrap: all;")

	noneLetter, noneBody := collectDropCapHits(t, noneRes)
	firstLetter, firstBody := collectDropCapHits(t, firstRes)
	allLetter, allBody := collectDropCapHits(t, allRes)

	noneX := noneBody[0].x
	firstX := firstBody[0].x
	allX := allBody[0].x

	if noneX > noneLetter.x+2 {
		t.Fatalf("wrap:none first body x=%.1f is beside letter x=%.1f; none must not exclude",
			noneX, noneLetter.x)
	}

	if allX <= allLetter.x+2 {
		t.Fatalf("wrap:all first body x=%.1f not to the right of letter x=%.1f", allX, allLetter.x)
	}

	if firstX <= firstLetter.x+2 {
		t.Fatalf("wrap:first first body x=%.1f not to the right of letter x=%.1f",
			firstX, firstLetter.x)
	}

	firstLater := laterLineX(firstBody, firstBody[0].y)
	allLater := laterLineX(allBody, allBody[0].y)
	if firstLater < 0 || allLater < 0 {
		t.Fatalf("need a later body line; first later=%.1f all later=%.1f firstHits=%v allHits=%v",
			firstLater, allLater, firstBody, allBody)
	}

	if firstLater > firstLetter.x+2 {
		t.Fatalf("wrap:first later-line x=%.1f still excluded (letter x=%.1f); first is first line only",
			firstLater, firstLetter.x)
	}

	if allLater <= allLetter.x+2 {
		t.Fatalf("wrap:all later-line x=%.1f not excluded (letter x=%.1f); all covers overlapping lines",
			allLater, allLetter.x)
	}
}

func laterLineX(hits []initialLetterHit, firstY float64) float64 {
	for _, h := range hits {
		if h.y > firstY+8 {
			return h.x
		}
	}

	return -1
}
