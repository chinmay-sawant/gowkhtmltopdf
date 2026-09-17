package layout

import (
	"testing"
)

// A line whose box crosses a page boundary snaps to the next page top and
// takes the chrome of its own row with it. A link run on the *previous* line
// carries an Op.H extended by its inline box's bottom border (see
// enableInlineChrome), so its box bottom sits exactly on the next line's
// baseline. The row-band test accepted that touch and dragged the previous
// line's link glyphs down with the snapped line: ana-de-armas p5 painted
// "Bond girl" one baseline below "In 2021 ... play a" (audit row
// ana-de-armas-13), and the wrapped citation titles on p11-p20 moved the same
// way (audit row ana-de-armas-14).
func TestSnapKeepsPreviousLineLinkRuns(t *testing.T) {
	t.Parallel()

	const (
		lineOneBaseline = 100.0
		lineTwoBaseline = 117.0
		contentH        = 132.0 // line two's box (117 + 16) crosses this boundary
	)

	textOp := func(text string, x, y, w, h float64) Op {
		return Op{
			Kind: OpText, X: x, Y: y, W: w, H: h, Text: text,
		}
	}

	res := &Result{
		Ops: []Op{
			textOp("play a ", 10, lineOneBaseline, 30, 16),
			// H = 17: the anchor's 1pt bottom border extends the run's op box
			// one point below the line box, exactly touching line two.
			textOp("Bond girl", 40, lineOneBaseline, 40, 17),
			textOp(" in ", 80, lineOneBaseline, 12, 16),
			textOp("next line start", 10, lineTwoBaseline, 60, 16),
		},
	}

	// paginateOps drives table fixpoints that need a real box tree; the snap
	// pass is the unit under test, so build its flow index and run it alone.
	ensureFlowIndex(res, contentH)

	if err := snapCrossingTextOps(t.Context(), res, contentH); err != nil {
		t.Fatal(err)
	}

	// Line two crosses the boundary and must move down; the first three runs
	// share one visual line and must keep one baseline.
	if res.Ops[3].Y <= contentH {
		t.Fatalf("snapped line y=%.3f, want below the %.0fpt boundary", res.Ops[3].Y, contentH)
	}

	for idx := 1; idx <= 2; idx++ {
		if res.Ops[idx].Y != lineOneBaseline {
			t.Fatalf("line one run %q y=%.3f, want %.3f: the snap dragged a run from the previous line",
				res.Ops[idx].Text, res.Ops[idx].Y, lineOneBaseline)
		}
	}
}

// TestSnapKeepsLineBaselinesTogether is the layout-shaped form of the same
// regression: every painted run of one visual line must keep one baseline
// through a snap, and the line above the boundary must not move at all.
func TestSnapKeepsLineBaselinesTogether(t *testing.T) {
	t.Parallel()

	res := layoutBondGirlParagraph(t)
	lines := groupTextLinesByBaseline(res)

	if len(lines) != 2 {
		t.Fatalf("paragraph laid out on %d text lines, want 2", len(lines))
	}

	lastLineY := maxTextLineY(lines)

	const (
		lineCrossMargin = 1.0
		lineBoxH        = 16.0
	)

	contentH := lastLineY + lineBoxH - lineCrossMargin
	ensureFlowIndex(res, contentH)

	if err := snapCrossingTextOps(t.Context(), res, contentH); err != nil {
		t.Fatal(err)
	}

	assertLineBaselinesStayTogether(t, res, lines, lastLineY)
}

func layoutBondGirlParagraph(t *testing.T) *Result {
	t.Helper()

	cssSheet := sheet(t, `
body { margin: 0 }
p { margin: 0; font-size: 12pt; line-height: 16pt; width: 500pt }
a { border-bottom: 1pt solid #000 }
`)

	src := `<html><body><p>In 2021, de Armas played a ` +
		`<a href="https://en.wikipedia.org/wiki/Bond_girl">Bond girl</a>.<br>` +
		`Fukunaga wrote the character of a Cuban CIA agent with de Armas in mind.</p></body></html>`

	return layoutHTML(t, src, cssSheet)
}

type textLineGroup struct {
	preY  float64
	delta float64
	ops   []int
}

func groupTextLinesByBaseline(res *Result) map[float64]*textLineGroup {
	lines := map[float64]*textLineGroup{}

	for idx, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		group, found := lines[paintOp.Y]
		if !found {
			group = &textLineGroup{preY: paintOp.Y}
			lines[paintOp.Y] = group
		}

		group.ops = append(group.ops, idx)
	}

	return lines
}

func maxTextLineY(lines map[float64]*textLineGroup) float64 {
	lastLineY := 0.0

	for lineY := range lines {
		if lineY > lastLineY {
			lastLineY = lineY
		}
	}

	return lastLineY
}

func assertLineBaselinesStayTogether(
	t *testing.T, res *Result, lines map[float64]*textLineGroup, lastLineY float64,
) {
	t.Helper()

	for _, group := range lines {
		group.delta = res.Ops[group.ops[0]].Y - group.preY

		for _, idx := range group.ops {
			if res.Ops[idx].Y-group.preY != group.delta {
				t.Fatalf("runs of one visual line moved apart: %q moved %.3f, %q moved %.3f",
					res.Ops[group.ops[0]].Text, group.delta,
					res.Ops[idx].Text, res.Ops[idx].Y-group.preY)
			}
		}
	}

	if lines[lastLineY].delta <= 0 {
		t.Fatalf("last line did not snap: delta=%.3f", lines[lastLineY].delta)
	}

	for lineY, group := range lines {
		if lineY == lastLineY {
			continue
		}

		if group.delta != 0 {
			t.Fatalf("line above the boundary moved by %.3f (%q): the snap dragged it along",
				group.delta, res.Ops[group.ops[0]].Text)
		}
	}
}
