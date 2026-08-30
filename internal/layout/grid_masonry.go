package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func isMasonryTrackList(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "masonry")
}

// stripMasonryKeyword clears a lone "masonry" column list so layout falls
// through to auto tracks. Row masonry is handled in emitMasonryItems.
func stripMasonryKeyword(raw string) string {
	if isMasonryTrackList(raw) {
		return ""
	}

	return raw
}

func shiftMasonryBox(e *engine, cblock *box, targetX, targetY float64) {
	dx := targetX - cblock.x
	dy := targetY - cblock.y

	if dx != 0 || dy != 0 {
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy
	}
}

func masonryMaxBottom(colBot []float64, rowGap float64) float64 {
	maxBot := 0.0
	for _, height := range colBot {
		if height > maxBot {
			maxBot = height
		}
	}

	if maxBot > rowGap {
		maxBot -= rowGap
	}

	return maxBot
}

// emitMasonryItems packs each item into the shortest column and keeps
// content-sized heights (CSS Grid L3 masonry lite).
func (e *engine) emitMasonryItems(
	boxNode *box, kids []*html.Node, cols []float64,
	columnGap, rowGap, contentX, posY, curY float64,
) float64 {
	nCols := len(cols)
	if nCols == 0 {
		return curY
	}

	colX := make([]float64, nCols)
	x := contentX

	for i, width := range cols {
		colX[i] = x
		x += width + columnGap
	}

	colBot := make([]float64, nCols)

	for _, kid := range kids {
		col, span, cellW := masonryPlacement(e.stylePtr(kid), cols, columnGap, colBot)
		targetY := posY + curY + masonrySpanStart(colBot, col, span)
		cblock := e.build(kid, cellW, colX[col], targetY)

		if cblock == nil {
			continue
		}

		shiftMasonryBox(e, cblock, colX[col], targetY)
		boxNode.children = append(boxNode.children, cblock)

		next := masonrySpanStart(colBot, col, span) + cblock.height + rowGap
		for i := range span {
			colBot[col+i] = next
		}
	}

	return curY + masonryMaxBottom(colBot, rowGap)
}

func masonryPlacement(
	stylePtr *ResolvedStyle, cols []float64, columnGap float64, colBot []float64,
) (int, int, float64) {
	nCols := len(cols)
	span := 1

	if stylePtr != nil && stylePtr.GridColumnSpan > 1 {
		span = stylePtr.GridColumnSpan
	}

	if span > nCols {
		span = nCols
	}

	var col int
	if stylePtr != nil && stylePtr.GridColumnStart > 0 {
		col = stylePtr.GridColumnStart - 1
		if col < 0 {
			col = 0
		}

		if col+span > nCols {
			col = nCols - span
		}
	} else {
		col = shortestMasonryColumn(colBot, span)
	}

	cellW := 0.0
	for i := range span {
		cellW += cols[col+i]
		if i > 0 {
			cellW += columnGap
		}
	}

	return col, span, cellW
}

func masonrySpanStart(colBot []float64, col, span int) float64 {
	start := colBot[col]
	for i := 1; i < span; i++ {
		if colBot[col+i] > start {
			start = colBot[col+i]
		}
	}

	return start
}

func shortestMasonryColumn(colBot []float64, span int) int {
	best, bestH := 0, math.Inf(1)

	for i := 0; i+span <= len(colBot); i++ {
		h := masonrySpanStart(colBot, i, span)
		if h < bestH {
			best, bestH = i, h
		}
	}

	return best
}
