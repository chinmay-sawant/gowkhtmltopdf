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
