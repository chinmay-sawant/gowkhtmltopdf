package layout

import (
	"strings"
	"testing"
)

// vertical-align: super raises text from the baseline and sub lowers it. Both
// consumers treat a positive shift as a raise (inline_paint.go baseline shift
// and alignedInlineTop), so this pins the sign contract at the op level.
func TestInlineTextSuperAndSubShiftSigns(t *testing.T) {
	t.Parallel()

	htmlSrc := `<html><body><p>base<span style="vertical-align:super">super</span>` +
		`<span style="vertical-align:sub">sub</span></p></body></html>`
	res := layoutHTML(t, htmlSrc, sheet(t, `body { margin: 0; font-size: 10pt; }`))
	base, sup, sub := verticalAlignOps(t, res.Ops)

	if sup.Y >= base.Y {
		t.Errorf("super Y=%.2f >= base Y=%.2f, want raised (smaller Y)", sup.Y, base.Y)
	}

	if sub.Y <= base.Y {
		t.Errorf("sub Y=%.2f <= base Y=%.2f, want lowered (larger Y)", sub.Y, base.Y)
	}

	if raise := base.Y - sup.Y; raise < 2.5 || raise > 5.5 {
		t.Errorf("super raise = %.2fpt, want ~4pt (0.4em at 10pt)", raise)
	}

	if lower := sub.Y - base.Y; lower < 0.5 || lower > 3.5 {
		t.Errorf("sub lower = %.2fpt, want ~2pt (0.2em at 10pt)", lower)
	}
}

// verticalAlignOps returns the base, super, and sub text ops, failing when any
// is missing.
func verticalAlignOps(t *testing.T, ops []Op) (Op, Op, Op) {
	t.Helper()

	var base, sup, sub Op

	for _, textOp := range ops {
		if textOp.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(textOp.Text, "super"):
			sup = textOp
		case strings.Contains(textOp.Text, "sub"):
			sub = textOp
		case strings.Contains(textOp.Text, "base"):
			base = textOp
		}
	}

	if base.Text == "" || sup.Text == "" || sub.Text == "" {
		t.Fatalf("missing ops: base=%+v super=%+v sub=%+v", base, sup, sub)
	}

	return base, sup, sub
}

// <sup> carries the browser UA declaration vertical-align: super plus
// font-size: smaller. Without it, wiki citation runs sit on the baseline at
// the parent size (ana-de-armas: 114 spans with a +0.52pt bottom delta where
// Chrome raises +5.27pt). The explicit-inline-style test above cannot catch a
// missing UA rule.
func TestSupElementRaisesRunBaseline(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p>base<sup>[1]</sup> tail</p></body></html>`,
		sheet(t, `body { margin: 0; font-size: 12pt }`))

	base := lastTextOpContaining(res, "base")

	sup := lastTextOpContaining(res, "[1]")
	if base.Text == "" || sup.Text == "" {
		t.Fatalf("missing ops: base=%+v sup=%+v", base, sup)
	}

	if sup.Y >= base.Y {
		t.Fatalf("sup baseline Y=%.2f not raised above base Y=%.2f; <sup> must honor vertical-align: super", sup.Y, base.Y)
	}

	if raise := base.Y - sup.Y; raise < 2.0 || raise > 6.0 {
		t.Errorf("sup raise = %.2fpt, want ~4pt (0.4em at the reduced sup size)", raise)
	}

	if sup.Size >= base.Size {
		t.Errorf("sup size = %.2fpt, want smaller than the base %.2fpt; "+
			"UA font-size: smaller must apply", sup.Size, base.Size)
	}
}

// lastTextOpContaining returns the last text op whose text contains needle.
func lastTextOpContaining(res *Result, needle string) Op {
	var found Op

	for _, textOp := range res.Ops {
		if textOp.Kind == OpText && strings.Contains(textOp.Text, needle) {
			found = textOp
		}
	}

	return found
}
