package layout

// paintChromeStyle is the small part of a resolved style needed by the
// post-pagination chrome repair pass. Keeping this view separate means that
// the pass does not depend on the cascade record's text, grid, or resource
// fields.
type paintChromeStyle struct {
	paddingBottom float64
	float         string
	position      string
	borderLeft    border
	borderRight   border
	borderTop     border
	borderBottom  border
}

func paintChromeStyleOf(boxNode *box) (paintChromeStyle, bool) {
	if boxNode == nil || boxNode.style == nil {
		return paintChromeStyle{}, false //nolint:exhaustruct // empty view for absent style
	}

	style := boxNode.style

	return paintChromeStyle{
		paddingBottom: style.PaddingBottom,
		float:         style.Float,
		position:      style.Position,
		borderLeft:    style.BorderLeft,
		borderRight:   style.BorderRight,
		borderTop:     style.BorderTop,
		borderBottom:  style.BorderBottom,
	}, true
}
