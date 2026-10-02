package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// TestMathLengthCalcRecursive covers the general calc() grammar: more than
// three tokens, parentheses, nesting, and mixed absolute units.
func TestMathLengthCalcRecursive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value      string
		fsize      float64
		containing float64
		want       float64
	}{
		{"calc(2 * (3px + 4px))", 12, 500, 10.5},           // 14px
		{"calc(100% - 2 * 10px)", 12, 500, 485},            // 500pt - 15pt
		{"calc(1in + 12pt)", 12, 500, 84},                  // 96px + 16px
		{"calc(1in + 1cm)", 12, 500, 72 + 72/2.54},         // 96px + 37.795px
		{"calc(50% + 10px)", 12, 500, 257.5},               // legacy 3-token form
		{"calc(100% / 4)", 12, 500, 125},                   // division
		{"calc(-5px + 15px)", 12, 500, 7.5},                // unary minus
		{"calc(2 * (1em + 2px))", 12, 500, 2 * (12 + 1.5)}, // em inside nesting
	}

	for _, tc := range cases {
		got, ok := mathLength(tc.value, tc.fsize, tc.containing)
		if !ok || !near(got, tc.want) {
			t.Fatalf("mathLength(%q) = %.4f, %v; want %.4f, true", tc.value, got, ok, tc.want)
		}
	}
}

// TestMathLengthMinMaxClamp covers min()/max()/clamp(): multiple args,
// nesting, percentages, and mixed absolute units.
func TestMathLengthMinMaxClamp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value      string
		containing float64
		want       float64
	}{
		{"min(10px, 20px)", 500, 7.5},
		{"max(10px, 20px, 15px)", 500, 15},
		{"min(10mm, 1in)", 500, 10 * 72 / 25.4},            // 28.346pt
		{"max(1cm, 0.5in)", 500, 36},                       // 1cm beats 0.5in
		{"clamp(24rem, 70%, 46rem)", 500, 350},             // 70% inside
		{"clamp(1px, 9999px, 5px)", 500, 3.75},             // high bound wins
		{"clamp(200px, 10px, 300px)", 500, 150},            // low bound wins
		{"min(max(10px, 30px), 20px)", 500, 15},            // nesting
		{"max(min(50%, 100px), 10px)", 200, 75},            // 100px floor
		{"min(30%, 100px)", 200, 60},                       // 30% of 200pt
		{"min(10dvh, 1in)", 800, 72},                       // dvh inside math
		{"calc(100% - clamp(10px, 20px, 30px))", 500, 485}, // calc wraps clamp
	}

	for _, tc := range cases {
		got, ok := mathLength(tc.value, 12, tc.containing)
		if !ok || !near(got, tc.want) {
			t.Fatalf("mathLength(%q) = %.4f, %v; want %.4f, true", tc.value, got, ok, tc.want)
		}
	}
}

func TestMathLengthInvalid(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"min(10px, )",
		"calc(1px +)",
		"calc(1px 2px)",
		"clamp(1px, 2px)",
		"clamp(1px, 2px, 3px, 4px)",
		"min(1px, foo)",
		"10px",
		"min(1px, 2foo)",
	} {
		if got, ok := mathLength(value, 12, 500); ok {
			t.Fatalf("mathLength(%q) = %.4f, true; want invalid", value, got)
		}
	}
}

// TestStyleMathWidth resolves min()/max()/clamp() through the width parser.
func TestStyleMathWidth(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="x">x</div></body></html>`)
	s := sheet(t, `.x { width: clamp(100px, 20%, 500px); }`)

	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	sty := styles[findElementByName(root, "div")]

	// 20% of the 500pt viewport sits between 100px (75pt) and 500px (375pt).
	if !near(sty.Width, 100) {
		t.Fatalf("width = %.2fpt, want 100 from clamp", sty.Width)
	}
}

func TestStyleNestedMinMaxWidth(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="x">x</div></body></html>`)
	s := sheet(t, `.x { width: min(max(300px, 20%), 100%); }`)

	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	sty := styles[findElementByName(root, "div")]

	// max(300px=225pt, 20%=100pt) = 225pt; min(225pt, 100%=500pt) = 225pt.
	if !near(sty.Width, 225) {
		t.Fatalf("width = %.2fpt, want 225 from nested min/max", sty.Width)
	}
}

// TestStyleViewportHeightUnits checks dvh/svh/lvh resolve like vh.
func TestStyleViewportHeightUnits(t *testing.T) {
	t.Parallel()

	for _, unit := range []string{"vh", "dvh", "svh", "lvh"} {
		root := mustParse(t, `<html><body><div class="x">x</div></body></html>`)
		s := sheet(t, `.x { height: 50`+unit+`; }`)

		styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
		sty := styles[findElementByName(root, "div")]

		if !near(sty.Height, 400) {
			t.Fatalf("height: 50%s = %.2fpt, want 400", unit, sty.Height)
		}
	}
}

func TestStyleFontSizeClamp(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="x">x</div></body></html>`)
	s := sheet(t, `.x { font-size: clamp(5pt, 2vw, 8pt); }`)

	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	sty := styles[findElementByName(root, "div")]

	// 2vw of the 500pt viewport is 10pt, above the 8pt ceiling.
	if !near(sty.FontSize, 8) {
		t.Fatalf("font-size = %.2fpt, want 8 from clamp", sty.FontSize)
	}
}
