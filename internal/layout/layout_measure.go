package layout

import (
	"math"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

func bandEmSize(ops []Op, indices []int) float64 {
	emSize := 8.0

	for _, i := range indices {
		if ops[i].Size > 0 {
			emSize = ops[i].Size

			break
		}
	}

	return emSize
}

// applyBandShifts moves every op onto the baseline of the nearest band.
func applyBandShifts(ops []Op, start, end int, shifts []bandShift, emSize float64) {
	for idx := start; idx < end && idx < len(ops); idx++ {
		posY := ops[idx].Y
		// Nearest band baseline.
		best, bestD := 0, math.Abs(posY-shifts[0].y0)

		for si := 1; si < len(shifts); si++ {
			d := math.Abs(posY - shifts[si].y0)
			if d < bestD {
				bestD, best = d, si
			}
		}

		if bestD <= emSize*1.5 {
			ops[idx].Y += shifts[best].dy
		}
	}
}

// band is a group of op indices sharing a baseline Y (average kept coherent).
type band struct {
	y   float64
	idx []int
}

// bandShift maps an old baseline to the delta that lands it on its target.
type bandShift struct{ y0, dy float64 }

// collectTextBands groups text/bullet ops into baseline bands.
func collectTextBands(ops []Op, start, end int, yEps float64) []band {
	var bands []band

	for idx := start; idx < end && idx < len(ops); idx++ {
		paintOp := ops[idx]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		placed := false

		for bi := range bands {
			if math.Abs(bands[bi].y-paintOp.Y) <= yEps {
				bands[bi].idx = append(bands[bi].idx, idx)
				// Keep average Y so multi-glyph lines stay coherent.
				n := float64(len(bands[bi].idx))
				bands[bi].y = (bands[bi].y*(n-1) + paintOp.Y) / n
				placed = true

				break
			}
		}

		if !placed {
			bands = append(bands, band{y: paintOp.Y, idx: []int{idx}})
		}
	}

	return bands
}

// sortBandsTopDown sorts bands by Y ascending.
func sortBandsTopDown(bands []band) {
	for i := 0; i < len(bands); i++ {
		for j := i + 1; j < len(bands); j++ {
			if bands[j].y < bands[i].y {
				bands[i], bands[j] = bands[j], bands[i]
			}
		}
	}
}

// interpolatedBandTargets places the first baseline ~0.7em into the cell, the
// last near the bottom, and interpolates the rest; nil when the cell is too
// small to redistribute.
func interpolatedBandTargets(ops []Op, bands []band, innerTop, innerBot float64) []float64 {
	if len(bands) == 1 {
		return nil
	}
	// Use first text size as em estimate.
	emSize := 8.0

	for _, i := range bands[0].idx {
		if ops[i].Size > 0 {
			emSize = ops[i].Size

			break
		}
	}

	first := innerTop + emSize*firstLineEm
	last := innerBot - emSize*baselineInsetRatio

	if last <= first {
		return nil
	}

	targets := make([]float64, len(bands))
	for i := range bands {
		targets[i] = first + (last-first)*float64(i)/float64(len(bands)-1)
	}

	return targets
}

// measureCellContent returns the max-content border-box width of the cell
// (longest unwrapped line, not longest word). Using min-content here made
// auto tables shrink-wrap to a rivulet of columns and inflate row heights
// via forced wraps (wiki filmography / any dense multi-column table).
func (e *engine) measureCellContent(n *html.Node, st ResolvedStyle) float64 {
	minW, maxW := e.measureCellMinMax(n, st)
	_ = minW

	return maxW
}

// measureCellMinMax returns min-content and max-content border-box widths.
// min-content ≈ longest unbreakable word; max-content ≈ widest line if soft
// wraps are not taken (hard breaks from <br>/blocks still split lines).
// When word-break/overflow-wrap allow breaking long tokens, min-content uses
// the longest soft segment (or widest single rune) so URL-heavy cells can
// shrink instead of forcing the table past the page edge.
//
// Short nowrap-only lines (wiki Ref cells with [127][128]) use the full line
// as min-content so adjacent cite markers stay on one horizontal line instead
// of wrapping into a stacked, overlapping pair in a one-marker-wide column.
func (e *engine) measureCellMinMax(node *html.Node, style ResolvedStyle) (float64, float64) {
	cellMeas := &cellMeasure{ //nolint:exhaustruct // zero fields are the flushed-line state
		engine: e,
		em:     style.FontSize,
		style:  style,
	}
	cellMeas.walk(node, style, style.WhiteSpace == cssWhiteSpaceNowrap || style.WhiteSpace == cssWhiteSpacePre)
	cellMeas.flushLine()

	chrome := e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)
	minW := cellMeas.longestWord + chrome
	maxW := cellMeas.maxW + chrome

	if maxW < minW {
		maxW = minW
	}

	return minW, maxW
}

// cellMeasure accumulates min/max-content width contributions while walking a
// table cell's subtree (see measureCellMinMax).
type cellMeasure struct {
	engine         *engine
	style          ResolvedStyle
	em             float64
	lineW          float64
	maxW           float64
	longestWord    float64
	lineOnlyNowrap bool
	lineHasInk     bool
	// spaceW is the memoized width of one ASCII space for the last measured
	// style identity, so a cell with many text nodes does not re-measure " "
	// per node.
	spaceSet      bool
	spaceFamHash  uint64
	spaceWeight   int
	spaceItalic   bool
	spaceFontSize float64
	spaceLSpacing float64
	spaceW        float64
}

// flushLine folds the current line into maxW and resets the line state.
func (m *cellMeasure) flushLine() {
	if m.lineW > m.maxW {
		m.maxW = m.lineW
	}

	if m.lineHasInk && m.lineOnlyNowrap && m.lineW > m.longestWord {
		// Cap so a pathological nowrap paragraph does not freeze the table
		// at max-content; multi-cite clusters are well under ~10em.
		em := m.em
		if em < 1 {
			em = 10
		}

		if m.lineW <= em*10*m.engine.scale {
			m.longestWord = m.lineW
		}
	}

	m.lineW = 0
	m.lineOnlyNowrap = true
	m.lineHasInk = false
}

// spaceWidth returns the measured width of one ASCII space for sty. The
// advance depends only on the style's family hash, weight/italic face
// variant, font size and letter-spacing, so the result is cached per style
// identity and reused across text nodes in one cell.
func (m *cellMeasure) spaceWidth(sty ResolvedStyle) float64 {
	if m.spaceSet &&
		m.spaceFamHash == sty.famHash &&
		m.spaceWeight == sty.FontWeight &&
		m.spaceItalic == sty.FontItalic &&
		m.spaceFontSize == sty.FontSize &&
		m.spaceLSpacing == sty.LetterSpacing {
		return m.spaceW
	}

	m.spaceW = m.engine.measureTextFace(" ", sty)
	m.spaceSet = true
	m.spaceFamHash = sty.famHash
	m.spaceWeight = sty.FontWeight
	m.spaceItalic = sty.FontItalic
	m.spaceFontSize = sty.FontSize
	m.spaceLSpacing = sty.LetterSpacing

	return m.spaceW
}

// walk measures one node's contribution to the current line.
func (m *cellMeasure) walk(nodeN *html.Node, cstate ResolvedStyle, nowrap bool) {
	switch nodeN.Type {
	case html.TextNode:
		m.measureText(nodeN.Text, cstate, nowrap)
	case html.ElementNode:
		m.measureElement(nodeN, cstate, nowrap)
	case html.CommentNode, html.DoctypeNode:
		return
	}
}

// measureText accumulates a text run into the current line, using the same
// face selection as paint (measureTextFace) — mismatched metrics undersize
// columns and force emergency wraps on words that should fit.
//
//nolint:cyclop // word-scan and nowrap paths share state; splitting hurts readability
func (m *cellMeasure) measureText(text string, cstate ResolvedStyle, nowrap bool) {
	eng := m.engine

	if !nowrap {
		// Walk words without strings.Fields: no []string or word copies.
		// Matching white-space:normal — runs of HTML space collapse to one gap.
		if !hasNonHTMLSpace(text) {
			return
		}

		m.lineOnlyNowrap = false
		m.lineHasInk = true
		chromeW := inlineMeasurementChromeWidth(eng, cstate)
		m.lineW += chromeW

		spaceW := m.spaceWidth(cstate)

		// Leading space if original had leading WS and line already started.
		if m.lineW > 0 && len(text) > 0 && isHTMLSpace(text[0]) {
			m.lineW += spaceW
		}

		wStart := 0
		first := true

		for wStart < len(text) {
			for wStart < len(text) && isHTMLSpace(text[wStart]) {
				wStart++
			}

			if wStart >= len(text) {
				break
			}

			wEnd := wStart
			for wEnd < len(text) && !isHTMLSpace(text[wEnd]) {
				wEnd++
			}

			word := text[wStart:wEnd]

			if !first {
				m.lineW += spaceW
			}

			first = false
			wordW := eng.measureTextFace(word, cstate)
			m.lineW += wordW
			m.noteWord(eng.minContentWidth(word, cstate, wordW+chromeW))

			wStart = wEnd
		}

		return
	}

	full := eng.measureTextFace(text, cstate) + inlineMeasurementChromeWidth(eng, cstate)
	m.lineW += full
	m.noteWord(eng.minContentWidth(text, cstate, full))

	if hasNonHTMLSpace(text) {
		m.lineHasInk = true
	}
}

// inlineMeasurementChromeWidth keeps intrinsic flex/grid measurements in
// lockstep with inline paint. Without the padding and border contribution,
// flex items containing code/mark spans receive a box that is just narrower
// than their painted inline chrome and wrap unexpectedly.
func inlineMeasurementChromeWidth(eng *engine, st ResolvedStyle) float64 {
	if eng == nil || st.Display != cssDisplayInline {
		return 0
	}

	return eng.scalePt(st.PaddingLeft) + eng.scalePt(st.PaddingRight) +
		eng.scalePt(st.BorderLeft.Width) + eng.scalePt(st.BorderRight.Width)
}

// noteWord records a token's min-content width.
func (m *cellMeasure) noteWord(uw float64) {
	if uw > m.longestWord {
		m.longestWord = uw
	}
}

// measureElement handles br, replaced images, and block-level in-cell boxes.
func (m *cellMeasure) measureElement(nodeN *html.Node, childCS ResolvedStyle, nowrap bool) {
	if childCS.Display == cssDisplayNone {
		return
	}

	if nodeN.Name == "br" {
		m.flushLine()

		return
	}
	// Replaced images contribute their used CSS-pixel width (wiki thumbs).
	if nodeN.Name == cssTagImg {
		innerW := m.engine.measureImageWidth(nodeN, childCS)
		m.noteWord(innerW)
		m.lineOnlyNowrap = false
		m.lineHasInk = true
		m.lineW += innerW

		return
	}
	// Block-level in-cell boxes start a new line (simplified).
	blockish := isCellBlockish(childCS.Display)
	m.walkBlockChildren(nodeN, childCS, nowrap, blockish)
}

// isCellBlockish reports displays that break the current measured line.
func isCellBlockish(display string) bool {
	switch display {
	case displayBlock, displayTable, displayListItem, displayFlex, displayGrid:
		return true
	default:
		return false
	}
}

// walkBlockChildren walks an element's children, flushing the line before and
// after block-level boxes.
func (m *cellMeasure) walkBlockChildren(nodeN *html.Node, childCS ResolvedStyle, nowrap, blockish bool) {
	if blockish {
		m.flushLine()
	}

	childNowrap := nowrap || childCS.WhiteSpace == cssWhiteSpaceNowrap || childCS.WhiteSpace == cssWhiteSpacePre
	for _, child := range nodeN.Children {
		m.walk(child, childCS, childNowrap)
	}

	if blockish {
		m.flushLine()
	}
}

// wordBreakPolicy is the single table for "how may a token split?" —
// white-space, word-break and overflow-wrap combine into one enum.
// Shared by intrinsic min-content measurement and inline overflow packing.
type wordBreakPolicy int

const (
	breakNormal wordBreakPolicy = iota
	breakAll                    // word-break:break-all / overflow-wrap:anywhere
	breakWord                   // overflow-wrap:break-word (soft only)
	breakNever                  // white-space:nowrap|pre
)

func wordBreakOf(sty ResolvedStyle) wordBreakPolicy {
	if sty.WhiteSpace == cssWhiteSpaceNowrap || sty.WhiteSpace == cssWhiteSpacePre {
		return breakNever
	}

	if sty.WordBreak == "break-all" || sty.OverflowWrap == overflowWrapAnywhere {
		return breakAll
	}

	if sty.OverflowWrap == overflowWrapBreakWord {
		return breakWord
	}

	return breakNormal
}

// softModeOf maps a break policy to the soft-wrap rune table used by
// breakToken / splitTextToWidth. Emergency (breakNormal) uses URL-ish
// opportunities only so ordinary hyphenated words stay intact.
func softModeOf(pol wordBreakPolicy) softBreakMode {
	switch pol {
	case breakAll:
		return softBreakNone
	case breakWord:
		return softBreakWord
	case breakNever:
		return softBreakURL
	case breakNormal:
		return softBreakURL
	}

	return softBreakURL
}

// minContentWidth is the min-content contribution of a single token under
// the element's word-break / overflow-wrap policy (CSS min-content). full is
// the token's already-measured full advance (the caller measures it for the
// line anyway), so breakNormal/breakNever return it without re-measuring.
// Emergency print wrapping (tokens wider than the used line) is layout-only
// and must not shrink table column mins to a single rune.
func (e *engine) minContentWidth(cssSheet string, sty ResolvedStyle, full float64) float64 {
	if cssSheet == "" {
		return 0
	}

	switch wordBreakOf(sty) {
	case breakNever:
		return full
	case breakAll:
		return e.maxRuneWidth(cssSheet, sty)
	case breakWord:
		// Soft opportunities (/, ?, &, …) split the token for min-content.
		return e.maxSoftSegmentWidth(cssSheet, sty)
	case breakNormal:
		return full
	}

	return full
}

func (e *engine) maxRuneWidth(s string, st ResolvedStyle) float64 {
	var widest float64

	for _, r := range s {
		w := e.measureRuneFace(r, st)
		if w > widest {
			widest = w
		}
	}

	return widest
}

func (e *engine) maxSoftSegmentWidth(cssS string, sty ResolvedStyle) float64 {
	if cssS == "" {
		return 0
	}

	var widest, cur float64

	for _, r := range cssS {
		cur += e.measureRuneFace(r, sty)
		// Soft-break runes end a min-content segment; residual after the
		// last break is flushed below (covers tokens with no soft points).
		if isSoftWrapRune(r, softBreakWord) {
			if cur > widest {
				widest = cur
			}

			cur = 0
		}
	}

	if cur > widest {
		widest = cur
	}

	if widest <= 0 {
		return e.maxRuneWidth(cssS, sty)
	}

	return widest
}

// countLeadingTHRows returns how many consecutive leading rows are composed
// entirely of <th> cells (column header band without an explicit <thead>).
// Empty rows (no cells) are skipped so a leading blank tr does not block
// detection of the real header band.
func countLeadingTHRows(rows [][]*html.Node) int {
	node := 0

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		allTH := true

		for _, cell := range row {
			if cell == nil || cell.Name != "th" {
				allTH = false

				break
			}
		}

		if !allTH {
			break
		}

		node++
	}

	return node
}

// stripEmptyTableRows removes rows that have no table-cell elements.
// Safe: rowspan placement only creates geometry for rows that exist in the
// source list; empty tr nodes never start cells and only produced phantom
// min-height bands in the border-collapse grid.
func stripEmptyTableRows(rows [][]*html.Node) [][]*html.Node {
	if len(rows) == 0 {
		return rows
	}

	out := rows[:0]

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		out = append(out, row)
	}
	// If every row was empty, return a fresh empty slice (not a shared buf).
	if len(out) == 0 {
		return nil
	}

	return out
}

// rowCellsHaveNoInk reports whether every cell in the row is free of text,
// images, and other non-whitespace content (padding-only empty th/td).
// The ink flags are recorded once per cell at build time (buildCell), so
// this is a flat flag loop instead of a per-row subtree walk.
func rowCellsHaveNoInk(cells []*box) bool {
	for _, cell := range cells {
		if cell != nil && cell.hasInk {
			return false
		}
	}

	return true
}

func nodeHasTableInk(node *html.Node) bool {
	if node == nil {
		return false
	}

	switch node.Type {
	case html.TextNode:
		return strings.TrimSpace(node.Text) != ""
	case html.ElementNode:
		if isTableInkElement(node.Name) {
			return true
		}

		for _, child := range node.Children {
			if nodeHasTableInk(child) {
				return true
			}
		}
	case html.CommentNode, html.DoctypeNode:
		return false
	}

	return false
}

// isTableInkElement reports element names that carry ink without text.
func isTableInkElement(name string) bool {
	switch name {
	case cssTagImg, "svg", "video", "canvas", "br":
		return true
	}

	return false
}

func isHTMLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// hasNonHTMLSpace reports that s contains at least one non-HTML-whitespace byte.
// Used instead of strings.TrimSpace(s) != "" to avoid the TrimSpace string header.
func hasNonHTMLSpace(s string) bool {
	for i := range len(s) {
		if !isHTMLSpace(s[i]) {
			return true
		}
	}

	return false
}

// measureImageWidth returns the same used content width that buildImage and
// inline painting use. Keeping intrinsic/attribute/CSS/max/containing-block
// policy in usedImageSize prevents float/table shrink-to-fit from disagreeing
// with the eventually painted image.
func (e *engine) measureImageWidth(n *html.Node, st ResolvedStyle) float64 {
	if n == nil {
		return 0
	}

	return e.usedImageSize(n, st, e.resolveImage(n.Attribute("src"))).w
}

// measureLargestImageWidth walks n for the widest descendant <img>.
func (e *engine) measureLargestImageWidth(node *html.Node) float64 {
	if node == nil {
		return 0
	}

	var best float64

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Name == cssTagImg {
				st := e.styleVal(node)
				if w := e.measureImageWidth(node, st); w > best {
					best = w
				}
			}

			for _, c := range node.Children {
				walk(c)
			}
		}
	}
	walk(node)

	return best
}

// layoutCell measures the height of a cell's content (no ops emitted).
func (e *engine) layoutCell(n *html.Node, sty ResolvedStyle, width float64) float64 {
	_, contentW := e.contentBox(0, width, sty)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)
	enclose := e.pushBFCFloats(sty, 0, contentW)
	curY = e.flowChildren(nil, n.Children, sty, contentW, 0, 0, curY)

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(0, curY)
	}

	e.popBFCFloats(enclose)

	return curY + e.scalePt(sty.PaddingBottom) + e.scalePt(sty.BorderBottom.Width)
}

func colSpan(n *html.Node) int {
	return tableSpan(n.Attribute("colspan"))
}

func cellRowSpan(n *html.Node) int {
	return tableSpan(n.Attribute("rowspan"))
}

func tableSpan(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1
	}

	if v, err := strconv.Atoi(value); err == nil && v > 1 {
		return v
	}

	return 1
}

// tableColumnEnv is everything the column-sizing pass needs; no DOM, no ops.
type tableColumnEnv struct {
	colMin []float64 // min-content per column (content only)
	colW   []float64 // max-content per column (content only); mutated to used widths
	colPct []float64 // width:% hints, -1 = auto
	colAbs []float64 // width:pt hints, -1 = auto
	chrome float64   // spacing + border + padding
	availW float64
	// tableW is a definite table border-box width when >= 0; -1 means auto.
	tableW float64
}

// sizeTableColumns resolves used column widths and the table border-box width
// (CSS2.1-lite auto/fixed: sum, % and abs hints, min floors, definite scaling).
func sizeTableColumns(env tableColumnEnv) ([]float64, float64) {
	colW := env.colW
	colMin := env.colMin
	colPct := env.colPct
	colAbs := env.colAbs
	chrome := env.chrome
	availW := env.availW

	nCols := len(colW)
	if nCols == 0 {
		if env.tableW >= 0 {
			return colW, env.tableW
		}

		return colW, availW
	}

	sumMax, sumMin := columnWidthSums(colW, colMin, chrome)

	definiteTable := env.tableW >= 0
	tableW := env.tableW

	if !definiteTable {
		tableW = availW
		if sumMax < availW {
			// width:auto — shrink-wrap to max-content (not min-content).
			tableW = sumMax
		}
	}

	distributeColumnWidths(colW, colMin, colPct, colAbs, tableW, chrome, definiteTable, sumMax, sumMin, nCols)

	// Auto tables: border box covers used columns. Definite width keeps tableW.
	if !definiteTable {
		if sumCols := chrome + sumColWidths(colW); sumCols > tableW {
			tableW = sumCols
		}
	}

	return colW, tableW
}

// distributeColumnWidths applies the hint/extra/shrink strategy to the used
// column widths (extracted from sizeTableColumns for clarity).
func distributeColumnWidths(
	colW, colMin, colPct, colAbs []float64, tableW, chrome float64,
	definiteTable bool, sumMax, sumMin float64, nCols int,
) {
	switch {
	case definiteTable && hasColumnHints(colPct, colAbs):
		distributeColumnHints(colW, colMin, colPct, colAbs, tableW, chrome, nCols)
	case tableW > sumMax:
		distributeColumnExtra(colW, tableW, sumMax, nCols)
	case tableW < sumMax:
		distributeColumnShrink(colW, colMin, sumMax, sumMin, tableW, chrome, definiteTable)
	}
}

// columnWidthSums returns the max-content and min-content sums plus chrome.
func columnWidthSums(colW, colMin []float64, chrome float64) (float64, float64) {
	sumMax, sumMin := 0.0, 0.0

	for i := range colW {
		sumMax += colW[i]
		sumMin += colMin[i]
	}

	return sumMax + chrome, sumMin + chrome
}

// sumColWidths returns the sum of the used column widths.
func sumColWidths(colW []float64) float64 {
	sum := 0.0
	for i := range colW {
		sum += colW[i]
	}

	return sum
}

// hasColumnHints reports whether any column has a % or absolute width hint.
func hasColumnHints(colPct, colAbs []float64) bool {
	for i := range colPct {
		if colPct[i] >= 0 || colAbs[i] >= 0 {
			return true
		}
	}

	return false
}

// distributeColumnHints resolves a definite table width with % / absolute
// column hints: hinted columns take their share first, then the leftover is
// spread over auto columns (or by % share when every column is hinted).
func distributeColumnHints(colW, colMin, colPct, colAbs []float64, tableW, chrome float64, nCols int) {
	inner := tableW - chrome
	if inner < 0 {
		inner = 0
	}

	used, autoMax := applyHintedColumns(colW, colMin, colPct, colAbs, inner)

	remain := inner - used
	if remain < 0 {
		remain = 0
	}

	switch {
	case autoMax > 0 && remain > 0:
		spreadRemainderOverAuto(colW, colMin, colPct, colAbs, remain, autoMax)
	case autoMax == 0 && remain > 0:
		// All columns hinted — distribute leftover by % share, else evenly.
		spreadRemainderOverHinted(colW, colPct, remain, nCols)
	}
}

// applyHintedColumns sizes the hinted columns and returns the used width plus
// the total max-content of the auto columns (extracted from
// distributeColumnHints for readability).
func applyHintedColumns(colW, colMin, colPct, colAbs []float64, inner float64) (float64, float64) {
	used, autoMax := 0.0, 0.0

	for idx := range colW {
		switch {
		case colPct[idx] >= 0:
			colW[idx] = maxF(inner*colPct[idx]/cssPercent, colMin[idx])
			used += colW[idx]
		case colAbs[idx] >= 0:
			colW[idx] = maxF(colAbs[idx], colMin[idx])
			used += colW[idx]
		default:
			autoMax += colW[idx]
		}
	}

	return used, autoMax
}

// spreadRemainderOverAuto distributes leftover width over auto columns
// proportionally to their current share (min floor per column).
func spreadRemainderOverAuto(colW, colMin, colPct, colAbs []float64, remain, autoMax float64) {
	for i := range colW {
		if colPct[i] < 0 && colAbs[i] < 0 {
			colW[i] = maxF(remain*(colW[i]/autoMax), colMin[i])
		}
	}
}

// spreadRemainderOverHinted distributes leftover width by % share, else evenly.
func spreadRemainderOverHinted(colW, colPct []float64, remain float64, nCols int) {
	pctTotal := 0.0

	for i := range colPct {
		if colPct[i] > 0 {
			pctTotal += colPct[i]
		}
	}

	for i := range colW {
		if pctTotal > 0 && colPct[i] > 0 {
			colW[i] += remain * (colPct[i] / pctTotal)
		} else {
			colW[i] += remain / float64(nCols)
		}
	}
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

// distributeColumnExtra spreads a surplus evenly across all columns.
func distributeColumnExtra(colW []float64, tableW, sumMax float64, nCols int) {
	extra := (tableW - sumMax) / float64(nCols)
	for i := range colW {
		colW[i] += extra
	}
}

// distributeColumnShrink squeezes columns when the used max-content width
// overflows the table: grow from min toward max, scale mins into a definite
// box, or honor mins for auto tables (which may overflow).
func distributeColumnShrink(colW, colMin []float64, sumMax, sumMin, tableW, chrome float64, definiteTable bool) {
	innerAvail := tableW - chrome
	if innerAvail < 0 {
		innerAvail = 0
	}

	innerMax := sumMax - chrome
	innerMin := sumMin - chrome

	switch {
	case innerAvail >= innerMin && innerMax > innerMin:
		growFromMinTowardMax(colW, colMin, innerAvail, innerMin, innerMax)
	case innerMin > 0 && innerAvail < innerMin:
		// Narrower than min-content.
		if definiteTable {
			scaleMinsIntoBox(colW, colMin, innerAvail, innerMin)
		} else {
			// width:auto — honor mins (table may overflow) rather than
			// crushing text into emergency mid-word wraps.
			copy(colW, colMin)
		}
	case innerMax > 0:
		scaleToInnerWidth(colW, colMin, innerAvail, innerMax, definiteTable)
	}
}

// growFromMinTowardMax grows each column from min toward max proportionally
// to its free space.
func growFromMinTowardMax(colW, colMin []float64, innerAvail, innerMin, innerMax float64) {
	free := innerAvail - innerMin
	span := innerMax - innerMin

	for idx := range colW {
		grow := colW[idx] - colMin[idx]
		if grow < 0 {
			grow = 0
		}

		colW[idx] = colMin[idx] + free*(grow/span)
	}
}

// scaleMinsIntoBox scales the column mins into a definite, narrower box so
// max-width:100% images in a 22em float still shrink.
func scaleMinsIntoBox(colW, colMin []float64, innerAvail, innerMin float64) {
	scale := innerAvail / innerMin
	if scale < 0 {
		scale = 0
	}

	for i := range colW {
		colW[i] = colMin[i] * scale
	}
}

// scaleToInnerWidth scales columns down to the available inner width.
func scaleToInnerWidth(colW, colMin []float64, innerAvail, innerMax float64, definiteTable bool) {
	scale := innerAvail / innerMax
	if scale < 0 {
		scale = 0
	}

	for i := range colW {
		colW[i] *= scale
		if !definiteTable && colW[i] < colMin[i] {
			colW[i] = colMin[i]
		}
	}
}

// DeactivateOp marks an op so every painter (Paint, PaintBand) and every
// pagination helper ignores it while keeping its slot in Ops: the box tree
// stores op indices (opStart/opEnd) that Paint relies on, so entries must
// not be removed.
const opKindNoop OpKind = 255

func DeactivateOp(paintOp *Op) {
	if paintOp == nil {
		return
	}

	paintOp.Kind = opKindNoop
	paintOp.URI = ""
}
