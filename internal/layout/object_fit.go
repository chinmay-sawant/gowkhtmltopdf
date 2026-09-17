package layout

import "strings"

func replacedIntrinsicPt(e *engine, sty ResolvedStyle, ref *imageRef) (float64, float64) {
	if ref == nil || ref.w <= 0 || ref.h <= 0 {
		return 0, 0
	}

	scale := imageResolutionScale(sty, ref)
	w := pxToPt(float64(ref.w) * scale)
	h := pxToPt(float64(ref.h) * scale)
	if e != nil {
		return e.scalePt(w), e.scalePt(h)
	}

	return w, h
}

// objectFitPaintRect sizes and positions the replaced image inside its content
// box per object-fit / object-position. Returns the paint rect and whether the
// image must be clipped to the content box (cover / none / scale-down overflow).
func objectFitPaintRect(
	fit string, posX, posY string,
	boxX, boxY, boxW, boxH, intrinsicW, intrinsicH float64,
) (imgX, imgY, imgW, imgH float64, clip bool) {
	fit = strings.ToLower(strings.TrimSpace(fit))
	if fit == "" {
		fit = "fill"
	}

	imgW, imgH = objectFitSize(fit, boxW, boxH, intrinsicW, intrinsicH)
	imgX, imgY = resolveBackgroundPosition(posX, posY, boxX, boxY, boxW, boxH, imgW, imgH)
	clip = fit == "cover" || fit == "none" || fit == "scale-down"

	return imgX, imgY, imgW, imgH, clip && (imgW > boxW+0.01 || imgH > boxH+0.01 ||
		imgX < boxX-0.01 || imgY < boxY-0.01 ||
		imgX+imgW > boxX+boxW+0.01 || imgY+imgH > boxY+boxH+0.01)
}

func objectFitSize(fit string, boxW, boxH, intrinsicW, intrinsicH float64) (float64, float64) {
	switch fit {
	case "contain":
		return objectFitScale(boxW, boxH, intrinsicW, intrinsicH, false)
	case "cover":
		return objectFitScale(boxW, boxH, intrinsicW, intrinsicH, true)
	case "none":
		if intrinsicW > 0 && intrinsicH > 0 {
			return intrinsicW, intrinsicH
		}

		return boxW, boxH
	case "scale-down":
		noneW, noneH := intrinsicW, intrinsicH
		if noneW <= 0 || noneH <= 0 {
			return boxW, boxH
		}

		contW, contH := objectFitScale(boxW, boxH, intrinsicW, intrinsicH, false)
		if noneW <= contW && noneH <= contH {
			return noneW, noneH
		}

		return contW, contH
	default: // fill
		return boxW, boxH
	}
}

// applyObjectFitToPaint adjusts a content-box paint rect for object-fit /
// object-position given the image's CSS-px intrinsic size already scaled to pt.
func applyObjectFitToPaint(
	sty ResolvedStyle, boxX, boxY, boxW, boxH, intrinsicW, intrinsicH float64,
) (float64, float64, float64, float64) {
	fitX, fitY, fitW, fitH, _ := objectFitPaintRect(
		sty.ObjectFit, sty.ObjectPositionX, sty.ObjectPositionY,
		boxX, boxY, boxW, boxH, intrinsicW, intrinsicH,
	)

	return fitX, fitY, fitW, fitH
}

func objectFitScale(boxW, boxH, intrinsicW, intrinsicH float64, cover bool) (float64, float64) {
	if intrinsicW <= 0 || intrinsicH <= 0 || boxW <= 0 || boxH <= 0 {
		return boxW, boxH
	}

	scaleX := boxW / intrinsicW
	scaleY := boxH / intrinsicH
	scale := scaleX

	if cover {
		if scaleY > scaleX {
			scale = scaleY
		}
	} else if scaleY < scaleX {
		scale = scaleY
	}

	return intrinsicW * scale, intrinsicH * scale
}
