package layout

import (
	"strings"
	"testing"
)

// ana-de-armas-15: figure captions must wrap whole words. Wikipedia's
// figcaption sets word-break:break-word (overflow-wrap:anywhere); a normal
// word that fits the next line must move down whole instead of being split
// mid-word to fill the current line's remainder. The live page painted
// "with n" / "umber 4", "San Seba" / "stián", "Film Fes" / "tival".
func TestFigureCaptionBreaksWholeWordsWithAnywhere(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0; font-size: 10pt }
figure { display: table; float: right; clear: right; margin: 0.5em 0 1.3em 1.4em; border: 1px solid #c8ccd1; }
figure > figcaption {
	display: table-caption; caption-side: bottom; font-size: 94%;
	padding: 0 6px 6px 6px; word-break: break-word;
}
img { display: block; }
p { margin: 0 }`)

	res := audit2LayoutImages(t, `<html><body>
<figure><img width="250" height="175" src="thumb.png"><figcaption>`+
		`De Armas (standing at the center with number 4) with the cast of `+
		`El Internado: Laguna Negra (The Boarding School) in 2008`+
		`</figcaption></figure>
<figure><img width="250" height="175" src="thumb.png"><figcaption>`+
		`De Armas at the San Sebastián International Film Festival in 2022`+
		`</figcaption></figure>
<figure><img width="250" height="175" src="thumb.png"><figcaption>`+
		`She attended the 2019 San Diego Comic-Con in July.`+
		`</figcaption></figure>
<p>`+strings.Repeat("more ", 40)+`</p>
</body></html>`, cssSheet)

	var textOps []string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			textOps = append(textOps, op.Text)
		}
	}

	for _, word := range []string{"number", "Laguna", "Internado", "Sebastián", "International", "Festival", "Comic-Con"} {
		found := false

		for _, got := range textOps {
			if strings.Contains(got, word) {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("caption word %q never paints whole; a word that fits the next "+
				"line must not be split mid-word (ops: %q)", word, textOps)
		}
	}
}
