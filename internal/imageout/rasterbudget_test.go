package imageout

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestRasterDimensionBudgetErrorIsActionable checks the enriched dimension
// failure: CSS pixels, the named limit, the wrapped sentinel, and the floored
// fitting-zoom bound when one applies.
func TestRasterDimensionBudgetErrorIsActionable(t *testing.T) {
	t.Parallel()

	// 22415 supersampled px is the complex-css height: CSS 11208 px against
	// the 16384 px height limit at the default zoom.
	_, err := rasterDimension(22415, "maxRasterHeight", rasterBudget{remedy: true})
	if err == nil {
		t.Fatal("rasterDimension accepted an over-limit height")
	}

	if !errors.Is(err, errRasterTooLarge) {
		t.Fatalf("error %v does not wrap errRasterTooLarge", err)
	}

	message := err.Error()
	for _, want := range []string{
		"raster exceeds resource budget",
		"dimension 22415 px",
		"CSS 11208 px",
		"maxRasterHeight 16384 px",
		"--zoom <= 0.73",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("height error %q does not contain %q", message, want)
		}
	}

	// The viewport-fixed width gets the CSS size and limit but no zoom hint:
	// a smaller zoom cannot shrink it.
	_, err = rasterDimension(20000, "maxRasterWidth", rasterBudget{})
	if err == nil {
		t.Fatal("rasterDimension accepted an over-limit width")
	}

	message = err.Error()
	for _, want := range []string{"CSS 10000 px", "maxRasterWidth 16384 px"} {
		if !strings.Contains(message, want) {
			t.Errorf("width error %q does not contain %q", message, want)
		}
	}

	if strings.Contains(message, "--zoom") {
		t.Errorf("width error %q offers a zoom remedy it cannot honor", message)
	}

	_, err = rasterDimension(math.NaN(), "maxRasterWidth", rasterBudget{})
	if !errors.Is(err, errRasterTooLarge) {
		t.Fatalf("non-finite error = %v, want errRasterTooLarge", err)
	}
}

// TestRasterCanvasBudgetErrorIsActionable checks the pixel-budget failure:
// the canvas in supersampled and CSS pixels, the named limit, and the floored
// zoom bound with and without a caller zoom.
func TestRasterCanvasBudgetErrorIsActionable(t *testing.T) {
	t.Parallel()

	// 2048x48856 supersampled is the fixture-60 canvas: CSS 1024x24428.
	err := validateRasterCanvas(2048, 48856, rasterBudget{remedy: true})
	if err == nil {
		t.Fatal("validateRasterCanvas accepted an over-budget canvas")
	}

	if !errors.Is(err, errRasterTooLarge) {
		t.Fatalf("error %v does not wrap errRasterTooLarge", err)
	}

	message := err.Error()
	for _, want := range []string{
		"raster exceeds resource budget",
		"2048x48856 px",
		"CSS 1024x24428 px",
		"maxRasterPixels 67108864 px",
		"--zoom <= 0.67",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("pixel error %q does not contain %q", message, want)
		}
	}

	err = validateRasterCanvas(2048, 48856, rasterBudget{zoom: 2, remedy: true})
	if err == nil {
		t.Fatal("validateRasterCanvas accepted an over-budget canvas at zoom 2")
	}

	if !strings.Contains(err.Error(), "--zoom <= 1.34") {
		t.Errorf("zoom-2 pixel error %q does not contain the scaled bound %q", err, "--zoom <= 1.34")
	}

	if err := validateRasterCanvas(1024, 1024, rasterBudget{remedy: true}); err != nil {
		t.Fatalf("validateRasterCanvas rejected an in-budget canvas: %v", err)
	}
}

// TestRasterizeContextBudgetErrorReportsCSSAndZoom proves the plumbing: the
// context dimensions reach the error message as CSS pixels and the caller's
// zoom scales the suggested bound.
func TestRasterizeContextBudgetErrorReportsCSSAndZoom(t *testing.T) {
	t.Parallel()

	res := &layout.Result{Width: 100}

	_, err := rasterizeContext(t.Context(), res, 20000, false, 0, 0)
	if err == nil {
		t.Fatal("rasterizeContext accepted an over-limit height")
	}

	if !errors.Is(err, errRasterTooLarge) {
		t.Fatalf("error %v does not wrap errRasterTooLarge", err)
	}

	message := err.Error()
	for _, want := range []string{
		"raster exceeds resource budget",
		"maxRasterHeight 16384 px",
		"CSS 26667 px",
		"--zoom <= 0.30",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("context error %q does not contain %q", message, want)
		}
	}

	_, err = rasterizeContext(t.Context(), res, 20000, false, 0, 2)
	if err == nil {
		t.Fatal("rasterizeContext accepted an over-limit height at zoom 2")
	}

	if !strings.Contains(err.Error(), "--zoom <= 0.61") {
		t.Errorf("zoom-2 context error %q does not contain the scaled bound %q", err, "--zoom <= 0.61")
	}
}
