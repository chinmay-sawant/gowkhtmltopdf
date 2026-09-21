package layout

// withAspectRatioHeight returns a copy of style whose Height is filled from
// definiteWidthScaled / AspectRatio when height is auto. definiteWidthScaled
// is the content-box width for content-box sizing and the border-box width
// for border-box sizing (callers pass the matching axis).
func withAspectRatioHeight(style ResolvedStyle, definiteWidthScaled float64, eng *engine) ResolvedStyle {
	if h, ok := aspectRatioAutoHeightPt(style, definiteWidthScaled, eng); ok {
		style.Height = h
	}

	return style
}

// aspectRatioAutoHeightPt returns the unscaled CSS pt height implied by
// aspect-ratio when height is auto and width is definite.
func aspectRatioAutoHeightPt(style ResolvedStyle, definiteWidthScaled float64, eng *engine) (float64, bool) {
	if style.AspectRatio <= 0 || definiteWidthScaled <= 0 {
		return 0, false
	}

	if style.Height >= 0 || style.HeightPercent >= 0 {
		return 0, false
	}

	scale := 1.0
	if eng != nil && eng.scale > 0 {
		scale = eng.scale
	}

	return (definiteWidthScaled / style.AspectRatio) / scale, true
}

// aspectRatioAutoWidthPt returns the unscaled CSS pt width implied by
// aspect-ratio when width is auto and height is definite.
func aspectRatioAutoWidthPt(style ResolvedStyle, definiteHeightScaled float64, eng *engine) (float64, bool) {
	if style.AspectRatio <= 0 || definiteHeightScaled <= 0 {
		return 0, false
	}

	if style.Width >= 0 || style.WidthPercent >= 0 {
		return 0, false
	}

	scale := 1.0
	if eng != nil && eng.scale > 0 {
		scale = eng.scale
	}

	return (definiteHeightScaled * style.AspectRatio) / scale, true
}
