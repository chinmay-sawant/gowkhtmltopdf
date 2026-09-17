//nolint:all // targeted unit tests for font-width / font-stretch
package layout

import (
	"context"
	"testing"
)

func TestFontStretchAliasesToWidth(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontWidthProps(&style, "font-stretch", "condensed", 12, nil, nil, false) {
		t.Fatal("font-stretch not owned")
	}

	if style.FontWidth != 75 {
		t.Fatalf("font-stretch condensed = %v, want 75", style.FontWidth)
	}

	style = initialStyle()
	if !applyFontWidthProps(&style, "font-width", "125%", 12, nil, nil, false) {
		t.Fatal("font-width not owned")
	}

	if style.FontWidth != 125 {
		t.Fatalf("font-width 125%% = %v", style.FontWidth)
	}

	parent := initialStyle()
	parent.FontWidth = 75
	child := initialStyle()
	inheritProps(&child, &parent, nil)

	if child.FontWidth != 75 {
		t.Fatalf("inherited width = %v", child.FontWidth)
	}

	ctx := &styleContext{ctx: context.Background(), viewportW: 800}
	applyStyleProp(&child, "font-stretch", "expanded", 12, ctx, &parent, true)

	if child.FontWidth != 125 {
		t.Fatalf("declared stretch = %v, want 125", child.FontWidth)
	}
}

func TestFontWidthKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  float64
	}{
		{"ultra-condensed", 50},
		{"normal", 100},
		{"ultra-expanded", 200},
	}

	for _, testCase := range tests {
		style := initialStyle()
		if !applyFontWidthProps(&style, "font-width", testCase.value, 12, nil, nil, false) {
			t.Fatalf("%s not owned", testCase.value)
		}

		if style.FontWidth != testCase.want {
			t.Fatalf("%s = %v, want %v", testCase.value, style.FontWidth, testCase.want)
		}
	}
}
