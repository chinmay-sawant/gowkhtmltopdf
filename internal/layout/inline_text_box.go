package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"

// textBoxTrimFlags reports whether the block's first/last line should drop
// over/under half-leading (and optionally retarget edges via text-box-edge).
const (
	textBoxTrimStartKeyword = "trim-start"
	textBoxTrimBothKeyword  = "trim-both"
	textBoxEdgeCapKeyword   = "cap"
	textBoxEdgeAlphaKeyword = "alphabetic"
	textBoxCapRatio         = 0.7
	textBoxExRatio          = 0.5
)

func textBoxTrimFlags(block *ResolvedStyle, firstLine, lastLine bool) (bool, bool) {
	if block == nil {
		return false, false
	}

	switch block.TextBoxTrim {
	case textBoxTrimStartKeyword:
		return firstLine, false
	case "trim-end":
		return false, lastLine
	case textBoxTrimBothKeyword:
		return firstLine, lastLine
	default:
		return false, false
	}
}

// adjustTextBoxMetrics applies text-box-edge + trim to one item's ascent/
// descent/half-leading contribution. trimStart/trimEnd drop the matching
// half-leading; when trimming, cap/ex/alphabetic retarget the content edges.
//
//nolint:cyclop // text-box edge and trim rules combine independent CSS axes
func (e *engine) adjustTextBoxMetrics(
	style *ResolvedStyle, face *pdf.Font, size, ascent, descent, extra float64,
	trimStart, trimEnd bool,
) (float64, float64) {
	extraTop := extra
	extraBottom := extra

	if !trimStart && !trimEnd {
		return ascent + extraTop, descent + extraBottom
	}

	over := "auto"
	under := "auto"

	if style != nil {
		if style.TextBoxEdgeOver != "" {
			over = style.TextBoxEdgeOver
		}

		if style.TextBoxEdgeUnder != "" {
			under = style.TextBoxEdgeUnder
		}
	}

	contentAscent := ascent
	contentDescent := descent

	switch over {
	case textBoxEdgeCapKeyword:
		if capH := e.fontCapHeightFace(face, size); capH > 0 && capH < contentAscent {
			contentAscent = capH
		}
	case "ex":
		if exH := e.fontExHeightFace(face, size); exH > 0 && exH < contentAscent {
			contentAscent = exH
		}
	}

	if under == textBoxEdgeAlphaKeyword {
		contentDescent = 0
	}

	if trimStart {
		extraTop = 0
		ascent = contentAscent
	}

	if trimEnd {
		extraBottom = 0
		descent = contentDescent
	}

	return ascent + extraTop, descent + extraBottom
}

func (e *engine) fontCapHeightFace(face *pdf.Font, size float64) float64 {
	if face == nil || face.UnitsPerEm() <= 0 {
		return size * textBoxCapRatio
	}

	capHeight := face.CapHeight()
	if capHeight <= 0 {
		return size * textBoxCapRatio
	}

	return float64(capHeight) * size / float64(face.UnitsPerEm())
}

// fontExHeightFace approximates x-height when the face has no OS/2 xHeight.
func (e *engine) fontExHeightFace(face *pdf.Font, size float64) float64 {
	capHeight := e.fontCapHeightFace(face, size)
	if capHeight > 0 {
		return capHeight * textBoxCapRatio
	}

	return size * textBoxExRatio
}
