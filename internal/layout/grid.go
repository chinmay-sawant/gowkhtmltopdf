package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Grid auto-flow keywords (grid-auto-flow longhand values).
const (
	gridFlowColumnDense = "column dense"
	gridFlowDense       = "dense"
	gridFlowRowDense    = "row dense"
)

// buildGrid lays out a CSS grid Stage B/C subset: column/row tracks (incl.
// minmax/fr/auto/min-content/max-content lite), independent gaps, template
// areas + named grid-area, auto-flow row/column (sparse or dense), column/row
// spanning, and justify/align-items/self.
//
// display:subgrid copy-inherits the parent's template columns (and
// unspecified gaps). Tracks are re-resolved against the subgrid's own
// content box - not shared parent sizing.
// grid-template-rows: masonry packs items into the shortest column and
// keeps intrinsic heights (no shared row stretch).
func prepareGridTemplateStyles(sty *ResolvedStyle) bool {
	masonryRows := isMasonryTrackList(sty.GridTemplateRows)
	sty.GridTemplateColumns = stripMasonryKeyword(sty.GridTemplateColumns)

	if masonryRows {
		sty.GridTemplateRows = ""
	} else {
		sty.GridTemplateRows = stripMasonryKeyword(sty.GridTemplateRows)
	}

	return masonryRows
}

func (e *engine) buildGrid(node *html.Node, sty ResolvedStyle, availW, posX, posY float64) *box {
	inheritSubgridFromParent(e, node, &sty)

	masonryRows := prepareGridTemplateStyles(&sty)

	ml := e.scalePt(sty.MarginLeft)
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayBlock, x: posX + ml, y: posY,
	}
	boxNode.w = resolveUsedWidth(sty, availW, e)
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, sty)

	contentStart := len(e.ops)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)

	rowGap, columnGap := e.styleGaps(sty)

	areas := parseGridTemplateAreas(sty.GridTemplateAreas)
	colDefs := gridColumnDefs(sty.GridTemplateColumns, areas.cols)

	kids := collectGridKids(e, node)
	colDefs = expandGridColumnDefsForExplicitPlacement(colDefs, kids, areas, e, sty.GridAutoColumns)

	// Handle repeat(auto-fit/auto-fill, minmax(..., 1fr)) which parseGridTrackDefs collapses to auto.
	// Expand at layout time when contentW is known: n = floor((contentW+gap)/(min+gap)), min from minmax.
	if len(colDefs) == 1 && colDefs[0].min.kind == trackAuto && colDefs[0].max.kind == trackAuto {
		if autoDefs := parseAutoFitDefs(sty.GridTemplateColumns, contentW, columnGap, e, len(kids)); len(autoDefs) > 0 {
			colDefs = autoDefs
		}
	}

	// Intrinsic measure lite for min-content / max-content column mins.
	colIntrinsics := measureTrackIntrinsics(e, kids, len(colDefs), true)

	cols := resolveGridTrackSizes(colDefs, contentW, columnGap, e, colIntrinsics)
	if len(cols) == 0 {
		cols = []float64{contentW}
	}

	contentH := resolveContentHeight(sty, e)

	if masonryRows {
		usedH := e.emitMasonryItems(boxNode, kids, cols, columnGap, rowGap, contentX, posY, curY)
		usedH = resolveGridUsedHeight(e, sty, usedH, contentH)
		boxNode.height = usedH
		e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

		return boxNode
	}

	return e.layoutStandardGrid(
		boxNode, sty, kids, areas, cols, columnGap, rowGap, contentX, curY, posY, contentH, contentStart,
	)
}

func (e *engine) layoutStandardGrid(
	boxNode *box, sty ResolvedStyle, kids []*html.Node, areas gridTemplateAreasMap,
	cols []float64, columnGap, rowGap, contentX, curY, posY, contentH float64,
	contentStart int,
) *box {
	columnMajor, densePack := gridAutoFlowMode(sty.GridAutoFlow)
	explicitRows := len(parseGridTrackDefs(sty.GridTemplateRows))
	placed := placeGridItems(
		e, kids, areas, newGridOccupation(len(cols)), len(cols), columnMajor, densePack, explicitRows,
	)

	numRows := gridRowCount(placed, areas)

	rowTemplate := strings.TrimSpace(sty.GridTemplateRows)
	definiteRows := contentH >= 0 && rowTemplate != "" &&
		strings.ToLower(rowTemplate) != cssDisplayNone

	rows, numRows := resolveGridRows(e, sty, kids, numRows, contentH, rowGap, definiteRows)
	pboxes := measureGridPreferredHeights(e, placed, cols, columnGap, rowGap, contentX, curY, posY, rows, definiteRows)

	rowYs := emitGridBoxes(e, sty, boxNode, pboxes, rows, rowGap, posY, curY)

	usedH := curY
	if numRows > 0 {
		usedH = rowYs[numRows-1] + rows[numRows-1]
	}

	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)

	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

func inheritSubgridTracksAndGaps(sty, parentStyle *ResolvedStyle) {
	if strings.TrimSpace(sty.GridTemplateColumns) == "" &&
		strings.TrimSpace(parentStyle.GridTemplateColumns) != "" {
		sty.GridTemplateColumns = parentStyle.GridTemplateColumns
	}

	if sty.RowGap == 0 && sty.ColumnGap == 0 && sty.Gap == 0 {
		sty.RowGap = parentStyle.RowGap
		sty.ColumnGap = parentStyle.ColumnGap
		sty.Gap = parentStyle.Gap
	}
}

func inheritSubgridFromParent(eng *engine, node *html.Node, sty *ResolvedStyle) {
	if eng == nil || node == nil || sty == nil || sty.Display != displaySubgrid {
		return
	}

	sty.Display = displayGrid

	if node.Parent == nil {
		return
	}

	parentStyle := eng.stylePtr(node.Parent)
	if parentStyle == nil || parentStyle == &zeroResolvedStyle {
		return
	}

	inheritSubgridTracksAndGaps(sty, parentStyle)
}

// gridColumnDefs expands the column track list, padding with flexible tracks
// when fewer defs than template-area columns are declared.
func gridColumnDefs(raw string, areaCols int) []gridTrackDef {
	colDefs := parseGridTrackDefs(raw)

	if len(colDefs) == 0 {
		n := areaCols
		if n < 1 {
			n = 1
		}

		colDefs = make([]gridTrackDef, n)
		for i := range colDefs {
			colDefs[i] = flexibleTrack()
		}

		return colDefs
	}

	if need := areaCols - len(colDefs); need > 0 {
		padded := make([]gridTrackDef, areaCols)
		copy(padded, colDefs)

		for i := len(colDefs); i < areaCols; i++ {
			padded[i] = flexibleTrack()
		}

		return padded
	}

	return colDefs
}

// expandGridColumnDefsForExplicitPlacement grows the explicit column defs when
// an item is explicitly placed beyond the template (implicit tracks lite).
// Implicit tracks use grid-auto-columns when specified (auto or 1fr lite),
// otherwise flexibleTrack() (1fr).
func expandGridColumnDefsForExplicitPlacement(
	colDefs []gridTrackDef, kids []*html.Node, areas gridTemplateAreasMap, eng *engine, autoCols string,
) []gridTrackDef {
	need := len(colDefs)
	if areas.cols > need {
		need = areas.cols
	}

	for _, kid := range kids {
		need = gridExplicitNeedForKid(need, kid, areas, eng, colDefs)
	}

	if need <= len(colDefs) {
		return colDefs
	}

	// Cap implicit expansion to avoid unbounded allocations from bogus line numbers.
	if need > maxImplicitGridTracks {
		need = maxImplicitGridTracks
	}

	padded := make([]gridTrackDef, need)
	copy(padded, colDefs)

	autoDef := gridAutoTrackDef(autoCols)
	for i := len(colDefs); i < need; i++ {
		padded[i] = autoDef
	}

	return padded
}

// gridExplicitNeedForKid grows the needed column count for one child's
// line-based placement. Named-area items are covered by areas.cols.
func gridExplicitNeedForKid(
	need int, kid *html.Node, areas gridTemplateAreasMap, eng *engine, colDefs []gridTrackDef,
) int {
	if kid.Type != html.ElementNode || eng == nil {
		return need
	}

	itemStyle := eng.stylePtr(kid)
	if itemStyle == nil {
		return need
	}

	// Named area already covered by areas.cols; only check line-based placement.
	if strings.TrimSpace(itemStyle.GridArea) != "" {
		return need
	}

	// Handle grid-column: 1 / -1 where -1 means last line (nCols+1).
	// For implicit track expansion, -1 resolves to max(len(colDefs), areas.cols, need)+1.
	effEnd := resolveExplicitColumnEnd(itemStyle, colDefs, areas, need)

	return explicitNeedForLinePlacement(itemStyle, effEnd, need)
}

// resolveExplicitColumnEnd resolves a -1 column end to the current last line.
func resolveExplicitColumnEnd(
	itemStyle *ResolvedStyle, colDefs []gridTrackDef, areas gridTemplateAreasMap, need int,
) int {
	if itemStyle.GridColumnEnd != -1 {
		return itemStyle.GridColumnEnd
	}

	base := len(colDefs)
	if areas.cols > base {
		base = areas.cols
	}

	if need > base {
		base = need
	}

	return base + 1
}

// explicitNeedForLinePlacement returns the column count needed by one item's
// line-based start/end/span placement.
func explicitNeedForLinePlacement(itemStyle *ResolvedStyle, effEnd, need int) int {
	switch {
	case itemStyle.GridColumnStart > 0:
		end := itemStyle.GridColumnStart - 1 + itemStyle.GridColumnSpan
		if effEnd > itemStyle.GridColumnStart {
			end = effEnd - 1
		}

		if end < 1 {
			end = 1
		}

		if end > need {
			return end
		}

		return need
	case effEnd > 0:
		// e.g. grid-column-end: 5 without start -> need at least end-1 columns.
		if effEnd-1 > need {
			return effEnd - 1
		}

		return need
	default:
		// Bare span without start does not force extra explicit tracks; auto placement wraps.
		return need
	}
}

//nolint:cyclop,funlen,mnd // auto-fit calculation handles balanced track distribution
func parseAutoFitDefs(raw string, contentW, gap float64, eng *engine, kidCount int) []gridTrackDef {
	lower := strings.ToLower(strings.TrimSpace(raw))

	idx := strings.Index(lower, "repeat(")
	if idx < 0 {
		return nil
	}

	end := -1
	depth := 0

	for scanIdx := idx; scanIdx < len(raw); scanIdx++ {
		if raw[scanIdx] == '(' {
			depth++
		} else if raw[scanIdx] == ')' {
			depth--
			if depth == 0 {
				end = scanIdx

				break
			}
		}
	}

	if end < 0 {
		return nil
	}

	inner := raw[idx+len("repeat(") : end]
	parts := splitTopLevelComma(inner)

	if len(parts) != 2 {
		return nil
	}

	first := strings.TrimSpace(strings.ToLower(parts[0]))
	if first != "auto-fit" && first != "auto-fill" {
		return nil
	}

	trackStr := strings.TrimSpace(parts[1])
	def := parseOneTrackDef(trackStr)

	minW := 200.0

	switch {
	case def.min.kind == trackFixed && def.min.val >= 0:
		minW = eng.scalePt(def.min.val)
	case def.min.kind == trackFixed && def.min.val < 0:
		pct := -def.min.val
		minW = contentW * pct / 100.0
	}

	if minW <= 0 {
		minW = 200.0
	}

	cols := 1
	if contentW > 0 && minW+gap > 0 {
		cols = int((contentW + gap) / (minW + gap))
		if cols < 1 {
			cols = 1
		}

		if kidCount > 0 && cols > kidCount {
			cols = kidCount
		}

		if cols > 12 {
			cols = 12
		}
	}

	out := make([]gridTrackDef, cols)
	for i := range out {
		out[i] = def
	}

	return out
}

// collectGridKids returns the element children that participate in grid
// layout (display:none is skipped).
func collectGridKids(eng *engine, node *html.Node) []*html.Node {
	kids := make([]*html.Node, 0, len(node.Children))

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if eng.stylePtr(child).Display == cssDisplayNone {
			continue
		}

		kids = append(kids, child)
	}

	return kids
}

// gridPlacedBox is a measured grid item with its cell geometry.
type gridPlacedBox struct {
	gridCell
	b         *box
	cellW, cx float64
	prefH     float64
}

// gridCellExtent returns the content width and x offset for one placed cell.
func gridCellExtent(cols []float64, columnGap, contentX float64, page gridCell) (float64, float64) {
	contW := 0.0

	curX := contentX
	for j := range page.col {
		curX += cols[j] + columnGap
	}

	for j := range page.colSpan {
		contW += cols[page.col+j]
		if j > 0 {
			contW += columnGap
		}
	}

	return contW, curX
}

// measureGridPreferredHeights measures items without emitting ops so auto
// row tracks can be sized before final placement.
func measureGridPreferredHeights(
	eng *engine,
	placed []gridCell,
	cols []float64,
	columnGap, rowGap, contentX, curY, posY float64,
	rows []float64,
	definiteRows bool,
) []gridPlacedBox {
	pboxes := make([]gridPlacedBox, 0, len(placed))

	for _, page := range placed {
		contW, curX := gridCellExtent(cols, columnGap, contentX, page)

		was := eng.noEmit
		eng.noEmit = true
		mb := eng.build(page.n, contW, curX, posY+curY)
		eng.noEmit = was

		prefH := 0.0
		if mb != nil {
			prefH = mb.height
		}

		pboxes = append(pboxes, gridPlacedBox{
			gridCell: page,
			b:        nil,
			cellW:    contW,
			cx:       curX,
			prefH:    prefH,
		})

		if !definiteRows && page.rowSpan == 1 && prefH > rows[page.row] {
			rows[page.row] = prefH
		}
	}

	if !definiteRows {
		growSpanningGridRows(rows, pboxes, rowGap)
	}

	return pboxes
}

// growSpanningGridRows distributes extra height across the rows a spanning
// item occupies so its preferred height fits.
func growSpanningGridRows(rows []float64, pboxes []gridPlacedBox, rowGap float64) {
	for _, pbox := range pboxes {
		if pbox.rowSpan <= 1 {
			continue
		}

		sum := 0.0
		for r := range pbox.rowSpan {
			sum += rows[pbox.row+r]
			if r > 0 {
				sum += rowGap
			}
		}

		if pbox.prefH > sum {
			extra := (pbox.prefH - sum) / float64(pbox.rowSpan)
			for r := range pbox.rowSpan {
				rows[pbox.row+r] += extra
			}
		}
	}
}

// emitGridBoxes builds and positions each item, returning the row y-offsets.
func emitGridBoxes(
	eng *engine,
	sty ResolvedStyle,
	boxNode *box,
	pboxes []gridPlacedBox,
	rows []float64,
	rowGap, posY, curY float64,
) []float64 {
	rowYs := make([]float64, len(rows))
	rowYs[0] = curY

	for r := 1; r < len(rows); r++ {
		rowYs[r] = rowYs[r-1] + rows[r-1] + rowGap
	}

	containerJustify := sty.JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}

	containerAlign := sty.AlignItems
	if containerAlign == "" {
		containerAlign = fxStretch
	}

	for i := range pboxes {
		emitGridItem(eng, boxNode, &pboxes[i], rows, rowGap, posY, rowYs, containerJustify, containerAlign)
	}

	return rowYs
}

// emitGridItem builds one item's box and shifts it into its cell.
func emitGridItem(
	eng *engine,
	boxNode *box,
	pbox *gridPlacedBox,
	rows []float64,
	rowGap, posY float64,
	rowYs []float64,
	containerJustify, containerAlign string,
) {
	cellH := gridItemCellHeight(*pbox, rows, rowGap)

	targetX := pbox.cx
	targetY := posY + rowYs[pbox.row]

	cstate := eng.stylePtr(pbox.n)
	justify := gridItemJustify(*cstate, containerJustify)
	align := gridItemAlign(*cstate, containerAlign)

	buildH := gridStretchBuildHeight(align, cellH, *cstate)

	oldMax := eng.imgMaxW

	if pbox.cellW > 0 {
		eng.imgMaxW = pbox.cellW
	}

	var cblock *box

	if buildH > 0 {
		override := *cstate
		override.Height = buildH
		override.HeightPercent = -1
		override.BoxSizing = borderBox
		cblock = eng.buildWithStyle(pbox.n, &override, pbox.cellW, pbox.cx, targetY)
	} else {
		cblock = eng.build(pbox.n, pbox.cellW, pbox.cx, targetY)
	}

	eng.imgMaxW = oldMax

	if cblock == nil {
		return
	}

	pbox.b = cblock

	deltaX := targetX - pbox.b.x
	deltaY := targetY - pbox.b.y
	deltaX += gridAlignOffset(justify, pbox.cellW, pbox.b.w)
	deltaY += gridAlignOffset(align, cellH, pbox.b.height)

	eng.shiftBoxOps(pbox.b, deltaX, deltaY)
	pbox.b.x += deltaX
	pbox.b.y += deltaY
	boxNode.children = append(boxNode.children, pbox.b)
}

// gridItemCellHeight sums the track sizes a spanning item occupies (with gaps).
func gridItemCellHeight(pbox gridPlacedBox, rows []float64, rowGap float64) float64 {
	cellH := 0.0

	for r := range pbox.rowSpan {
		cellH += rows[pbox.row+r]
		if r > 0 {
			cellH += rowGap
		}
	}

	return cellH
}

// gridItemJustify resolves the effective justify-self for one item.
func gridItemJustify(cstate ResolvedStyle, container string) string {
	justify := cstate.JustifySelf
	if justify == "" || justify == overflowAuto {
		justify = container
	}

	return justify
}

// gridItemAlign resolves the effective align-self for one item.
func gridItemAlign(cstate ResolvedStyle, container string) string {
	align := cstate.AlignSelf
	if align == "" || align == overflowAuto {
		align = container
	}

	return align
}

// gridStretchBuildHeight returns the box height used to stretch a grid item
// into its cell, or -1 when the item is not stretched.
func gridStretchBuildHeight(align string, cellH float64, cstate ResolvedStyle) float64 {
	// Default stretch: border box fills the grid area (CSS Grid Section 10.2).
	if (align == fxStretch || align == "") && cellH > 0 &&
		cstate.Height < 0 && cstate.HeightPercent < 0 {
		return cellH
	}

	return -1
}

// resolveGridUsedHeight bumps the used height to the definite height and the
// min border-box floor.
func resolveGridUsedHeight(eng *engine, sty ResolvedStyle, usedH, contentH float64) float64 {
	usedH += eng.scalePt(sty.PaddingBottom)

	if sty.Height >= 0 {
		height := eng.scalePt(sty.Height)
		if sty.BoxSizing != borderBox {
			height += eng.scalePt(sty.PaddingTop) + eng.scalePt(sty.PaddingBottom) +
				eng.scalePt(sty.BorderTop.Width) + eng.scalePt(sty.BorderBottom.Width)
		}

		if usedH < height {
			usedH = height
		}
	}

	minBorderH := eng.scalePt(sty.PaddingTop) + eng.scalePt(sty.BorderTop.Width) +
		eng.scalePt(sty.PaddingBottom) + eng.scalePt(sty.BorderBottom.Width)
	if contentH >= 0 {
		minBorderH += contentH
	}

	if usedH < minBorderH {
		usedH = minBorderH
	}

	return usedH
}

// --- Width / height helpers -------------------------------------------------

// resolveUsedWidth computes border-box width. WidthPercent against a
// non-positive (indefinite) availW is treated as auto (fill remaining).
// Shared by flex/grid/multicol (block keeps its own min/max/margin-auto path).
func resolveUsedWidth(sty ResolvedStyle, availW float64, engN *engine) float64 {
	ml, mr := engN.scalePt(sty.MarginLeft), engN.scalePt(sty.MarginRight)

	width := availW - ml - mr
	if width < 0 {
		width = 0
	}

	if sty.WidthPercent >= 0 {
		// Cyclic % -> auto (keep fill-remaining width).
		if availW > 0 && !math.IsInf(availW, 0) && availW < 1e12 {
			width = availW * sty.WidthPercent / oneHundred
		}
	} else if sty.Width >= 0 {
		width = engN.scalePt(sty.Width)
		if sty.BoxSizing != borderBox {
			width += engN.scalePt(sty.PaddingLeft) + engN.scalePt(sty.PaddingRight) +
				engN.scalePt(sty.BorderLeft.Width) + engN.scalePt(sty.BorderRight.Width)
		}
	}

	return width
}

// resolveContentHeight returns definite content-box height, or -1 when auto.
// HeightPercent only resolves when Height was already made definite by a parent
// stretch; unresolved HeightPercent (indefinite CB) is treated as auto.
// Shared by flex/grid/multicol.
func resolveContentHeight(sty ResolvedStyle, engN *engine) float64 {
	if sty.HeightPercent >= 0 && sty.Height < 0 {
		// Cyclic % honesty: indefinite containing block -> auto.
		return -1
	}

	if sty.Height < 0 {
		return -1
	}

	height := engN.scalePt(sty.Height)
	if sty.BoxSizing == borderBox {
		height -= engN.scalePt(sty.PaddingTop) + engN.scalePt(sty.PaddingBottom) +
			engN.scalePt(sty.BorderTop.Width) + engN.scalePt(sty.BorderBottom.Width)
	}

	if height < 0 {
		height = 0
	}

	return height
}

// --- Gaps / alignment -------------------------------------------------------

// styleGaps returns scaled row/column gaps. When both longhands are unset (0),
// fall back to the Gap shorthand for both axes. Shared by flex and grid.
func (e *engine) styleGaps(sty ResolvedStyle) (float64, float64) {
	if sty.RowGap == 0 && sty.ColumnGap == 0 {
		g := e.scalePt(sty.Gap)

		return g, g
	}

	return e.scalePt(sty.RowGap), e.scalePt(sty.ColumnGap)
}

// gridAlignOffset returns the inline/block offset for start/end/center/stretch.
// stretch is treated as start (lite).
func gridAlignOffset(value string, cell, item float64) float64 {
	switch value {
	case fxFlexEnd, fxEnd, cssVerticalAlignBottom, floatRight:
		if cell > item {
			return cell - item
		}
	case fxCenter:
		if cell > item {
			return (cell - item) / two
		}
	}

	return 0
}
