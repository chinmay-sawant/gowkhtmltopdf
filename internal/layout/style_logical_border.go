//nolint:all
package layout

func logicalRadiusCorner(style *ResolvedStyle, corner string) (*float64, *float64) {
	switch corner {
	case "start-start":
		return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY
	case "start-end":
		return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY
	case "end-start":
		return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY
	case "end-end":
		return &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY
	default:
		return nil, nil
	}
}

// applyLogicalRadiusLonghand handles CSS logical corner radii and logical side radii.
func applyLogicalRadiusLonghand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-start-start-radius":
		dx, dy := logicalRadiusCorner(style, "start-start")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-start-end-radius":
		dx, dy := logicalRadiusCorner(style, "start-end")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-end-start-radius":
		dx, dy := logicalRadiusCorner(style, "end-start")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-end-end-radius":
		dx, dy := logicalRadiusCorner(style, "end-end")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-block-start-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-start")
		d2x, d2y := logicalRadiusCorner(style, "start-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-block-end-radius":
		d1x, d1y := logicalRadiusCorner(style, "end-start")
		d2x, d2y := logicalRadiusCorner(style, "end-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-inline-start-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-start")
		d2x, d2y := logicalRadiusCorner(style, "end-start")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-inline-end-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-end")
		d2x, d2y := logicalRadiusCorner(style, "end-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	default:
		return false
	}

	return true
}
