package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// applyFontSizeAdjustProps owns font-size-adjust (ex-height number form).
func applyFontSizeAdjustProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop != "font-size-adjust" {
		return false
	}

	if adjust, set, ok := parseFontSizeAdjustValue(value); ok {
		style.FontSizeAdjust = adjust
		style.FontSizeAdjustSet = set
	}

	return true
}

func parseFontSizeAdjustValue(raw string) (float64, bool, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return 0, false, false
	}

	if value == "none" {
		return 0, false, true
	}

	// from-font / metric keywords need richer face probing; drop for now.
	if strings.Contains(value, " ") || value == "from-font" {
		return 0, false, false
	}

	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n < 0 {
		return 0, false, false
	}

	return n, true, true
}

// usedFontSize returns the font size after font-size-adjust. When the face has
// no usable x-height aspect, the specified size is returned unchanged.
func usedFontSize(sty *ResolvedStyle, face *pdf.Font) float64 {
	if sty == nil {
		return 0
	}

	size := sty.FontSize
	if !sty.FontSizeAdjustSet || face == nil {
		return size
	}

	aspect := face.XHeightAspect()
	if aspect <= 0 {
		return size
	}

	return size * (sty.FontSizeAdjust / aspect)
}
