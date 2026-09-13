package imageout

import (
	"fmt"
	"math"
)

const (
	// fitZoomEpsilon absorbs float division noise so a bound that is exactly
	// on a hundredth does not floor one step lower.
	fitZoomEpsilon = 1e-9
	// fitZoomScale converts the ratio to a two-decimal percentage.
	fitZoomScale = 100
	// minFitZoom is the smallest zoom bound worth suggesting as a remedy.
	minFitZoom = 0.01
)

// rasterBudget is the zoom context a supersampled-canvas budget error needs.
// A zero zoom means the layout default of 1; remedy marks measurements a
// smaller layout zoom can shrink (height and pixel area, not the fixed
// viewport width).
type rasterBudget struct {
	zoom   float64
	remedy bool
}

// effectiveZoom returns the zoom the layout used, mapping 0 to 1.
func (b rasterBudget) effectiveZoom() float64 {
	if b.zoom > 0 {
		return b.zoom
	}

	return 1
}

// fitZoom floors to two decimals the zoom whose linear scale brings measured
// down to limit. ok is false when no remedy applies or the bound would be
// zero or meaningless.
func (b rasterBudget) fitZoom(measured, limit float64) (float64, bool) {
	if !b.remedy || measured <= 0 || limit <= 0 || limit >= measured {
		return 0, false
	}

	fit := math.Floor(b.effectiveZoom()*limit/measured*fitZoomScale+fitZoomEpsilon) / fitZoomScale
	if fit < minFitZoom {
		return 0, false
	}

	return fit, true
}

// cssDimension converts a supersampled canvas measurement in one axis back to
// CSS pixels (supersampled/rasterSS).
func cssDimension(canvasPx float64) int {
	return int(math.Round(canvasPx / rasterSS))
}

// rasterDimension checks one supersampled canvas dimension against the
// maxRasterWidth limit, rounding the way the canvas allocation does. A failure
// names the limit, reports the CSS-pixel size, and adds a deterministic --zoom
// bound when the budget carries a zoom remedy.
func rasterDimension(value float64, limitName string, budget rasterBudget) (int, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%w: non-finite dimension", errRasterTooLarge)
	}

	if value <= 0 {
		return 1, nil
	}

	rounded := math.Round(value)
	if rounded < 1 {
		return 1, nil
	}

	if rounded <= float64(maxRasterWidth) {
		return int(rounded), nil
	}

	message := fmt.Sprintf(
		"dimension %.0f px (CSS %d px) exceeds %s %d px",
		rounded,
		cssDimension(rounded),
		limitName,
		maxRasterWidth,
	)

	if fit, ok := budget.fitZoom(rounded, float64(maxRasterWidth)); ok {
		message += fmt.Sprintf("; try --zoom <= %.2f", fit)
	}

	return 0, fmt.Errorf("%w: %s", errRasterTooLarge, message)
}

// validateRasterSize checks a final-resolution image against the raster
// budgets. newRasterImage and onWhite call it with output pixels, not
// supersampled canvas pixels, so its messages name the limits without a CSS
// conversion.
func validateRasterSize(width, height int) error {
	if width < 1 || height < 1 || width > maxRasterWidth || height > maxRasterHeight {
		return fmt.Errorf(
			"%w: dimensions %dx%d px exceed maxRasterWidth %d px / maxRasterHeight %d px",
			errRasterTooLarge,
			width,
			height,
			maxRasterWidth,
			maxRasterHeight,
		)
	}

	pixels := int64(width) * int64(height)
	if pixels > maxRasterPixels || pixels*4 > maxRasterBytes {
		return fmt.Errorf(
			"%w: %d px exceed maxRasterPixels %d px",
			errRasterTooLarge,
			pixels,
			maxRasterPixels,
		)
	}

	return nil
}

// validateRasterCanvas checks the supersampled paint canvas after both
// dimensions passed rasterDimension. Its failure names the pixel limit,
// reports the canvas in supersampled and CSS pixels, and adds a
// deterministic --zoom bound when the budget carries a zoom remedy.
func validateRasterCanvas(width, height int, budget rasterBudget) error {
	pixels := int64(width) * int64(height)
	if pixels <= maxRasterPixels && pixels*4 <= maxRasterBytes {
		return nil
	}

	message := fmt.Sprintf(
		"%dx%d px (CSS %dx%d px) exceeds maxRasterPixels %d px",
		width,
		height,
		cssDimension(float64(width)),
		cssDimension(float64(height)),
		maxRasterPixels,
	)

	if fit, ok := budget.fitZoom(float64(pixels), float64(maxRasterPixels)); ok {
		message += fmt.Sprintf("; try --zoom <= %.2f", fit)
	}

	return fmt.Errorf("%w: %s", errRasterTooLarge, message)
}
