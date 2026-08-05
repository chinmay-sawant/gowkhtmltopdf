package layout

import (
	"strings"
	"testing"
)

// Blocks after a tall float must pack beside it (Chrome), not clear below
// unless clear is set. Non-BFC parents must not grow around floats — that
// used to push the next <section> below the entire floated infobox.
func TestBlockHeadingPacksBesideTallFloat(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
.box { float: right; width: 100pt; display: table; border: 1px solid #000; }
.box td { padding: 4pt; }
p { margin: 0 0 6pt 0; }
.hd { overflow: hidden; border-bottom: 2px solid #000; margin: 0.25em 0; font-size: 14pt; font-weight: bold; }
.hd h2 { display: inline; margin: 0; padding: 0; border: 0; font: inherit; }
`)
	res := layoutHTML(t, `<html><body>
<section class="lead">
<table class="box"><tr><td>PHOTO<br/>2<br/>3<br/>4<br/>5<br/>6<br/>7<br/>8<br/>9<br/>10<br/>11<br/>12<br/>Born<br/>Occ<br/>Spouse</td></tr></table>
<p>Lead one wraps beside the float with enough words to fill a couple of lines here.</p>
<p>Lead two ends with Ballerina.</p>
</section>
<section class="next">
<div class="hd"><h2>Early life</h2></div>
<p>Born in Havana text that should also sit beside the float if room remains.</p>
</section>
</body></html>`, s)

	var earlyY, leadEnd, floatBottom float64
	earlyY = -1
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch {
		case op.Text == "Early life":
			earlyY = op.Y
		case strings.Contains(op.Text, "Ballerina"):
			if op.Y > leadEnd {
				leadEnd = op.Y
			}
		case strings.Contains(op.Text, "Spouse") || strings.Contains(op.Text, "PHOTO"):
			if op.Y > floatBottom {
				floatBottom = op.Y
			}
		}
	}
	if earlyY < 0 {
		t.Fatal("Early life text not found")
	}
	t.Logf("leadEnd=%.1f earlyY=%.1f floatBottom=%.1f", leadEnd, earlyY, floatBottom)
	if floatBottom > leadEnd+20 && earlyY > floatBottom-5 {
		t.Fatalf("heading cleared below float: earlyY=%.1f floatBottom=%.1f leadEnd=%.1f", earlyY, floatBottom, leadEnd)
	}
	if earlyY > leadEnd+40 {
		t.Fatalf("too much gap before heading: earlyY=%.1f leadEnd=%.1f", earlyY, leadEnd)
	}
}
