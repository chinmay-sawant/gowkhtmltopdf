//nolint:wsl,cyclop,mnd,nlreturn // compositing equations follow the CSS color formulas directly
package layout

import (
	"math"
	"strings"
)

const (
	blendNormal     = "normal"
	blendMultiply   = "multiply"
	blendScreen     = "screen"
	blendOverlay    = "overlay"
	blendDarken     = "darken"
	blendLighten    = "lighten"
	blendColorDodge = "color-dodge"
	blendColorBurn  = "color-burn"
	blendHardLight  = "hard-light"
	blendSoftLight  = "soft-light"
	blendDifference = "difference"
	blendExclusion  = "exclusion"
	blendHue        = "hue"
	blendSaturation = "saturation"
	blendColor      = "color"
	blendLuminosity = "luminosity"
)

var supportedBlendModes = map[string]struct{}{ //nolint:gochecknoglobals // immutable CSS mode vocabulary
	blendNormal:     {},
	blendMultiply:   {},
	blendScreen:     {},
	blendOverlay:    {},
	blendDarken:     {},
	blendLighten:    {},
	blendColorDodge: {},
	blendColorBurn:  {},
	blendHardLight:  {},
	blendSoftLight:  {},
	blendDifference: {},
	blendExclusion:  {},
	blendHue:        {},
	blendSaturation: {},
	blendColor:      {},
	blendLuminosity: {},
}

func normalizeBlendMode(value string) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(value))
	if _, ok := supportedBlendModes[mode]; !ok {
		return "", false
	}

	return mode, true
}

func backgroundBlendModeForLayer(value string, layerIndex int) string {
	parts := splitCommaLayers(value)
	if len(parts) == 0 {
		return blendNormal
	}
	if layerIndex < 0 {
		layerIndex = 0
	}
	if layerIndex >= len(parts) {
		layerIndex = len(parts) - 1
	}

	mode, ok := normalizeBlendMode(parts[layerIndex])
	if !ok {
		return blendNormal
	}

	return mode
}

// BlendColor returns the unpremultiplied RGB result of blending source over
// backdrop. Alpha compositing is performed by the image output adapter.
func BlendColor(mode string, backdrop, source [3]float64) [3]float64 {
	backdrop = clampColor(backdrop)
	source = clampColor(source)

	switch mode {
	case blendMultiply:
		return separableColor(backdrop, source, func(cb, cs float64) float64 { return cb * cs })
	case blendScreen:
		return separableColor(backdrop, source, func(cb, cs float64) float64 {
			return cb + cs - cb*cs
		})
	case blendOverlay:
		return separableColor(backdrop, source, overlayChannel)
	case blendDarken:
		return separableColor(backdrop, source, math.Min)
	case blendLighten:
		return separableColor(backdrop, source, math.Max)
	case blendColorDodge:
		return separableColor(backdrop, source, colorDodgeChannel)
	case blendColorBurn:
		return separableColor(backdrop, source, colorBurnChannel)
	case blendHardLight:
		return separableColor(backdrop, source, func(cb, cs float64) float64 {
			return overlayChannel(cs, cb)
		})
	case blendSoftLight:
		return separableColor(backdrop, source, softLightChannel)
	case blendDifference:
		return separableColor(backdrop, source, func(cb, cs float64) float64 {
			return math.Abs(cb - cs)
		})
	case blendExclusion:
		return separableColor(backdrop, source, func(cb, cs float64) float64 {
			return cb + cs - 2*cb*cs
		})
	case blendHue:
		return setLum(setSat(source, saturation(backdrop)), luminance(backdrop))
	case blendSaturation:
		return setLum(setSat(backdrop, saturation(source)), luminance(backdrop))
	case blendColor:
		return setLum(source, luminance(backdrop))
	case blendLuminosity:
		return setLum(backdrop, luminance(source))
	case blendNormal:
		fallthrough
	default:
		return source
	}
}

func separableColor(backdrop, source [3]float64, channel func(float64, float64) float64) [3]float64 {
	return [3]float64{
		clamp01(channel(backdrop[0], source[0])),
		clamp01(channel(backdrop[1], source[1])),
		clamp01(channel(backdrop[2], source[2])),
	}
}

func overlayChannel(backdrop, source float64) float64 {
	if backdrop <= 0.5 {
		return 2 * backdrop * source
	}

	return 1 - 2*(1-backdrop)*(1-source)
}

func colorDodgeChannel(backdrop, source float64) float64 {
	if source >= 1 {
		return 1
	}

	return math.Min(1, backdrop/(1-source))
}

func colorBurnChannel(backdrop, source float64) float64 {
	if source <= 0 {
		return 0
	}

	return 1 - math.Min(1, (1-backdrop)/source)
}

func softLightChannel(backdrop, source float64) float64 {
	if source <= 0.5 {
		return backdrop - (1-2*source)*backdrop*(1-backdrop)
	}

	d := softLightD(backdrop)
	return backdrop + (2*source-1)*(d-backdrop)
}

func softLightD(backdrop float64) float64 {
	if backdrop <= 0.25 {
		return ((16*backdrop-12)*backdrop + 4) * backdrop
	}

	return math.Sqrt(backdrop)
}

func luminance(value [3]float64) float64 {
	return 0.3*value[0] + 0.59*value[1] + 0.11*value[2]
}

func saturation(value [3]float64) float64 {
	minValue := math.Min(value[0], math.Min(value[1], value[2]))
	maxValue := math.Max(value[0], math.Max(value[1], value[2]))

	return maxValue - minValue
}

func setSat(value [3]float64, target float64) [3]float64 {
	minIndex, maxIndex := 0, 0
	for index := 1; index < len(value); index++ {
		if value[index] < value[minIndex] {
			minIndex = index
		}
		if value[index] > value[maxIndex] {
			maxIndex = index
		}
	}

	if minIndex == maxIndex {
		return [3]float64{}
	}

	middleIndex := 3 - minIndex - maxIndex
	minValue, maxValue := value[minIndex], value[maxIndex]
	span := maxValue - minValue
	result := [3]float64{}
	result[middleIndex] = (value[middleIndex] - minValue) * target / span
	result[maxIndex] = target

	return result
}

func setLum(value [3]float64, target float64) [3]float64 {
	delta := target - luminance(value)
	for index := range value {
		value[index] += delta
	}

	return clipColor(value)
}

func clipColor(value [3]float64) [3]float64 {
	lum := luminance(value)
	minValue := math.Min(value[0], math.Min(value[1], value[2]))
	maxValue := math.Max(value[0], math.Max(value[1], value[2]))

	if minValue < 0 && lum != minValue {
		for index := range value {
			value[index] = lum + (value[index]-lum)*lum/(lum-minValue)
		}
	}
	if maxValue > 1 && maxValue != lum {
		for index := range value {
			value[index] = lum + (value[index]-lum)*(1-lum)/(maxValue-lum)
		}
	}

	return clampColor(value)
}

func clampColor(value [3]float64) [3]float64 {
	for index := range value {
		value[index] = clamp01(value[index])
	}

	return value
}
