package layout

import (
	"fmt"
	"strings"
	"testing"
)

// maxTextRight returns the right edge of the widest text op in the result.
func maxTextRight(res *Result) float64 {
	right := 0.0

	for _, op := range res.Ops {
		if op.Kind == OpText && op.X+op.W > right {
			right = op.X + op.W
		}
	}

	return right
}

// A soft wrap must not count collapsible spaces at the end of the candidate
// line: they hang at the line edge. The gobyexample intro wrapped "to" one
// word early although the line fit by 1.13pt because the trailing space
// (3.0pt) was included in the fit (313.875pt text vs 316.875pt threshold at a
// 315.00pt container).
func TestLineFitExcludesTrailingSpace(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0 } p { margin: 0; font-size: 12pt }`)
	prefix := "release Go and may use new language features. Try to upgrade to"

	// Measure the candidate line in a nowrap block: the op width is its raw
	// advance, without the trailing space that follows "to".
	nowrap := layoutHTML(t, `<html><body><p style="white-space:nowrap">`+prefix+`</p></body></html>`, cssSheet)
	candidateW := firstText(nowrap).W

	// A 0.5pt-margin container keeps "to" on the first line only when the
	// trailing space hangs instead of counting against the width.
	src := fmt.Sprintf(`<html><body><p style="width:%.3fpt">%s the latest version.</p></body></html>`,
		candidateW+0.5, prefix)
	res := layoutHTML(t, src, cssSheet)

	line1 := ""

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "upgrade") {
			line1 = op.Text
		}
	}

	if !strings.HasSuffix(line1, "upgrade to") {
		t.Fatalf("first line %q broke before \"to\"; the %.3fpt candidate line fits the %.3fpt "+
			"container once the trailing space hangs", line1, candidateW, candidateW+0.5)
	}
}

// An unbreakable citation chain ("[" + "41" + "]") that fits a fresh line must
// move down whole instead of gluing its digits behind an opening bracket that
// happens to fit. The wiki line "role,[41][18]" painted 15.6pt past the
// content box (page 3 of ana-de-armas); Chrome breaks before the cluster.
func TestSupCitationClusterDoesNotOverflowContentBox(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0 }
p { margin: 0; font-size: 12pt }
sup.reference { white-space: nowrap; font-size: 80% }`)

	prose := strings.Repeat("alpha bravo charlie delta echo foxtrot golf hotel india juliet ", 3) + "role,"
	openBracket := `<sup class="reference"><a href="#c41"><span class="cite-bracket">[</span></a></sup>`
	cluster := `<sup class="reference"><a href="#c41">` +
		`<span class="cite-bracket">[</span>41<span class="cite-bracket">]</span></a></sup>` +
		`<sup class="reference"><a href="#c18">` +
		`<span class="cite-bracket">[</span>18<span class="cite-bracket">]</span></a></sup>`

	// Width of the prose plus the opening bracket: at this width "[" fits and
	// the digits after it do not, the exact trigger for the glued overflow.
	measure := layoutHTML(t, `<html><body><p style="white-space:nowrap">`+prose+openBracket+`</p></body></html>`, cssSheet)
	limit := maxTextRight(measure) + 0.5

	res := layoutHTML(t,
		fmt.Sprintf(`<html><body><p style="width:%.3fpt">%s%s</p></body></html>`, limit, prose, cluster),
		cssSheet)

	if right := maxTextRight(res); right > limit+0.01 {
		t.Fatalf("citation cluster paints to %.3fpt past the %.3fpt content edge; "+
			"the unbreakable chain fits a fresh line and must wrap whole", right-limit, limit)
	}
}
