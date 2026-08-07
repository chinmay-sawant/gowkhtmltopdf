package layout

import (
	"strings"
	"testing"
)

// Wiki .hlist uses li::after{content:"\a0 · "} between citizenship entries.
func TestHListAfterSeparator(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
.hlist li { display: inline; margin: 0; }
.hlist li::after { content: "\a0 · "; font-weight: bold; }
.hlist li:last-child::after { content: none; }
`)
	res := layoutHTML(t, `<html><body>
<ul class="hlist">
<li><a href="/c">Cuba</a></li>
<li><a href="/s">Spain</a></li>
<li><a href="/u">United States</a></li>
</ul>
</body></html>`, s)

	var got string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			got += op.Text
		}
	}

	if strings.Contains(got, "CubaSpain") || strings.Contains(got, "SpainUnited") {
		t.Fatalf("missing hlist separators: %q", got)
	}

	if !strings.Contains(got, "Cuba") || !strings.Contains(got, "Spain") {
		t.Fatalf("missing labels: %q", got)
	}
	// NBSP + middle dot between items
	if !strings.Contains(got, "·") {
		t.Fatalf("expected middle-dot separator in %q", got)
	}
}
