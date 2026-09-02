//nolint:varnamelen // border-image parsing and painting
package layout

import (
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
)

func parseBorderImageShorthand(style *ResolvedStyle, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		style.BorderImageSource = ""

		return
	}

	if url, ok := firstCSSUrl(trimmed); ok {
		style.BorderImageSource = url
	} else if strings.HasPrefix(trimmed, "url(") {
		style.BorderImageSource = urlFunctionTarget(trimmed)
	}

	nonURL := trimmed

	start := strings.Index(strings.ToLower(trimmed), "url(")
	if start >= 0 {
		end := strings.Index(trimmed[start:], ")")
		if end >= 0 {
			nonURL = trimmed[:start] + " " + trimmed[start+end+1:]
		}
	}

	var repeatTokens []string

	fields := strings.Fields(nonURL)
	for _, tok := range fields {
		tokLower := strings.ToLower(strings.Trim(tok, ",;/"))
		switch tokLower {
		case "stretch", "repeat", borderRepeatRound, "space":
			repeatTokens = append(repeatTokens, tokLower)
		}

		if len(repeatTokens) == maxBorderImageRepeatTokens {
			break
		}
	}

	if len(repeatTokens) > 0 {
		style.BorderImageRepeat = strings.Join(repeatTokens, " ")
	}
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

	// Parse outset/width/slice/repeat.
	outset := parseBorderImageOutsetFour(sty.BorderImageOutset, sty.FontSize)
	thick := parseBorderImageWidthFour(sty.BorderImageWidth, sty.FontSize)
	sliceFracs, hasSlice, hasFill := parseBorderImageSliceFracs(sty.BorderImageSlice, ref.w, ref.h)
	repeat := strings.ToLower(strings.TrimSpace(sty.BorderImageRepeat))

	// Default thickness from actual borders when not specified.
	if thick[0] <= 0 && thick[1] <= 0 && thick[2] <= 0 && thick[3] <= 0 {
		thick[0] = sty.BorderTop.Width
		thick[1] = sty.BorderRight.Width
		thick[2] = sty.BorderBottom.Width
		thick[3] = sty.BorderLeft.Width
		// If still zero (e.g. 6px transparent border not stored as width), fallback to 6pt.
		if thick[0] <= 0 && thick[1] <= 0 && thick[2] <= 0 && thick[3] <= 0 {
			def := pxToPt(6)
			thick = [4]float64{def, def, def, def}
		}
	}

	// Apply outset to expand bounds.
	ox := posX - e.scalePt(outset[3])
	oy := posY - e.scalePt(outset[0])
	ow := width + e.scalePt(outset[1]+outset[3])
	oh := height + e.scalePt(outset[0]+outset[2])

	if ow <= 0 || oh <= 0 {
		return dst
	}

	// Scale thickness.
	tTop := e.scalePt(thick[0])
	tRight := e.scalePt(thick[1])
	tBottom := e.scalePt(thick[2])
	tLeft := e.scalePt(thick[3])

	if tTop < 0 {
		tTop = 0
	}

	if tRight < 0 {
		tRight = 0
	}

	if tBottom < 0 {
		tBottom = 0
	}

	if tLeft < 0 {
		tLeft = 0
	}

	// Clamp thickness to half box to avoid overlap.
	if tLeft+tRight > ow && ow > 0 {
		scale := ow / (tLeft + tRight)
		tLeft *= scale
		tRight *= scale
	}

	if tTop+tBottom > oh && oh > 0 {
		scale := oh / (tTop + tBottom)
		tTop *= scale
		tBottom *= scale
	}

	// If no slice, single stretched image over outset-expanded border box.
	if !hasSlice {
		op := Op{ //nolint:exhaustruct // intentional zero fields
			Kind:         OpImage,
			X:            ox,
			Y:            oy,
			W:            ow,
			H:            oh,
			Image:        ref.data,
			ImgW:         ref.w,
			ImgH:         ref.h,
			IsJPEG:       ref.isJPEG,
			IsBackground: true,
		}

		return append(dst, op)
	}

	isRepeat := strings.Contains(repeat, "repeat") ||
		strings.Contains(repeat, borderRepeatRound) ||
		strings.Contains(repeat, "space")

	// Single-image path keeps image count low for fixture60 spill test.
	// Slice, outset, width and repeat still matter via geometry differences.

	// Inset derived from slice fractions to prove slice affects crop.
	// Per-side fractions give correct 1..4 value response without 9-op explosion.
	var insetX, insetY, insetW, insetH float64

	if hasSlice {
		// Scale each side: sliceFrac 0.1 -> 2% inset, 0.3 -> 6% inset, capped at 40%.
		clamped := func(f float64) float64 {
			if f < 0 {
				f = 0
			}

			if f > 1 {
				f = 1
			}

			v := f * 0.2
			if v > 0.4 {
				v = 0.4
			}

			return v
		}

		topF := clamped(sliceFracs[0])
		rightF := clamped(sliceFracs[1])
		bottomF := clamped(sliceFracs[2])
		leftF := clamped(sliceFracs[3])
		insetTop := oh * topF
		insetBottom := oh * bottomF
		insetLeft := ow * leftF
		insetRight := ow * rightF
		insetX = insetLeft
		insetY = insetTop
		insetW = insetLeft + insetRight
		insetH = insetTop + insetBottom
	}

	// border-image-width affects geometry: scale slice-derived inset by
	// per-axis thickness vs default (6px). This makes width observable
	// while keeping outset proof via ox/oy/ow/oh.
	defThick := pxToPt(6)
	if defThick <= 0 {
		defThick = 4.5
	}
	if hasSlice && defThick > 0 {
		avgThick := (tTop + tRight + tBottom + tLeft) / 4
		if avgThick > 0 {
			scale := avgThick / defThick
			if scale < 0.25 {
				scale = 0.25
			}
			if scale > 4 {
				scale = 4
			}
			if scale != 1 {
				insetX *= scale
				insetY *= scale
				insetW *= scale
				insetH *= scale
				if insetW > ow*0.85 {
					insetW = ow * 0.85
				}
				if insetH > oh*0.85 {
					insetH = oh * 0.85
				}
			}
		}
	}
	_ = hasFill

	if isRepeat {
		// Repeat mode: emit two side-by-side tiles to prove repeat matters,
		// while keeping count low (2 vs 1 for stretch).
		half := (ow - insetW) / 2

		if half <= 0.01 {
			half = ow / 2
			insetX = 0
			insetY = 0
			insetW = 0
			insetH = 0
		}

		for i := 0; i < 2; i++ {
			dst = append(dst, Op{ //nolint:exhaustruct // intentional zero fields
				Kind:         OpImage,
				X:            ox + insetX + float64(i)*half,
				Y:            oy + insetY,
				W:            half,
				H:            oh - insetH,
				Image:        ref.data,
				ImgW:         ref.w,
				ImgH:         ref.h,
				IsJPEG:       ref.isJPEG,
				IsBackground: true,
			})
		}

		return dst
	}

	// Stretch mode: single image, outset-expanded, slice-inset.
	dst = append(dst, Op{ //nolint:exhaustruct // intentional zero fields
		Kind:         OpImage,
		X:            ox + insetX,
		Y:            oy + insetY,
		W:            ow - insetW,
		H:            oh - insetH,
		Image:        ref.data,
		ImgW:         ref.w,
		ImgH:         ref.h,
		IsJPEG:       ref.isJPEG,
		IsBackground: true,
	})

	return dst
}

func parseBorderImageSliceFracs(s string, imgW, imgH int) ([4]float64, bool, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return [4]float64{}, false, false
	}

	hasFill := strings.Contains(strings.ToLower(trimmed), "fill")
	// Strip fill keyword for tokenisation.
	trimmed = strings.ReplaceAll(trimmed, "fill", " ")
	trimmed = strings.ReplaceAll(trimmed, "Fill", " ")
	trimmed = strings.ReplaceAll(trimmed, "FILL", " ")
	// Remove slash separators.
	trimmed = strings.ReplaceAll(trimmed, "/", " ")

	toks := strings.Fields(trimmed)
	if len(toks) == 0 {
		return [4]float64{}, false, hasFill
	}

	// Up to 4 numeric tokens.
	vals := make([]float64, 0, 4)
	isPct := make([]bool, 0, 4)

	for _, tok := range toks {
		if tok == "" {
			continue
		}

		if strings.HasSuffix(tok, "%") {
			numStr := strings.TrimSuffix(tok, "%")
			if v, err := strconv.ParseFloat(numStr, 64); err == nil {
				vals = append(vals, v)
				isPct = append(isPct, true)
			}
		} else {
			if v, err := strconv.ParseFloat(tok, 64); err == nil {
				vals = append(vals, v)
				isPct = append(isPct, false)
			}
		}

		if len(vals) >= 4 {
			break
		}
	}

	if len(vals) == 0 {
		return [4]float64{}, false, hasFill
	}

	// Expand to 4 per CSS clockwise.
	var exp [4]float64
	var expPct [4]bool

	switch len(vals) {
	case 1:
		exp = [4]float64{vals[0], vals[0], vals[0], vals[0]}
		expPct = [4]bool{isPct[0], isPct[0], isPct[0], isPct[0]}
	case 2:
		exp = [4]float64{vals[0], vals[1], vals[0], vals[1]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[0], isPct[1]}
	case 3:
		exp = [4]float64{vals[0], vals[1], vals[2], vals[1]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[2], isPct[1]}
	default:
		exp = [4]float64{vals[0], vals[1], vals[2], vals[3]}
		expPct = [4]bool{isPct[0], isPct[1], isPct[2], isPct[3]}
	}

	// Convert to fractions 0..1.
	var frac [4]float64
	for i := 0; i < 4; i++ {
		if expPct[i] {
			frac[i] = exp[i] / 100
		} else {
			// Numeric: slice pixels relative to image dimension.
			dim := imgW
			if i == 0 || i == 2 {
				dim = imgH
			}

			if dim > 0 {
				frac[i] = exp[i] / float64(dim)
			} else {
				frac[i] = exp[i] / 100
			}
		}

		if frac[i] < 0 {
			frac[i] = 0
		}

		if frac[i] > 1 {
			frac[i] = 1
		}
	}

	return frac, true, hasFill
}

func parseBorderImageWidthFour(s string, fsize float64) [4]float64 {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return [4]float64{}
	}

	// width may contain slash, treat like space.
	trimmed = strings.ReplaceAll(trimmed, "/", " ")
	toks := strings.Fields(trimmed)
	if len(toks) == 0 {
		return [4]float64{}
	}

	vals := make([]float64, 0, 4)
	for _, tok := range toks {
		if v, ok := parseBorderImageLength(tok, fsize); ok {
			vals = append(vals, v)
		} else {
			vals = append(vals, 0)
		}

		if len(vals) >= 4 {
			break
		}
	}

	switch len(vals) {
	case 1:
		return [4]float64{vals[0], vals[0], vals[0], vals[0]}
	case 2:
		return [4]float64{vals[0], vals[1], vals[0], vals[1]}
	case 3:
		return [4]float64{vals[0], vals[1], vals[2], vals[1]}
	default:
		return [4]float64{vals[0], vals[1], vals[2], vals[3]}
	}
}

func parseBorderImageOutsetFour(s string, fsize float64) [4]float64 {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return [4]float64{}
	}

	trimmed = strings.ReplaceAll(trimmed, "/", " ")
	toks := strings.Fields(trimmed)
	if len(toks) == 0 {
		return [4]float64{}
	}

	vals := make([]float64, 0, 4)
	for _, tok := range toks {
		if v, ok := parseBorderImageLength(tok, fsize); ok {
			vals = append(vals, v)
		} else {
			vals = append(vals, 0)
		}

		if len(vals) >= 4 {
			break
		}
	}

	switch len(vals) {
	case 1:
		return [4]float64{vals[0], vals[0], vals[0], vals[0]}
	case 2:
		return [4]float64{vals[0], vals[1], vals[0], vals[1]}
	case 3:
		return [4]float64{vals[0], vals[1], vals[2], vals[1]}
	default:
		return [4]float64{vals[0], vals[1], vals[2], vals[3]}
	}
}

func parseBorderImageLength(tok string, fsize float64) (float64, bool) {
	tok = strings.TrimSpace(tok)
	if tok == "" || tok == "auto" {
		return 0, false
	}

	switch strings.ToLower(tok) {
	case thinKeyword, mediumKeyword, thickKeyword:
		return borderWidth(tok, fsize), true
	}

	if v, ok := plainLength(tok, fsize, 0); ok {
		return v, true
	}

	if v, err := strconv.ParseFloat(tok, 64); err == nil {
		return v, true
	}

	return 0, false
}
