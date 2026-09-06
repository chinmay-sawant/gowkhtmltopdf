package layout

import "strings"

const (
	backgroundRepeatShorthandTwoValues = 2
	backgroundRepeatRepeat             = "repeat"
	backgroundRepeatNoRepeat           = "no-repeat"
)

// backgroundRepeatAxes resolves the independent horizontal and vertical
// repeat values. An axis longhand starts the other axis at its CSS initial
// value, repeat, before applying the explicit axis value.
func backgroundRepeatAxes(sty ResolvedStyle) (string, string) {
	repeatX, repeatY := backgroundRepeatBase(sty)
	repeatX = overrideBackgroundRepeat(repeatX, sty.BackgroundRepeatX)
	repeatY = overrideBackgroundRepeat(repeatY, sty.BackgroundRepeatY)

	return applyLogicalBackgroundRepeat(sty, repeatX, repeatY)
}

func backgroundRepeatBase(sty ResolvedStyle) (string, string) {
	repeatX, repeatY := parseBackgroundRepeatShorthand(sty.BackgroundRepeat)

	if sty.BackgroundRepeat == "" && hasBackgroundRepeatAxisLonghand(sty) {
		return backgroundRepeatRepeat, backgroundRepeatRepeat
	}

	return repeatX, repeatY
}

func hasBackgroundRepeatAxisLonghand(sty ResolvedStyle) bool {
	return sty.BackgroundRepeatX != "" || sty.BackgroundRepeatY != "" ||
		sty.BackgroundRepeatBlock != "" || sty.BackgroundRepeatInline != ""
}

func overrideBackgroundRepeat(current, value string) string {
	if normalized := normalizeBackgroundRepeatValue(value); normalized != "" {
		return normalized
	}

	return current
}

func applyLogicalBackgroundRepeat(sty ResolvedStyle, repeatX, repeatY string) (string, string) {
	block := normalizeBackgroundRepeatValue(sty.BackgroundRepeatBlock)
	inline := normalizeBackgroundRepeatValue(sty.BackgroundRepeatInline)

	if backgroundRepeatIsVerticalWritingMode(sty.WritingMode) {
		return overrideBackgroundRepeat(repeatX, block), overrideBackgroundRepeat(repeatY, inline)
	}

	return overrideBackgroundRepeat(repeatX, inline), overrideBackgroundRepeat(repeatY, block)
}

func backgroundRepeatIsVerticalWritingMode(writingMode string) bool {
	return writingMode == writingModeVerticalRL || writingMode == writingModeVerticalLR
}

func parseBackgroundRepeatShorthand(value string) (string, string) {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(parts) == 0 {
		return backgroundRepeatRepeat, backgroundRepeatRepeat
	}

	if len(parts) >= backgroundRepeatShorthandTwoValues {
		return normalizedRepeatOrDefault(parts[0]), normalizedRepeatOrDefault(parts[1])
	}

	switch parts[0] {
	case "repeat-x":
		return backgroundRepeatRepeat, backgroundRepeatNoRepeat
	case "repeat-y":
		return backgroundRepeatNoRepeat, backgroundRepeatRepeat
	case "repeat-block":
		return backgroundRepeatNoRepeat, backgroundRepeatRepeat
	case "repeat-inline":
		return backgroundRepeatRepeat, backgroundRepeatNoRepeat
	default:
		value := normalizedRepeatOrDefault(parts[0])

		return value, value
	}
}

func normalizeBackgroundRepeatValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	return normalizedRepeatOrDefault(value)
}

func normalizedRepeatOrDefault(value string) string {
	switch value {
	case backgroundRepeatRepeat, backgroundRepeatNoRepeat, "space", borderRepeatRound:
		return value
	default:
		return backgroundRepeatRepeat
	}
}

func setBackgroundRepeatShorthand(style *ResolvedStyle, value string) {
	style.BackgroundRepeat = strings.TrimSpace(value)
	style.BackgroundRepeatX, style.BackgroundRepeatY = parseBackgroundRepeatShorthand(value)
	style.BackgroundRepeatBlock = ""
	style.BackgroundRepeatInline = ""
}
