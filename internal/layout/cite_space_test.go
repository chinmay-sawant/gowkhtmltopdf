package layout

import (
	"strings"
	"testing"
)

// Cite markers like <span>[</span>111<span>]</span> must not gain artificial
// spaces between text nodes ("[ 111 ]"), which blows narrow table columns and
// stacks glyphs on top of each other.
func TestCiteBracketNoArtificialSpaces(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.reference { white-space: nowrap; }
table { border-collapse: collapse; width: 200pt; }
td { border: 1px solid #aaa; padding: 2pt; width: 20pt; }
`)
	res := layoutHTML(t, `<html><body><table><tr><td>
<sup class="reference"><a href="#c"><span class="cite-bracket">[</span>111<span class="cite-bracket">]</span></a></sup>
</td></tr></table></body></html>`, cssSheet)

	var got string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			got += op.Text
		}
	}

	got = strings.ReplaceAll(got, " ", "")
	if !strings.Contains(got, "[111]") {
		t.Fatalf("cite text=%q, want [111] without interstitial spaces in layout", got)
	}
	// Also ensure we did not emit spaced form as separate ops glued poorly.
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "[ ") {
			t.Fatalf("artificial space after bracket: %q", op.Text)
		}
	}
}
