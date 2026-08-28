//nolint:varnamelen // border-image parsing and painting
package layout

import (
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

func parseBorderImageShorthand(style *ResolvedStyle, value string) {
	trimmed := strings.TrimSpace(value)
	if url, ok := firstCSSUrl(trimmed); ok {
		style.BorderImageSource = url
	} else if strings.HasPrefix(trimmed, "url(") {
		style.BorderImageSource = urlFunctionTarget(trimmed)
	} else {
		style.BorderImageSource = trimmed
	}
}

// appendBorderImage renders border-image when BorderImageSource is set and resolvable.
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

	return append(dst, op)
}
