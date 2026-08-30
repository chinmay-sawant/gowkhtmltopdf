//nolint:cyclop,funlen,gocyclo,mnd,wsl,nlreturn,varnamelen,lll,goconst // centralized property dispatch for Wave 80.5
package layout

import (
	"math"
	"strconv"
	"strings"
)

// applyLeftoversProps handles SVG presentation and individual transform properties.
func applyLeftoversProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
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
	case "rotate":
		applyRotateProperty(style, value)
	case "scale":
		applyScaleProperty(style, value)
	case "translate":
		applyTranslateProperty(style, value, fsize)

	default:
		return false
	}

	return true
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
