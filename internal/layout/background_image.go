//nolint:varnamelen,cyclop,wsl,intrange,nlreturn,funlen,mnd,goconst,gocognit // background layers
package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	backgroundURLPrefix = "url("
	gradientFuncMark    = "gradient("
	maxBackgroundTiles  = 64
)

// appendBackgroundImage paints all background-image layers (gradients, images)
// with position, size, repeat, origin, and clip.
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

	originX, originY, originW, originH := resolveBackgroundOriginBox(sty, posX, posY, width, height)
	if originW <= 0 || originH <= 0 {
		originX, originY, originW, originH = posX, posY, width, height
	}

	for i := len(layers) - 1; i >= 0; i-- {
		layer := strings.TrimSpace(layers[i])
		if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
			continue
		}

		if isGradientFunc(layer) {
			if pngData, imgW, imgH, ok := renderGradientPNG(layer, originW, originH, sty.Color); ok {
				op := Op{ //nolint:exhaustruct // intentional zero fields
					Kind:   OpImage,
					X:      originX,
					Y:      originY,
					W:      originW,
					H:      originH,
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

		imgW := float64(ref.w)
		imgH := float64(ref.h)
		if imgW <= 0 {
			imgW = originW
		}
		if imgH <= 0 {
			imgH = originH
		}

		destW, destH := resolveBackgroundSize(sty.BackgroundSize, originW, originH, imgW, imgH)
		if (sty.BackgroundSize == "" || strings.EqualFold(sty.BackgroundSize, "auto")) &&
			(strings.EqualFold(sty.BackgroundRepeat, "repeat") ||
				strings.EqualFold(sty.BackgroundRepeat, "repeat-x") ||
				strings.EqualFold(sty.BackgroundRepeat, "repeat-y")) {
			if imgW > 0 && imgH > 0 {
				destW, destH = imgW, imgH
			}
		}
		destX, destY := resolveBackgroundPosition(
			sty.BackgroundPosX, sty.BackgroundPosY, originX, originY, originW, originH, destW, destH,
		)

		baseOp := Op{ //nolint:exhaustruct // intentional zero fields
			Kind:   OpImage,
			X:      destX,
			Y:      destY,
			W:      destW,
			H:      destH,
			Image:  ref.data,
			ImgW:   ref.w,
			ImgH:   ref.h,
			IsJPEG: ref.isJPEG,
		}
		if sty.Filter != "" {
			filters := parseFilterList(sty.Filter, sty.Color, sty.FontSize)
			baseOp.Image = applyImageFilterToImage(baseOp.Image, filters)
		}

		dst = tileBackgroundRepeat(
			dst, baseOp, sty.BackgroundRepeat, originX, originY, originW, originH, destX, destY, destW, destH,
		)
	}

	return dst
}

func resolveBackgroundOriginBox(
	sty ResolvedStyle, posX, posY, width, height float64,
) (float64, float64, float64, float64) {
	switch sty.BackgroundOrigin {
	case "content-box":
		x := posX + sty.BorderLeft.Width + sty.PaddingLeft
		y := posY + sty.BorderTop.Width + sty.PaddingTop
		w := width - sty.BorderLeft.Width - sty.BorderRight.Width - sty.PaddingLeft - sty.PaddingRight
		h := height - sty.BorderTop.Width - sty.BorderBottom.Width - sty.PaddingTop - sty.PaddingBottom
		return x, y, math.Max(0, w), math.Max(0, h)
	case "border-box":
		return posX, posY, width, height
	default: // padding-box
		x := posX + sty.BorderLeft.Width
		y := posY + sty.BorderTop.Width
		w := width - sty.BorderLeft.Width - sty.BorderRight.Width
		h := height - sty.BorderTop.Width - sty.BorderBottom.Width
		return x, y, math.Max(0, w), math.Max(0, h)
	}
}

func resolveBackgroundSize(sizeSpec string, originW, originH, imgW, imgH float64) (float64, float64) {
	spec := strings.ToLower(strings.TrimSpace(sizeSpec))
	switch spec {
	case "cover":
		scale := math.Max(originW/imgW, originH/imgH)
		return imgW * scale, imgH * scale
	case "contain":
		scale := math.Min(originW/imgW, originH/imgH)
		return imgW * scale, imgH * scale
	case "", "auto":
		return originW, originH
	}

	parts := strings.Fields(spec)
	if len(parts) == 1 {
		w := parseBgDim(parts[0], originW, imgW)
		h := imgH * (w / imgW)
		return w, h
	}
	if len(parts) >= 2 {
		w := parseBgDim(parts[0], originW, imgW)
		h := parseBgDim(parts[1], originH, imgH)
		return w, h
	}

	return originW, originH
}

func parseBgDim(token string, containerDim, fallback float64) float64 {
	token = strings.TrimSpace(token)
	if token == "auto" || token == "" {
		return fallback
	}
	if strings.HasSuffix(token, "%") {
		if val, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64); err == nil {
			return containerDim * val / 100.0
		}
	}
	if val, unit, ok := css.ParseLength(token); ok {
		if pt, converted := lengthToPt(val, unit, defaultFontSizePt); converted {
			return pt
		}
		return val
	}
	return fallback
}

func resolveBackgroundPosition(
	posSpecX, posSpecY string, originX, originY, originW, originH, destW, destH float64,
) (float64, float64) {
	offX := 0.0
	specX := strings.ToLower(strings.TrimSpace(posSpecX))
	switch specX {
	case "center", "50%":
		offX = (originW - destW) / 2.0
	case "right", "100%":
		offX = originW - destW
	case "left", "0%", "":
		offX = 0
	default:
		if strings.HasSuffix(specX, "%") {
			if val, err := strconv.ParseFloat(strings.TrimSuffix(specX, "%"), 64); err == nil {
				offX = (originW - destW) * val / 100.0
			}
		} else if val, unit, ok := css.ParseLength(specX); ok {
			if pt, converted := lengthToPt(val, unit, defaultFontSizePt); converted {
				offX = pt
			}
		}
	}

	offY := 0.0
	specY := strings.ToLower(strings.TrimSpace(posSpecY))
	switch specY {
	case "center", "50%":
		offY = (originH - destH) / 2.0
	case "bottom", "100%":
		offY = originH - destH
	case "top", "0%", "":
		offY = 0
	default:
		if strings.HasSuffix(specY, "%") {
			if val, err := strconv.ParseFloat(strings.TrimSuffix(specY, "%"), 64); err == nil {
				offY = (originH - destH) * val / 100.0
			}
		} else if val, unit, ok := css.ParseLength(specY); ok {
			if pt, converted := lengthToPt(val, unit, defaultFontSizePt); converted {
				offY = pt
			}
		}
	}

	return originX + offX, originY + offY
}

func tileBackgroundRepeat(
	dst []Op, baseOp Op, repeatSpec string,
	originX, originY, originW, originH, destX, destY, destW, destH float64,
) []Op {
	spec := strings.ToLower(strings.TrimSpace(repeatSpec))
	switch spec {
	case "no-repeat", "":
		dst = append(dst, baseOp)
		return dst
	case "repeat-x":
		if destW <= 0 {
			dst = append(dst, baseOp)
			return dst
		}
		count := int(math.Ceil(originW / destW))
		if count > maxBackgroundTiles {
			count = maxBackgroundTiles
		}
		for k := 0; k < count; k++ {
			op := baseOp
			op.X = originX + float64(k)*destW
			op.Y = destY
			if op.X+op.W > originX+originW {
				op.W = originX + originW - op.X
			}
			if op.Y+op.H > originY+originH {
				op.H = originY + originH - op.Y
			}
			if op.W > 0 && op.H > 0 {
				dst = append(dst, op)
			}
		}
		return dst
	case "repeat-y":
		if destH <= 0 {
			dst = append(dst, baseOp)
			return dst
		}
		count := int(math.Ceil(originH / destH))
		if count > maxBackgroundTiles {
			count = maxBackgroundTiles
		}
		for k := 0; k < count; k++ {
			op := baseOp
			op.X = destX
			op.Y = originY + float64(k)*destH
			if op.X+op.W > originX+originW {
				op.W = originX + originW - op.X
			}
			if op.Y+op.H > originY+originH {
				op.H = originY + originH - op.Y
			}
			if op.W > 0 && op.H > 0 {
				dst = append(dst, op)
			}
		}
		return dst
	case "repeat":
		if (destW >= originW && destH >= originH) || destW <= 0 || destH <= 0 {
			dst = append(dst, baseOp)
			return dst
		}
		countX := int(math.Ceil(originW / destW))
		countY := int(math.Ceil(originH / destH))
		if countX*countY > maxBackgroundTiles {
			countX = int(math.Sqrt(maxBackgroundTiles))
			countY = countX
		}
		for yk := 0; yk < countY; yk++ {
			for xk := 0; xk < countX; xk++ {
				op := baseOp
				op.X = originX + float64(xk)*destW
				op.Y = originY + float64(yk)*destH
				if op.X+op.W > originX+originW {
					op.W = originX + originW - op.X
				}
				if op.Y+op.H > originY+originH {
					op.H = originY + originH - op.Y
				}
				if op.W > 0 && op.H > 0 {
					dst = append(dst, op)
				}
			}
		}
		return dst
	default:
		dst = append(dst, baseOp)
		return dst
	}
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
