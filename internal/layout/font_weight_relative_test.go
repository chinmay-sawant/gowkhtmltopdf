package layout

import (
	"testing"
)

// CSS Fonts 3 relative font weights: bolder and lighter are a mapping table,
// not current +/- 100 (tutorialspoint-cpp finding 2: bolder from 400 must
// resolve to 700 so <b>/<strong> paint bold).
func TestResolveFontWeightRelativeKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		current int
		val     string
		want    int
	}{
		{100, "bolder", 400},
		{200, "bolder", 400},
		{300, "bolder", 400},
		{400, "bolder", 700},
		{500, "bolder", 700},
		{600, "bolder", 900},
		{700, "bolder", 900},
		{800, "bolder", 900},
		{900, "bolder", 900},
		{100, "lighter", 100},
		{200, "lighter", 100},
		{300, "lighter", 100},
		{400, "lighter", 100},
		{500, "lighter", 100},
		{600, "lighter", 400},
		{700, "lighter", 400},
		{800, "lighter", 700},
		{900, "lighter", 700},
	}

	for _, tc := range cases {
		if got := resolveFontWeight(tc.current, tc.val); got != tc.want {
			t.Errorf("resolveFontWeight(%d, %q) = %d, want %d", tc.current, tc.val, got, tc.want)
		}
	}
}
