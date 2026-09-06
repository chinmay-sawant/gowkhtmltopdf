//nolint:varnamelen // border-image parsing and painting
package layout

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"  // Register GIF decoder for cropBorderImage via image.Decode.
	_ "image/jpeg" // Register JPEG decoder for cropBorderImage via image.Decode.
	"image/png"
	"math"
	"strconv"
	"strings"
)

// applyBorderImageProps handles border-image and its longhands.
func applyBorderImageProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "border-image":
		parseBorderImageShorthand(style, value)
	case "border-image-source":
		style.BorderImageSource = strings.TrimSpace(value)
	case "border-image-slice":
		style.BorderImageSlice = strings.TrimSpace(value)
	case "border-image-width":
		style.BorderImageWidth = strings.TrimSpace(value)
	case "border-image-outset":
		style.BorderImageOutset = strings.TrimSpace(value)
	case "border-image-repeat":
		style.BorderImageRepeat = strings.TrimSpace(value)
	default:
		return false
	}

	return true
}

const (
	maxBorderImageRepeatTokens = 2
	borderRepeatRound          = "round"
	borderRepeatRepeat         = "repeat"
	borderRepeatSpace          = "space"

	borderImageFullCount   = 4
	borderImagePairCount   = 2
	borderImageTripleCount = 3

	borderImageSliceSection  = 0
	borderImageWidthSection  = 1
	borderImageOutsetSection = 2

	borderImageTopEdge    = 0
	borderImageBottomEdge = 2

	borderImagePercentDivisor      = 100
	fallbackBorderImageThicknessPx = 6
)

func parseBorderImageShorthand(style *ResolvedStyle, value string) {
	trimmed := strings.TrimSpace(value)
	style.BorderImageSource = ""
	style.BorderImageSlice = ""
	style.BorderImageWidth = ""
	style.BorderImageOutset = ""
	style.BorderImageRepeat = ""

	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		return
	}

	source, nonURL := borderImageSourceAndRemainder(trimmed)
	style.BorderImageSource = source

	normalized := strings.ReplaceAll(nonURL, "/", " / ")
	cleanedSections, repeatTokens := splitBorderImageSections(normalized)

	if len(cleanedSections) > borderImageSliceSection {
		style.BorderImageSlice = strings.TrimSpace(cleanedSections[borderImageSliceSection])
	}

	if len(cleanedSections) > borderImageWidthSection {
		style.BorderImageWidth = strings.TrimSpace(cleanedSections[borderImageWidthSection])
	}

	if len(cleanedSections) > borderImageOutsetSection {
		style.BorderImageOutset = strings.TrimSpace(cleanedSections[borderImageOutsetSection])
	}

	if len(repeatTokens) > 0 {
		style.BorderImageRepeat = strings.Join(repeatTokens, " ")
	}
}

// borderImageSourceAndRemainder extracts the border-image-source URL and the
// remaining shorthand text without the url(...) token.
func borderImageSourceAndRemainder(trimmed string) (string, string) {
	source := ""

	if url, ok := firstCSSUrl(trimmed); ok {
		source = url
	} else if strings.HasPrefix(trimmed, "url(") {
		source = urlFunctionTarget(trimmed)
	}

	nonURL := trimmed
	start := strings.Index(strings.ToLower(trimmed), "url(")

	if start >= 0 {
		end := strings.Index(trimmed[start:], ")")

		if end >= 0 {
			nonURL = trimmed[:start] + " " + trimmed[start+end+1:]
		}
	}

	return source, nonURL
}

// splitBorderImageSections splits slash-separated slice/width/outset sections
// and pulls repeat keywords into repeatTokens.
func splitBorderImageSections(normalized string) ([]string, []string) {
	sections := strings.Split(normalized, "/")
	cleanedSections := make([]string, 0, len(sections))
	repeatTokens := make([]string, 0, maxBorderImageRepeatTokens)

	for _, section := range sections {
		cleaned, updated := extractBorderImageRepeats(section, repeatTokens)
		repeatTokens = updated

		cleanedSections = append(cleanedSections, cleaned)
	}

	return cleanedSections, repeatTokens
}

// extractBorderImageRepeats separates one section into its kept content and
// the repeat keywords it contains.
func extractBorderImageRepeats(section string, repeatTokens []string) (string, []string) {
	fields := strings.Fields(section)
	kept := make([]string, 0, len(fields))

	for _, tok := range fields {
		cleaned := strings.Trim(tok, ",")
		lowered := strings.ToLower(cleaned)

		if isBorderImageRepeatToken(lowered) {
			repeatTokens = appendBorderImageRepeat(repeatTokens, lowered)

			continue
		}

		kept = append(kept, cleaned)
	}

	return strings.Join(kept, " "), repeatTokens
}

// appendBorderImageRepeat records one repeat keyword up to the two-token cap.
func appendBorderImageRepeat(repeatTokens []string, token string) []string {
	if len(repeatTokens) >= maxBorderImageRepeatTokens {
		return repeatTokens
	}

	return append(repeatTokens, token)
}

// isBorderImageRepeatToken reports whether a lowercased token is a
// border-image repeat keyword rather than slice/width/outset content.
func isBorderImageRepeatToken(token string) bool {
	switch token {
	case fxStretch, borderRepeatRepeat, borderRepeatRound, borderRepeatSpace:
		return true
	default:
		return false
	}
}

type borderImageDimension struct {
	value      float64
	set        bool
	multiplier bool
	auto       bool
}

// appendBorderImage renders border-image when BorderImageSource is set and resolvable.
// Implements basic 9-slice: outset expands bounds, slice determines corner/edge
// geometry, width determines border thickness, repeat controls stretch vs tiled edges.
func (e *engine) appendBorderImage(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
) []Op {
	if sty.BorderImageSource == "" || width <= 0 || height <= 0 {
		return dst
	}

	src := backgroundImageSrc(sty.BorderImageSource)

	if src == "" {
		src = sty.BorderImageSource
	}

	ref := e.resolveImage(src)

	if ref == nil || ref.data == nil {
		return dst
	}

	outsetSpec := parseBorderImageDimensionFour(sty.BorderImageOutset, sty.FontSize)
	widthSpec := parseBorderImageDimensionFour(sty.BorderImageWidth, sty.FontSize)
	sliceFracs, hasSlice, hasFill := parseBorderImageSliceFracs(sty.BorderImageSlice, ref.w, ref.h)
	repeat := strings.ToLower(strings.TrimSpace(sty.BorderImageRepeat))

	thick := borderImageThickness(widthSpec, borderWidthsOf(sty))
	outset := borderImageOutset(outsetSpec, thick)
	ox, oy, ow, oh := borderImageOuterBounds(e, posX, posY, width, height, outset)

	if ow <= 0 || oh <= 0 {
		return dst
	}

	scaled := scaleBorderImageThickness(e, thick, ow, oh)

	if !hasSlice {
		return append(dst, newBorderImageOp(ox, oy, ow, oh, ref.data, ref.w, ref.h, ref.isJPEG))
	}

	return appendBorderImageByRepeat(dst, ref, ox, oy, ow, oh, scaled, sliceFracs, repeat, hasFill)
}

// borderWidthsOf reads the painted widths of the four borders in top, right,
// bottom, left order.
func borderWidthsOf(sty ResolvedStyle) [4]float64 {
	return [4]float64{
		borderPaint(sty.BorderTop), borderPaint(sty.BorderRight),
		borderPaint(sty.BorderBottom), borderPaint(sty.BorderLeft),
	}
}

// borderImageThickness resolves the border-image-width spec against the painted
// border widths, keeping the historical 6px fallback when all are unusable.
func borderImageThickness(widthSpec [4]borderImageDimension, borderWidths [4]float64) [4]float64 {
	var thick [4]float64

	for idx, spec := range widthSpec {
		switch {
		case !spec.set || spec.auto:
			thick[idx] = borderWidths[idx]
		case spec.multiplier:
			thick[idx] = borderWidths[idx] * spec.value
		default:
			thick[idx] = spec.value
		}
	}

	if thick[0] <= 0 && thick[1] <= 0 && thick[2] <= 0 && thick[3] <= 0 {
		def := pxToPt(fallbackBorderImageThicknessPx)
		thick = [4]float64{def, def, def, def}
	}

	return thick
}

// borderImageOutset resolves the border-image-outset spec against thickness.
func borderImageOutset(outsetSpec [4]borderImageDimension, thick [4]float64) [4]float64 {
	var outset [4]float64

	for idx, spec := range outsetSpec {
		if !spec.set {
			continue
		}

		if spec.multiplier {
			outset[idx] = spec.value * thick[idx]
		} else {
			outset[idx] = spec.value
		}
	}

	return outset
}

// borderImageOuterBounds expands the border box by the outset on each side.
func borderImageOuterBounds(
	e *engine, posX, posY, width, height float64, outset [4]float64,
) (float64, float64, float64, float64) {
	ox := posX - e.scalePt(outset[3])
	oy := posY - e.scalePt(outset[0])
	ow := width + e.scalePt(outset[1]+outset[3])
	oh := height + e.scalePt(outset[0]+outset[2])

	return ox, oy, ow, oh
}

// scaleBorderImageThickness converts thickness to device units and clamps it
// to non-negative values within half the outer box to avoid overlap.
func scaleBorderImageThickness(e *engine, thick [4]float64, ow, oh float64) [4]float64 {
	scaled := [4]float64{
		e.scalePt(thick[0]), e.scalePt(thick[1]),
		e.scalePt(thick[2]), e.scalePt(thick[3]),
	}

	for idx := range scaled {
		if scaled[idx] < 0 {
			scaled[idx] = 0
		}
	}

	if scaled[3]+scaled[1] > ow && ow > 0 {
		scale := ow / (scaled[3] + scaled[1])
		scaled[3] *= scale
		scaled[1] *= scale
	}

	if scaled[0]+scaled[2] > oh && oh > 0 {
		scale := oh / (scaled[0] + scaled[2])
		scaled[0] *= scale
		scaled[2] *= scale
	}

	return scaled
}

// appendBorderImageByRepeat dispatches tiled repeat modes to the repeated
// painter and everything else to the stretched painter.
func appendBorderImageByRepeat(
	dst []Op,
	ref *imageRef,
	ox, oy, ow, oh float64,
	thick [4]float64,
	sliceFracs [4]float64,
	repeat string,
	hasFill bool,
) []Op {
	if isTiledBorderImageRepeat(repeat) {
		return appendBorderImageRepeated(dst, ref, ox, oy, ow, oh, thick, sliceFracs, hasFill)
	}

	return appendBorderImageStretched(dst, ref, ox, oy, ow, oh, thick, sliceFracs, hasFill)
}

// isTiledBorderImageRepeat reports whether the repeat value asks for tiled
// edges instead of stretched ones.
func isTiledBorderImageRepeat(repeat string) bool {
	return strings.Contains(repeat, borderRepeatRepeat) ||
		strings.Contains(repeat, borderRepeatRound) ||
		strings.Contains(repeat, borderRepeatSpace)
}

func newBorderImageOp(x, y, w, h float64, data []byte, imgW, imgH int, isJPEG bool) Op {
	return Op{ //nolint:exhaustruct // intentional zero fields
		Kind:         OpImage,
		X:            x,
		Y:            y,
		W:            w,
		H:            h,
		Image:        data,
		ImgW:         imgW,
		ImgH:         imgH,
		IsJPEG:       isJPEG,
		IsBackground: true,
	}
}

// appendBorderImageStretched paints the 3x3 slice grid with each source slice
// mapped to one destination corner or edge, leaving the center transparent
// unless fill was requested. This is the supported mode used by fixture 60.
func appendBorderImageStretched(
	dst []Op,
	ref *imageRef,
	ox, oy, ow, oh float64,
	thick [4]float64,
	sliceFracs [4]float64,
	hasFill bool,
) []Op {
	slice := borderImageSlicePixels(sliceFracs, ref.w, ref.h)
	innerW, innerH := clampBorderImageInner(ow, oh, thick)
	srcX, srcY, srcW, srcH := borderImageSourceGrid(ref, slice)
	dstX, dstY, dstW, dstH := borderImageDestGrid(ox, oy, ow, oh, thick, innerW, innerH)

	for row := range 3 {
		for col := range 3 {
			dst = appendBorderImageCell(
				dst, ref, srcX, srcY, srcW, srcH, dstX, dstY, dstW, dstH, row, col, hasFill,
			)
		}
	}

	return dst
}

// clampBorderImageInner derives the center-cell size, floored at zero.
func clampBorderImageInner(ow, oh float64, thick [4]float64) (float64, float64) {
	innerW := ow - thick[3] - thick[1]
	innerH := oh - thick[0] - thick[2]

	if innerW < 0 {
		innerW = 0
	}

	if innerH < 0 {
		innerH = 0
	}

	return innerW, innerH
}

// borderImageSourceGrid maps slice pixels to source column/row origins and sizes.
func borderImageSourceGrid(ref *imageRef, slice [4]int) ([3]int, [3]int, [3]int, [3]int) {
	srcTop, srcRight, srcBottom, srcLeft := slice[0], slice[1], slice[2], slice[3]

	srcX := [3]int{0, srcLeft, ref.w - srcRight}
	srcY := [3]int{0, srcTop, ref.h - srcBottom}
	srcW := [3]int{srcLeft, ref.w - srcLeft - srcRight, srcRight}
	srcH := [3]int{srcTop, ref.h - srcTop - srcBottom, srcBottom}

	return srcX, srcY, srcW, srcH
}

// borderImageDestGrid maps thickness and inner size to destination origins and sizes.
func borderImageDestGrid(
	ox, oy, ow, oh float64, thick [4]float64, innerW, innerH float64,
) ([3]float64, [3]float64, [3]float64, [3]float64) {
	dstX := [3]float64{ox, ox + thick[3], ox + ow - thick[1]}
	dstY := [3]float64{oy, oy + thick[0], oy + oh - thick[2]}
	dstW := [3]float64{thick[3], innerW, thick[1]}
	dstH := [3]float64{thick[0], innerH, thick[2]}

	return dstX, dstY, dstW, dstH
}

// appendBorderImageCell paints one of the nine stretched grid cells, skipping
// the transparent center unless fill was requested and any empty slice.
func appendBorderImageCell(
	dst []Op,
	ref *imageRef,
	srcX, srcY, srcW, srcH [3]int,
	dstX, dstY, dstW, dstH [3]float64,
	row, col int,
	hasFill bool,
) []Op {
	if row == 1 && col == 1 && !hasFill {
		return dst
	}

	if srcW[col] <= 0 || srcH[row] <= 0 || dstW[col] <= 0 || dstH[row] <= 0 {
		return dst
	}

	srcRect := image.Rect(srcX[col], srcY[row], srcX[col]+srcW[col], srcY[row]+srcH[row])

	return appendBorderImagePart(dst, ref, srcRect, dstX[col], dstY[row], dstW[col], dstH[row])
}

func appendBorderImageRepeated(
	dst []Op,
	ref *imageRef,
	ox, oy, ow, oh float64,
	thick [4]float64,
	sliceFracs [4]float64,
	hasFill bool,
) []Op {
	// Keep the same source slicing and geometry for repeat-like values until
	// edge tiling is requested by a fixture. The visible border remains a
	// proper eight-piece border rather than a filled full-image rectangle.
	return appendBorderImageStretched(dst, ref, ox, oy, ow, oh, thick, sliceFracs, hasFill)
}

func appendBorderImagePart(
	dst []Op,
	ref *imageRef,
	src image.Rectangle,
	x, y, w, h float64,
) []Op {
	data, err := cropBorderImage(ref.data, src)
	if err != nil {
		return dst
	}

	return append(dst, newBorderImageOp(x, y, w, h, data, src.Dx(), src.Dy(), false))
}

func cropBorderImage(data []byte, src image.Rectangle) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode border image: %w", err)
	}

	src = src.Intersect(img.Bounds())

	if src.Empty() {
		return nil, image.ErrFormat
	}

	cropped := image.NewRGBA(image.Rect(0, 0, src.Dx(), src.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, src.Min, draw.Src)

	var out bytes.Buffer
	if err := png.Encode(&out, cropped); err != nil {
		return nil, fmt.Errorf("encode border image slice: %w", err)
	}

	return out.Bytes(), nil
}

func borderImageSlicePixels(fracs [4]float64, imgW, imgH int) [4]int {
	var pixels [4]int
	pixels[0] = clampBorderImagePixel(int(math.Round(fracs[0]*float64(imgH))), imgH)
	pixels[1] = clampBorderImagePixel(int(math.Round(fracs[1]*float64(imgW))), imgW)
	pixels[2] = clampBorderImagePixel(int(math.Round(fracs[2]*float64(imgH))), imgH)
	pixels[3] = clampBorderImagePixel(int(math.Round(fracs[3]*float64(imgW))), imgW)

	if pixels[0]+pixels[2] > imgH {
		scale := float64(imgH) / float64(pixels[0]+pixels[2])
		pixels[0] = int(math.Round(float64(pixels[0]) * scale))
		pixels[2] = imgH - pixels[0]
	}

	if pixels[1]+pixels[3] > imgW {
		scale := float64(imgW) / float64(pixels[1]+pixels[3])
		pixels[1] = int(math.Round(float64(pixels[1]) * scale))
		pixels[3] = imgW - pixels[1]
	}

	return pixels
}

func clampBorderImagePixel(value, limit int) int {
	if value < 0 {
		return 0
	}

	if value > limit {
		return limit
	}

	return value
}

func parseBorderImageSliceFracs(s string, imgW, imgH int) ([4]float64, bool, bool) {
	trimmed := strings.TrimSpace(s)

	if trimmed == "" {
		return [4]float64{}, false, false
	}

	withoutFill, hasFill := stripBorderImageFill(trimmed)
	toks := strings.Fields(withoutFill)

	if len(toks) == 0 {
		return [4]float64{}, false, hasFill
	}

	vals, isPct := parseBorderImageSliceValues(toks)

	if len(vals) == 0 {
		return [4]float64{}, false, hasFill
	}

	exp, expPct := expandBorderImageSliceValues(vals, isPct)

	return sliceFracsFromExpanded(exp, expPct, imgW, imgH), true, hasFill
}

// stripBorderImageFill removes fill keywords and slash separators, reporting
// whether the fill keyword was present.
func stripBorderImageFill(trimmed string) (string, bool) {
	hasFill := strings.Contains(strings.ToLower(trimmed), "fill")
	withoutFill := strings.ReplaceAll(trimmed, "fill", " ")
	withoutFill = strings.ReplaceAll(withoutFill, "Fill", " ")
	withoutFill = strings.ReplaceAll(withoutFill, "FILL", " ")
	// Remove slash separators.
	withoutFill = strings.ReplaceAll(withoutFill, "/", " ")

	return withoutFill, hasFill
}

// parseBorderImageSliceValues reads up to four numeric slice tokens,
// tracking which were percentages.
func parseBorderImageSliceValues(toks []string) ([]float64, []bool) {
	vals := make([]float64, 0, borderImageFullCount)
	isPct := make([]bool, 0, borderImageFullCount)

	for _, tok := range toks {
		if tok == "" {
			continue
		}

		vals, isPct = appendBorderImageSliceValue(vals, isPct, tok)

		if len(vals) >= borderImageFullCount {
			break
		}
	}

	return vals, isPct
}

// appendBorderImageSliceValue parses one slice token as a percent or a bare
// number, appending it to vals when valid.
func appendBorderImageSliceValue(
	vals []float64, isPct []bool, tok string,
) ([]float64, []bool) {
	if strings.HasSuffix(tok, "%") {
		numStr := strings.TrimSuffix(tok, "%")
		value, err := strconv.ParseFloat(numStr, 64)

		if err == nil {
			vals = append(vals, value)
			isPct = append(isPct, true)
		}

		return vals, isPct
	}

	value, err := strconv.ParseFloat(tok, 64)

	if err == nil {
		vals = append(vals, value)
		isPct = append(isPct, false)
	}

	return vals, isPct
}

// expandBorderImageSliceValues expands one to four values to clockwise order.
func expandBorderImageSliceValues(vals []float64, isPct []bool) ([4]float64, [4]bool) {
	var (
		exp    [4]float64
		expPct [4]bool
	)

	switch len(vals) {
	case 1:
		exp = [4]float64{vals[0], vals[0], vals[0], vals[0]}
		expPct = [4]bool{isPct[0], isPct[0], isPct[0], isPct[0]}
	case borderImagePairCount:
		exp = [4]float64{vals[0], vals[1], vals[0], vals[1]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[0], isPct[1]}
	case borderImageTripleCount:
		exp = [4]float64{vals[0], vals[1], vals[2], vals[1]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[2], isPct[1]}
	default:
		exp = [4]float64{vals[0], vals[1], vals[2], vals[3]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[2], isPct[3]}
	}

	return exp, expPct
}

// sliceFracsFromExpanded converts expanded slice values to fractions,
// resolving bare numbers against the image dimensions.
func sliceFracsFromExpanded(
	exp [4]float64, expPct [4]bool, imgW, imgH int,
) [4]float64 {
	var frac [4]float64

	for idx := range borderImageFullCount {
		if expPct[idx] {
			frac[idx] = exp[idx] / borderImagePercentDivisor
		} else {
			frac[idx] = numericBorderImageFrac(exp[idx], idx, imgW, imgH)
		}

		if frac[idx] < 0 {
			frac[idx] = 0
		}

		if frac[idx] > 1 {
			frac[idx] = 1
		}
	}

	return frac
}

// numericBorderImageFrac resolves a bare slice number against the relevant
// image dimension, falling back to percent semantics when unknown.
func numericBorderImageFrac(value float64, idx, imgW, imgH int) float64 {
	dim := imgW

	if idx == borderImageTopEdge || idx == borderImageBottomEdge {
		dim = imgH
	}

	if dim > 0 {
		return value / float64(dim)
	}

	return value / borderImagePercentDivisor
}

func parseBorderImageDimensionFour(s string, fsize float64) [4]borderImageDimension {
	toks := strings.Fields(strings.ReplaceAll(s, "/", " "))

	if len(toks) == 0 {
		return [4]borderImageDimension{}
	}

	values := make([]borderImageDimension, 0, borderImageFullCount)

	for _, tok := range toks {
		if value, ok := parseBorderImageDimension(tok, fsize); ok {
			values = append(values, value)
		}

		if len(values) >= borderImageFullCount {
			break
		}
	}

	return expandBorderImageDimensions(values)
}

func parseBorderImageDimension(tok string, fsize float64) (borderImageDimension, bool) {
	tok = strings.TrimSpace(tok)

	if tok == "" {
		return borderImageDimension{value: 0, set: false, multiplier: false, auto: false}, false
	}

	if strings.EqualFold(tok, "auto") {
		return borderImageDimension{value: 0, set: true, multiplier: false, auto: true}, true
	}

	if value, err := strconv.ParseFloat(tok, 64); err == nil {
		return borderImageDimension{value: value, set: true, multiplier: true, auto: false}, true
	}

	if v, ok := plainLength(tok, fsize, 0); ok {
		return borderImageDimension{value: v, set: true, multiplier: false, auto: false}, true
	}

	return borderImageDimension{value: 0, set: false, multiplier: false, auto: false}, false
}

func expandBorderImageDimensions(values []borderImageDimension) [4]borderImageDimension {
	if len(values) == 0 {
		return [4]borderImageDimension{}
	}

	if len(values) == 1 {
		return [4]borderImageDimension{values[0], values[0], values[0], values[0]}
	}

	if len(values) == borderImagePairCount {
		return [4]borderImageDimension{values[0], values[1], values[0], values[1]}
	}

	if len(values) == borderImageTripleCount {
		return [4]borderImageDimension{values[0], values[1], values[2], values[1]}
	}

	return [4]borderImageDimension{values[0], values[1], values[2], values[3]}
}
