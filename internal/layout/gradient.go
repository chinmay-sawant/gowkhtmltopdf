//nolint:varnamelen,funlen,cyclop,mnd,wsl,intrange,nlreturn,nestif,gocognit,gocritic,nonamedreturns // grad
package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

type gradientStop struct {
	r, g, b float64
	a       float64
	pos     float64 // 0..1
	hasPos  bool
}

type linearGradientSpec struct {
	angleDeg  float64
	stops     []gradientStop
	repeating bool
}

type radialGradientSpec struct {
	cx, cy    float64
	rx, ry    float64
	stops     []gradientStop
	repeating bool
}

func isGradientFunc(val string) bool {
	low := strings.ToLower(strings.TrimSpace(val))
	return strings.HasPrefix(low, "linear-gradient(") ||
		strings.HasPrefix(low, "repeating-linear-gradient(") ||
		strings.HasPrefix(low, "radial-gradient(") ||
		strings.HasPrefix(low, "repeating-radial-gradient(")
}

func parseGradientStops(parts []string, current [3]float64) []gradientStop {
	var rawStops []gradientStop
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			continue
		}

		// Look for color in tokens
		var (
			colR, colG, colB = current[0], current[1], current[2]
			colA             = 1.0
			pos              = 0.0
			hasPos           = false
			colorFound       = false
		)

		for i := 0; i < len(tokens); i++ {
			tok := tokens[i]
			// Try color
			if r, g, b, a, ok := css.ParseColor(tok); ok {
				colR = float64(r) / 255.0
				colG = float64(g) / 255.0
				colB = float64(b) / 255.0
				colA = a
				colorFound = true
				continue
			}
			if isCurrentColor(tok) {
				colR, colG, colB = current[0], current[1], current[2]
				colA = 1.0
				colorFound = true
				continue
			}

			// Try percentage or length
			if strings.HasSuffix(tok, "%") {
				if v, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64); err == nil {
					pos = v / 100.0
					hasPos = true
				}
				continue
			}
			if v, ok := plainLength(tok, 12, 0); ok {
				pos = v // length in points
				hasPos = true
				continue
			}
		}

		if colorFound || hasPos {
			rawStops = append(rawStops, gradientStop{
				r: colR, g: colG, b: colB, a: colA,
				pos: pos, hasPos: hasPos,
			})
		}
	}

	if len(rawStops) == 0 {
		return nil
	}
	if len(rawStops) == 1 {
		rawStops = append(rawStops, rawStops[0])
		rawStops[0].pos = 0
		rawStops[0].hasPos = true
		rawStops[1].pos = 1
		rawStops[1].hasPos = true
		return rawStops
	}

	// Normalize first and last stop
	if !rawStops[0].hasPos {
		rawStops[0].pos = 0
		rawStops[0].hasPos = true
	}
	n := len(rawStops)
	if !rawStops[n-1].hasPos {
		rawStops[n-1].pos = 1
		rawStops[n-1].hasPos = true
	}

	// Fill in unpositioned intermediate stops
	lastPosIdx := 0
	for i := 1; i < n; i++ {
		if rawStops[i].hasPos {
			if i > lastPosIdx+1 {
				startPos := rawStops[lastPosIdx].pos
				endPos := rawStops[i].pos
				count := float64(i - lastPosIdx)
				for j := lastPosIdx + 1; j < i; j++ {
					fraction := float64(j-lastPosIdx) / count
					rawStops[j].pos = startPos + fraction*(endPos-startPos)
					rawStops[j].hasPos = true
				}
			}
			lastPosIdx = i
		}
	}

	return rawStops
}

func parseLinearGradient(raw string, current [3]float64) (*linearGradientSpec, bool) {
	raw = strings.TrimSpace(raw)
	low := strings.ToLower(raw)
	repeating := false
	var args string
	if strings.HasPrefix(low, "repeating-linear-gradient(") {
		repeating = true
		args = raw[len("repeating-linear-gradient(") : len(raw)-1]
	} else if strings.HasPrefix(low, "linear-gradient(") {
		args = raw[len("linear-gradient(") : len(raw)-1]
	} else {
		return nil, false
	}

	parts := splitFunctionArgs(args)
	if len(parts) == 0 {
		return nil, false
	}

	angle := 180.0 // default to bottom
	firstPart := strings.TrimSpace(parts[0])
	firstLow := strings.ToLower(firstPart)
	isDirection := false

	if strings.HasPrefix(firstLow, "to ") {
		isDirection = true
		dir := strings.TrimSpace(firstLow[3:])
		switch dir {
		case "top":
			angle = 0
		case "right":
			angle = 90
		case "bottom":
			angle = 180
		case "left":
			angle = 270
		case "top right", "right top":
			angle = 45
		case "bottom right", "right bottom":
			angle = 135
		case "bottom left", "left bottom":
			angle = 225
		case "top left", "left top":
			angle = 315
		}
	} else if deg, ok := parseAngleDeg(firstPart); ok {
		angle = deg
		isDirection = true
	}

	stopParts := parts
	if isDirection {
		stopParts = parts[1:]
	}

	stops := parseGradientStops(stopParts, current)
	if len(stops) < 2 {
		return nil, false
	}

	return &linearGradientSpec{
		angleDeg:  angle,
		stops:     stops,
		repeating: repeating,
	}, true
}

func parseRadialGradient(raw string, current [3]float64) (*radialGradientSpec, bool) {
	raw = strings.TrimSpace(raw)
	low := strings.ToLower(raw)
	repeating := false
	var args string
	if strings.HasPrefix(low, "repeating-radial-gradient(") {
		repeating = true
		args = raw[len("repeating-radial-gradient(") : len(raw)-1]
	} else if strings.HasPrefix(low, "radial-gradient(") {
		args = raw[len("radial-gradient(") : len(raw)-1]
	} else {
		return nil, false
	}

	parts := splitFunctionArgs(args)
	if len(parts) == 0 {
		return nil, false
	}

	cx, cy := 0.5, 0.5
	rx, ry := 0.5, 0.5
	stopParts := parts

	firstPart := strings.TrimSpace(parts[0])
	firstLow := strings.ToLower(firstPart)
	if strings.Contains(firstLow, "circle") || strings.Contains(firstLow, "ellipse") || strings.Contains(firstLow, "at ") {
		stopParts = parts[1:]
		if atIdx := strings.Index(firstLow, "at "); atIdx >= 0 {
			posStr := strings.TrimSpace(firstLow[atIdx+3:])
			posParts := strings.Fields(posStr)
			if len(posParts) >= 1 {
				cx = parseGradientPosComponent(posParts[0])
			}
			if len(posParts) >= 2 {
				cy = parseGradientPosComponent(posParts[1])
			}
		}
	}

	stops := parseGradientStops(stopParts, current)
	if len(stops) < 2 {
		return nil, false
	}

	return &radialGradientSpec{
		cx: cx, cy: cy, rx: rx, ry: ry,
		stops: stops, repeating: repeating,
	}, true
}

func parseGradientPosComponent(tok string) float64 {
	switch tok {
	case "left", "top":
		return 0
	case "center":
		return 0.5
	case "right", "bottom":
		return 1.0
	}
	if strings.HasSuffix(tok, "%") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64); err == nil {
			return v / 100.0
		}
	}
	return 0.5
}

func splitFunctionArgs(args string) []string {
	return splitParenArgs(args, ',')
}

func sampleStops(stops []gradientStop, t float64) (r, g, b, a float64) {
	if len(stops) == 0 {
		return 0, 0, 0, 0
	}
	if t <= stops[0].pos {
		return stops[0].r, stops[0].g, stops[0].b, stops[0].a
	}
	n := len(stops)
	if t >= stops[n-1].pos {
		return stops[n-1].r, stops[n-1].g, stops[n-1].b, stops[n-1].a
	}

	for i := 0; i < n-1; i++ {
		if t >= stops[i].pos && t <= stops[i+1].pos {
			span := stops[i+1].pos - stops[i].pos
			if span <= 0 {
				return stops[i+1].r, stops[i+1].g, stops[i+1].b, stops[i+1].a
			}
			f := (t - stops[i].pos) / span
			return stops[i].r + f*(stops[i+1].r-stops[i].r),
				stops[i].g + f*(stops[i+1].g-stops[i].g),
				stops[i].b + f*(stops[i+1].b-stops[i].b),
				stops[i].a + f*(stops[i+1].a-stops[i].a)
		}
	}

	return stops[n-1].r, stops[n-1].g, stops[n-1].b, stops[n-1].a
}

func renderGradientPNG(raw string, width, height float64, current [3]float64) ([]byte, int, int, bool) {
	if width <= 0 || height <= 0 {
		return nil, 0, 0, false
	}

	const maxGradDim = 512

	imgW := int(math.Ceil(width))
	imgH := int(math.Ceil(height))

	if imgW < 1 {
		imgW = 1
	}

	if imgH < 1 {
		imgH = 1
	}

	if imgW > maxGradDim {
		imgW = maxGradDim
	}

	if imgH > maxGradDim {
		imgH = maxGradDim
	}

	if lin, ok := parseLinearGradient(raw, current); ok {
		return rasterizeLinearGradient(lin, imgW, imgH)
	}

	if rad, ok := parseRadialGradient(raw, current); ok {
		return rasterizeRadialGradient(rad, imgW, imgH)
	}

	return nil, 0, 0, false
}

const (
	halfPixel      = 0.5
	degToRadFactor = 180.0
	maxRGBFloat    = 255.0
)

func rasterizeLinearGradient(lin *linearGradientSpec, imgW, imgH int) ([]byte, int, int, bool) {
	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	rad := lin.angleDeg * math.Pi / degToRadFactor
	sinA := math.Sin(rad)
	cosA := math.Cos(rad)

	wF := float64(imgW)
	hF := float64(imgH)
	centerX := wF / 2.0
	centerY := hF / 2.0
	length := math.Abs(wF*sinA) + math.Abs(hF*cosA)

	if length <= 0 {
		length = 1
	}

	firstPos := lin.stops[0].pos
	lastPos := lin.stops[len(lin.stops)-1].pos
	period := lastPos - firstPos

	if period <= 0 {
		period = 1
	}

	for y := range imgH {
		dy := (float64(y) + halfPixel) - centerY

		for x := range imgW {
			dx := (float64(x) + halfPixel) - centerX
			proj := dx*sinA - dy*cosA
			t := halfPixel + proj/length

			if lin.repeating {
				t = math.Mod(t-firstPos, period)
				if t < 0 {
					t += period
				}

				t += firstPos
			}

			cr, cg, cb, ca := sampleStops(lin.stops, t)
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(clamp01(cr) * maxRGBFloat)),
				G: uint8(math.Round(clamp01(cg) * maxRGBFloat)),
				B: uint8(math.Round(clamp01(cb) * maxRGBFloat)),
				A: uint8(math.Round(clamp01(ca) * maxRGBFloat)),
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, false
	}

	return buf.Bytes(), imgW, imgH, true
}

func rasterizeRadialGradient(rad *radialGradientSpec, imgW, imgH int) ([]byte, int, int, bool) {
	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	wF := float64(imgW)
	hF := float64(imgH)
	centerX := wF * rad.cx
	centerY := hF * rad.cy
	radiusX := wF * rad.rx
	radiusY := hF * rad.ry

	if radiusX <= 0 {
		radiusX = 1
	}

	if radiusY <= 0 {
		radiusY = 1
	}

	firstPos := rad.stops[0].pos
	lastPos := rad.stops[len(rad.stops)-1].pos
	period := lastPos - firstPos

	if period <= 0 {
		period = 1
	}

	for y := range imgH {
		dy := ((float64(y) + halfPixel) - centerY) / radiusY

		for x := range imgW {
			dx := ((float64(x) + halfPixel) - centerX) / radiusX
			dist := math.Sqrt(dx*dx + dy*dy)
			t := dist

			if rad.repeating {
				t = math.Mod(t-firstPos, period)
				if t < 0 {
					t += period
				}

				t += firstPos
			}

			cr, cg, cb, ca := sampleStops(rad.stops, t)
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(clamp01(cr) * maxRGBFloat)),
				G: uint8(math.Round(clamp01(cg) * maxRGBFloat)),
				B: uint8(math.Round(clamp01(cb) * maxRGBFloat)),
				A: uint8(math.Round(clamp01(ca) * maxRGBFloat)),
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, false
	}

	return buf.Bytes(), imgW, imgH, true
}
