//nolint:varnamelen,mnd,cyclop,gocognit,goconst // clip-path parsing and raster mask
package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	clipPathNone     = "none"
	clipPathEvenOdd  = "evenodd"
	clipPathNonZero  = "nonzero"
	clipPathAtMarker = " at "
)

// clipPathKind is the parsed basic-shape family. Zero means no shape.
type clipPathKind uint8

const (
	clipPathInset clipPathKind = iota + 1
	clipPathCircle
	clipPathEllipse
	clipPathPolygon
)

// clipPathLength is one component of a basic shape: an absolute length in
// points, a fraction of a reference dimension, or a closest/farthest-side
// keyword.
type clipPathLength struct {
	value   float64
	percent bool
	keyword string // "closest-side" | "farthest-side" | ""
}

func (l clipPathLength) resolve(base float64) float64 {
	if l.percent {
		return l.value * base
	}

	return l.value
}

type clipPathPoint struct {
	x, y clipPathLength
}

// clipPathShape is a parsed clip-path basic shape. Lengths resolve against the
// reference box (border box) at mask time.
type clipPathShape struct {
	kind     clipPathKind
	top      clipPathLength
	right    clipPathLength
	bottom   clipPathLength
	left     clipPathLength
	radius   clipPathLength
	radiusX  clipPathLength
	radiusY  clipPathLength
	centerX  clipPathLength
	centerY  clipPathLength
	points   []clipPathPoint
	fillRule string
}

// applyClipPathProps owns clip-path. Supported values are stored canonically;
// unsupported or invalid values leave the previous declaration intact so the
// renderer keeps today's no-clip behavior for them.
func applyClipPathProps(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop != "clip-path" {
		return false
	}

	normalized := normalizeCSSValue(value)
	if normalized == clipPathNone {
		style.ClipPath = ""

		return true
	}

	if _, ok := parseClipPathShape(normalized, fsize); ok {
		style.ClipPath = normalized
	}

	return true
}

// parseClipPathShape parses none | inset() | circle() | ellipse() | polygon().
// "none" and unsupported functions yield ok=false.
func parseClipPathShape(raw string, fsize float64) (clipPathShape, bool) {
	value := normalizeCSSValue(raw)
	if value == "" || value == clipPathNone {
		return clipPathShape{}, false //nolint:exhaustruct // no shape
	}

	name, args, ok := splitShapeFunction(value)
	if !ok {
		return clipPathShape{}, false //nolint:exhaustruct // malformed function
	}

	switch name {
	case "inset":
		return parseClipPathInset(args, fsize)
	case "circle":
		return parseClipPathCircle(args, fsize)
	case "ellipse":
		return parseClipPathEllipse(args, fsize)
	case "polygon":
		return parseClipPathPolygon(args, fsize)
	default:
		return clipPathShape{}, false //nolint:exhaustruct // unsupported shape
	}
}

func parseClipPathInset(args string, fsize float64) (clipPathShape, bool) {
	main, _ := splitViewBoxRound(args) // round radii are accepted and ignored
	fields := strings.Fields(main)
	if len(fields) < 1 || len(fields) > 4 {
		return clipPathShape{}, false //nolint:exhaustruct // invalid inset
	}

	lengths := make([]clipPathLength, 4)
	for i, field := range fields {
		length, ok := parseClipPathLength(field, fsize)
		if !ok {
			return clipPathShape{}, false //nolint:exhaustruct // invalid inset length
		}

		lengths[i] = length
	}

	shape := clipPathShape{ //nolint:exhaustruct // remaining fields unused
		kind: clipPathInset,
	}

	switch len(fields) {
	case 1:
		shape.top, shape.right, shape.bottom, shape.left = lengths[0], lengths[0], lengths[0], lengths[0]
	case 2:
		shape.top, shape.bottom = lengths[0], lengths[0]
		shape.right, shape.left = lengths[1], lengths[1]
	case 3:
		shape.top, shape.right, shape.bottom = lengths[0], lengths[1], lengths[2]
		shape.left = lengths[1]
	default:
		shape.top, shape.right, shape.bottom, shape.left = lengths[0], lengths[1], lengths[2], lengths[3]
	}

	return shape, true
}

func parseClipPathCircle(args string, fsize float64) (clipPathShape, bool) {
	radiusPart, position := splitClipPathAt(args)
	radius := clipPathLength{keyword: "closest-side"}

	if radiusPart != "" {
		if radiusPart == "closest-side" || radiusPart == "farthest-side" {
			radius = clipPathLength{keyword: radiusPart}
		} else {
			parsed, ok := parseClipPathLength(radiusPart, fsize)
			if !ok {
				return clipPathShape{}, false //nolint:exhaustruct // invalid radius
			}

			radius = parsed
		}
	}

	centerX, centerY, ok := parseClipPathPosition(position, fsize)
	if !ok {
		return clipPathShape{}, false //nolint:exhaustruct // invalid position
	}

	return clipPathShape{ //nolint:exhaustruct // remaining fields unused
		kind: clipPathCircle, radius: radius, centerX: centerX, centerY: centerY,
	}, true
}

func parseClipPathEllipse(args string, fsize float64) (clipPathShape, bool) {
	radiiPart, position := splitClipPathAt(args)
	radiusX := clipPathLength{keyword: "closest-side"}
	radiusY := clipPathLength{keyword: "closest-side"}

	if radiiPart != "" {
		fields := strings.Fields(radiiPart)
		if len(fields) != 2 {
			return clipPathShape{}, false //nolint:exhaustruct // invalid radii count
		}

		parsedX, okX := parseClipPathRadius(fields[0], fsize)
		parsedY, okY := parseClipPathRadius(fields[1], fsize)
		if !okX || !okY {
			return clipPathShape{}, false //nolint:exhaustruct // invalid radius
		}

		radiusX, radiusY = parsedX, parsedY
	}

	centerX, centerY, ok := parseClipPathPosition(position, fsize)
	if !ok {
		return clipPathShape{}, false //nolint:exhaustruct // invalid position
	}

	return clipPathShape{ //nolint:exhaustruct // remaining fields unused
		kind: clipPathEllipse, radiusX: radiusX, radiusY: radiusY, centerX: centerX, centerY: centerY,
	}, true
}

func parseClipPathPolygon(args string, fsize float64) (clipPathShape, bool) {
	fillRule := ""
	if idx := strings.IndexByte(args, ','); idx >= 0 {
		first := strings.TrimSpace(args[:idx])
		if first == clipPathEvenOdd || first == clipPathNonZero {
			fillRule = first
			args = strings.TrimSpace(args[idx+1:])
		}
	}

	var points []clipPathPoint

	for _, part := range splitCommaLayers(args) {
		fields := strings.Fields(part)
		if len(fields) != 2 {
			return clipPathShape{}, false //nolint:exhaustruct // invalid vertex
		}

		x, okX := parseClipPathCoord(fields[0], true, fsize)
		y, okY := parseClipPathCoord(fields[1], false, fsize)
		if !okX || !okY {
			return clipPathShape{}, false //nolint:exhaustruct // invalid vertex
		}

		points = append(points, clipPathPoint{x: x, y: y})
	}

	if len(points) < 3 {
		return clipPathShape{}, false //nolint:exhaustruct // not a polygon
	}

	return clipPathShape{ //nolint:exhaustruct // remaining fields unused
		kind: clipPathPolygon, points: points, fillRule: fillRule,
	}, true
}

// splitClipPathAt splits "radius at position" or "at position" into its parts.
func splitClipPathAt(args string) (string, string) {
	args = strings.TrimSpace(args)
	if strings.HasPrefix(args, "at ") {
		return "", strings.TrimSpace(args[len("at "):])
	}

	if idx := strings.Index(args, clipPathAtMarker); idx >= 0 {
		return strings.TrimSpace(args[:idx]), strings.TrimSpace(args[idx+len(clipPathAtMarker):])
	}

	return args, ""
}

func parseClipPathRadius(token string, fsize float64) (clipPathLength, bool) {
	if token == "closest-side" || token == "farthest-side" {
		return clipPathLength{keyword: token}, true
	}

	return parseClipPathLength(token, fsize)
}

// parseClipPathPosition parses 0-2 position tokens: keywords plus lengths or
// percentages. An empty position is center.
func parseClipPathPosition(position string, fsize float64) (clipPathLength, clipPathLength, bool) {
	half := clipPathLength{value: 0.5, percent: true}
	centerX, centerY := half, half
	fields := strings.Fields(position)

	if len(fields) > 2 {
		return clipPathLength{}, clipPathLength{}, false //nolint:exhaustruct // unsupported offsets
	}

	haveX := false

	for i, field := range fields {
		switch field {
		case "left", "right":
			centerX, haveX = clipPathKeyword(field), true
		case "top", "bottom":
			centerY = clipPathKeyword(field)
		case "center":
			if !haveX {
				centerX, haveX = half, true
			} else {
				centerY = half
			}
		default:
			length, ok := parseClipPathLength(field, fsize)
			if !ok {
				return clipPathLength{}, clipPathLength{}, false //nolint:exhaustruct // invalid position
			}

			if i == 0 {
				centerX, haveX = length, true
			} else {
				centerY = length
			}
		}
	}

	return centerX, centerY, true
}

func clipPathKeyword(token string) clipPathLength {
	switch token {
	case "left", "top":
		return clipPathLength{value: 0}
	case "right", "bottom":
		return clipPathLength{value: 1, percent: true}
	default:
		return clipPathLength{value: 0.5, percent: true}
	}
}

func parseClipPathCoord(token string, horizontal bool, fsize float64) (clipPathLength, bool) {
	switch token {
	case "center":
		return clipPathLength{value: 0.5, percent: true}, true
	case "left", "right":
		if horizontal {
			return clipPathKeyword(token), true
		}
	case "top", "bottom":
		if !horizontal {
			return clipPathKeyword(token), true
		}
	}

	return parseClipPathLength(token, fsize)
}

// parseClipPathLength parses a non-negative length or percentage.
func parseClipPathLength(token string, fsize float64) (clipPathLength, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return clipPathLength{}, false //nolint:exhaustruct // empty token
	}

	if strings.HasSuffix(token, "%") {
		value, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64)
		if err != nil || value < 0 {
			return clipPathLength{}, false //nolint:exhaustruct // invalid percentage
		}

		return clipPathLength{value: value / 100.0, percent: true}, true
	}

	value, unit, ok := css.ParseLength(token)
	if !ok || value < 0 {
		return clipPathLength{}, false //nolint:exhaustruct // invalid length
	}

	pt, converted := lengthToPt(value, unit, fsize)
	if !converted {
		return clipPathLength{}, false //nolint:exhaustruct // unsupported unit
	}

	return clipPathLength{value: pt}, true
}

// contains reports whether a point in reference-box coordinates lies inside
// the shape. w and h are the reference box dimensions.
func (s clipPathShape) contains(x, y, w, h float64) bool {
	switch s.kind {
	case clipPathInset:
		return x >= s.left.resolve(w) && x <= w-s.right.resolve(w) &&
			y >= s.top.resolve(h) && y <= h-s.bottom.resolve(h)
	case clipPathCircle:
		cx, cy := s.centerX.resolve(w), s.centerY.resolve(h)
		radius := resolveCircleRadius(s.radius, cx, cy, w, h)
		if radius <= 0 {
			return false
		}

		dx, dy := x-cx, y-cy

		return dx*dx+dy*dy <= radius*radius
	case clipPathEllipse:
		cx, cy := s.centerX.resolve(w), s.centerY.resolve(h)
		rx, ry := resolveEllipseRadii(s.radiusX, s.radiusY, cx, cy, w, h)
		if rx <= 0 || ry <= 0 {
			return false
		}

		dx, dy := (x-cx)/rx, (y-cy)/ry

		return dx*dx+dy*dy <= 1
	case clipPathPolygon:
		return polygonContains(s.points, s.fillRule, x, y, w, h)
	default:
		return true
	}
}

func resolveCircleRadius(radius clipPathLength, cx, cy, w, h float64) float64 {
	switch radius.keyword {
	case "closest-side":
		return math.Min(math.Min(cx, w-cx), math.Min(cy, h-cy))
	case "farthest-side":
		return math.Max(math.Max(cx, w-cx), math.Max(cy, h-cy))
	default:
		return radius.resolve(shapeReferenceDiagonal(w, h))
	}
}

func resolveEllipseRadii(rx, ry clipPathLength, cx, cy, w, h float64) (float64, float64) {
	if rx.keyword == "farthest-side" || ry.keyword == "farthest-side" {
		return math.Max(cx, w-cx), math.Max(cy, h-cy)
	}

	if rx.keyword == "closest-side" || ry.keyword == "closest-side" {
		return math.Min(cx, w-cx), math.Min(cy, h-cy)
	}

	return rx.resolve(w), ry.resolve(h)
}

func polygonContains(points []clipPathPoint, fillRule string, x, y, w, h float64) bool {
	inside := false
	winding := 0
	prev := len(points) - 1

	for i := range points {
		xi, yi := points[i].x.resolve(w), points[i].y.resolve(h)
		xj, yj := points[prev].x.resolve(w), points[prev].y.resolve(h)

		if (yi > y) != (yj > y) {
			cross := xi + (y-yi)*(xj-xi)/(yj-yi)
			if x < cross {
				inside = !inside

				if yj > yi {
					winding++
				} else {
					winding--
				}
			}
		}

		prev = i
	}

	if fillRule == clipPathEvenOdd {
		return inside
	}

	return winding != 0
}

// maskClipPathOps masks every image op in ops to shape. The shape resolves
// against the reference box; each op's image maps onto its drawn rect.
func maskClipPathOps(ops []Op, shape clipPathShape, refX, refY, refW, refH float64) {
	for i := range ops {
		op := &ops[i]
		if op.Kind != OpImage || len(op.Image) == 0 || op.W <= 0 || op.H <= 0 {
			continue
		}

		masked := maskImageWithClipPath(
			op.Image, shape, op.X, op.Y, op.W, op.H, refX, refY, refW, refH,
		)
		if masked == nil {
			continue
		}

		op.setImage(masked, op.ImgW, op.ImgH, op.Alt)
		op.IsJPEG = false
	}
}

// maskImageWithClipPath returns PNG bytes with alpha zeroed outside shape.
// nil means the mask could not be applied; callers keep the original bytes.
func maskImageWithClipPath(
	data []byte, shape clipPathShape, drawnX, drawnY, drawnW, drawnH, refX, refY, refW, refH float64,
) []byte {
	if len(data) == 0 || drawnW <= 0 || drawnH <= 0 || refW <= 0 || refH <= 0 || shape.kind == 0 {
		return nil
	}

	src, err := decodeImageBytes(data)
	if err != nil {
		return nil
	}

	nrgba := toNRGBA(src)
	bounds := nrgba.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= 0 || h <= 0 {
		return nil
	}

	for y := range h {
		for x := range w {
			c := nrgba.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}

			coverage := 0
			for subY := range 2 {
				for subX := range 2 {
					px := drawnX + (float64(x)+(float64(subX)+0.5)/2)*drawnW/float64(w) - refX
					py := drawnY + (float64(y)+(float64(subY)+0.5)/2)*drawnH/float64(h) - refY
					if shape.contains(px, py, refW, refH) {
						coverage++
					}
				}
			}

			c.A = uint8(float64(c.A) * float64(coverage) / 4)
			nrgba.SetNRGBA(x, y, c)
		}
	}

	out, err := encodePNGImage(nrgba)
	if err != nil {
		return nil
	}

	return out
}
