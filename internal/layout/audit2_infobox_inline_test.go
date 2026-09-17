package layout

import (
	"math"
	"strings"
	"testing"
)

// ana-de-armas-17: the infobox Spouse row wraps to three line boxes because
// the marriage-line div is a block-level child (line-height:0; the second
// zero-width div is inline-block) inside an inline context. line-height:0 was
// folded into the "normal" sentinel, so the starved block line box reserved a
// full strut, and the row spread 36.4pt where Chrome keeps it compact (8.2pt).
func TestInfoboxMarriageLineStaysCompact(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0; font-size: 8.8pt; line-height: 1.5 }
table { border-collapse: collapse }
td { padding: 2pt 4pt }
.marriage-display-inline { display: inline }
.marriage-line-margin2px { line-height: 0; margin-bottom: -2px }`)

	res := audit2LayoutImages(t, `<html><body><table><tr><td style="width:100pt">
<div class="marriage-display-inline"><div style="display:inline-block;`+
		`line-height:normal;margin-top:1px;white-space:normal;">`+
		`<a href="#">Marc Clotet</a></div>
   <div class="marriage-line-margin2px"><span>&#8203;</span></div>`+
		`<span> </span><div style="display:inline-block;margin-bottom:1px;">`+
		`<span>&#8203;</span></div><span>(</span>m.<span> </span>2011`+
		`<span>;</span><span> </span>div.<span> </span>2013<span>)</span></div>
</td></tr></table></body></html>`, cssSheet)

	marc, foundMarc := textOpContaining(res, "Marc Clotet")
	if !foundMarc {
		t.Fatal("no painted 'Marc Clotet' run")
	}

	marriage, foundMarriage := textOpContaining(res, "(m.")
	if !foundMarriage {
		t.Fatal("no painted '(m.' run")
	}

	gap := marriage.Y - marc.Y
	if math.Abs(gap) < 1 {
		// Same baseline is not the reference shape either (the block-in-inline
		// split must still break the line).
		t.Errorf("marriage line and spouse line share one baseline (gap %.2f); want two compact lines", gap)
	}

	// Chrome keeps the spouse/marriage baselines about 8.2pt apart. After the
	// block-in-inline split + line-height:0 collapse the budget is 12pt.
	if gap > 12 {
		t.Errorf("spouse-line gap = %.2fpt, want <= 12pt (target ~8pt); line-height:0 "+
			"block-in-inline must not reserve a full strut", gap)
	}
}

// ana-de-armas-16a: an empty inline span with horizontal padding must still
// contribute its padding. cs1-kern-left spans (padding-left:0.2em) keep the
// two Wikipedia quote glyphs apart; gowk painted them touching.
func TestEmptyInlineSpanContributesPadding(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0; font-size: 10pt } p { margin: 0 }`)

	res := layoutHTML(t, `<html><body><p>a<span style="padding-left:0.2em"></span>b</p></body></html>`, cssSheet)

	aOp, foundA := textOpContaining(res, "a")
	if !foundA {
		t.Fatal("no painted 'a' run")
	}

	bOp, foundB := textOpContaining(res, "b")
	if !foundB {
		t.Fatal("no painted 'b' run")
	}

	gap := bOp.X - (aOp.X + aOp.W)
	if math.Abs(gap-2.0) > 0.35 { // 0.2em at 10pt = 2pt
		t.Errorf("gap after empty padded span = %.2fpt, want 2.00pt (0.2em); "+
			"an empty inline span must still contribute its padding", gap)
	}
}

// ana-de-armas-16b: the explicit whitespace-only span before a nowrap IPA run
// is a real collapsible space. The bracket heuristic dropped it because the
// next run starts with "[", merging "pronunciation:" and "[ˈana".
func TestSpaceBeforeNowrapBracketSpanKept(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0; font-size: 10pt } p { margin: 0 }`)

	res := layoutHTML(t, `<html><body><p>Spanish pronunciation:<span> </span>`+
		`<span style="white-space:nowrap">[ˈana]</span></p></body></html>`, cssSheet)

	var joined string

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText {
			joined += paintOp.Text
		}
	}

	if !strings.Contains(joined, ": [") {
		t.Errorf("text layer = %q, want a space between the label and the IPA "+
			"bracket; the explicit separator span was dropped", joined)
	}
}

func textOpContaining(res *Result, needle string) (Op, bool) {
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, needle) {
			return op, true
		}
	}

	return Op{}, false
}
