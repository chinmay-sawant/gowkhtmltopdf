//nolint:cyclop,exhaustruct,mnd,varnamelen // CSS shape-outside float exclusion geometry
package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// shapeExclusion is one float's shape-outside contour in canvas coordinates.
// lineBounds / floatState.exclusion consult it for per-line intervals.
type shapeExclusion struct {
	kind shapeKind
	// ellipse / circle center and radii (radii already include shape-margin).
	cx, cy, rx, ry float64
	// inset rectangle edges (already expanded by shape-margin).
	x0, y0, x1, y1 float64
}

type shapeKind uint8

const (
	shapeKindNone    shapeKind = iota
	shapeKindEllipse           // circle is an ellipse with rx==ry
	shapeKindInset
	shapeEllipseKeyword      = "ellipse"
	shapeClosestSideKeyword  = "closest-side"
	shapeFarthestSideKeyword = "farthest-side"
)

// rightEdgeAt returns the rightmost x of the shape at canvas y, if the
// horizontal ray intersects the contour.
func (s *shapeExclusion) rightEdgeAt(y float64) (float64, bool) {
	left, right, ok := s.intervalAt(y)
	if !ok {
		return 0, false
	}

	_ = left

	return right, true
}

// leftEdgeAt returns the leftmost x of the shape at canvas y.
func (s *shapeExclusion) leftEdgeAt(y float64) (float64, bool) {
	left, right, ok := s.intervalAt(y)
	if !ok {
		return 0, false
	}

	_ = right

	return left, true
}

func (s *shapeExclusion) intervalAt(y float64) (float64, float64, bool) {
	if s == nil {
		return 0, 0, false
	}

	switch s.kind {
	case shapeKindNone:
		return 0, 0, false
	case shapeKindEllipse:
		return ellipseInterval(s.cx, s.cy, s.rx, s.ry, y)
	case shapeKindInset:
		if y < s.y0 || y > s.y1 {
			return 0, 0, false
		}

		return s.x0, s.x1, true
	default:
		return 0, 0, false
	}
}

func ellipseInterval(cx, cy, rx, ry, y float64) (float64, float64, bool) {
	if rx <= 0 || ry <= 0 {
		return 0, 0, false
	}

	dy := y - cy
	if math.Abs(dy) > ry {
		return 0, 0, false
	}

	// x = cx ± rx * sqrt(1 - (dy/ry)^2)
	half := rx * math.Sqrt(math.Max(0, 1-(dy*dy)/(ry*ry)))

	return cx - half, cx + half, true
}

// buildShapeExclusion resolves sty.ShapeOutside against the float's margin box.
// Returns nil when shape-outside is none / unsupported / unresolvable so the
// caller keeps rectangular exclusion.
func buildShapeExclusion(
	sty ResolvedStyle, fbox *box, margL, margR, scale float64,
) *shapeExclusion {
	raw := strings.TrimSpace(sty.ShapeOutside)
	if raw == "" || raw == shapeOutsideNone {
		return nil
	}

	shape, _, ok := splitShapeOutsideParts(normalizeCSSValue(raw))
	if !ok {
		// Canonical form without a box keyword is the whole string.
		shape = normalizeCSSValue(raw)
	}

	name, args, ok := splitShapeFunction(shape)
	if !ok {
		return nil
	}

	refX := fbox.x - margL
	refY := fbox.y
	refW := fbox.w + margL + margR
	refH := fbox.height

	if refW <= 0 || refH <= 0 {
		return nil
	}

	margin := resolveShapeMargin(sty, refW, refH, scale)

	switch name {
	case listStyleCircle:
		return resolveCircleExclusion(args, refX, refY, refW, refH, margin, sty.FontSize, scale)
	case shapeEllipseKeyword:
		return resolveEllipseExclusion(args, refX, refY, refW, refH, margin, sty.FontSize, scale)
	case "inset":
		return resolveInsetExclusion(args, refX, refY, refW, refH, margin, sty.FontSize, scale)
	default:
		return nil
	}
}

func resolveShapeMargin(sty ResolvedStyle, refW, refH, scale float64) float64 {
	if sty.ShapeMarginPercent >= 0 {
		return sty.ShapeMarginPercent / 100 * shapeReferenceDiagonal(refW, refH)
	}

	return sty.ShapeMargin * scale
}

func resolveCircleExclusion(
	args string, refX, refY, refW, refH, margin, fsize, scale float64,
) *shapeExclusion {
	radii, position, ok := splitShapeRadiusPosition(args, 1)
	if !ok {
		return nil
	}

	cx, cy := resolveShapeCenter(position, refX, refY, refW, refH, fsize, scale)
	r := closestSideRadius(cx, cy, refX, refY, refW, refH)

	if len(radii) == 1 {
		switch radii[0] {
		case shapeClosestSideKeyword:
			// keep default
		case shapeFarthestSideKeyword:
			r = farthestSideRadius(cx, cy, refX, refY, refW, refH)
		default:
			v, ok := resolveShapeRadius(radii[0], refW, refH, fsize, scale, true)
			if !ok {
				return nil
			}

			r = v
		}
	}

	r += margin
	if r <= 0 {
		return nil
	}

	return &shapeExclusion{kind: shapeKindEllipse, cx: cx, cy: cy, rx: r, ry: r}
}

func resolveEllipseExclusion(
	args string, refX, refY, refW, refH, margin, fsize, scale float64,
) *shapeExclusion {
	radii, position, ok := splitShapeRadiusPosition(args, 2)
	if !ok {
		return nil
	}

	cx, cy := resolveShapeCenter(position, refX, refY, refW, refH, fsize, scale)
	// closest-side defaults: distance to nearest horizontal / vertical edge.
	rx := minY(cx-refX, refX+refW-cx)
	ry := minY(cy-refY, refY+refH-cy)

	switch len(radii) {
	case 0:
		// defaults already set
	case 2:
		if radii[0] != shapeClosestSideKeyword && radii[0] != shapeFarthestSideKeyword {
			if v, ok := resolveShapeRadius(radii[0], refW, refH, fsize, scale, false); ok {
				rx = v
			} else {
				return nil
			}
		}

		if radii[1] != shapeClosestSideKeyword && radii[1] != shapeFarthestSideKeyword {
			if v, ok := resolveShapeRadius(radii[1], refH, refW, fsize, scale, false); ok {
				ry = v
			} else {
				return nil
			}
		}
	default:
		return nil
	}

	rx += margin
	ry += margin

	if rx <= 0 || ry <= 0 {
		return nil
	}

	return &shapeExclusion{kind: shapeKindEllipse, cx: cx, cy: cy, rx: rx, ry: ry}
}

func resolveInsetExclusion(
	args string, refX, refY, refW, refH, margin, fsize, scale float64,
) *shapeExclusion {
	main, _ := splitViewBoxRound(args)
	parts := strings.Fields(main)

	top, right, bottom, left, ok := viewBoxInsetParts(parts)
	if !ok {
		return nil
	}

	t, okT := resolveShapeInsetEdge(top, refH, fsize, scale)
	r, okR := resolveShapeInsetEdge(right, refW, fsize, scale)
	b, okB := resolveShapeInsetEdge(bottom, refH, fsize, scale)
	l, okL := resolveShapeInsetEdge(left, refW, fsize, scale)

	if !okT || !okR || !okB || !okL {
		return nil
	}

	x0 := refX + l - margin
	y0 := refY + t - margin
	x1 := refX + refW - r + margin
	y1 := refY + refH - b + margin

	if x1 <= x0 || y1 <= y0 {
		return nil
	}

	return &shapeExclusion{kind: shapeKindInset, x0: x0, y0: y0, x1: x1, y1: y1}
}

func resolveShapeCenter(
	position string, refX, refY, refW, refH, fsize, scale float64,
) (float64, float64) {
	cx := refX + refW/2
	cy := refY + refH/2

	if position == "" {
		return cx, cy
	}

	fields := strings.Fields(position)
	// Lite: 1-2 length/keyword tokens → (x,y); keywords map to edges/center.
	switch len(fields) {
	case 1:
		cx = resolveShapePos1D(fields[0], refX, refW, fsize, scale, true)
	case 2:
		cx = resolveShapePos1D(fields[0], refX, refW, fsize, scale, true)
		cy = resolveShapePos1D(fields[1], refY, refH, fsize, scale, false)
	}

	return cx, cy
}

func resolveShapePos1D(token string, origin, size, fsize, scale float64, horizontal bool) float64 {
	switch token {
	case "center":
		return origin + size/2
	case "left", "top":
		if horizontal && token == "top" {
			return origin + size/2
		}

		if !horizontal && token == "left" {
			return origin + size/2
		}

		return origin
	case "right", "bottom":
		if horizontal && token == "bottom" {
			return origin + size/2
		}

		if !horizontal && token == "right" {
			return origin + size/2
		}

		return origin + size
	}

	if v, ok := resolveShapeLength(token, size, fsize, scale); ok {
		return origin + v
	}

	return origin + size/2
}

func resolveShapeRadius(
	token string, primary, secondary, fsize, scale float64, circle bool,
) (float64, bool) {
	val, unit, ok := css.ParseLength(token)
	if !ok {
		return 0, false
	}

	if unit == "%" {
		if circle {
			return val / 100 * shapeReferenceDiagonal(primary, secondary), true
		}

		return primary * val / 100, true
	}

	pt, converted := lengthToPt(val, unit, fsize)
	if !converted {
		return 0, false
	}

	return pt * scale, true
}

func resolveShapeLength(token string, basis, fsize, scale float64) (float64, bool) {
	val, unit, ok := css.ParseLength(token)
	if !ok {
		return 0, false
	}

	if unit == "%" {
		return basis * val / 100, true
	}

	pt, converted := lengthToPt(val, unit, fsize)
	if !converted {
		return 0, false
	}

	return pt * scale, true
}

func resolveShapeInsetEdge(token string, basis, fsize, scale float64) (float64, bool) {
	return resolveShapeLength(token, basis, fsize, scale)
}

func closestSideRadius(cx, cy, refX, refY, refW, refH float64) float64 {
	r := math.MaxFloat64
	if refW > 0 {
		r = minY(r, minY(cx-refX, refX+refW-cx))
	}

	if refH > 0 {
		r = minY(r, minY(cy-refY, refY+refH-cy))
	}

	if r == math.MaxFloat64 || r < 0 {
		return 0
	}

	return r
}

func farthestSideRadius(cx, cy, refX, refY, refW, refH float64) float64 {
	return maxY(
		maxY(cx-refX, refX+refW-cx),
		maxY(cy-refY, refY+refH-cy),
	)
}
