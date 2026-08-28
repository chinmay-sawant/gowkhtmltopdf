//nolint:cyclop,funlen,gocyclo,mnd,wsl,nlreturn,varnamelen,lll,goconst // centralized property dispatch for Wave 80.5
package layout

import (
	"math"
	"strconv"
	"strings"
)

// applyLeftoversProps handles SVG presentation, visual overflow, transforms, page, and ruby properties.
func applyLeftoversProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "clip-path", "clip":
		style.ClipPath = strings.TrimSpace(value)
	case "overflow-clip-margin":
		if m, ok := plainLength(value, fsize, 0); ok && m >= 0 {
			style.OverflowClipMargin = m
		}
	case "scroll-margin":
		applyScrollMarginShorthand(style, value, fsize)
	case "scroll-margin-top":
		if m, ok := plainLength(value, fsize, 0); ok {
			style.ScrollMarginTop = m
		}
	case "scroll-margin-right":
		if m, ok := plainLength(value, fsize, 0); ok {
			style.ScrollMarginRight = m
		}
	case "scroll-margin-bottom":
		if m, ok := plainLength(value, fsize, 0); ok {
			style.ScrollMarginBottom = m
		}
	case "scroll-margin-left":
		if m, ok := plainLength(value, fsize, 0); ok {
			style.ScrollMarginLeft = m
		}

	case "stroke-dasharray":
		applyStrokeDashArray(style, value, fsize)
	case "stroke-dashoffset":
		if off, ok := plainLength(value, fsize, 0); ok {
			style.StrokeDashOffset = off
		}
	case "stroke-linecap":
		style.StrokeLineCap = strings.ToLower(strings.TrimSpace(value))
	case "stroke-linejoin":
		style.StrokeLineJoin = strings.ToLower(strings.TrimSpace(value))
	case "stroke-miterlimit":
		if n, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && n >= 1 {
			style.StrokeMiterLimit = n
		}
	case "fill-rule":
		style.FillRule = strings.ToLower(strings.TrimSpace(value))
	case "clip-rule":
		style.ClipRule = strings.ToLower(strings.TrimSpace(value))
	case "color-interpolation":
		style.ColorInterpolation = strings.TrimSpace(value)
	case "color-interpolation-filters":
		style.ColorInterpolationFilters = strings.TrimSpace(value)
	case "shape-rendering":
		style.ShapeRendering = strings.TrimSpace(value)
	case "text-anchor":
		style.TextAnchor = strings.ToLower(strings.TrimSpace(value))
	case "dominant-baseline":
		style.DominantBaseline = strings.ToLower(strings.TrimSpace(value))
	case "alignment-baseline":
		style.AlignmentBaseline = strings.ToLower(strings.TrimSpace(value))

	case "transform-box":
		style.TransformBox = strings.ToLower(strings.TrimSpace(value))
	case "transform-style":
		style.TransformStyle = strings.ToLower(strings.TrimSpace(value))
	case "perspective":
		if p, ok := plainLength(value, fsize, 0); ok && p > 0 {
			style.Perspective = p
		}
	case "perspective-origin":
		applyPerspectiveOrigin(style, value, fsize)
	case "backface-visibility":
		style.BackfaceVisibility = strings.ToLower(strings.TrimSpace(value))
	case "rotate":
		applyRotateProperty(style, value)
	case "scale":
		applyScaleProperty(style, value)
	case "translate":
		applyTranslateProperty(style, value, fsize)

	case "page":
		style.Page = strings.TrimSpace(value)

	case "ruby-align":
		style.RubyAlign = strings.ToLower(strings.TrimSpace(value))
	case "ruby-position":
		style.RubyPosition = strings.ToLower(strings.TrimSpace(value))
	case "ruby-merge":
		style.RubyMerge = strings.ToLower(strings.TrimSpace(value))
	case "ruby-overhang":
		style.RubyOverhang = strings.ToLower(strings.TrimSpace(value))

	default:
		return false
	}

	return true
}

func applyScrollMarginShorthand(style *ResolvedStyle, value string, fsize float64) {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return
	}
	var vals []float64
	for _, p := range parts {
		if m, ok := plainLength(p, fsize, 0); ok {
			vals = append(vals, m)
		}
	}
	switch len(vals) {
	case 1:
		style.ScrollMarginTop, style.ScrollMarginRight, style.ScrollMarginBottom, style.ScrollMarginLeft = vals[0], vals[0], vals[0], vals[0]
	case 2:
		style.ScrollMarginTop, style.ScrollMarginBottom = vals[0], vals[0]
		style.ScrollMarginRight, style.ScrollMarginLeft = vals[1], vals[1]
	case 3:
		style.ScrollMarginTop = vals[0]
		style.ScrollMarginRight, style.ScrollMarginLeft = vals[1], vals[1]
		style.ScrollMarginBottom = vals[2]
	case 4:
		style.ScrollMarginTop, style.ScrollMarginRight, style.ScrollMarginBottom, style.ScrollMarginLeft = vals[0], vals[1], vals[2], vals[3]
	}
}

func applyStrokeDashArray(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if val == "none" || val == "" {
		style.StrokeDashArray = nil
		return
	}
	raw := strings.ReplaceAll(val, ",", " ")
	for _, tok := range strings.Fields(raw) {
		if length, ok := plainLength(tok, fsize, 0); ok && length >= 0 {
			style.StrokeDashArray = append(style.StrokeDashArray, length)
		} else if n, err := strconv.ParseFloat(tok, 64); err == nil && n >= 0 {
			style.StrokeDashArray = append(style.StrokeDashArray, n)
		}
	}
}

func applyPerspectiveOrigin(style *ResolvedStyle, value string, fsize float64) {
	parts := strings.Fields(value)
	if len(parts) >= 2 {
		if x, ok := plainLength(parts[0], fsize, 0); ok {
			style.PerspectiveOrigin[0] = x
		}
		if y, ok := plainLength(parts[1], fsize, 0); ok {
			style.PerspectiveOrigin[1] = y
		}
	}
}

func applyRotateProperty(style *ResolvedStyle, value string) {
	rad, ok := parseCSSAngle(value)
	if !ok {
		return
	}
	cos := math.Cos(rad)
	sin := math.Sin(rad)
	rot := Matrix2D{A: cos, B: sin, C: -sin, D: cos, E: 0, F: 0}
	style.Transform = style.Transform.Mul(rot)
	style.HasTransform = true
}

func applyScaleProperty(style *ResolvedStyle, value string) {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return
	}
	sx, err1 := strconv.ParseFloat(parts[0], 64)
	if err1 != nil {
		return
	}
	sy := sx
	if len(parts) >= 2 {
		if s2, err2 := strconv.ParseFloat(parts[1], 64); err2 == nil {
			sy = s2
		}
	}
	sc := Matrix2D{A: sx, B: 0, C: 0, D: sy, E: 0, F: 0}
	style.Transform = style.Transform.Mul(sc)
	style.HasTransform = true
}

func applyTranslateProperty(style *ResolvedStyle, value string, fsize float64) {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return
	}
	tx, ok1 := plainLength(parts[0], fsize, 0)
	if !ok1 {
		return
	}
	ty := 0.0
	if len(parts) >= 2 {
		if t2, ok2 := plainLength(parts[1], fsize, 0); ok2 {
			ty = t2
		}
	}
	tr := Matrix2D{A: 1, B: 0, C: 0, D: 1, E: tx, F: ty}
	style.Transform = style.Transform.Mul(tr)
	style.HasTransform = true
}

func parseCSSAngle(value string) (float64, bool) {
	v := strings.ToLower(strings.TrimSpace(value))
	if strings.HasSuffix(v, "deg") {
		deg, err := strconv.ParseFloat(strings.TrimSuffix(v, "deg"), 64)
		return deg * math.Pi / 180.0, err == nil
	}
	if strings.HasSuffix(v, "rad") {
		rad, err := strconv.ParseFloat(strings.TrimSuffix(v, "rad"), 64)
		return rad, err == nil
	}
	if strings.HasSuffix(v, "grad") {
		grad, err := strconv.ParseFloat(strings.TrimSuffix(v, "grad"), 64)
		return grad * math.Pi / 200.0, err == nil
	}
	if strings.HasSuffix(v, "turn") {
		turn, err := strconv.ParseFloat(strings.TrimSuffix(v, "turn"), 64)
		return turn * 2 * math.Pi, err == nil
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		return n * math.Pi / 180.0, true
	}

	return 0, false
}
