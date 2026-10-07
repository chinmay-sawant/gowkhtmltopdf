package css

import (
	"math"
	"strconv"
	"strings"
)

// Modern CSS Color 4 functions: oklab(), oklch(), color-mix(in srgb, ...)
// and light-dark(). They share ParseColor's return shape: RGB 0..255 and
// alpha 0..1, ok=false for anything unrecognized.

const (
	oklabPercentABScale = 0.4 // 100% = 0.4 for oklab a/b and oklch C
	oklabHueDeg         = 360
	oklabHalfTurn       = 2
	oklabChannelCount   = 3
	lightDarkArgCount   = 2
	colorMixEvenSplit   = 0.5
)

// Ottosson's Oklab -> linear sRGB matrices. The comments show the formula
// coefficient each constant stands for in oklabToRGB.
const (
	oklabLFromA     = 0.3963377774 // l = L + a*oklabLFromA + b*oklabLFromB
	oklabLFromB     = 0.2158037573
	oklabMFromA     = 0.1055613458 // m = L - a*oklabMFromA - b*oklabMFromB
	oklabMFromB     = 0.0638541728
	oklabSFromA     = 0.0894841775 // s = L - a*oklabSFromA - b*oklabSFromB
	oklabSFromB     = 1.2914855480
	oklabRedFromL   = 4.0767416621 // r = l*oklabRedFromL - m*oklabRedFromM + s*oklabRedFromS
	oklabRedFromM   = 3.3077115913
	oklabRedFromS   = 0.2309699292
	oklabGreenFromL = 1.2684380046 // g = -l*oklabGreenFromL + m*oklabGreenFromM - s*oklabGreenFromS
	oklabGreenFromM = 2.6097574011
	oklabGreenFromS = 0.3413193965
	oklabBlueFromL  = 0.0041960863 // b = -l*oklabBlueFromL - m*oklabBlueFromM + s*oklabBlueFromS
	oklabBlueFromM  = 0.7034186147
	oklabBlueFromS  = 1.7076147010
)

// sRGB transfer function constants (linear <-> gamma encoding).
const (
	srgbLinearThreshold = 0.0031308
	srgbLinearSlope     = 12.92
	srgbGammaScale      = 1.055
	srgbGammaExponent   = 2.4
	srgbGammaOffset     = 0.055
)

// parseModernColor dispatches the modern color functions. low is the
// lower-cased value; val keeps the original case for nested parsing.
func parseModernColor(val, low string) (int, int, int, float64, bool) {
	switch {
	case strings.HasPrefix(low, "oklch("):
		return parseOKLCHColor(val)
	case strings.HasPrefix(low, "oklab("):
		return parseOKLabColor(val)
	case strings.HasPrefix(low, "color-mix("):
		return parseColorMix(val)
	case strings.HasPrefix(low, "light-dark("):
		return parseLightDark(val)
	}

	return 0, 0, 0, 0, false
}

// colorFunctionBody returns the text between the first '(' and the last ')'.
func colorFunctionBody(val string) (string, bool) {
	open := strings.IndexByte(val, '(')
	closeIdx := strings.LastIndexByte(val, ')')

	if open < 0 || closeIdx <= open {
		return "", false
	}

	return strings.TrimSpace(val[open+1 : closeIdx]), true
}

// splitColorAlpha splits a space-separated channel list from an optional
// '/ alpha' tail at paren depth 0.
func splitColorAlpha(body string) (string, string, bool) {
	depth := 0

	for index := range len(body) {
		switch body[index] {
		case '(':
			depth++
		case ')':
			depth--
		case '/':
			if depth == 0 {
				return strings.TrimSpace(body[:index]), strings.TrimSpace(body[index+1:]), true
			}
		}
	}

	return body, "", false
}

// splitTopLevelCommas splits text on commas outside parentheses.
func splitTopLevelCommas(text string) []string {
	var parts []string

	depth, start := 0, 0

	for index := range len(text) {
		switch text[index] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(text[start:index]))

				start = index + 1
			}
		}
	}

	return append(parts, strings.TrimSpace(text[start:]))
}

func parseOKLabColor(val string) (int, int, int, float64, bool) {
	body, found := colorFunctionBody(val)
	if !found {
		return 0, 0, 0, 0, false
	}

	channels, alphaRaw, hasAlpha := splitColorAlpha(body)
	fields := strings.Fields(channels)

	if len(fields) != oklabChannelCount {
		return 0, 0, 0, 0, false
	}

	light, found := parseOKLabLight(fields[0])
	if !found {
		return 0, 0, 0, 0, false
	}

	aComp, found := parseOKLabAxis(fields[1])
	if !found {
		return 0, 0, 0, 0, false
	}

	bComp, found := parseOKLabAxis(fields[2])
	if !found {
		return 0, 0, 0, 0, false
	}

	alpha, found := parseModernAlpha(alphaRaw, hasAlpha)
	if !found {
		return 0, 0, 0, 0, false
	}

	red, green, blue := oklabToRGB(light, aComp, bComp)

	return red, green, blue, alpha, true
}

func parseOKLCHColor(val string) (int, int, int, float64, bool) {
	body, found := colorFunctionBody(val)
	if !found {
		return 0, 0, 0, 0, false
	}

	channels, alphaRaw, hasAlpha := splitColorAlpha(body)
	fields := strings.Fields(channels)

	if len(fields) != oklabChannelCount {
		return 0, 0, 0, 0, false
	}

	light, found := parseOKLabLight(fields[0])
	if !found {
		return 0, 0, 0, 0, false
	}

	chroma, found := parseOKLabAxis(fields[1])
	if !found {
		return 0, 0, 0, 0, false
	}

	if chroma < 0 {
		chroma = 0
	}

	hue, found := parseHueChannel(fields[2])
	if !found {
		return 0, 0, 0, 0, false
	}

	alpha, found := parseModernAlpha(alphaRaw, hasAlpha)
	if !found {
		return 0, 0, 0, 0, false
	}

	hue = math.Mod(hue, oklabHueDeg)
	if hue < 0 {
		hue += oklabHueDeg
	}

	rad := hue * math.Pi / (oklabHueDeg / oklabHalfTurn)

	red, green, blue := oklabToRGB(light, chroma*math.Cos(rad), chroma*math.Sin(rad))

	return red, green, blue, alpha, true
}

// parseOKLabLight parses oklab/oklch L: number 0..1 or percentage, clamped.
func parseOKLabLight(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return 0, false
		}

		return clampUnit(f / percentScale), true
	}

	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}

	return clampUnit(f), true
}

// parseOKLabAxis parses oklab a/b or oklch C: number, or percentage where
// 100% maps to 0.4.
func parseOKLabAxis(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return 0, false
		}

		return f / percentScale * oklabPercentABScale, true
	}

	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

// parseModernAlpha parses the optional '/ alpha' tail: number or percentage.
func parseModernAlpha(raw string, hasAlpha bool) (float64, bool) {
	if !hasAlpha {
		return 1, true
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}

	if strings.HasSuffix(raw, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return 0, false
		}

		return clampAlpha(f / percentScale), true
	}

	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}

	return clampAlpha(f), true
}

// oklabToRGB converts an Oklab triplet to sRGB bytes (Ottosson's matrices),
// clamping out-of-gamut results.
func oklabToRGB(light, aComp, bComp float64) (int, int, int) {
	lmsL := light + oklabLFromA*aComp + oklabLFromB*bComp
	lmsM := light - oklabMFromA*aComp - oklabMFromB*bComp
	lmsS := light - oklabSFromA*aComp - oklabSFromB*bComp

	lmsL = lmsL * lmsL * lmsL
	lmsM = lmsM * lmsM * lmsM
	lmsS = lmsS * lmsS * lmsS

	red := linearToSRGB(oklabRedFromL*lmsL - oklabRedFromM*lmsM + oklabRedFromS*lmsS)
	green := linearToSRGB(-oklabGreenFromL*lmsL + oklabGreenFromM*lmsM - oklabGreenFromS*lmsS)
	blue := linearToSRGB(-oklabBlueFromL*lmsL - oklabBlueFromM*lmsM + oklabBlueFromS*lmsS)

	return clampByte(red * maxRGBChannel),
		clampByte(green * maxRGBChannel),
		clampByte(blue * maxRGBChannel)
}

func linearToSRGB(channel float64) float64 {
	if channel <= srgbLinearThreshold {
		return srgbLinearSlope * channel
	}

	return srgbGammaScale*math.Pow(channel, 1.0/srgbGammaExponent) - srgbGammaOffset
}

type colorMixStop struct {
	red, green, blue int
	alpha            float64
	percent          float64
	hasPercent       bool
}

// zeroColorMixStop is the all-zero stop: no channels and no percentage.
func zeroColorMixStop() colorMixStop {
	return colorMixStop{
		red:        0,
		green:      0,
		blue:       0,
		alpha:      0,
		percent:    0,
		hasPercent: false,
	}
}

func parseColorMix(val string) (int, int, int, float64, bool) {
	body, ok := colorFunctionBody(val)
	if !ok {
		return 0, 0, 0, 0, false
	}

	parts := splitTopLevelCommas(body)
	if len(parts) != 3 || !strings.EqualFold(strings.Join(strings.Fields(parts[0]), " "), "in srgb") {
		return 0, 0, 0, 0, false
	}

	first, okFirst := parseColorMixStop(parts[1])
	second, okSecond := parseColorMixStop(parts[2])

	if !okFirst || !okSecond {
		return 0, 0, 0, 0, false
	}

	weightFirst, weightSecond := colorMixWeights(first, second)

	red, green, blue, alpha := mixPremultiplied(first, second, weightFirst, weightSecond)

	return red, green, blue, alpha, true
}

// parseColorMixStop parses '<color> <percentage>?'.
func parseColorMixStop(raw string) (colorMixStop, bool) {
	raw = strings.TrimSpace(raw)
	stop := zeroColorMixStop()

	if strings.HasSuffix(raw, "%") {
		space := strings.LastIndexByte(raw, ' ')
		if space < 0 {
			return zeroColorMixStop(), false
		}

		f, err := strconv.ParseFloat(strings.TrimSpace(raw[space+1:len(raw)-1]), 64)
		if err != nil {
			return zeroColorMixStop(), false
		}

		stop.percent = clampPercent(f)
		stop.hasPercent = true

		raw = strings.TrimSpace(raw[:space])
	}

	red, green, blue, alpha, ok := ParseColor(raw)
	if !ok {
		return zeroColorMixStop(), false
	}

	stop.red, stop.green, stop.blue, stop.alpha = red, green, blue, alpha

	return stop, true
}

// colorMixWeights normalizes the stop percentages: both omitted means an even
// split, one omitted takes the remainder, both given are scaled to sum to 100%.
func colorMixWeights(first, second colorMixStop) (float64, float64) {
	switch {
	case first.hasPercent && second.hasPercent:
		sum := first.percent + second.percent
		if sum == 0 {
			return 0, 0
		}

		return first.percent / sum, second.percent / sum
	case first.hasPercent:
		w := first.percent / percentScale

		return w, 1 - w
	case second.hasPercent:
		w := second.percent / percentScale

		return 1 - w, w
	default:
		return colorMixEvenSplit, colorMixEvenSplit
	}
}

func clampPercent(percent float64) float64 {
	if percent < 0 {
		return 0
	}

	if percent > percentScale {
		return percentScale
	}

	return percent
}

// mixPremultiplied blends two sRGB colors with premultiplied alpha, per the
// color-mix interpolation rules.
func mixPremultiplied(first, second colorMixStop, weightFirst, weightSecond float64) (int, int, int, float64) {
	outAlpha := first.alpha*weightFirst + second.alpha*weightSecond
	if outAlpha == 0 {
		return 0, 0, 0, 0
	}

	red := (float64(first.red)*first.alpha*weightFirst + float64(second.red)*second.alpha*weightSecond) / outAlpha
	green := (float64(first.green)*first.alpha*weightFirst + float64(second.green)*second.alpha*weightSecond) / outAlpha
	blue := (float64(first.blue)*first.alpha*weightFirst + float64(second.blue)*second.alpha*weightSecond) / outAlpha

	return clampByte(red), clampByte(green), clampByte(blue), clampAlpha(outAlpha)
}

// parseLightDark returns the light-scheme color. Both arguments must parse.
func parseLightDark(val string) (int, int, int, float64, bool) {
	body, ok := colorFunctionBody(val)
	if !ok {
		return 0, 0, 0, 0, false
	}

	parts := splitTopLevelCommas(body)
	if len(parts) != lightDarkArgCount {
		return 0, 0, 0, 0, false
	}

	lightR, lightG, lightB, lightA, okLight := ParseColor(parts[0])
	darkR, darkG, darkB, darkA, okDark := ParseColor(parts[1])

	if !okLight || !okDark {
		return 0, 0, 0, 0, false
	}

	// The engine always renders the light scheme; the dark channels are
	// parsed only to reject a malformed pair.
	_ = darkR
	_ = darkG
	_ = darkB
	_ = darkA

	return lightR, lightG, lightB, lightA, true
}
