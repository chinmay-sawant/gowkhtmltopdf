//nolint:cyclop,exhaustruct,mnd,varnamelen,wsl // image adjustment property parsers
package layout

import (
	"image"
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// Image adjustment support group: image-orientation, image-resolution, and
// object-view-box. The paint and sizing consumers live in layout_images.go;
// the raster transforms and image metadata parsers live in image_exif.go.
//
// Supported subset:
//   - image-orientation: from-image | none | [ <angle> || flip ]. from-image
//     reads the JPEG EXIF orientation; none ignores it; an explicit angle
//     replaces EXIF. Non-quarter-turn angles rotate inside the original
//     canvas, so content outside the canvas is dropped.
//   - image-resolution: from-image | <resolution> | [ from-image &&
//     <resolution> ]. dpi, dpcm, dppx, and the x alias are converted to DPI.
//   - object-view-box: inset, xywh, and rect crop the source image. circle,
//     ellipse, and polygon parse and store but leave the image uncropped.
const (
	anglePrecision      = 1e6
	resolutionPrecision = 1e3

	// imageAdjustFromImage is the "from-image" keyword shared by
	// image-orientation and image-resolution.
	imageAdjustFromImage = "from-image"

	// objectViewBoxRectShape is the rect() spelling shared by the canonical
	// parser, the auto-edge check, and the rect resolver.
	objectViewBoxRectShape = "rect"
)

// applyImageAdjustProps owns image-orientation, image-resolution, and
// object-view-box. Invalid values leave the previous value in place, matching
// the other apply groups.
func applyImageAdjustProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "image-orientation":
		if canonical, deg, ok := parseImageOrientation(value); ok {
			style.ImageOrientation = canonical
			style.ImageOrientationAngle = deg
		}
	case "image-resolution":
		if canonical, dpi, ok := parseImageResolution(value); ok {
			style.ImageResolution = canonical
			style.ImageResolutionDPI = dpi
		}
	case "object-view-box":
		if canonical, ok := parseObjectViewBox(value); ok {
			style.ObjectViewBox = canonical
		}
	default:
		return false
	}

	return true
}

// parseImageOrientation parses from-image | none | [ <angle> || flip ].
// The canonical value is "from-image", "none", or "<degrees>deg" with an
// optional " flip" suffix. The return value carries the used degrees so the
// consumer never re-parses the string.
func parseImageOrientation(raw string) (string, float64, bool) {
	value := normalizeCSSValue(raw)
	switch value {
	case "":
		return "", 0, false
	case imageAdjustFromImage, "none":
		return value, 0, true
	}

	deg := 0.0
	hasAngle := false
	flip := false

	for _, token := range strings.Fields(value) {
		if token == "flip" {
			if flip {
				return "", 0, false
			}

			flip = true

			continue
		}

		if hasAngle {
			return "", 0, false
		}

		parsed, ok := parseImageOrientationAngle(token)
		if !ok {
			return "", 0, false
		}

		deg, hasAngle = parsed, true
	}

	if !hasAngle && !flip {
		return "", 0, false
	}

	deg = snapAngle(normalizeDegrees(deg))

	canonical := formatCSSNumber(deg) + "deg"
	if flip {
		canonical += " flip"
	}

	return canonical, deg, true
}

// parseImageOrientationAngle parses one CSS angle token. Unlike the transform
// grammar, image-orientation requires a unit.
func parseImageOrientationAngle(token string) (float64, bool) {
	switch {
	case strings.HasSuffix(token, "grad"):
		return scaleImageAngle(token, "grad", gradToDegFactor)
	case strings.HasSuffix(token, "turn"):
		return scaleImageAngle(token, "turn", fullTurnDegrees)
	case strings.HasSuffix(token, "deg"):
		return scaleImageAngle(token, "deg", 1)
	case strings.HasSuffix(token, "rad"):
		return scaledImageRadians(token)
	default:
		return 0, false
	}
}

func scaleImageAngle(token, unit string, factor float64) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSuffix(token, unit), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	return value * factor, true
}

func scaledImageRadians(token string) (float64, bool) {
	value, ok := scaleImageAngle(token, "rad", 1)
	if !ok {
		return 0, false
	}

	return value * degreesInHalfCircle / math.Pi, true
}

// parseImageResolution parses from-image | <resolution> |
// [ from-image && <resolution> ]. The canonical string stores the resolution
// in DPI so equivalent units intern to one record.
func parseImageResolution(raw string) (string, float64, bool) {
	value := normalizeCSSValue(raw)
	switch value {
	case "":
		return "", 0, false
	case imageAdjustFromImage:
		return value, 0, true
	}

	fromImage := false
	dpi := 0.0
	hasResolution := false

	for _, token := range strings.Fields(value) {
		if token == imageAdjustFromImage {
			if fromImage {
				return "", 0, false
			}

			fromImage = true

			continue
		}

		if hasResolution {
			return "", 0, false
		}

		parsed, ok := parseResolutionDPI(token)
		if !ok {
			return "", 0, false
		}

		dpi, hasResolution = parsed, true
	}

	if !hasResolution {
		return "", 0, false
	}

	dpi = math.Round(dpi*resolutionPrecision) / resolutionPrecision

	canonical := formatCSSNumber(dpi) + "dpi"
	if fromImage {
		canonical = "from-image " + canonical
	}

	return canonical, dpi, true
}

// parseResolutionDPI converts one resolution token to DPI. Resolution must be
// positive; zero and negative values are invalid.
func parseResolutionDPI(token string) (float64, bool) {
	var unit string

	switch {
	case strings.HasSuffix(token, "dppx"):
		unit = "dppx"
	case strings.HasSuffix(token, "dpcm"):
		unit = "dpcm"
	case strings.HasSuffix(token, "dpi"):
		unit = "dpi"
	case strings.HasSuffix(token, "x"):
		unit = "x"
	default:
		return 0, false
	}

	value, err := strconv.ParseFloat(strings.TrimSuffix(token, unit), 64)
	if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	switch unit {
	case "dppx", "x":
		return value * defaultImageResolutionDPI, true
	case "dpcm":
		return value * inchesToCentimeters, true
	default:
		return value, true
	}
}

// parseObjectViewBox validates and canonicalizes object-view-box:
// none | inset(...) | xywh(...) | rect(...) | circle(...) | ellipse(...) |
// polygon(...). The shape functions are stored as written; objectViewBoxRect
// leaves them uncropped.
func parseObjectViewBox(raw string) (string, bool) {
	value := normalizeCSSValue(raw)
	if value == "" {
		return "", false
	}

	if value == "none" {
		return value, true
	}

	name, args, ok := splitCSSFunction(value)
	if !ok {
		return "", false
	}

	switch name {
	case "inset":
		return canonicalInsetViewBox(args)
	case "xywh", objectViewBoxRectShape:
		return canonicalRectViewBox(name, args)
	case "circle", "ellipse", "polygon":
		return name + "(" + args + ")", true
	default:
		return "", false
	}
}

func canonicalInsetViewBox(args string) (string, bool) {
	main, radius := splitViewBoxRound(args)
	parts := strings.Fields(main)
	if len(parts) < 1 || len(parts) > 4 {
		return "", false
	}

	for _, part := range parts {
		if _, ok := parseViewBoxLength(part); !ok {
			return "", false
		}
	}

	if radius != "" && !validViewBoxRound(radius) {
		return "", false
	}

	out := "inset(" + strings.Join(parts, " ")
	if radius != "" {
		out += " round " + radius
	}

	return out + ")", true
}

func canonicalRectViewBox(name, args string) (string, bool) {
	main, radius := splitViewBoxRound(args)
	parts := strings.Fields(main)
	if len(parts) != 4 {
		return "", false
	}

	for _, part := range parts {
		if part == "auto" && name == objectViewBoxRectShape {
			continue
		}

		if _, ok := parseViewBoxLength(part); !ok {
			return "", false
		}
	}

	if radius != "" && !validViewBoxRound(radius) {
		return "", false
	}

	out := name + "(" + strings.Join(parts, " ")
	if radius != "" {
		out += " round " + radius
	}

	return out + ")", true
}

// validViewBoxRound accepts the optional round <border-radius> clause without
// applying it: the supported crop is rectangular. The grammar allows one or
// two radius lists separated by "/".
func validViewBoxRound(radius string) bool {
	sides := strings.Split(radius, "/")
	if len(sides) > 2 {
		return false
	}

	for _, side := range sides {
		fields := strings.Fields(side)
		if len(fields) < 1 || len(fields) > 4 {
			return false
		}

		for _, field := range fields {
			if _, ok := parseViewBoxLength(field); !ok {
				return false
			}
		}
	}

	return true
}

// objectViewBoxRect resolves a canonical object-view-box value to a source
// crop rectangle for an image of w x h pixels. ok is false for "none",
// malformed values, an empty resolved rectangle, and the shape functions
// (circle/ellipse/polygon): those leave the image uncropped.
func objectViewBoxRect(value string, w, h int) (image.Rectangle, bool) {
	if w <= 0 || h <= 0 {
		return image.Rectangle{}, false
	}

	name, args, ok := splitCSSFunction(strings.TrimSpace(value))
	if !ok {
		return image.Rectangle{}, false
	}

	main, _ := splitViewBoxRound(args)
	parts := strings.Fields(main)

	switch name {
	case "inset":
		top, right, bottom, left, ok := viewBoxInsetParts(parts)
		if !ok {
			return image.Rectangle{}, false
		}

		return viewBoxRectFromEdges(top, right, bottom, left, w, h, false)
	case objectViewBoxRectShape:
		if len(parts) != 4 {
			return image.Rectangle{}, false
		}

		return viewBoxRectFromEdges(parts[0], parts[1], parts[2], parts[3], w, h, true)
	case "xywh":
		return viewBoxRectXYWH(parts, w, h)
	default:
		return image.Rectangle{}, false
	}
}

// viewBoxInsetParts expands the inset() 1-4 value shorthand into top, right,
// bottom, and left edge tokens.
func viewBoxInsetParts(parts []string) (string, string, string, string, bool) {
	switch len(parts) {
	case 1:
		return parts[0], parts[0], parts[0], parts[0], true
	case 2:
		return parts[0], parts[1], parts[0], parts[1], true
	case 3:
		return parts[0], parts[1], parts[2], parts[1], true
	case 4:
		return parts[0], parts[1], parts[2], parts[3], true
	default:
		return "", "", "", "", false
	}
}

// viewBoxRectFromEdges maps inset/rect edges to a source rect. auto is legal
// only for rect(), where it means the start edge (top/left) or the end edge
// (bottom/right).
func viewBoxRectFromEdges(top, right, bottom, left string, w, h int, allowAuto bool) (image.Rectangle, bool) {
	topPx, okTop := viewBoxEdgePixels(top, h, allowAuto, true)
	rightPx, okRight := viewBoxEdgePixels(right, w, allowAuto, false)
	bottomPx, okBottom := viewBoxEdgePixels(bottom, h, allowAuto, false)
	leftPx, okLeft := viewBoxEdgePixels(left, w, allowAuto, true)
	if !okTop || !okRight || !okBottom || !okLeft {
		return image.Rectangle{}, false
	}

	rect := image.Rect(
		int(math.Round(leftPx)), int(math.Round(topPx)),
		int(math.Round(float64(w)-rightPx)), int(math.Round(float64(h)-bottomPx)),
	)

	return clampViewBoxRect(rect, w, h)
}

func viewBoxRectXYWH(parts []string, w, h int) (image.Rectangle, bool) {
	if len(parts) != 4 {
		return image.Rectangle{}, false
	}

	x, okX := viewBoxEdgePixels(parts[0], w, false, true)
	y, okY := viewBoxEdgePixels(parts[1], h, false, true)
	width, okW := viewBoxEdgePixels(parts[2], w, false, true)
	height, okH := viewBoxEdgePixels(parts[3], h, false, true)
	if !okX || !okY || !okW || !okH {
		return image.Rectangle{}, false
	}

	rect := image.Rect(
		int(math.Round(x)), int(math.Round(y)),
		int(math.Round(x+width)), int(math.Round(y+height)),
	)

	return clampViewBoxRect(rect, w, h)
}

func viewBoxEdgePixels(token string, extent int, allowAuto, startEdge bool) (float64, bool) {
	if token == "auto" {
		if !allowAuto {
			return 0, false
		}

		if startEdge {
			return 0, true
		}

		return float64(extent), true
	}

	length, ok := parseViewBoxLength(token)
	if !ok {
		return 0, false
	}

	if length.percent {
		return length.value * float64(extent) / oneHundred, true
	}

	return length.value, true
}

func clampViewBoxRect(rect image.Rectangle, w, h int) (image.Rectangle, bool) {
	rect = rect.Intersect(image.Rect(0, 0, w, h))
	if rect.Empty() {
		return image.Rectangle{}, false
	}

	return rect, true
}

// viewBoxLength is one object-view-box coordinate: an absolute length in CSS
// px or a percentage of the image's pixel extent.
type viewBoxLength struct {
	value   float64
	percent bool
}

// parseViewBoxLength validates one length-percentage token. The image
// coordinate system has no font or viewport context, so em/rem/ex/ch/vw/vh
// are rejected; absolute units convert at 96dpi.
func parseViewBoxLength(token string) (viewBoxLength, bool) {
	value, unit, ok := css.ParseLength(token)
	if !ok || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return viewBoxLength{}, false
	}

	switch unit {
	case "%":
		return viewBoxLength{value: value, percent: true}, true
	case "px":
		return viewBoxLength{value: value}, true
	case "in", "cm", "mm", "pt", "pc":
		pt, converted := css.LengthToPt(value, unit, 16)
		if !converted {
			return viewBoxLength{}, false
		}

		return viewBoxLength{value: pt / pxToPt(1)}, true
	default:
		return viewBoxLength{}, false
	}
}

// splitViewBoxRound separates the optional " round <radius>" clause.
func splitViewBoxRound(args string) (string, string) {
	const marker = " round "

	idx := strings.Index(args, marker)
	if idx < 0 {
		return args, ""
	}

	return strings.TrimSpace(args[:idx]), strings.TrimSpace(args[idx+len(marker):])
}

// splitCSSFunction splits "name(args)" into name and args. It rejects empty
// names, empty argument lists, and unbalanced argument parentheses.
func splitCSSFunction(value string) (string, string, bool) {
	open := strings.IndexByte(value, '(')
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return "", "", false
	}

	name := strings.TrimSpace(value[:open])
	args := strings.TrimSpace(value[open+1 : len(value)-1])
	if name == "" || args == "" || strings.ContainsAny(name, "()") {
		return "", "", false
	}

	if !balancedParens(args) {
		return "", "", false
	}

	return name, args, true
}

func balancedParens(value string) bool {
	depth := 0

	for idx := range len(value) {
		switch value[idx] {
		case '(':
			depth++
		case ')':
			depth--

			if depth < 0 {
				return false
			}
		}
	}

	return depth == 0
}

// normalizeCSSValue lowercases and collapses whitespace so canonical stored
// values compare equal for equivalent declarations.
func normalizeCSSValue(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func formatCSSNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

// snapAngle rounds to six decimals so 100grad and 0.25turn canonicalize to
// the same "90deg".
func snapAngle(deg float64) float64 {
	return math.Round(deg*anglePrecision) / anglePrecision
}
