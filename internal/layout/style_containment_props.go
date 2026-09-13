//nolint:cyclop,mnd,varnamelen // containment property parser and layout helpers
package layout

import "strings"

// CSS Containment support. Contain stores the normalized authored keyword
// list ("none", "strict", "content", or space-separated size/layout/paint/
// style); the strict/content shorthands expand through the contains* helpers
// below. contain-intrinsic-* lengths are stored in CSS points with -1 as the
// auto/unset sentinel. contain: style and content-visibility: auto are parsed
// for cascade fidelity but have no print-observable effect.
const (
	containValueNone     = "none"
	containValueStrict   = "strict"
	containValueContent  = "content"
	containKeywordSize   = "size"
	containKeywordLayout = "layout"
	containKeywordPaint  = "paint"
	containKeywordStyle  = "style"

	containIntrinsicAuto = "auto"
	containIntrinsicNone = "none"

	contentVisibilityVisible = "visible"
	contentVisibilityAuto    = "auto"
	contentVisibilityHidden  = "hidden"
)

// applyContainmentProps owns contain, the contain-intrinsic-* family, and
// content-visibility. It writes ResolvedStyle.Contain,
// .ContainIntrinsicWidth, .ContainIntrinsicHeight,
// .ContainIntrinsicBlockSize, .ContainIntrinsicInlineSize, and
// .ContentVisibility; the layout consumers live in layout_flow.go
// (flowChildren, flowAbsCB, floatIntrinsicAvail) and layout_measure.go
// (measureCellMinMax).
func applyContainmentProps(
	style *ResolvedStyle, prop, value string, fsize float64,
	_ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	case "contain":
		return applyContainKeyword(style, val)
	case "contain-intrinsic-size":
		width, height, ok := parseContainIntrinsicSizes(val, fsize, 2)
		if !ok {
			return false
		}

		style.ContainIntrinsicWidth = width
		style.ContainIntrinsicHeight = height

		return true
	case "contain-intrinsic-width":
		return applyContainIntrinsicAxis(&style.ContainIntrinsicWidth, val, fsize)
	case "contain-intrinsic-height":
		return applyContainIntrinsicAxis(&style.ContainIntrinsicHeight, val, fsize)
	case "contain-intrinsic-block-size":
		return applyContainIntrinsicAxis(&style.ContainIntrinsicBlockSize, val, fsize)
	case "contain-intrinsic-inline-size":
		return applyContainIntrinsicAxis(&style.ContainIntrinsicInlineSize, val, fsize)
	case "content-visibility":
		switch val {
		case contentVisibilityVisible, contentVisibilityAuto, contentVisibilityHidden:
			style.ContentVisibility = val

			return true
		}
	}

	return false
}

// applyContainKeyword validates and stores contain's keyword list. The
// grammar is none | strict | content | [ size || layout || paint || style ];
// the single-keyword shorthands keep their authored form and the multi-keyword
// list rejects unknown names and repeats.
func applyContainKeyword(style *ResolvedStyle, val string) bool {
	tokens := strings.Fields(val)

	if len(tokens) == 1 {
		switch tokens[0] {
		case containValueNone, containValueStrict, containValueContent:
			style.Contain = tokens[0]

			return true
		}
	}

	seen := make(map[string]struct{}, len(tokens))

	for _, token := range tokens {
		switch token {
		case containKeywordSize, containKeywordLayout, containKeywordPaint, containKeywordStyle:
		default:
			return false
		}

		if _, dup := seen[token]; dup {
			return false
		}

		seen[token] = struct{}{}
	}

	if len(seen) == 0 {
		return false
	}

	style.Contain = strings.Join(tokens, " ")

	return true
}

// parseContainIntrinsicSizes parses contain-intrinsic-size (maxEntries 2) or
// one longhand (maxEntries 1). Each entry is auto? [ none | <length> ]; the
// shorthand mirrors a single entry to both axes. This engine has no
// last-remembered-size state for auto, so "auto <length>" stores the length
// and a bare auto stores the -1 unset sentinel.
func parseContainIntrinsicSizes(val string, fsize float64, maxEntries int) (float64, float64, bool) {
	tokens := strings.Fields(val)

	var sizes [2]float64

	count := 0

	for i := 0; i < len(tokens); {
		if count == maxEntries {
			return 0, 0, false
		}

		hadAuto := false
		if tokens[i] == containIntrinsicAuto {
			hadAuto = true

			i++
		}

		size := -1.0
		tookValue := false

		if i < len(tokens) {
			switch {
			case tokens[i] == containIntrinsicNone:
				tookValue = true

				i++
			default:
				if v, ok := parseAdvancedLengthOK(tokens[i], fsize); ok && v >= 0 {
					size = v
					tookValue = true

					i++
				}
			}
		}

		if !tookValue && !hadAuto {
			return 0, 0, false
		}

		sizes[count] = size
		count++
	}

	if count == 0 {
		return 0, 0, false
	}

	if count == 1 {
		sizes[1] = sizes[0]
	}

	return sizes[0], sizes[1], true
}

// applyContainIntrinsicAxis stores one longhand value, accepting a length,
// "none", or "auto [length]" with the same auto handling as the shorthand.
func applyContainIntrinsicAxis(dst *float64, val string, fsize float64) bool {
	width, _, ok := parseContainIntrinsicSizes(val, fsize, 1)
	if !ok {
		return false
	}

	*dst = width

	return true
}

// containHasKeyword reports whether the normalized contain list includes
// keyword, expanding strict (size layout paint style) and content (layout
// paint style).
func containHasKeyword(contain, keyword string) bool {
	if contain == "" || contain == containValueNone {
		return false
	}

	if contain == keyword {
		return true
	}

	switch contain {
	case containValueStrict:
		switch keyword {
		case containKeywordSize, containKeywordLayout, containKeywordPaint, containKeywordStyle:
			return true
		}

		return false
	case containValueContent:
		switch keyword {
		case containKeywordLayout, containKeywordPaint, containKeywordStyle:
			return true
		}

		return false
	}

	for _, token := range strings.Fields(contain) {
		if token == keyword {
			return true
		}
	}

	return false
}

// containsSize reports size containment: contain: size or strict.
func containsSize(st ResolvedStyle) bool {
	return containHasKeyword(st.Contain, containKeywordSize)
}

// containsLayout reports layout containment: contain: layout, strict, content.
func containsLayout(st ResolvedStyle) bool {
	return containHasKeyword(st.Contain, containKeywordLayout)
}

// containsPaint reports paint containment: contain: paint, strict, content.
// content-visibility hidden/auto imply paint containment too; hidden is
// handled where descendants are skipped and auto has no print-observable
// effect.
func containsPaint(st ResolvedStyle) bool {
	return containHasKeyword(st.Contain, containKeywordPaint)
}

// containmentIntrinsicWidth returns the physical-axis content width in
// unscaled CSS points from the contain-intrinsic-* fields, or -1 when unset.
// Physical longhands win over logical ones; the logical axis maps through the
// writing mode (horizontal-tb: inline is width, block is height; vertical
// modes swap them).
func containmentIntrinsicWidth(st ResolvedStyle) float64 {
	if st.ContainIntrinsicWidth >= 0 {
		return st.ContainIntrinsicWidth
	}

	if isVerticalWritingMode(st.WritingMode) {
		return st.ContainIntrinsicBlockSize
	}

	return st.ContainIntrinsicInlineSize
}

// containmentIntrinsicHeight returns the physical-axis content height in
// unscaled CSS points from the contain-intrinsic-* fields, or -1 when unset.
func containmentIntrinsicHeight(st ResolvedStyle) float64 {
	if st.ContainIntrinsicHeight >= 0 {
		return st.ContainIntrinsicHeight
	}

	if isVerticalWritingMode(st.WritingMode) {
		return st.ContainIntrinsicInlineSize
	}

	return st.ContainIntrinsicBlockSize
}
