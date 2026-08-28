//nolint:varnamelen,cyclop,wsl,intrange,nlreturn,funlen // multi-layer background image and gradient rendering
package layout

import "strings"

const (
	backgroundURLPrefix = "url("
	gradientFuncMark    = "gradient("
)

// appendBackgroundImage paints all background-image layers (gradients, images)
// in reverse order so the first specified layer paints on top of subsequent layers.
func (e *engine) appendBackgroundImage(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
) []Op {
	if !e.opts.Background || width <= 0 || height <= 0 || sty.BackgroundImage == "" {
		return dst
	}

	layers := splitCommaLayers(sty.BackgroundImage)
	if len(layers) == 0 {
		return dst
	}

	for i := len(layers) - 1; i >= 0; i-- {
		layer := strings.TrimSpace(layers[i])
		if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
			continue
		}

		if isGradientFunc(layer) {
			if pngData, imgW, imgH, ok := renderGradientPNG(layer, width, height, sty.Color); ok {
				op := Op{ //nolint:exhaustruct // intentional zero fields
					Kind:   OpImage,
					X:      posX,
					Y:      posY,
					W:      width,
					H:      height,
					Image:  pngData,
					ImgW:   imgW,
					ImgH:   imgH,
					IsJPEG: false,
				}
				if sty.Filter != "" {
					filters := parseFilterList(sty.Filter, sty.Color, sty.FontSize)
					op.Image = applyImageFilterToImage(op.Image, filters)
				}
				dst = append(dst, op)
			}
			continue
		}

		src := backgroundImageSrc(layer)
		if src == "" {
			continue
		}

		ref := e.resolveImage(src)
		if ref == nil || ref.data == nil {
			continue
		}

		op := Op{ //nolint:exhaustruct // intentional zero fields
			Kind:   OpImage,
			X:      posX,
			Y:      posY,
			W:      width,
			H:      height,
			Image:  ref.data,
			ImgW:   ref.w,
			ImgH:   ref.h,
			IsJPEG: ref.isJPEG,
		}
		if sty.Filter != "" {
			filters := parseFilterList(sty.Filter, sty.Color, sty.FontSize)
			op.Image = applyImageFilterToImage(op.Image, filters)
		}
		dst = append(dst, op)
	}

	return dst
}

// backgroundImageSrc returns the fetch target for a layer. Accepts url("x"),
// url('x'), url(x), or a bare path. empty / none / gradients yield "".
func backgroundImageSrc(layer string) string {
	layers := splitCommaLayers(layer)
	if len(layers) > 1 {
		layer = layers[0]
	}
	layer = strings.TrimSpace(layer)
	if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
		return ""
	}

	lower := strings.ToLower(layer)
	if idx := strings.Index(lower, backgroundURLPrefix); idx >= 0 {
		return urlFunctionTarget(layer[idx:])
	}

	if strings.Contains(layer, "(") {
		return ""
	}

	switch lower {
	case inheritKeyword, cssKeywordInitial, cssKeywordUnset, cssKeywordRevert:
		return ""
	}

	return strings.Trim(layer, `"'`)
}

// splitCommaLayers splits raw by top-level commas, not splitting commas inside
// quotes or parentheses.
func splitCommaLayers(raw string) []string {
	var layers []string
	depth := 0
	inQuote := byte(0)
	start := 0

	for idx := 0; idx < len(raw); idx++ {
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
				layers = append(layers, strings.TrimSpace(raw[start:idx]))
				start = idx + 1
			}
		}
	}

	if start < len(raw) {
		trimmed := strings.TrimSpace(raw[start:])
		if trimmed != "" {
			layers = append(layers, trimmed)
		}
	}

	return layers
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
