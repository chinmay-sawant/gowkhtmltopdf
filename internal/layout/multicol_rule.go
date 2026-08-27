package layout

// columnRulePaints reports a CSS column-rule that should stroke. Style none
// or a zero gap (no gutter to sit in) suppresses the rule.
func columnRulePaints(sty ResolvedStyle, gap float64) bool {
	if gap <= layoutEpsilon || sty.ColumnRuleWidth <= 0 {
		return false
	}

	switch sty.ColumnRuleStyle {
	case solidKeyword, borderStyleDashed, borderStyleDotted:
		return true
	}

	return false
}

func columnRuleStrokeColor(sty ResolvedStyle) (float64, float64, float64) {
	if sty.ColumnRuleColorSet {
		return sty.ColumnRuleColor[0], sty.ColumnRuleColor[1], sty.ColumnRuleColor[2]
	}

	return sty.Color[0], sty.Color[1], sty.Color[2]
}

// paintColumnRules strokes a vertical rule in the center of each column-gap
// for one multicol line. Horizontal column-axis is out of scope (this engine
// only lays columns along the row axis).
func (e *engine) paintColumnRules(
	sty ResolvedStyle, nCols int, colW, gap, contentX, topY, lineH float64,
) {
	if e == nil || e.noEmit || !columnRulePaints(sty, gap) || nCols < 2 || lineH <= 0 {
		return
	}

	width := e.scalePt(sty.ColumnRuleWidth)
	if width <= 0 {
		return
	}

	red, green, blue := columnRuleStrokeColor(sty)
	ruleX := contentX + colW + gap/two

	for col := 0; col < nCols-1; col++ {
		ops := appendBorderLineOps(
			nil, ruleX, topY, 0, lineH, width, sty.ColumnRuleStyle, red, green, blue,
		)
		for i := range ops {
			e.add(ops[i])
		}

		ruleX += colW + gap
	}
}
