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
	maxBackgroundTiles  = 1024
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
	repeatX, repeatY := backgroundRepeatAxes(sty)

	originX, originY, originW, originH := resolveBackgroundOriginBox(sty, posX, posY, width, height)
	if originW <= 0 || originH <= 0 {
		originX, originY, originW, originH = posX, posY, width, height
	}
	// Backgrounds always paint inside the background-clip box (CSS Backgrounds).
	clipX, clipY, clipW, clipH := resolveBackgroundClipBox(sty, posX, posY, width, height)
	if clipW <= 0 || clipH <= 0 {
		clipX, clipY, clipW, clipH = posX, posY, width, height
	}
	clip := clipRect{x: clipX, y: clipY, w: clipW, h: clipH}

	layerStart := len(dst)
	for i := len(layers) - 1; i >= 0; i-- {
		layer := strings.TrimSpace(layers[i])
		if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
			continue
		}

		if isGradientFunc(layer) {
			sizeSpec := backgroundSizeForLayer(sty.BackgroundSize, i)
			destW, destH := resolveBackgroundSize(sizeSpec, originW, originH, originW, originH)
			if sizeSpec == "" || strings.EqualFold(sizeSpec, "auto") {
				destW, destH = originW, originH
			}
			destX, destY := resolveBackgroundPosition(
				sty.BackgroundPosX, sty.BackgroundPosY, originX, originY, originW, originH, destW, destH,
			)
			if pngData, imgW, imgH, ok := renderGradientPNG(layer, destW, destH, sty.Color); ok {
				baseOp := Op{ //nolint:exhaustruct // intentional zero fields
					Kind:         OpImage,
					X:            destX,
					Y:            destY,
					W:            destW,
					H:            destH,
					Image:        pngData,
					ImgW:         imgW,
					ImgH:         imgH,
					IsJPEG:       false,
					IsBackground: true,
					BlendMode:    backgroundBlendModeForLayer(sty.BackgroundBlendMode, i),
				}
				if sty.Filter != "" {
					filters := parseFilterList(sty.Filter, sty.Color, sty.FontSize)
					baseOp.Image = applyImageFilterToImage(baseOp.Image, filters)
				}
				dst = tileBackgroundRepeat(
					dst, baseOp, repeatX, repeatY, clip, destX, destY, destW, destH,
				)
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

		// Bitmap pixels are CSS px; match <img> / inline image sizing (pxToPt).
		imgW := e.scalePt(pxToPt(float64(ref.w)))
		imgH := e.scalePt(pxToPt(float64(ref.h)))
		if imgW <= 0 {
			imgW = originW
		}
		if imgH <= 0 {
			imgH = originH
		}

		sizeSpec := backgroundSizeForLayer(sty.BackgroundSize, i)
		destW, destH := resolveBackgroundSize(sizeSpec, originW, originH, imgW, imgH)
		if (sizeSpec == "" || strings.EqualFold(sizeSpec, "auto")) &&
			(strings.EqualFold(repeatX, backgroundRepeatRepeat) ||
				strings.EqualFold(repeatY, backgroundRepeatRepeat)) {
			if imgW > 0 && imgH > 0 {
				destW, destH = imgW, imgH
			}
		}
		destX, destY := resolveBackgroundPosition(
			sty.BackgroundPosX, sty.BackgroundPosY, originX, originY, originW, originH, destW, destH,
		)

		baseOp := Op{ //nolint:exhaustruct // intentional zero fields
			Kind:         OpImage,
			X:            destX,
			Y:            destY,
			W:            destW,
			H:            destH,
			Image:        ref.data,
			ImgW:         ref.w,
			ImgH:         ref.h,
			IsJPEG:       ref.isJPEG,
			IsBackground: true,
			BlendMode:    backgroundBlendModeForLayer(sty.BackgroundBlendMode, i),
		}
		if sty.Filter != "" {
			filters := parseFilterList(sty.Filter, sty.Color, sty.FontSize)
			baseOp.Image = applyImageFilterToImage(baseOp.Image, filters)
		}

		dst = tileBackgroundRepeat(
			dst, baseOp, repeatX, repeatY, clip, destX, destY, destW, destH,
		)
	}
	// Clip every layer (including no-repeat / cover overflow) to background-clip.
	clipOpsSlice(dst[layerStart:], clip)

	return dst
}

// backgroundSizeForLayer picks the comma-separated background-size token for
// layer index (0 = first declared layer, topmost, per CSS Backgrounds).
func backgroundSizeForLayer(sizeSpec string, layerIndex int) string {
	parts := splitCommaLayers(sizeSpec)
	if len(parts) == 0 {
		return sizeSpec
	}
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0])
	}
	if layerIndex < 0 {
		layerIndex = 0
	}
	return strings.TrimSpace(parts[layerIndex%len(parts)])
}

func resolveBackgroundClipBox(
	sty ResolvedStyle, posX, posY, width, height float64,
) (float64, float64, float64, float64) {
	keyword := strings.ToLower(strings.TrimSpace(sty.BackgroundClip))
	if keyword == "" || keyword == "border-box" {
		return posX, posY, width, height
	}

	return backgroundBoxForKeyword(keyword, sty, posX, posY, width, height)
}

func resolveBackgroundOriginBox(
	sty ResolvedStyle, posX, posY, width, height float64,
) (float64, float64, float64, float64) {
	keyword := strings.ToLower(strings.TrimSpace(sty.BackgroundOrigin))

	return backgroundBoxForKeyword(keyword, sty, posX, posY, width, height)
}

func backgroundBoxForKeyword(
	keyword string, sty ResolvedStyle, posX, posY, width, height float64,
) (float64, float64, float64, float64) {
	switch keyword {
	case "content-box":
		x := posX + sty.BorderLeft.Width + sty.PaddingLeft
		y := posY + sty.BorderTop.Width + sty.PaddingTop
		w := width - sty.BorderLeft.Width - sty.BorderRight.Width - sty.PaddingLeft - sty.PaddingRight
		h := height - sty.BorderTop.Width - sty.BorderBottom.Width - sty.PaddingTop - sty.PaddingBottom

		return x, y, math.Max(0, w), math.Max(0, h)
	case "border-box":
		return posX, posY, width, height
	case "text":
		// background-clip:text clips to glyph bounds; PDF path requires vector
		// text clip which we approximate as content-box (conservative) since
		// no glyph vector clipping is implemented. Treat as content-box.
		x := posX + sty.BorderLeft.Width + sty.PaddingLeft
		y := posY + sty.BorderTop.Width + sty.PaddingTop
		w := width - sty.BorderLeft.Width - sty.BorderRight.Width - sty.PaddingLeft - sty.PaddingRight
		h := height - sty.BorderTop.Width - sty.BorderBottom.Width - sty.PaddingTop - sty.PaddingBottom

		return x, y, math.Max(0, w), math.Max(0, h)
	default: // padding-box (also the background-origin initial value)
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
		return imgW, imgH
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
		if pt, converted := lengthToPt(val, unit, 12); converted {
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
			if pt, converted := lengthToPt(val, unit, 12); converted {
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
			if pt, converted := lengthToPt(val, unit, 12); converted {
				offY = pt
			}
		}
	}

	return originX + offX, originY + offY
}

// background-attachment:fixed is intentionally a no-op in paginated PDF
// output (no viewport scroll); it paints as scroll. No paint change needed
// beyond documentation here; tileBackgroundRepeat branches remain correct.
//
// Tiling is anchored on the positioned tile (destX/destY, resolved against
// the background-origin positioning area) and covers the background-clip
// painting area. The origin box is never a clip bound: a no-repeat image may
// overflow the positioning area and paint out to the clip edge.
func tileBackgroundRepeat(
	dst []Op, baseOp Op, repeatX, repeatY string,
	clip clipRect, destX, destY, destW, destH float64,
) []Op {
	switch {
	case repeatX == backgroundRepeatNoRepeat && repeatY == backgroundRepeatNoRepeat:
		return appendBackgroundTileNoRepeat(dst, baseOp, clip)
	case repeatX == backgroundRepeatRepeat && repeatY == backgroundRepeatNoRepeat:
		return appendBackgroundTileRepeatX(dst, baseOp, clip, destX, destY, destW)
	case repeatX == backgroundRepeatNoRepeat && repeatY == backgroundRepeatRepeat:
		return appendBackgroundTileRepeatY(dst, baseOp, clip, destX, destY, destH)
	case repeatX == backgroundRepeatRepeat && repeatY == backgroundRepeatRepeat:
		return appendBackgroundTileRepeat(dst, baseOp, clip, destX, destY, destW, destH)
	default:
		dst = append(dst, baseOp)

		return dst
	}
}

func appendBackgroundTileNoRepeat(dst []Op, baseOp Op, clip clipRect) []Op {
	op := baseOp
	clipOpToPaintArea(&op, clip)

	if op.W > 0 && op.H > 0 {
		dst = append(dst, op)
	}

	return dst
}

func appendBackgroundTileRepeatX(dst []Op, baseOp Op, clip clipRect, destX, destY, destW float64) []Op {
	if destW <= 0 {
		dst = append(dst, baseOp)

		return dst
	}

	end := clip.x + clip.w

	for x, n := tileCoverStart(destX, clip.x, destW), 0; x < end && n < maxBackgroundTiles; x, n = x+destW, n+1 {
		op := baseOp
		op.X = x
		op.Y = destY
		clipOpToPaintArea(&op, clip)

		if op.W > 0 && op.H > 0 {
			dst = append(dst, op)
		}
	}

	return dst
}

func appendBackgroundTileRepeatY(dst []Op, baseOp Op, clip clipRect, destX, destY, destH float64) []Op {
	if destH <= 0 {
		dst = append(dst, baseOp)

		return dst
	}

	end := clip.y + clip.h

	for y, n := tileCoverStart(destY, clip.y, destH), 0; y < end && n < maxBackgroundTiles; y, n = y+destH, n+1 {
		op := baseOp
		op.X = destX
		op.Y = y
		clipOpToPaintArea(&op, clip)

		if op.W > 0 && op.H > 0 {
			dst = append(dst, op)
		}
	}

	return dst
}

func appendBackgroundTileRepeat(dst []Op, baseOp Op, clip clipRect, destX, destY, destW, destH float64) []Op {
	if destW <= 0 || destH <= 0 {
		dst = append(dst, baseOp)

		return dst
	}

	endX := clip.x + clip.w
	endY := clip.y + clip.h
	startX := tileCoverStart(destX, clip.x, destW)
	startY := tileCoverStart(destY, clip.y, destH)
	emitted := 0

	for y := startY; y < endY && emitted < maxBackgroundTiles; y += destH {
		for x := startX; x < endX && emitted < maxBackgroundTiles; x += destW {
			op := baseOp
			op.X = x
			op.Y = y
			clipOpToPaintArea(&op, clip)

			if op.W > 0 && op.H > 0 {
				dst = append(dst, op)
				emitted++
			}
		}
	}

	return dst
}

// tileCoverStart returns the first tile position at or before coverStart on
// the grid fixed by the positioned tile at pos, so repeat runs fill the
// painting area on both sides of the positioned tile.
func tileCoverStart(pos, coverStart, tile float64) float64 {
	return pos - math.Ceil((pos-coverStart)/tile)*tile
}

func clipOpToPaintArea(op *Op, clip clipRect) {
	if op.X < clip.x {
		op.W -= clip.x - op.X
		op.X = clip.x
	}

	if op.Y < clip.y {
		op.H -= clip.y - op.Y
		op.Y = clip.y
	}

	if op.X+op.W > clip.x+clip.w {
		op.W = clip.x + clip.w - op.X
	}

	if op.Y+op.H > clip.y+clip.h {
		op.H = clip.y + clip.h - op.Y
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
// quotes or parentheses. Delegates to splitParenArgs helper.
func splitCommaLayers(raw string) []string {
	return splitParenArgs(raw, ',')
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
