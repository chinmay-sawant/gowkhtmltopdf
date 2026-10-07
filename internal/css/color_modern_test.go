package css

import "testing"

func TestNamedColorTableComplete(t *testing.T) {
	t.Parallel()

	checkNamedColorTable(t)

	cases := []struct {
		name    string
		r, g, b int
	}{
		{"aliceblue", 240, 248, 255},
		{"yellowgreen", 154, 205, 50},
		{"mediumspringgreen", 0, 250, 154},
		{"darkslategray", 47, 79, 79},
		{"rebeccapurple", 102, 51, 153},
		{"grey", 128, 128, 128},
		{"gray", 128, 128, 128},
	}

	for _, tc := range cases {
		r, g, b, a, ok := ParseColor(tc.name)
		if !ok || r != tc.r || g != tc.g || b != tc.b || a != 1 {
			t.Errorf("ParseColor(%q) = (%d,%d,%d,%v,%v), want (%d,%d,%d,1,true)",
				tc.name, r, g, b, a, ok, tc.r, tc.g, tc.b)
		}
	}
}

// checkNamedColorTable validates the cached named-color table: complete for
// CSS Color 4 and every channel inside the byte range.
func checkNamedColorTable(t *testing.T) {
	t.Helper()

	if got := len(namedColorTable); got != 148 {
		t.Fatalf("namedColorTable has %d entries, want the 148 CSS Color 4 names", got)
	}

	for name, rgb := range namedColorTable {
		for _, ch := range rgb {
			if ch < 0 || ch > maxRGBChannel {
				t.Errorf("named color %q has channel %d outside 0..%d", name, ch, maxRGBChannel)
			}
		}
	}
}

func TestParseColorOKLab(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src     string
		r, g, b int
		alpha   float64
		ok      bool
	}{
		// Spec reference pairs: sRGB red and green in Oklab.
		{"oklch(0.627955 0.257683 29.2339)", 255, 0, 0, 1, true},
		{"oklab(0.627955 0.224863 0.125846)", 255, 0, 0, 1, true},
		{"oklab(0.86644 -0.23389 0.1795)", 0, 255, 0, 1, true},
		{"oklch(1 0 0)", 255, 255, 255, 1, true},
		{"oklch(0 0 0)", 0, 0, 0, 1, true},
		{"oklch(62.7955% 0.257683 29.2339)", 255, 0, 0, 1, true},
		{"oklch(1 0 0 / 0.5)", 255, 255, 255, 0.5, true},
		{"oklch(1 0 0 / 50%)", 255, 255, 255, 0.5, true},
		{"OKLCH(1 0 0)", 255, 255, 255, 1, true},
		{"oklch(0.7 0.1 150)", 111, 176, 125, 1, true},
		{"oklab(50% 25% -25%)", 129, 69, 154, 1, true},
		{"oklch(0.5 0.1)", 0, 0, 0, 0, false},
		{"oklab(0.5)", 0, 0, 0, 0, false},
		{"oklch(1 0 0 /)", 0, 0, 0, 0, false},
		{"oklch(1 0 0 / x)", 0, 0, 0, 0, false},
		{"oklch(nope 0 0)", 0, 0, 0, 0, false},
	}

	checkColorCases(t, cases)
}

func TestParseColorMix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src     string
		r, g, b int
		alpha   float64
		ok      bool
	}{
		{"color-mix(in srgb, red, blue)", 128, 0, 128, 1, true},
		{"color-mix(in srgb, red 50%, blue 50%)", 128, 0, 128, 1, true},
		// 20/30 normalize to 40/60.
		{"color-mix(in srgb, red 20%, blue 30%)", 102, 0, 153, 1, true},
		// One percentage omitted takes the remainder.
		{"color-mix(in srgb, red 25%, blue)", 64, 0, 191, 1, true},
		{"color-mix(in srgb, red, blue 75%)", 64, 0, 191, 1, true},
		// Premultiplied alpha: (85,0,170) at 0.75, not the naive (128,0,128).
		{"color-mix(in srgb, rgba(255,0,0,0.5) 50%, blue 50%)", 85, 0, 170, 0.75, true},
		{"color-mix(in srgb, rgb(0, 128, 0) 10%, rgb(0, 0, 255) 90%)", 0, 13, 230, 1, true},
		{"COLOR-MIX(IN SRGB, red, blue)", 128, 0, 128, 1, true},
		{"color-mix(in oklab, red, blue)", 0, 0, 0, 0, false},
		{"color-mix(in srgb, red)", 0, 0, 0, 0, false},
		{"color-mix(in srgb, red, blue, green)", 0, 0, 0, 0, false},
		{"color-mix(in srgb, notacolor, blue)", 0, 0, 0, 0, false},
		{"color-mix(in srgb, red 200%, blue -50%)", 255, 0, 0, 1, true},
	}

	checkColorCases(t, cases)
}

func TestParseColorLightDark(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src     string
		r, g, b int
		alpha   float64
		ok      bool
	}{
		{"light-dark(red, blue)", 255, 0, 0, 1, true},
		{"light-dark(rgb(1,2,3), #fff)", 1, 2, 3, 1, true},
		{"LIGHT-DARK(white, black)", 255, 255, 255, 1, true},
		{"light-dark(oklch(1 0 0 / 0.5), blue)", 255, 255, 255, 0.5, true},
		{"light-dark(red)", 0, 0, 0, 0, false},
		{"light-dark(red, notacolor)", 0, 0, 0, 0, false},
	}

	checkColorCases(t, cases)
}

func checkColorCases(t *testing.T, cases []struct {
	src     string
	r, g, b int
	alpha   float64
	ok      bool
},
) {
	t.Helper()

	for _, tc := range cases {
		r, g, b, a, ok := ParseColor(tc.src)
		if ok != tc.ok || (ok && (r != tc.r || g != tc.g || b != tc.b || a != tc.alpha)) {
			t.Errorf("ParseColor(%q) = (%d,%d,%d,%v,%v), want (%d,%d,%d,%v,%v)",
				tc.src, r, g, b, a, ok, tc.r, tc.g, tc.b, tc.alpha, tc.ok)
		}
	}
}
