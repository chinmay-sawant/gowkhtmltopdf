package layout

import (
	"errors"
	"math"
	"testing"
)

// TestValidatePaintPageIndicesStillRejects pins the single validation point
// that replaced the three per-Paint scans: an out-of-range op Y and an
// invalid content height must still fail instead of writing an empty PDF.
func TestValidatePaintPageIndicesStillRejects(t *testing.T) {
	t.Parallel()

	contentH := 100.0

	res := layoutHTML(t, `<html><body><p>alpha</p></body></html>`)

	res.Ops[0].Y = float64(maxFlowPageIndex+1) * contentH

	if err := validatePaintPageIndices(res.Ops, contentH); !errors.Is(err, errOutOfRangePageIndex) {
		t.Fatalf("out-of-range Y error = %v, want errOutOfRangePageIndex", err)
	}

	for _, bad := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if err := validatePaintPageIndices(nil, bad); !errors.Is(err, errInvalidContentHeight) {
			t.Fatalf("contentH %v error = %v, want errInvalidContentHeight", bad, err)
		}
	}
}
