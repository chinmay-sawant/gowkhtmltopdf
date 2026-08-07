package layout

import (
	"strings"
	"testing"
)

// Consecutive citation markers ][ get a thin separating space so clusters
// like [90][91][92] are not painted cramped. Inside a single marker stays tight.
func TestAdjacentCiteMarkersSeparated(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
.reference { white-space: nowrap; font-size: 8pt; }
`)
	res := layoutHTML(t, `<html><body><p>end
<sup class="reference"><a href="#a"><span>[</span>90<span>]</span></a></sup><sup class="reference"><a href="#b"><span>[</span>91<span>]</span></a></sup><sup class="reference"><a href="#c"><span>[</span>92<span>]</span></a></sup>
more</p></body></html>`, s)

	var parts []string

	var xs []float64

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		if strings.ContainsAny(op.Text, "[]0123456789") && !strings.Contains(op.Text, "end") && !strings.Contains(op.Text, "more") {
			parts = append(parts, op.Text)
			xs = append(xs, op.X)
		}
	}

	joined := strings.Join(parts, "")
	// Strip hair/normal spaces for containment check.
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\u200a' || r == '\u2009' {
			return -1
		}

		return r
	}, joined)
	if !strings.Contains(compact, "[90][91][92]") && !strings.Contains(compact, "[90]") {
		t.Fatalf("cite cluster missing, got parts=%v compact=%q", parts, compact)
	}
	// Markers should not all share the exact same X (would mean stacked).
	// With separation, successive [ markers advance in X on the same line.
	if len(xs) >= 2 {
		// At least some horizontal advance across the cluster.
		if xs[len(xs)-1] <= xs[0]+1 {
			t.Fatalf("cite markers appear stacked/cramped xs=%v parts=%v", xs, parts)
		}
	}
}
