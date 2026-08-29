package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	calcParts     = 3
	mediumKeyword = "medium"
	thinKeyword   = "thin"
	thickKeyword  = "thick"
)

func parseColumnsShorthand(sty *ResolvedStyle, value string, fsize, viewportW float64) {
	sty.ColumnCount = 0
	sty.ColumnWidth = -1

	for _, tok := range strings.Fields(value) {
		tok = strings.TrimSpace(tok)
		if tok == "" || tok == overflowAuto {
			continue
		}

		if n, err := strconv.Atoi(tok); err == nil && n >= 1 {
			sty.ColumnCount = n

			continue
		}

		if v, ok := lengthBox(tok, fsize, viewportW, overflowAuto); ok && v >= 0 {
			sty.ColumnWidth = v
		}
	}
}

// isMulticol reports whether st establishes a multi-column container
// (column-count or column-width is not auto).
func isMulticol(st ResolvedStyle) bool {
	return st.ColumnCount > 0 || st.ColumnWidth >= 0
}

// parseFontShorthand handles "font: italic bold 12px/1.4 Arial, sans-serif".
func parseFontShorthand(style *ResolvedStyle, value string, remBase float64) {
	parts := strings.Fields(value)
	for idx, page := range parts {
		if applyFontStyleKeyword(style, page) {
			continue
		}

		if n, ok := css.ParseNumber(page); ok && n >= 100 && n <= 900 {
			style.FontWeight = int(n)

			continue
		}
		// first size token
		sizePart, linePart, hasLineH := strings.Cut(page, "/")
		style.FontSize = fontSize(sizePart, style.FontSize, remBase)

		if hasLineH {
			lineH := lineHeight(linePart, style.FontSize)
			if lineH >= 0 {
				setFontLineHeight(style, page, lineH)
			}
		}

		if idx+1 < len(parts) {
			if fam := css.ParseFontFamily(strings.Join(parts[idx+1:], " ")); len(fam) > 0 {
				style.FontFamily = fam
			}
		}

		return
	}
}

// applyFontStyleKeyword handles the italic/oblique/bold style keywords; false
// when the token is not a font style keyword.
func applyFontStyleKeyword(style *ResolvedStyle, page string) bool {
	switch page {
	case "italic", "oblique":
		style.FontItalic = true
	case "bold":
		style.FontWeight = fontWeightBold
	default:
		return false
	}

	return true
}

// parseFlexShorthand handles flex: none | auto | <grow> | <grow> <shrink> | <grow> <shrink> <basis>.
func parseFlexShorthand(style *ResolvedStyle, value string, fontSize, pctBase float64) {
	value = strings.TrimSpace(value)
	switch value {
	case cssDisplayNone:
		style.FlexGrow, style.FlexShrink = 0, 0
		style.FlexBasis, style.FlexBasisPercent = -1, -1

		return
	case overflowAuto:
		style.FlexGrow, style.FlexShrink = 1, 1
		style.FlexBasis, style.FlexBasisPercent = -1, -1

		return
	}

	parts := strings.Fields(value)
	switch len(parts) {
	case 0:
		return
	case 1:
		parseFlexOne(style, parts[0], fontSize, pctBase)
	case two:
		parseFlexTwo(style, parts, fontSize, pctBase)
	default:
		parseFlexThree(style, parts, fontSize, pctBase)
	}
}

// flexIsBasis reports whether a token can be a flex-basis value.
func flexIsBasis(tok string) bool {
	if tok == overflowAuto || tok == "content" {
		return true
	}

	_, _, ok := css.ParseLength(tok)

	return ok
}

// flexSetBasis writes the basis longhands from a token.
func flexSetBasis(style *ResolvedStyle, tok string, fontSize, pctBase float64) {
	if tok == overflowAuto || tok == "content" {
		style.FlexBasis = -1
		style.FlexBasisPercent = -1

		return
	}

	if v, unit, ok := css.ParseLength(tok); ok && unit == "%" {
		style.FlexBasisPercent = v
		style.FlexBasis = -1

		return
	}

	if v, ok := lengthBox(tok, fontSize, pctBase, overflowAuto); ok {
		style.FlexBasis = v
		style.FlexBasisPercent = -1
	}
}

func parseFlexOne(style *ResolvedStyle, part string, fontSize, pctBase float64) {
	if g, err := strconv.ParseFloat(part, 64); err == nil {
		// flex: <number> → grow <number>, shrink 1, basis 0%
		style.FlexGrow = g
		style.FlexShrink = 1
		style.FlexBasis = -1
		style.FlexBasisPercent = 0

		return
	}

	if flexIsBasis(part) {
		style.FlexGrow, style.FlexShrink = 1, 1

		flexSetBasis(style, part, fontSize, pctBase)
	}
}

func parseFlexTwo(style *ResolvedStyle, parts []string, fontSize, pctBase float64) {
	g, errG := strconv.ParseFloat(parts[0], 64)
	if errG != nil {
		return
	}

	style.FlexGrow = g
	if sh, err := strconv.ParseFloat(parts[1], 64); err == nil {
		style.FlexShrink = sh
		style.FlexBasis = -1
		style.FlexBasisPercent = 0

		return
	}

	style.FlexShrink = 1

	if flexIsBasis(parts[1]) {
		flexSetBasis(style, parts[1], fontSize, pctBase)
	}
}

func parseFlexThree(style *ResolvedStyle, parts []string, fontSize, pctBase float64) {
	gap, errG := strconv.ParseFloat(parts[0], 64)
	shval, errS := strconv.ParseFloat(parts[1], 64)

	if errG != nil || errS != nil {
		return
	}

	style.FlexGrow, style.FlexShrink = gap, shval

	flexSetBasis(style, parts[2], fontSize, pctBase)
}

// parseFlexFlow expands flex-flow to flex-direction + flex-wrap. Omitted
// longhands reset to CSS initials (row, nowrap). Invalid tokens drop the
// declaration.
func parseFlexFlow(style *ResolvedStyle, value string) {
	dir, wrap := fxRow, cssWhiteSpaceNowrap
	found := false

	for _, tok := range strings.Fields(value) {
		switch tok {
		case fxRow, fxCol, fxRowRev, fxColRev:
			dir = tok
			found = true
		case cssWhiteSpaceNowrap, fxWrap, fxWrapRev:
			wrap = tok
			found = true
		default:
			return
		}
	}

	if !found {
		return
	}

	style.FlexDirection = dir
	style.FlexWrap = wrap
}

// splitPlacePair splits a place-* shorthand: one token is copied to both
// longhands; two tokens are align then justify.
func splitPlacePair(value string) (string, string, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	switch len(parts) {
	case 1:
		return parts[0], parts[0], true
	case two:
		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}

func parsePlaceContent(style *ResolvedStyle, value string) {
	align, justify, ok := splitPlacePair(value)
	if !ok {
		return
	}

	if isPlaceDistributionKeyword(align) && (isPlaceAlignmentKeyword(justify) || justify == fxStretch) {
		setJustifyContentValue(style, align)
		setAlignContentValue(style, justify)
		setAlignItemsValue(style, justify)

		return
	}

	setAlignContentValue(style, align)
	setJustifyContentValue(style, justify)
}

func isPlaceDistributionKeyword(val string) bool {
	return val == fxBetween || val == fxAround || val == fxEvenly
}

func isPlaceAlignmentKeyword(val string) bool {
	return val == fxCenter || val == fxStart || val == fxEnd || val == fxFlexStart || val == fxFlexEnd
}

func parsePlaceItems(style *ResolvedStyle, value string) {
	align, justify, ok := splitPlacePair(value)
	if !ok {
		return
	}

	setAlignItemsValue(style, align)
	setJustifyItemsValue(style, justify)
}

func parsePlaceSelf(style *ResolvedStyle, value string) {
	align, justify, ok := splitPlacePair(value)
	if !ok {
		return
	}

	setAlignSelfValue(style, align)
	setJustifySelfValue(style, justify)
}

// setFourMargin applies a margin shorthand and tracks auto margins on all axes.
func setFourMargin(sty *ResolvedStyle, value string, fsize, ctxW float64) {
	var val [4]string
	count := splitSpaceTokens(value, val[:])
	sty.MarginTopAuto, sty.MarginBottomAuto = false, false
	sty.MarginLeftAuto, sty.MarginRightAuto = false, false

	switch count {
	case 0:
		return
	case 1:
		sty.MarginTop, sty.MarginTopAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginBottom, sty.MarginBottomAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[0], fsize, ctxW)
	case two:
		sty.MarginTop, sty.MarginTopAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom, sty.MarginBottomAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[1], fsize, ctxW)
	case three:
		sty.MarginTop, sty.MarginTopAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom, sty.MarginBottomAuto = marginLenAuto(val[2], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[1], fsize, ctxW)
	default:
		sty.MarginTop, sty.MarginTopAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom, sty.MarginBottomAuto = marginLenAuto(val[2], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[3], fsize, ctxW)
	}
}

func setFour(_ *ResolvedStyle, value string, top, right, bottom, left *float64, fsVal, ctxW float64) {
	var val [4]string

	count := splitSpaceTokens(value, val[:])
	if count == 0 {
		return
	}

	if count == 1 {
		*top = marginLen(val[0], fsVal, ctxW)
		*right, *bottom, *left = *top, *top, *top

		return
	}

	if count == two {
		*top = marginLen(val[0], fsVal, ctxW)
		*right = marginLen(val[1], fsVal, ctxW)
		*bottom, *left = *top, *right

		return
	}

	if count == three {
		*top = marginLen(val[0], fsVal, ctxW)
		*right = marginLen(val[1], fsVal, ctxW)
		*bottom = marginLen(val[2], fsVal, ctxW)
		*left = *right

		return
	}

	*top = marginLen(val[0], fsVal, ctxW)
	*right = marginLen(val[1], fsVal, ctxW)
	*bottom = marginLen(val[2], fsVal, ctxW)
	*left = marginLen(val[3], fsVal, ctxW)
}

func parseBorder(value string, fsize float64, current [3]float64) (border, bool) { //nolint:cyclop
	var boxNode border

	for start := 0; ; {
		face, next, ok := nextSpaceToken(value, start)
		if !ok {
			break
		}

		switch face {
		case solidKeyword, borderStyleDashed, borderStyleDotted:
			boxNode.Style = face
		case cssDisplayNone, overflowHidden:
			boxNode.Style = cssDisplayNone
		default:
			if isCurrentColor(face) {
				boxNode.Color = current
			} else if r, g, bb, _, ok := css.ParseColor(face); ok {
				boxNode.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(bb) / 255}
			} else if v, unit, ok := css.ParseLength(face); ok {
				boxNode.Width = v
				if pt, converted := lengthToPt(v, unit, fsize); converted {
					boxNode.PaintWidth = pt
				}
			}
		}

		start = next
	}

	if boxNode.Style == "" {
		boxNode.Style = solidKeyword
	}

	if boxNode.Width == 0 {
		boxNode.Width = 1
		boxNode.PaintWidth = 1
	}

	return boxNode, boxNode.Style != cssDisplayNone
}

// splitSpaceTokens writes up to len(tokens) CSS whitespace-separated tokens
// into tokens and returns the actual token count. Counts above the capacity
// preserve strings.Fields' len>4 behavior without allocating a larger slice.
func splitSpaceTokens(value string, tokens []string) int {
	count := 0

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			return count
		}

		if count < len(tokens) {
			tokens[count] = token
		}

		count++
		start = next
	}
}

func nextSpaceToken(value string, start int) (string, int, bool) {
	for start < len(value) && isCSSSpace(value[start]) {
		start++
	}

	if start == len(value) {
		return "", start, false
	}

	end := start
	depth := 0

tokenLoop:
	for end < len(value) {
		char := value[end]
		switch {
		case char == '(':
			depth++
		case char == ')' && depth > 0:
			depth--
		case depth == 0 && isCSSSpace(char):
			break tokenLoop
		}

		end++
	}

	return value[start:end], end, true
}

func isCSSSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\v' || value == '\f' || value == '\r'
}

func borderWidth(value string, _ float64) float64 {
	switch value {
	case thinKeyword:
		return pxToPt(1)
	case mediumKeyword:
		return pxToPt(three)
	case thickKeyword:
		return pxToPt(borderWidthMediumPx)
	}

	if v, _, ok := css.ParseLength(value); ok {
		return v
	}

	return 0
}

func borderPaintWidth(value string, fsize float64) float64 {
	switch value {
	case thinKeyword, mediumKeyword, thickKeyword:
		return borderWidth(value, fsize)
	}

	if v, unit, ok := css.ParseLength(value); ok {
		if pt, converted := lengthToPt(v, unit, fsize); converted {
			return pt
		}
	}

	return 0
}

func setFontLineHeight(style *ResolvedStyle, page string, lineH float64) {
	style.LineHeightUnitless = 0

	if _, after, ok := strings.Cut(page, "/"); ok {
		if ratio, numeric := css.ParseNumber(after); numeric {
			style.LineHeightUnitless = ratio
		}
	}

	style.LineHeight = lineH
}

func fontSize(value string, parent, remBase float64) float64 {
	if remBase <= 0 {
		remBase = pxToPt(cssPxRoot)
	}

	if pt, ok := fontSizeKeyword(value, parent); ok {
		return pt
	}

	if val, unit, ok := css.ParseLength(value); ok {
		switch unit {
		case "%":
			return parent * val / cssPercent
		case remUnit:
			return remBase * val
		default:
			if pt, ok := lengthToPt(val, unit, parent); ok {
				return pt
			}
		}
	}

	return parent
}

// fontSizeKeyword resolves the named font-size keywords relative to parent.
func fontSizeKeyword(value string, parent float64) (float64, bool) {
	switch value {
	case "xx-small":
		return pxToPt(fontSizeXSmallPx), true
	case "x-small":
		return pxToPt(fontSizeSmallPx), true
	case "small":
		return pxToPt(fontSizeMediumPx), true
	case mediumKeyword:
		return pxToPt(cssPxRoot), true
	case "large":
		return pxToPt(fontSizeLargePx), true
	case "x-large":
		return pxToPt(twoLineRoomPt), true
	case "xx-large":
		return pxToPt(fontSizeXXXLargePx), true
	case "smaller":
		return parent * smallerFontRatio, true
	case "larger":
		return parent * defaultLineHeightRatio, true
	}

	return 0, false
}

func lineHeight(value string, fsize float64) float64 {
	if value == contentNormal {
		return 0
	}

	if v, ok := css.ParseNumber(value); ok {
		return v * fsize
	}

	if v, unit, ok := css.ParseLength(value); ok {
		if unit == "%" {
			return fsize * v / cssPercent
		}

		if pt, ok := lengthToPt(v, unit, fsize); ok {
			return pt
		}
	}

	return 0
}

// parseOverflowKeyword accepts CSS overflow keywords used for sticky scrollport
// detection. clip is treated like hidden (scroll container, no user scroll).
func parseOverflowKeyword(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case visibleKeyword, overflowHidden, overflowScroll, overflowAuto, "clip":
		return strings.ToLower(strings.TrimSpace(value)), true
	}

	return "", false
}

// parseListStyleType returns a known list-style-type keyword, or "" if unknown.
func parseListStyleType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case listStyleDisc, "circle", listStyleSquare, listStyleDecimal, "decimal-leading-zero",
		"lower-roman", "upper-roman", "lower-alpha", "lower-latin",
		"upper-alpha", "upper-latin", cssDisplayNone:
		return strings.ToLower(strings.TrimSpace(value))
	}

	return ""
}

func parseListStylePosition(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case listPosInside, listPosOutside:
		return strings.ToLower(strings.TrimSpace(value))
	}

	return ""
}

func parseOutlineStyle(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case solidKeyword, borderStyleDashed, borderStyleDotted, cssDisplayNone:
		return strings.ToLower(strings.TrimSpace(value)), true
	}

	return "", false
}

func parseOutlineWidth(value string, fsize float64) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case thinKeyword, mediumKeyword, thickKeyword:
		return borderWidth(value, fsize), true
	}

	if _, _, ok := css.ParseLength(value); ok {
		return borderWidth(value, fsize), true
	}

	return 0, false
}

// parseQuotesPair reads the first two CSS strings from a quotes value.
// quotes: none yields empty open/close with ok true.
func parseQuotesPair(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}

	if strings.EqualFold(value, cssDisplayNone) {
		return "", "", true
	}

	openQuote, next, foundOpen := nextQuotedCSSString(value, 0)
	if !foundOpen {
		return "", "", false
	}

	closeQuote, _, foundClose := nextQuotedCSSString(value, next)
	if !foundClose {
		return "", "", false
	}

	return openQuote, closeQuote, true
}

func nextQuotedCSSString(value string, start int) (string, int, bool) {
	start = skipCSSWhitespace(value, start)
	if start >= len(value) {
		return "", start, false
	}

	quote := value[start]
	if quote != '"' && quote != '\'' {
		return "", start, false
	}

	end, ok := scanQuotedContent(value, start+1, quote)
	if !ok {
		return "", start, false
	}

	return decodeCSSString(value[start+1 : end]), end + 1, true
}

// overflowCreatesStickyScrollport reports whether overflow establishes a sticky
// scrollport (CSS Position 3 / Overflow 3). PDF has no user scroll, so sticky
// inside these boxes clamps at scroll offset 0 against the box edges.
func overflowCreatesStickyScrollport(overflow string) bool {
	switch overflow {
	case overflowAuto, "scroll", overflowHidden, "clip":
		return true
	}

	return false
}

// vminVmaxPt parses Nvmin / Nvmax as a percent of min/max(viewportW, viewportH).
func vminVmaxPt(value string, viewportW, viewportH float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	lower := strings.ToLower(value)

	var useMax bool

	switch {
	case strings.HasSuffix(lower, "vmin"):
		value = strings.TrimSpace(value[:len(value)-len("vmin")])
	case strings.HasSuffix(lower, "vmax"):
		useMax = true
		value = strings.TrimSpace(value[:len(value)-len("vmax")])
	default:
		return 0, false
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}

	base := viewportW

	if useMax {
		if viewportH > viewportW {
			base = viewportH
		}
	} else if viewportH < viewportW {
		base = viewportH
	}

	return base * parsed / cssPercent, true
}

// lengthBox parses a length for box-sizing properties. "auto" maps to -1,
// "none" to -1 as well (max-*). Percentages resolve against the containing
// block dimension (approximated by viewport at this phase).
func lengthBox(value string, fsize, containing float64, autoValue string) (float64, bool) {
	if value == autoValue || value == inheritKeyword || value == cssKeywordInitial {
		return -1, true
	}

	if pt, parsed := lengthBoxResolved(value, fsize, containing); parsed {
		return pt, true
	}

	return lengthBoxFromUnit(value, fsize, containing)
}

func lengthBoxResolved(value string, fsize, containing float64) (float64, bool) {
	if pt, parsed := vminVmaxPt(value, containing, containing); parsed {
		return pt, true
	}

	if point, parsed := calcLength(value, fsize, containing); parsed {
		return point, true
	}

	return clampLength(value, fsize, containing)
}

func lengthBoxFromUnit(value string, fsize, containing float64) (float64, bool) {
	val, unit, parsed := css.ParseLength(value)
	if !parsed {
		return 0, false
	}

	switch unit {
	case "%", "vw", "vh":
		return containing * val / cssPercent, true
	default:
		point, converted := lengthToPt(val, unit, fsize)
		if !converted {
			return 0, false
		}

		// rem uses LengthToPt's 16px root; keep remBase-independent path
		// matching prior lengthBox (rem → 12pt * v via pxToPt(16)).
		if unit == remUnit {
			return pxToPt(cssPxRoot) * val, true
		}

		return point, true
	}
}

// marginLenAuto parses a horizontal margin; auto yields (0, true).
func marginLenAuto(value string, fs, ctxW float64) (float64, bool) {
	if value == overflowAuto {
		return 0, true
	}

	return marginLen(value, fs, ctxW), false
}

// marginLen parses a margin/padding/letter-spacing length in points; 0 when
// unparseable. Percentages resolve against the containing width.
func marginLen(value string, fsize, ctxW float64) float64 {
	if value == overflowAuto || value == inheritKeyword || value == cssKeywordInitial {
		return 0
	}

	if pt, parsed := marginLenResolved(value, fsize, ctxW); parsed {
		return pt
	}

	return marginLenFromUnit(value, fsize, ctxW)
}

func marginLenResolved(value string, fsize, ctxW float64) (float64, bool) {
	if pt, parsed := vminVmaxPt(value, ctxW, ctxW); parsed {
		return pt, true
	}

	if point, parsed := calcLength(value, fsize, ctxW); parsed {
		return point, true
	}

	return clampLength(value, fsize, ctxW)
}

func marginLenFromUnit(value string, fsize, ctxW float64) float64 {
	val, unit, parsed := css.ParseLength(value)
	if !parsed {
		return 0
	}

	if unit == "%" {
		return ctxW * val / cssPercent
	}

	if unit == remUnit {
		return pxToPt(cssPxRoot) * val
	}

	if pt, converted := lengthToPt(val, unit, fsize); converted {
		return pt
	}

	return 0
}

const clampPrefix = "clamp("

// clampLength evaluates clamp(min, pref, max) on lengths. Nested calc() in an
// argument is accepted when calcLength can compute it. Unparseable clamps stay
// invalid so a later fallback can win.
func clampLength(value string, fsize, containing float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if len(value) < len("clamp()") ||
		!strings.EqualFold(value[:len(clampPrefix)], clampPrefix) ||
		value[len(value)-1] != ')' {
		return 0, false
	}

	args := splitCommaArgs(value[len(clampPrefix) : len(value)-1])
	if len(args) != three {
		return 0, false
	}

	minLen, minOK := resolvedLength(strings.TrimSpace(args[0]), fsize, containing)
	prefLen, prefOK := resolvedLength(strings.TrimSpace(args[1]), fsize, containing)
	maxLen, maxOK := resolvedLength(strings.TrimSpace(args[2]), fsize, containing)

	if !minOK || !prefOK || !maxOK {
		return 0, false
	}

	used := prefLen
	if used > maxLen {
		used = maxLen
	}

	if used < minLen {
		used = minLen
	}

	return used, true
}

func splitCommaArgs(value string) []string {
	parts := make([]string, 0, three)
	depth := 0
	start := 0

	for idx := range len(value) {
		switch value[idx] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, value[start:idx])
				start = idx + 1
			}
		}
	}

	return append(parts, value[start:])
}

func resolvedLength(value string, fsize, containing float64) (float64, bool) {
	if point, ok := calcLength(value, fsize, containing); ok {
		return point, true
	}

	val, unit, ok := css.ParseLength(value)
	if !ok {
		return 0, false
	}

	switch unit {
	case "%", "vw", "vh":
		return containing * val / cssPercent, true
	case remUnit:
		return pxToPt(cssPxRoot) * val, true
	default:
		return lengthToPt(val, unit, fsize)
	}
}

// calcLength evaluates the small arithmetic subset needed by the print CSS:
// one length plus/minus another length, or one length multiplied by a number.
// Unsupported calc expressions remain invalid and keep the existing fallback.
//
//nolint:cyclop // compact calc operator grammar
func calcLength(value string, fsize, containing float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if len(value) < len("calc()") || !strings.EqualFold(value[:5], "calc(") || value[len(value)-1] != ')' {
		return 0, false
	}

	parts := strings.Fields(value[5 : len(value)-1])
	if len(parts) != calcParts {
		return 0, false
	}

	left, ok := plainLength(parts[0], fsize, containing)
	if !ok {
		return 0, false
	}

	switch parts[1] {
	case "*":
		factor, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, false
		}

		return left * factor, true
	case "+", "-":
		right, rightOK := plainLength(parts[2], fsize, containing)
		if !rightOK {
			return 0, false
		}

		if parts[1] == "-" {
			return left - right, true
		}

		return left + right, true
	default:
		return 0, false
	}
}

func plainLength(value string, fsize, containing float64) (float64, bool) {
	val, unit, ok := css.ParseLength(value)
	if !ok {
		return 0, false
	}

	if unit == "%" {
		return containing * val / cssPercent, true
	}

	if unit == remUnit {
		return pxToPt(cssPxRoot) * val, true
	}

	return lengthToPt(val, unit, fsize)
}

func pxToPt(px float64) float64 { return px * pxToPtFactor }

// parseGridAutoFlowValue normalizes grid-auto-flow to one of:
// "row", "column", "dense", "row dense", "column dense".
func parseGridAutoFlowValue(value string) string {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	row, col, dense := false, false, false

	for _, t := range toks {
		switch t {
		case fxRow:
			row = true
		case fxCol:
			col = true
		case gridFlowDense:
			dense = true
		}
	}

	return gridAutoFlowName(row, col, dense)
}

// gridAutoFlowName maps the parsed tokens onto the canonical keyword.
func gridAutoFlowName(row, col, dense bool) string {
	switch {
	case col && dense:
		return gridFlowColumnDense
	case row && dense:
		return gridFlowRowDense
	case dense:
		return gridFlowDense
	case col:
		return fxCol
	default:
		return fxRow
	}
}

func clearGridTemplate(style *ResolvedStyle) {
	style.GridTemplateRows = ""
	style.GridTemplateColumns = ""
	style.GridTemplateAreas = ""
}

func gridValueHasMasonry(value string) bool {
	for _, tok := range strings.Fields(strings.ToLower(value)) {
		if strings.Trim(tok, `"'/`) == "masonry" {
			return true
		}
	}

	return false
}

func gridPartHasAutoFlow(part string) bool {
	for _, tok := range strings.Fields(strings.ToLower(part)) {
		if tok == "auto-flow" {
			return true
		}
	}

	return false
}

func gridPartHasAreaString(part string) bool {
	return strings.Contains(part, `"`) || strings.Contains(part, "'")
}

// splitGridTemplateSlash splits rows/areas from columns on the first `/`
// outside quotes. CSS grid-template has at most one such slash.
func splitGridTemplateSlash(value string) (string, string, bool) {
	inQuote := byte(0)

	for idx := range len(value) {
		char := value[idx]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}

			continue
		}

		if char == '"' || char == '\'' {
			inQuote = char

			continue
		}

		if char == '/' {
			return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx+1:]), true
		}
	}

	return strings.TrimSpace(value), "", false
}

// parseGridTemplateShorthand is a Partial expansion into the existing
// template-rows/columns/areas fields. Masonry values are skipped.
func parseGridTemplateShorthand(style *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if value == "" || gridValueHasMasonry(value) {
		return
	}

	if strings.EqualFold(value, cssDisplayNone) {
		clearGridTemplate(style)

		return
	}

	before, after, hasSlash := splitGridTemplateSlash(value)
	applyGridTemplateParts(style, before, after, hasSlash)
}

func applyGridTemplateParts(style *ResolvedStyle, before, after string, hasSlash bool) {
	if gridPartHasAreaString(before) {
		areas, rows := splitGridAreasAndRowSizes(before)
		style.GridTemplateAreas = areas
		style.GridTemplateRows = rows
		style.GridTemplateColumns = ""

		if hasSlash {
			style.GridTemplateColumns = after
		}

		return
	}

	style.GridTemplateAreas = ""
	style.GridTemplateRows = before
	style.GridTemplateColumns = ""

	if hasSlash {
		style.GridTemplateColumns = after
	}
}

// parseGridShorthand is a Partial parse of `grid`. Template forms reuse
// parseGridTemplateShorthand and reset auto-flow to row. auto-flow forms set
// GridAutoFlow plus the explicit template axis; masonry is skipped.
func parseGridShorthand(style *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if value == "" || gridValueHasMasonry(value) {
		return
	}

	if strings.EqualFold(value, cssDisplayNone) {
		clearGridTemplate(style)
		style.GridAutoFlow = fxRow

		return
	}

	before, after, hasSlash := splitGridTemplateSlash(value)

	if gridPartHasAutoFlow(before) || gridPartHasAutoFlow(after) {
		applyGridAutoFlowShorthand(style, before, after, hasSlash)

		return
	}

	parseGridTemplateShorthand(style, value)
	style.GridAutoFlow = fxRow
}

func applyGridAutoFlowShorthand(style *ResolvedStyle, before, after string, hasSlash bool) {
	clearGridTemplate(style)

	if gridPartHasAutoFlow(before) {
		style.GridAutoFlow = parseGridAutoFlowValue(before)
		if hasSlash {
			style.GridTemplateColumns = after
		}

		return
	}

	style.GridTemplateRows = before
	style.GridAutoFlow = gridColumnAutoFlow(after)
}

func gridColumnAutoFlow(after string) string {
	for _, tok := range strings.Fields(strings.ToLower(after)) {
		if tok == gridFlowDense {
			return gridFlowColumnDense
		}
	}

	return fxCol
}

// splitGridAreasAndRowSizes pulls quoted area rows and leftover track sizes
// out of the grid-template areas form. Line names in [] are dropped.
func splitGridAreasAndRowSizes(part string) (string, string) {
	var areas, rows []string

	for idx := 0; idx < len(part); {
		for idx < len(part) && isCSSSpace(part[idx]) {
			idx++
		}

		if idx >= len(part) {
			break
		}

		idx = takeGridTemplateToken(part, idx, &areas, &rows)
	}

	return strings.Join(areas, " "), strings.Join(rows, " ")
}

func takeGridTemplateToken(part string, pos int, areas, rows *[]string) int {
	switch part[pos] {
	case '"', '\'':
		end := scanQuotedGridToken(part, pos)

		*areas = append(*areas, part[pos:end])

		return end
	case '[':
		return skipGridLineNames(part, pos)
	default:
		end := pos
		for end < len(part) && !isCSSSpace(part[end]) &&
			part[end] != '"' && part[end] != '\'' && part[end] != '[' {
			end++
		}

		*rows = append(*rows, part[pos:end])

		return end
	}
}

func scanQuotedGridToken(part string, pos int) int {
	quote := part[pos]
	end := pos + 1

	for end < len(part) && part[end] != quote {
		end++
	}

	if end < len(part) {
		end++
	}

	return end
}

func skipGridLineNames(part string, i int) int {
	end := i + 1
	for end < len(part) && part[end] != ']' {
		end++
	}

	if end < len(part) {
		end++
	}

	return end
}

// parseGridArea handles grid-area: <custom-ident> or the lite line form
// row-start / column-start / row-end / column-end (and 1–2 slash forms).
func parseGridArea(sty *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, overflowAuto) {
		sty.GridArea = ""

		return
	}

	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) == 1 {
		tok := parts[0]
		if _, err := strconv.Atoi(tok); err == nil {
			// Single line index → row-start (CSS shorthand lite).
			parseGridRow(sty, tok)
			sty.GridArea = ""

			return
		}

		if strings.HasPrefix(tok, "span ") {
			parseGridRow(sty, tok)
			sty.GridArea = ""

			return
		}
		// Named area.
		sty.GridArea = tok

		return
	}

	sty.GridArea = ""

	switch len(parts) {
	case two:
		// CSS: row-start / column-start (omitted ends copy starts → span 1).
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
	case three:
		// row-start / column-start / row-end
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
		applyGridLineEnd(sty, true, parts[2])
	default:
		// row-start / column-start / row-end / column-end
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
		applyGridLineEnd(sty, true, parts[2])
		applyGridLineEnd(sty, false, parts[3])
	}
}

// applyGridLineEnd sets span from an end line or "span N" on row (isRow) or column.
func applyGridLineEnd(style *ResolvedStyle, isRow bool, end string) {
	target := gridTarget(style, isRow)

	end = strings.TrimSpace(end)
	if strings.HasPrefix(end, "span ") {
		node, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(end, "span ")))
		if err != nil || node < 1 {
			return
		}

		*target.span = node

		return
	}

	val, err := strconv.Atoi(end)
	if err != nil {
		return
	}

	if *target.start > 0 {
		sp := val - *target.start
		if sp < 1 {
			sp = 1
		}

		*target.span = sp
	}
}

func parseGridColumn(st *ResolvedStyle, value string) { parseGridLineAt(colGridTarget(st), value) }

func parseGridRow(st *ResolvedStyle, value string) { parseGridLineAt(rowGridTarget(st), value) }

// gridLineTarget points at the start/span fields of one grid axis.
type gridLineTarget struct {
	start *int
	span  *int
}

func rowGridTarget(st *ResolvedStyle) gridLineTarget {
	return gridLineTarget{start: &st.GridRowStart, span: &st.GridRowSpan}
}

func colGridTarget(st *ResolvedStyle) gridLineTarget {
	return gridLineTarget{start: &st.GridColumnStart, span: &st.GridColumnSpan}
}

func gridTarget(st *ResolvedStyle, isRow bool) gridLineTarget {
	if isRow {
		return rowGridTarget(st)
	}

	return colGridTarget(st)
}

// parseGridLineAt handles "N", "span N", "N / M" and "N / span M" for one
// grid axis.
func parseGridLineAt(target gridLineTarget, value string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "span ") {
		applyGridSpanToken(target, strings.TrimSpace(strings.TrimPrefix(value, "span ")))

		return
	}

	// "1 / 3" or "1 / span 2"
	parts := strings.Split(value, "/")
	if len(parts) == 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
			*target.start = v
			*target.span = 1
		}

		return
	}

	setGridStartToken(target, parts[0])
	applyGridEndToken(target, parts[1])
}

// setGridStartToken applies a positive start line index.
func setGridStartToken(target gridLineTarget, token string) {
	if v, err := strconv.Atoi(strings.TrimSpace(token)); err == nil && v > 0 {
		*target.start = v
	}
}

// applyGridEndToken applies a "span N" or absolute end line; absolute ends
// become spans relative to the start line.
func applyGridEndToken(target gridLineTarget, end string) {
	end = strings.TrimSpace(end)
	if strings.HasPrefix(end, "span ") {
		applyGridSpanToken(target, strings.TrimSpace(strings.TrimPrefix(end, "span ")))

		return
	}

	if v, err := strconv.Atoi(end); err == nil && *target.start > 0 {
		sp := v - *target.start
		if sp < 1 {
			sp = 1
		}

		*target.span = sp
	}
}

// applyGridSpanToken sets the span when token is a positive integer.
func applyGridSpanToken(target gridLineTarget, token string) {
	if n, err := strconv.Atoi(token); err == nil && n > 0 {
		*target.span = n
	}
}

func applyGridEndOnly(style *ResolvedStyle, isRow bool, value string) {
	target := gridTarget(style, isRow)
	trimmed := strings.TrimSpace(value)

	if strings.HasPrefix(trimmed, "span ") {
		applyGridSpanToken(target, strings.TrimSpace(strings.TrimPrefix(trimmed, "span ")))

		return
	}

	applyGridEndNumeric(style, target, isRow, trimmed)
}

func applyGridEndNumeric(style *ResolvedStyle, target gridLineTarget, isRow bool, value string) {
	val, err := strconv.Atoi(value)
	if err != nil || val <= 0 {
		return
	}

	if isRow {
		style.GridRowEnd = val
	} else {
		style.GridColumnEnd = val
	}

	if *target.start > 0 {
		sp := val - *target.start
		if sp < 1 {
			sp = 1
		}

		*target.span = sp
	}
}

// uaDecls is the user-agent declaration table for element names. Lookup is
// per element; unknown names get the initial values.
var uaDecls = map[string][]css.Declaration{ //nolint:gochecknoglobals // static UA table
	"html": {{Prop: "display", Value: "block"}}, //nolint:exhaustruct // intentional zero fields
	"body": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "8px"},    //nolint:exhaustruct // intentional zero fields
	},
	divElementName: {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	htmlSection: {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"article": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"header": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"footer": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"main": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"aside": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"nav": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"form": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"fieldset": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"figure": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"figcaption": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"blockquote": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"address": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"dl": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"dd": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"details": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"summary": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"p": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"pre": {
		// Match browser UA: preserve newlines/spaces; monospace is a
		// soft preference (we fall back to Liberation Sans metrics).
		{Prop: "display", Value: "block"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},          //nolint:exhaustruct // intentional zero fields
		{Prop: "white-space", Value: "pre"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"code": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"kbd": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"samp": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"h1": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "2em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.67em 0"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h2": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1.5em"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.83em 0"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h3": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1.17em"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},     //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h4": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"h5": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"h6": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"ul": {
		{Prop: "display", Value: "block"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "disc"}, //nolint:exhaustruct // intentional zero fields
	},
	"menu": {
		{Prop: "display", Value: "block"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "disc"}, //nolint:exhaustruct // intentional zero fields
	},
	"ol": {
		{Prop: "display", Value: "block"},           //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},            //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "decimal"}, //nolint:exhaustruct // intentional zero fields
	},
	"li": {
		{Prop: "display", Value: "list-item"}, //nolint:exhaustruct // intentional zero fields
	},
	"table": {
		{Prop: "display", Value: "table"},      //nolint:exhaustruct // intentional zero fields
		{Prop: "border-spacing", Value: "2px"}, //nolint:exhaustruct // intentional zero fields
	},
	"thead": {
		{Prop: "display", Value: "table-header-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tfoot": {
		{Prop: "display", Value: "table-footer-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tbody": {
		{Prop: "display", Value: "table-row-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tr": {
		{Prop: "display", Value: "table-row"}, //nolint:exhaustruct // intentional zero fields
	},
	"td": {
		{Prop: "display", Value: "table-cell"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "padding", Value: "1px"},        //nolint:exhaustruct // intentional zero fields
	},
	"th": {
		{Prop: "display", Value: "table-cell"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "padding", Value: "1px"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "text-align", Value: "center"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"},   //nolint:exhaustruct // intentional zero fields
	},
	"img": {
		{Prop: "display", Value: "inline-block"}, //nolint:exhaustruct // intentional zero fields
	},
	"meter": {
		{Prop: "display", Value: "inline-block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "width", Value: "10em"},           //nolint:exhaustruct // intentional zero fields
		{Prop: "height", Value: "1em"},           //nolint:exhaustruct // intentional zero fields
	},
	"progress": {
		{Prop: "display", Value: "inline-block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "width", Value: "10em"},           //nolint:exhaustruct // intentional zero fields
		{Prop: "height", Value: "1em"},           //nolint:exhaustruct // intentional zero fields
	},
	"hr": {
		{Prop: "display", Value: "block"},     //nolint:exhaustruct // intentional zero fields
		{Prop: "border", Value: "1px inset"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.5em auto"}, //nolint:exhaustruct // intentional zero fields
	},
	"a": {
		{Prop: "color", Value: "#0000ee"},             //nolint:exhaustruct // intentional zero fields
		{Prop: "text-decoration", Value: "underline"}, //nolint:exhaustruct // intentional zero fields
	},
	"b": {
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"strong": {
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"i": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"em": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"cite": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"dfn": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"var": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"u": {
		{Prop: "text-decoration", Value: "underline"}, //nolint:exhaustruct // intentional zero fields
	},
	"s": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"strike": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"del": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"small": {
		{Prop: "font-size", Value: "smaller"}, //nolint:exhaustruct // intentional zero fields
	},
	"big": {
		{Prop: "font-size", Value: "larger"}, //nolint:exhaustruct // intentional zero fields
	},
	"center": {
		{Prop: "text-align", Value: "center"}, //nolint:exhaustruct // intentional zero fields
	},
	"title": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	styleElement: {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"script": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"meta": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"link": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"head": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"textarea": {
		{Prop: "white-space", Value: "pre"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"br": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
}

// uaRules returns the user-agent declarations for an element name.
func uaRules(name string) []css.Declaration {
	return uaDecls[name]
}
