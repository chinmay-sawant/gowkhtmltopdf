package css_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// Property names repeated across the @supports fixtures.
const (
	displayProp = "display"
	colorProp   = "color"
)

func parseSheet(t *testing.T, src string) *css.Stylesheet {
	t.Helper()

	s, err := css.Parse(src)
	if err != nil {
		t.Fatalf("css.Parse(%q): %v", src, err)
	}

	return s
}

func resolveStyles(t *testing.T, src string, sheets ...*css.Stylesheet,
) (*html.Node, map[*html.Node]*layout.ResolvedStyle) {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatalf("html.Parse(%q): %v", src, err)
	}

	styles, err := layout.ResolveStyles(t.Context(), root, layout.Options{
		Width: 600, Height: 800, Sheets: sheets,
	})
	if err != nil {
		t.Fatalf("layout.ResolveStyles: %v", err)
	}

	return root, styles
}

func findNode(n *html.Node, name string) *html.Node {
	if n.Name == name {
		return n
	}

	for _, child := range n.Children {
		if found := findNode(child, name); found != nil {
			return found
		}
	}

	return nil
}

// testColors holds the primary colors the layout tests compare against.
type testColors struct {
	red   [3]float64
	blue  [3]float64
	green [3]float64
	black [3]float64
}

// testPalette returns freshly initialized colors so parallel tests share no
// mutable global state.
func testPalette() testColors {
	return testColors{
		red:   [3]float64{1, 0, 0},
		blue:  [3]float64{0, 0, 1},
		green: [3]float64{0, 1, 0},
		black: [3]float64{0, 0, 0},
	}
}

func colorOf(t *testing.T, root *html.Node, styles map[*html.Node]*layout.ResolvedStyle, tag string) [3]float64 {
	t.Helper()

	node := findNode(root, tag)
	if node == nil {
		t.Fatalf("no <%s> node", tag)
	}

	style := styles[node]
	if style == nil {
		t.Fatalf("no resolved style for <%s>", tag)
	}

	return style.Color
}

func supportsCond(t *testing.T, cond string) *css.SupportsCondition {
	t.Helper()

	sheet := parseSheet(t, "@supports "+cond+" { p { color: #f00 } }")
	if len(sheet.Rules) != 1 {
		t.Fatalf("@supports %s: got %d rules, want 1", cond, len(sheet.Rules))
	}

	if sheet.Rules[0].Supports == nil {
		t.Fatalf("@supports %s: nil Supports condition", cond)
	}

	return sheet.Rules[0].Supports
}

func TestSupportsConditionMatching(t *testing.T) {
	t.Parallel()

	supported := func(prop, _ string) bool {
		return prop == displayProp || prop == colorProp
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"(display: grid)", true},
		{"(unknown-prop: 1)", false},
		{"(display: grid) and (color: #f00)", true},
		{"(display: grid) and (unknown-prop: 1)", false},
		{"(unknown-prop: 1) or (color: #f00)", true},
		{"not (unknown-prop: 1)", true},
		{"not (display: grid)", false},
		{"((display: grid) and (color: #f00)) or (unknown-prop: 1)", true},
		{"(display: grid) and ((color: #f00) or (unknown-prop: 1))", true},
		{"(display grid)", false},
		{"selector(p)", false},
	}

	for _, tc := range cases {
		if got := css.SupportsMatches(supportsCond(t, tc.cond), supported); got != tc.want {
			t.Errorf("SupportsMatches(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

func TestSupportsInvalidPreludeDropsBlock(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, "@supports display: grid { p { color: #f00 } }")

	if len(sheet.Rules) != 0 {
		t.Fatalf("invalid @supports kept %d rules, want 0", len(sheet.Rules))
	}
}

func TestSupportsNestedConditionCombines(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@supports (display: grid) {
			@supports (color: #f00) {
				p { color: #f00 }
			}
		}`)
	if len(sheet.Rules) != 1 {
		t.Fatalf("nested @supports: got %d rules, want 1", len(sheet.Rules))
	}

	cond := sheet.Rules[0].Supports
	supported := func(prop, _ string) bool { return prop == displayProp }

	if css.SupportsMatches(cond, supported) {
		t.Fatal("nested @supports matched with color unsupported, want false")
	}

	supported = func(prop, _ string) bool { return prop == displayProp || prop == colorProp }

	if !css.SupportsMatches(cond, supported) {
		t.Fatal("nested @supports did not match with both supported, want true")
	}
}

func TestSupportsGatesRulesInLayout(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@supports (display: grid) { p { color: #f00 } }
		@supports (unknown-prop: 1) { h1 { color: #f00 } }`)

	root, styles := resolveStyles(t, "<p>x</p><h1>y</h1>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.red {
		t.Fatalf("supported @supports rule: p color = %v, want red", got)
	}

	if got := colorOf(t, root, styles, "h1"); got != colors.black {
		t.Fatalf("unsupported @supports rule: h1 color = %v, want black", got)
	}
}

func TestSupportsInsideMedia(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@media all {
			@supports (unknown-prop: 1) { p { color: #f00 } }
			@supports (display: grid) { h1 { color: #f00 } }
		}`)

	root, styles := resolveStyles(t, "<p>x</p><h1>y</h1>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.black {
		t.Fatalf("unsupported nested @supports: p color = %v, want black", got)
	}

	if got := colorOf(t, root, styles, "h1"); got != colors.red {
		t.Fatalf("supported nested @supports: h1 color = %v, want red", got)
	}
}

func TestLayerRanksAndOrder(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@layer base, theme;
		@layer theme { p { color: #00f } }
		@layer base { p { color: #f00 } }
		p { color: #0f0 }`)

	ranks := map[string]int{}

	for _, r := range sheet.Rules {
		if len(r.Decls) == 1 && r.Decls[0].Prop == colorProp {
			ranks[r.Decls[0].Value] = r.Layer
		}
	}

	if ranks["#f00"] != 1 || ranks["#00f"] != 2 || ranks["#0f0"] != 0 {
		t.Fatalf("layer ranks = %v, want base 1, theme 2, unlayered 0", ranks)
	}

	if len(sheet.Layers) != 2 || sheet.Layers[0] != "base" || sheet.Layers[1] != "theme" {
		t.Fatalf("Layers = %v, want [base theme]", sheet.Layers)
	}
}

func TestLayerCascadeOrder(t *testing.T) {
	t.Parallel()

	colors := testPalette()

	cases := []struct {
		name  string
		sheet string
		want  [3]float64
	}{
		{
			name:  "unlayered beats layered",
			sheet: `@layer a { p { color: #00f } } p { color: #0f0 }`,
			want:  colors.green,
		},
		{
			name:  "later layer beats earlier",
			sheet: `@layer a, b; @layer a { p { color: #f00 } } @layer b { p { color: #00f } }`,
			want:  colors.blue,
		},
		{
			name:  "anonymous layers follow order",
			sheet: `@layer { p { color: #f00 } } @layer { p { color: #00f } }`,
			want:  colors.blue,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, styles := resolveStyles(t, "<p>x</p>", parseSheet(t, testCase.sheet))

			if got := colorOf(t, root, styles, "p"); got != testCase.want {
				t.Fatalf("color = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPropertyRegistrationParse(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@property --brand {
			syntax: "<color>";
			initial-value: #f00;
			inherits: false;
		}`)

	if len(sheet.Properties) != 1 {
		t.Fatalf("got %d @property registrations, want 1", len(sheet.Properties))
	}

	prop := sheet.Properties[0]
	if prop.Name != "--brand" || prop.Syntax != "<color>" || prop.Initial != "#f00" {
		t.Fatalf("@property = %+v, want --brand <color> #f00", prop)
	}

	if prop.Inherits || !prop.InheritsSet {
		t.Fatalf("inherits = %v set = %v, want false set", prop.Inherits, prop.InheritsSet)
	}
}

func TestPropertyInitialValueUsedByVar(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@property --brand { syntax: "<color>"; initial-value: #f00; inherits: false; }
		p { color: var(--brand) }`)

	root, styles := resolveStyles(t, "<p>x</p>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.red {
		t.Fatalf("var(--brand) color = %v, want initial red", got)
	}
}

func TestPropertyVarFallbackWithoutInitial(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@property --brand { syntax: "*"; inherits: false; }
		p { color: var(--brand, #0f0) }`)

	root, styles := resolveStyles(t, "<p>x</p>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.green {
		t.Fatalf("var(--brand, #0f0) color = %v, want fallback green", got)
	}
}

func TestPropertyInheritsFlag(t *testing.T) {
	t.Parallel()

	colors := testPalette()

	cases := []struct {
		name     string
		inherits string
		want     [3]float64
	}{
		{name: "inherits true", inherits: "true", want: colors.blue},
		{name: "inherits false", inherits: "false", want: colors.red},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sheet := parseSheet(t, `
				@property --brand { syntax: "<color>"; initial-value: #f00; inherits: `+testCase.inherits+`; }
				div { --brand: #00f }
				p { color: var(--brand) }`)

			root, styles := resolveStyles(t, "<div><p>x</p></div>", sheet)

			if got := colorOf(t, root, styles, "p"); got != testCase.want {
				t.Fatalf("child color = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestUnknownAtRulesStillSkipped(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, "@unknown foo { p { color: #f00 } } p { color: #0f0 }")

	if len(sheet.Rules) != 1 || sheet.Rules[0].Decls[0].Value != "#0f0" {
		t.Fatalf("unknown at-rule changed rule list: %+v", sheet.Rules)
	}
}
