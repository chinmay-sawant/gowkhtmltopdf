package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	synthSmallCapsScale  = 0.8
	synthSubSuperScale   = 0.65
	synthSubShiftRatio   = 0.2
	synthSuperShiftRatio = 0.35
	synthObliqueSkew     = 0.22
)

func fontWidthScale(sty *ResolvedStyle) float64 {
	if sty == nil || sty.FontWidth <= 0 || sty.FontWidth == 100 {
		return 1
	}

	return sty.FontWidth / 100
}

func synthesizeSmallCapsText(sty *ResolvedStyle, text string) (string, float64) {
	if sty == nil || text == "" || !sty.FontSynthesisSmallCaps {
		return text, 1
	}

	switch sty.FontVariantCaps {
	case "small-caps", "all-small-caps", "petite-caps", "all-petite-caps":
		return strings.ToUpper(text), synthSmallCapsScale
	default:
		return text, 1
	}
}

func synthesizePositionAdjust(sty *ResolvedStyle, size float64) (scale, baselineShift float64) {
	if sty == nil || !sty.FontSynthesisPosition {
		return 1, 0
	}

	switch sty.FontVariantPosition {
	case "sub":
		return synthSubSuperScale, size * synthSubShiftRatio
	case "super":
		return synthSubSuperScale, -size * synthSuperShiftRatio
	default:
		return 1, 0
	}
}

func needsFakeOblique(sty *ResolvedStyle, face *pdf.Font) bool {
	if sty == nil || !sty.FontItalic || !sty.FontSynthesisStyle {
		return false
	}

	if face == nil {
		return true
	}

	return !face.Italic()
}
