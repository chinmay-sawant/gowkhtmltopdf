package layout

import (
	"slices"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// flowFlexVerticalRow maps a row flex container's logical inline axis onto
// physical Y. The ordinary row path is intentionally horizontal-only; this
// path keeps vertical writing modes from silently laying their main axis on X.
//
//nolint:wsl // logical-axis setup stays adjacent to its placement phases
func (e *engine) flowFlexVerticalRow(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, gap float64,
) float64 {
	contentH := resolveFlexColumnContentHeight(style, e)
	items := e.flexColumnItems(kids, contentW, contentH)
	if len(items) == 0 {
		return curY
	}

	if flexRowPhysicalReverse(style) {
		slices.Reverse(items)
	}

	heights := e.flexColumnHeights(items, contentH, gap)
	sumH := 0.0
	for idx, height := range heights {
		sumH += height + flexColumnMainMargins(e, items[idx])
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	startY, justifyGap := justifyColumnStart(
		flexMainJustify(style), contentH, curY, sumH+gaps, sumH, gap, len(items),
	)
	endY := e.buildVerticalRowItems(
		parent, style, items, heights, contentW, contentX, topY, curY, startY, justifyGap,
	)

	if contentH >= 0 && endY < curY+contentH {
		return curY + contentH
	}

	return endY
}

//nolint:wsl // vertical row placement mirrors the column placement phases
func (e *engine) buildVerticalRowItems(
	parent *box, style ResolvedStyle, items []flexColMeas, heights []float64,
	contentW, contentX, topY, curY, startY, justifyGap float64,
) float64 {
	leftY := startY
	endY := curY
	poll := newCtxPoll(e.ctx)

	for idx, item := range items {
		if poll.poll() {
			e.err = poll.err

			return endY
		}

		cstate := e.stylePtr(item.n)
		if cstate.MarginTopAuto {
			leftY += 0
		} else {
			leftY += e.scalePt(cstate.MarginTop)
		}

		override := e.flexColumnItemOverride(item.n, *cstate, style, heights[idx], contentW)
		cblock := e.buildWithStyle(item.n, &override, contentW, contentX, topY+leftY)
		if cblock != nil {
			targetX := verticalRowCrossX(style, contentX, contentW, *cstate, cblock.w)
			dx := targetX - cblock.x
			dy := (topY + leftY) - cblock.y
			e.shiftBoxOps(cblock, dx, dy)
			cblock.x += dx
			cblock.y += dy

			if parent != nil {
				parent.children = append(parent.children, cblock)
			}
		}

		leftY += heights[idx]
		if cstate.MarginBottomAuto {
			leftY += 0
		} else {
			leftY += e.scalePt(cstate.MarginBottom)
		}
		endY = leftY
		if idx < len(items)-1 {
			leftY += justifyGap
			endY = leftY
		}
	}

	return endY
}

func verticalRowCrossX(
	containerStyle ResolvedStyle, contentX, contentW float64,
	itemStyle ResolvedStyle, itemW float64,
) float64 {
	align := containerStyle.AlignItems
	if itemStyle.AlignSelf != "" && itemStyle.AlignSelf != fxAuto {
		align = itemStyle.AlignSelf
	}

	if align == fxCenter {
		return contentX + (contentW-itemW)/2
	}

	if containerStyle.WritingMode == writingModeVerticalRL || align == fxFlexEnd || align == fxEnd {
		return contentX + contentW - itemW
	}

	return contentX
}

// flexRowPhysicalReverse reports whether the row main axis progresses toward
// the physical left or top edge when the logical direction is applied.
// direction reverses row and row-reverse in horizontal and vertical writing.
func flexRowPhysicalReverse(style ResolvedStyle) bool {
	reverse := style.FlexDirection == fxRowRev
	if style.Direction == cssDirectionRTL {
		return !reverse
	}

	return reverse
}

// flexRowNormalMargins returns the non-auto outer margins that consume row
// main-axis space. Auto margins absorb only the positive space left after
// flex sizing.
//
//nolint:varnamelen,wsl // e is the package's established engine receiver name
func flexRowNormalMargins(e *engine, item flexMeas) float64 {
	style := e.stylePtr(item.n)
	margins := 0.0
	if !style.MarginLeftAuto {
		margins += e.scalePt(style.MarginLeft)
	}
	if !style.MarginRightAuto {
		margins += e.scalePt(style.MarginRight)
	}

	return margins
}

// flexColumnMainMargins returns the normal, non-auto margins that consume
// space on a column flex container's main axis. Flex items do not collapse
// their vertical margins with one another.
//
//nolint:varnamelen,wsl // e is the package's established engine receiver name
func flexColumnMainMargins(e *engine, item flexColMeas) float64 {
	style := e.stylePtr(item.n)
	margins := 0.0
	if !style.MarginTopAuto {
		margins += e.scalePt(style.MarginTop)
	}
	if !style.MarginBottomAuto {
		margins += e.scalePt(style.MarginBottom)
	}

	return margins
}
