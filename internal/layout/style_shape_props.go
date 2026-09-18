//nolint:cyclop,mnd,varnamelen // shape-outside / shape-margin parsers
package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// CSS Shapes apply group (css-shapes-1 lite):
//   - shape-outside: none | circle() | ellipse() | inset() [ || <shape-box> ]
//   - shape-margin: <length-percentage>
//
// polygon()/path()/url() parse as rejected (leave prior value). shape-inside,
// shape-padding, and shape-image-threshold stay Unsupported (no apply arms).

const (
	shapeOutsideNone = "none"
	shapeBoxMargin   = "margin-box"
	shapeBoxBorder   = "border-box"
	shapeBoxPadding  = "padding-box"
	shapeBoxContent  = "content-box"
)

// applyShapeProps owns shape-outside and shape-margin.
func applyShapeProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "shape-outside":
		if canonical, ok := parseShapeOutside(value); ok {
			style.ShapeOutside = canonical
		}
	case "shape-margin":
		if pt, pct, ok := parseShapeMargin(value, fsize, ctx); ok {
			style.ShapeMargin = pt
			style.ShapeMarginPercent = pct
		}
	default:
		return false
	}

	return true
}

// parseShapeOutside validates and canonicalizes the supported shape-outside
// subset. Shape-box keywords are accepted and stored; resolution always uses
// the float margin box at place time (lite).
func parseShapeOutside(raw string) (string, bool) {
	value := normalizeCSSValue(raw)
	if value == "" {
		return "", false
	}

	if value == shapeOutsideNone {
		return shapeOutsideNone, true
	}

	shape, box, ok := splitShapeOutsideParts(value)
	if !ok {
		return "", false
	}

	name, args, ok := splitShapeFunction(shape)
	if !ok {
		return "", false
	}

	var canonical string

	switch name {
	case "circle":
		canonical, ok = canonicalShapeCircle(args)
	case "ellipse":
		canonical, ok = canonicalShapeEllipse(args)
	case "inset":
		canonical, ok = canonicalShapeInset(args)
	default:
		return "", false
	}

	if !ok {
		return "", false
	}

	if box != "" {
		return canonical + " " + box, true
	}

	return canonical, true
}

func splitShapeOutsideParts(value string) (string, string, bool) {
	box := ""
	rest := value

	for _, keyword := range []string{shapeBoxMargin, shapeBoxBorder, shapeBoxPadding, shapeBoxContent} {
		if rest == keyword {
			return "", "", false // box alone is not a supported lite shape
		}

		prefix := keyword + " "
		suffix := " " + keyword

		switch {
		case strings.HasPrefix(rest, prefix):
			box = keyword
			rest = strings.TrimSpace(rest[len(prefix):])
		case strings.HasSuffix(rest, suffix):
			box = keyword
			rest = strings.TrimSpace(rest[:len(rest)-len(suffix)])
		}
	}

	if rest == "" || !strings.Contains(rest, "(") {
		return "", "", false
	}

	return rest, box, true
}

func splitShapeFunction(value string) (string, string, bool) {
	open := strings.IndexByte(value, '(')
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return "", "", false
	}

	name := strings.TrimSpace(value[:open])
	args := strings.TrimSpace(value[open+1 : len(value)-1])

	if name == "" || strings.ContainsAny(name, "()") {
		return "", "", false
	}

	if !balancedParens(args) {
		return "", "", false
	}

	return name, args, true
}

func canonicalShapeCircle(args string) (string, bool) {
	if args == "" {
		return "circle()", true
	}

	radius, position, ok := splitShapeRadiusPosition(args, 1)
	if !ok {
		return "", false
	}

	out := "circle(" + strings.Join(radius, " ")
	if position != "" {
		out += " at " + position
	}

	return out + ")", true
}

func canonicalShapeEllipse(args string) (string, bool) {
	if args == "" {
		return "ellipse()", true
	}

	radii, position, ok := splitShapeRadiusPosition(args, 2)
	if !ok {
		return "", false
	}

	out := "ellipse(" + strings.Join(radii, " ")
	if position != "" {
		out += " at " + position
	}

	return out + ")", true
}

func canonicalShapeInset(args string) (string, bool) {
	main, _ := splitViewBoxRound(args) // ignore round radii in lite wrap
	parts := strings.Fields(main)

	if len(parts) < 1 || len(parts) > 4 {
		return "", false
	}

	for _, part := range parts {
		if !validShapeLength(part) {
			return "", false
		}
	}

	return "inset(" + strings.Join(parts, " ") + ")", true
}

func splitShapeRadiusPosition(args string, wantRadii int) ([]string, string, bool) {
	at := strings.Index(args, " at ")
	main := args
	position := ""

	if at >= 0 {
		main = strings.TrimSpace(args[:at])
		position = strings.TrimSpace(args[at+4:])

		if position == "" || !validShapePosition(position) {
			return nil, "", false
		}
	}

	fields := strings.Fields(main)
	if len(fields) == 0 {
		// Empty or "at …" alone → default closest-side radii.
		return nil, position, true
	}

	if len(fields) != wantRadii {
		return nil, "", false
	}

	for _, field := range fields {
		if field == "closest-side" || field == "farthest-side" {
			continue
		}

		if !validShapeLength(field) {
			return nil, "", false
		}
	}

	return fields, position, true
}

func validShapePosition(position string) bool {
	fields := strings.Fields(position)
	if len(fields) == 0 || len(fields) > 4 {
		return false
	}

	for _, field := range fields {
		switch field {
		case "left", "right", "top", "bottom", "center":
			continue
		default:
			if !validShapeLength(field) {
				return false
			}
		}
	}

	return true
}

func validShapeLength(token string) bool {
	if token == "0" {
		return true
	}

	val, unit, ok := css.ParseLength(token)
	if !ok || val < 0 {
		return false
	}

	switch unit {
	case "", "px", "pt", "em", "rem", "ex", "ch", "cm", "mm", "in", "pc", "q", "%":
		return true
	default:
		return false
	}
}

func parseShapeMargin(raw string, fsize float64, ctx *styleContext) (float64, float64, bool) {
	value := normalizeCSSValue(raw)
	if value == "" {
		return 0, -1, false
	}

	val, unit, ok := css.ParseLength(value)
	if !ok || val < 0 {
		return 0, -1, false
	}

	if unit == "%" {
		return 0, val, true
	}

	pt, converted := lengthToPt(val, unit, fsize)
	if !converted {
		return 0, -1, false
	}

	_ = ctx // percentages of viewport are not used; % is shape-reference relative

	return pt, -1, true
}

// shapeReferenceDiagonal returns the CSS Shapes %-basis: sqrt(w²+h²)/√2.
func shapeReferenceDiagonal(w, h float64) float64 {
	return math.Sqrt(w*w+h*h) / math.Sqrt2
}
