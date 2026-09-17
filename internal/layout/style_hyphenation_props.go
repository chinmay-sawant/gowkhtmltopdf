//nolint:cyclop // hyphenation limit + hanging-punctuation parsers
package layout

import (
	"strconv"
	"strings"
)

// applyHyphenationProps owns hanging-punctuation and hyphenate-limit-*.
// Existing hyphens / hyphenate-character stay in applyTextPropsWave3; this
// group only adds the limit longhands and hanging punctuation.
func applyHyphenationProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	case "hanging-punctuation":
		if isHangingPunctuationValue(val) {
			style.HangingPunctuation = val
		}
	case "hyphenate-limit-chars":
		applyHyphenateLimitChars(style, val)
	case "hyphenate-limit-last":
		if isHyphenateLimitLastValue(val) {
			style.HyphenateLimitLast = val
		}
	case "hyphenate-limit-lines":
		applyHyphenateLimitLines(style, val)
	case "hyphenate-limit-zone":
		applyHyphenateLimitZone(style, val, fsize, ctx)
	default:
		return false
	}

	return true
}

func isHangingPunctuationValue(val string) bool {
	if val == cssDisplayNone {
		return true
	}

	for _, tok := range strings.Fields(val) {
		switch tok {
		case "first", "last", "allow-end", "force-end":
			continue
		default:
			return false
		}
	}

	return true
}

func applyHyphenateLimitChars(style *ResolvedStyle, val string) {
	if val == "auto" {
		style.HyphenateLimitMinWord = 0
		style.HyphenateLimitMinBefore = 0
		style.HyphenateLimitMinAfter = 0

		return
	}

	parts := strings.Fields(val)
	nums := make([]int, 0, 3)

	for _, part := range parts {
		if part == "auto" {
			nums = append(nums, 0)

			continue
		}

		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return
		}

		nums = append(nums, n)
	}

	switch len(nums) {
	case 1:
		style.HyphenateLimitMinWord = nums[0]
		style.HyphenateLimitMinBefore = 0
		style.HyphenateLimitMinAfter = 0
	case 2:
		style.HyphenateLimitMinWord = nums[0]
		style.HyphenateLimitMinBefore = nums[1]
		style.HyphenateLimitMinAfter = nums[1]
	case 3:
		style.HyphenateLimitMinWord = nums[0]
		style.HyphenateLimitMinBefore = nums[1]
		style.HyphenateLimitMinAfter = nums[2]
	}
}

func isHyphenateLimitLastValue(val string) bool {
	switch val {
	case cssDisplayNone, "always", "column", "page", "spread":
		return true
	default:
		return false
	}
}

func applyHyphenateLimitLines(style *ResolvedStyle, val string) {
	if val == "no-limit" {
		style.HyphenateLimitLines = -1

		return
	}

	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return
	}

	style.HyphenateLimitLines = n
}

func applyHyphenateLimitZone(style *ResolvedStyle, val string, fsize float64, ctx *styleContext) {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "%") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
		if err != nil || n < 0 {
			return
		}

		style.HyphenateLimitZonePct = n
		style.HyphenateLimitZonePt = 0

		return
	}

	viewport := 0.0
	if ctx != nil {
		viewport = ctx.viewportW
	}

	if length, ok := plainLength(val, fsize, viewport); ok && length >= 0 {
		style.HyphenateLimitZonePt = length
		style.HyphenateLimitZonePct = -1
	}
}
