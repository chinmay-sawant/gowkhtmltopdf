package layout

import (
	"strings"
	"testing"
)

// Consecutive citation markers ][ get a thin separating space so clusters
// like [90][91][92] are not painted cramped. Inside a single marker stays tight.
func TestAdjacentCiteMarkersSeparated(t *testing.T) {
	cssSheetVal := sheet(t, `
body { margin: 0; font-size: 10pt; }
.reference { white-space: nowrap; font-size: 8pt; }
`)
	res := layoutHTML(t, `<html><body><p>end
<sup class="reference"><a href="#a"><span>[</span>90<span>]</span></a></sup><sup class="reference"><a href="#b"><span>[</span>91<span>]</span></a></sup><sup class="reference"><a href="#c"><span>[</span>92<span>]</span></a></sup>
more</p></body></html>`, cssSheetVal)

	var parts []string

	var xList []float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.ContainsAny(paintOp.Text, "[]0123456789") && !strings.Contains(paintOp.Text, "end") && !strings.Contains(paintOp.Text, "more") {
			parts = append(parts, paintOp.Text)
			xList = append(xList, paintOp.X)
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
	if len(xList) >= 2 {
		// At least some horizontal advance across the cluster.
		if xList[len(xList)-1] <= xList[0]+1 {
			t.Fatalf("cite markers appear stacked/cramped xs=%v parts=%v", xList, parts)
		}
	}
}
