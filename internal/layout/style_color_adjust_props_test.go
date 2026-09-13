//nolint:varnamelen // targeted unit tests for the color-adjust property family
package layout

import "testing"

// colorAdjustField reads one field of the family so the table can share one
// assertion loop across five properties.
type colorAdjustField struct {
	name string
	get  func(*ResolvedStyle) string
}

var colorAdjustFields = []colorAdjustField{ //nolint:gochecknoglobals // static test table
	{name: "color-adjust", get: func(s *ResolvedStyle) string { return s.ColorAdjust }},
	{name: "print-color-adjust", get: func(s *ResolvedStyle) string { return s.ColorAdjust }},
	{name: "forced-color-adjust", get: func(s *ResolvedStyle) string { return s.ForcedColorAdjust }},
	{name: "color-scheme", get: func(s *ResolvedStyle) string { return s.ColorScheme }},
	{name: "dynamic-range-limit", get: func(s *ResolvedStyle) string { return s.DynamicRangeLimit }},
}

func colorAdjustFieldFor(t *testing.T, prop string) func(*ResolvedStyle) string {
	t.Helper()

	for _, field := range colorAdjustFields {
		if field.name == prop {
			return field.get
		}
	}

	t.Fatalf("no field reader for %q", prop)

	return nil
}

func TestColorAdjustPropsAcceptLegalKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  string
	}{
		{prop: "color-adjust", value: "exact", want: colorAdjustExact},
		{prop: "color-adjust", value: "economy", want: colorAdjustEconomy},
		{prop: "print-color-adjust", value: "exact", want: colorAdjustExact},
		{prop: "print-color-adjust", value: "  EXACT ", want: colorAdjustExact},
		{prop: "forced-color-adjust", value: "auto", want: forcedColorAdjustAuto},
		{prop: "forced-color-adjust", value: "none", want: forcedColorAdjustNone},
		{prop: "forced-color-adjust", value: "preserve-parent-color", want: forcedColorAdjustPreserveParentColor},
		{prop: "color-scheme", value: "normal", want: colorSchemeNormal},
		{prop: "color-scheme", value: "light", want: colorSchemeLight},
		{prop: "color-scheme", value: "dark", want: colorSchemeDark},
		{prop: "color-scheme", value: "Light  Dark", want: "light dark"},
		{prop: "color-scheme", value: "dark light", want: "dark light"},
		{prop: "color-scheme", value: "only light", want: "only light"},
		{prop: "color-scheme", value: "ONLY DARK", want: "only dark"},
		{prop: "dynamic-range-limit", value: "no-limit", want: dynamicRangeLimitNoLimit},
		{prop: "dynamic-range-limit", value: "standard", want: dynamicRangeLimitStandard},
		{prop: "dynamic-range-limit", value: "high", want: dynamicRangeLimitHigh},
		{prop: "dynamic-range-limit", value: "constrained-high", want: dynamicRangeLimitConstrainedHigh},
		{prop: "dynamic-range-limit", value: "constrained", want: dynamicRangeLimitConstrainedHigh},
	}

	for _, tc := range cases {
		style := initialStyle()

		if !applyColorAdjustProps(&style, tc.prop, tc.value, 12, nil, nil, false) {
			t.Errorf("group did not own %s", tc.prop)

			continue
		}

		if got := colorAdjustFieldFor(t, tc.prop)(&style); got != tc.want {
			t.Errorf("%s: %q = %q, want %q", tc.prop, tc.value, got, tc.want)
		}
	}
}

func TestColorAdjustPropsRejectIllegalKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
	}{
		{prop: "color-adjust", value: "auto"},
		{prop: "print-color-adjust", value: "bogus"},
		{prop: "forced-color-adjust", value: "true"},
		{prop: "forced-color-adjust", value: "preserve"},
		{prop: "color-scheme", value: "dark only"},
		{prop: "color-scheme", value: "my-scheme"},
		{prop: "color-scheme", value: "only"},
		{prop: "dynamic-range-limit", value: "unlimited"},
		{prop: "dynamic-range-limit", value: "mixed(1 2 3)"},
	}

	for _, tc := range cases {
		style := initialStyle()
		read := colorAdjustFieldFor(t, tc.prop)
		before := read(&style)

		if !applyColorAdjustProps(&style, tc.prop, tc.value, 12, nil, nil, false) {
			t.Errorf("group did not own %s", tc.prop)

			continue
		}

		if got := read(&style); got != before {
			t.Errorf("%s: illegal %q rewrote %q to %q", tc.prop, tc.value, before, got)
		}
	}
}

func TestColorAdjustPropsCSSWideKeywords(t *testing.T) {
	t.Parallel()

	parent := initialStyle()
	parent.ColorAdjust = colorAdjustExact
	parent.ForcedColorAdjust = forcedColorAdjustNone
	parent.ColorScheme = colorSchemeDark
	parent.DynamicRangeLimit = dynamicRangeLimitHigh

	child := initialStyle()
	for _, field := range colorAdjustFields {
		applyColorAdjustProps(&child, field.name, "inherit", 12, nil, &parent, true)
	}

	if child.ColorAdjust != colorAdjustExact ||
		child.ForcedColorAdjust != forcedColorAdjustNone ||
		child.ColorScheme != colorSchemeDark ||
		child.DynamicRangeLimit != dynamicRangeLimitHigh {
		t.Errorf("inherit did not copy the parent: %+v", child)
	}

	unset := initialStyle()
	applyColorAdjustProps(&unset, "color-scheme", "unset", 12, nil, &parent, true)

	if unset.ColorScheme != colorSchemeDark {
		t.Errorf("unset = %q, want parent dark", unset.ColorScheme)
	}

	reset := initialStyle()
	reset.ColorAdjust = colorAdjustExact // simulate the inherited copy
	applyColorAdjustProps(&reset, "color-adjust", "initial", 12, nil, &parent, true)

	if reset.ColorAdjust != colorAdjustEconomy {
		t.Errorf("initial = %q, want economy", reset.ColorAdjust)
	}

	revert := initialStyle()
	revert.DynamicRangeLimit = dynamicRangeLimitHigh
	applyColorAdjustProps(&revert, "dynamic-range-limit", "revert", 12, nil, &parent, true)

	if revert.DynamicRangeLimit != dynamicRangeLimitNoLimit {
		t.Errorf("revert = %q, want no-limit", revert.DynamicRangeLimit)
	}
}

func TestColorAdjustPropsForeignProperty(t *testing.T) {
	t.Parallel()

	style := initialStyle()

	if applyColorAdjustProps(&style, "color", "red", 12, nil, nil, false) {
		t.Fatal("color-adjust group claimed the color property")
	}
}

// TestColorAdjustPropsReachRestPass proves the group is reached through the
// real cascade dispatch (applyRestProps), not only by direct calls.
func TestColorAdjustPropsReachRestPass(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       t.Context(),
		viewportW: 800,
	}

	style := initialStyle()
	applyRestProps(&style, map[string]string{
		"print-color-adjust":  "exact",
		"forced-color-adjust": "none",
		"color-scheme":        "only dark",
		"dynamic-range-limit": "constrained-high",
	}, ctx, nil)

	if style.ColorAdjust != colorAdjustExact ||
		style.ForcedColorAdjust != forcedColorAdjustNone ||
		style.ColorScheme != "only dark" ||
		style.DynamicRangeLimit != dynamicRangeLimitConstrainedHigh {
		t.Errorf("cascade dispatch missed a value: %+v", style)
	}
}
