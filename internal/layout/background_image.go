package layout

import "strings"

const (
	backgroundURLPrefix = "url("
	gradientFuncMark    = "gradient("
)

// appendBackgroundImage paints the first background-image layer, no-repeat,
// at the box origin, sized to the box. Missing images are skipped so layout
// does not fail. Gradients are ignored.
func (e *engine) appendBackgroundImage(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
) []Op {
	if op, ok := e.backgroundImageOp(sty, posX, posY, width, height); ok {
		return append(dst, op)
	}

	return dst
}

func (e *engine) backgroundImageOp(
	sty ResolvedStyle, posX, posY, width, height float64,
) (Op, bool) {
	if !e.opts.Background || width <= 0 || height <= 0 {
		return Op{}, false //nolint:exhaustruct // empty op on skip
	}

	src := backgroundImageSrc(sty.BackgroundImage)
	if src == "" {
		return Op{}, false //nolint:exhaustruct // empty op on skip
	}

	ref := e.resolveImage(src)
	if ref == nil || ref.data == nil {
		return Op{}, false //nolint:exhaustruct // empty op on skip
	}

	return Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpImage,
		X:      posX,
		Y:      posY,
		W:      width,
		H:      height,
		Image:  ref.data,
		ImgW:   ref.w,
		ImgH:   ref.h,
		IsJPEG: ref.isJPEG,
	}, true
}

// backgroundImageSrc returns the first-layer fetch target. Accepts url("x"),
// url('x'), url(x), or a bare path. empty / none / gradients yield "".
func backgroundImageSrc(raw string) string {
	layer := strings.TrimSpace(firstCommaLayer(strings.TrimSpace(raw)))
	if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
		return ""
	}

	lower := strings.ToLower(layer)
	if idx := strings.Index(lower, backgroundURLPrefix); idx >= 0 {
		if gidx := strings.Index(lower, gradientFuncMark); gidx >= 0 && gidx < idx {
			return ""
		}

		return urlFunctionTarget(layer[idx:])
	}

	if strings.Contains(layer, "(") || strings.Contains(lower, gradientFuncMark) {
		return ""
	}

	switch lower {
	case inheritKeyword, cssKeywordInitial, cssKeywordUnset, cssKeywordRevert:
		return ""
	}

	return strings.Trim(layer, `"'`)
}

// firstCommaLayer returns the first comma-separated background layer, not
// splitting on commas inside quotes or parentheses.
func firstCommaLayer(raw string) string {
	depth := 0
	inQuote := byte(0)

	for idx := range len(raw) {
		char := raw[idx]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}

			continue
		}

		switch char {
		case '"', '\'':
			inQuote = char
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				return raw[:idx]
			}
		}
	}

	return raw
}

func urlFunctionTarget(layer string) string {
	start := len(backgroundURLPrefix)
	if len(layer) < start {
		return ""
	}

	inQuote := byte(0)

	for idx := start; idx < len(layer); idx++ {
		char := layer[idx]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}

			continue
		}

		switch char {
		case '"', '\'':
			inQuote = char
		case ')':
			inner := strings.TrimSpace(layer[start:idx])

			return strings.Trim(inner, `"'`)
		}
	}

	return ""
}
