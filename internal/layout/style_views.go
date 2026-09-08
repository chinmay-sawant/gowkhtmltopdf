package layout

// boxModelStyle is the layout-owned view of the resolved CSS box model.
// ResolvedStyle remains the cascade record, but sizing helpers no longer
// accept unrelated text, grid, paint, and resource state.
type boxModelStyle struct {
	boxSizing string

	width, widthPercent       float64
	minWidth, minWidthPercent float64
	maxWidth, maxWidthPercent float64

	height, heightPercent       float64
	minHeight, minHeightPercent float64
	maxHeight, maxHeightPercent float64

	marginLeft, marginRight         float64
	marginLeftAuto, marginRightAuto bool
	paddingTop, paddingRight        float64
	paddingBottom, paddingLeft      float64
	borderTop, borderRight          border
	borderBottom, borderLeft        border
	borderImageSource               string
}

func boxModelStyleOf(style *ResolvedStyle) boxModelStyle {
	if style == nil {
		return boxModelStyle{} //nolint:exhaustruct // empty view for absent style
	}

	return boxModelStyle{
		boxSizing:         style.BoxSizing,
		width:             style.Width,
		widthPercent:      style.WidthPercent,
		minWidth:          style.MinWidth,
		minWidthPercent:   style.MinWidthPercent,
		maxWidth:          style.MaxWidth,
		maxWidthPercent:   style.MaxWidthPercent,
		height:            style.Height,
		heightPercent:     style.HeightPercent,
		minHeight:         style.MinHeight,
		minHeightPercent:  style.MinHeightPercent,
		maxHeight:         style.MaxHeight,
		maxHeightPercent:  style.MaxHeightPercent,
		marginLeft:        style.MarginLeft,
		marginRight:       style.MarginRight,
		marginLeftAuto:    style.MarginLeftAuto,
		marginRightAuto:   style.MarginRightAuto,
		paddingTop:        style.PaddingTop,
		paddingRight:      style.PaddingRight,
		paddingBottom:     style.PaddingBottom,
		paddingLeft:       style.PaddingLeft,
		borderTop:         style.BorderTop,
		borderRight:       style.BorderRight,
		borderBottom:      style.BorderBottom,
		borderLeft:        style.BorderLeft,
		borderImageSource: style.BorderImageSource,
	}
}

func (style boxModelStyle) horizontalChrome(eng *engine) float64 {
	return eng.scalePt(style.paddingLeft) + eng.scalePt(style.paddingRight) +
		eng.scalePt(style.borderLeft.Width) + eng.scalePt(style.borderRight.Width)
}

func (style boxModelStyle) verticalChrome(eng *engine) float64 {
	return eng.scalePt(style.paddingTop) + eng.scalePt(style.paddingBottom) +
		eng.scalePt(style.borderTop.Width) + eng.scalePt(style.borderBottom.Width)
}
