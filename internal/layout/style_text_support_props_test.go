//nolint:cyclop // text-decoration skip shorthand and longhand tests
package layout

import "testing"

// applySupportProp runs the text support apply arm against a fresh initial
// style. The context and parent arguments are unused by the arm.
func applySupportProp(prop, value string, fsize float64) (ResolvedStyle, bool) {
	sty := initialStyle()
	handled := applyTextSupportProps(&sty, prop, value, fsize, nil, nil, false)

	return sty, handled
}

func TestApplyTextSupportPropsUnicodeBidi(t *testing.T) {
	t.Parallel()

	for _, val := range []string{"normal", "embed", "isolate", "bidi-override", "isolate-override", "plaintext"} {
		sty, handled := applySupportProp("unicode-bidi", val, 12)
		if !handled {
			t.Fatalf("unicode-bidi %q: handled = false, want true", val)
		}

		if sty.UnicodeBidi != val {
			t.Fatalf("UnicodeBidi = %q, want %q", sty.UnicodeBidi, val)
		}
	}

	sty, handled := applySupportProp("unicode-bidi", "BIDI-OVERRIDE", 12)
	if !handled || sty.UnicodeBidi != "bidi-override" {
		t.Fatalf("case-insensitive value: handled=%v UnicodeBidi=%q", handled, sty.UnicodeBidi)
	}

	// Malformed keeps the initial value.
	sty, _ = applySupportProp("unicode-bidi", "sideways", 12)
	if sty.UnicodeBidi != "normal" {
		t.Fatalf("malformed UnicodeBidi = %q, want initial normal", sty.UnicodeBidi)
	}

	if _, handled := applySupportProp("font-size", "12pt", 12); handled {
		t.Fatal("font-size: handled = true, want false (not this group's property)")
	}
}

func TestApplyTextSupportPropsTextOrientation(t *testing.T) {
	t.Parallel()

	for _, val := range []string{"mixed", "upright", "sideways"} {
		sty, handled := applySupportProp("text-orientation", val, 12)
		if !handled || sty.TextOrientation != val {
			t.Fatalf("text-orientation %q: handled=%v TextOrientation=%q", val, handled, sty.TextOrientation)
		}
	}

	sty, _ := applySupportProp("text-orientation", "upright-ish", 12)
	if sty.TextOrientation != "mixed" {
		t.Fatalf("malformed TextOrientation = %q, want initial mixed", sty.TextOrientation)
	}
}

func TestApplyTextSupportPropsTextCombineUpright(t *testing.T) {
	t.Parallel()

	for _, val := range []string{"none", "all"} {
		sty, handled := applySupportProp("text-combine-upright", val, 12)
		if !handled || sty.TextCombineUpright != val {
			t.Fatalf("text-combine-upright %q: handled=%v value=%q", val, handled, sty.TextCombineUpright)
		}
	}

	for val, want := range map[string]string{
		"digits 1":  "digits 1",
		"digits 4":  "digits 4",
		"digits 04": "digits 4",
	} {
		sty, handled := applySupportProp("text-combine-upright", val, 12)
		if !handled || sty.TextCombineUpright != want {
			t.Fatalf("text-combine-upright %q: handled=%v value=%q, want %q", val, handled, sty.TextCombineUpright, want)
		}
	}

	for _, val := range []string{"digits", "digits 0", "digits -1", "digits two", "digits 2 3", "combine"} {
		sty, _ := applySupportProp("text-combine-upright", val, 12)
		if sty.TextCombineUpright != "none" {
			t.Fatalf("malformed text-combine-upright %q left %q, want initial none", val, sty.TextCombineUpright)
		}
	}
}

func TestApplyTextSupportPropsTextDecorationInset(t *testing.T) {
	t.Parallel()

	cases := map[string]float64{
		"8px":     6, // 96dpi px -> pt
		"2pt":     2,
		"1em":     16,
		"1rem":    12, // 16px root at 96dpi -> 12pt
		"2pt 4pt": 2,  // first endpoint wins for the single stored field
		"auto":    0,
	}

	for val, want := range cases {
		sty, handled := applySupportProp("text-decoration-inset", val, 16)
		if !handled {
			t.Fatalf("text-decoration-inset %q: handled = false", val)
		}

		if !near(sty.TextDecorationInset, want) {
			t.Fatalf("TextDecorationInset(%q) = %v, want %v", val, sty.TextDecorationInset, want)
		}
	}

	sty := initialStyle()

	applyTextSupportProps(&sty, "text-decoration-inset", "6pt", 16, nil, nil, false)
	applyTextSupportProps(&sty, "text-decoration-inset", "50%", 16, nil, nil, false)

	if !near(sty.TextDecorationInset, 6) {
		t.Fatalf("percentage inset changed stored value to %v, want 6 (rejected)", sty.TextDecorationInset)
	}

	applyTextSupportProps(&sty, "text-decoration-inset", "1pt 2pt 3pt", 16, nil, nil, false)

	if !near(sty.TextDecorationInset, 6) {
		t.Fatalf("three-value inset changed stored value to %v, want 6 (rejected)", sty.TextDecorationInset)
	}
}

func TestApplyTextSupportPropsTextDecorationSkipShorthand(t *testing.T) {
	t.Parallel()

	sty, handled := applySupportProp("text-decoration-skip", "none", 12)
	if !handled {
		t.Fatal("text-decoration-skip none: handled = false")
	}

	if sty.TextDecorationSkip != "none" || sty.TextDecorationSkipSelf != "no-skip" ||
		sty.TextDecorationSkipBox != "none" || sty.TextDecorationSkipSpaces != "none" ||
		sty.TextDecorationSkipInk != "none" {
		t.Fatalf("skip none expanded to self=%q box=%q spaces=%q ink=%q",
			sty.TextDecorationSkipSelf, sty.TextDecorationSkipBox,
			sty.TextDecorationSkipSpaces, sty.TextDecorationSkipInk)
	}

	sty, handled = applySupportProp("text-decoration-skip", "auto", 12)
	if !handled {
		t.Fatal("text-decoration-skip auto: handled = false")
	}

	if sty.TextDecorationSkip != "auto" || sty.TextDecorationSkipSelf != "auto" ||
		sty.TextDecorationSkipBox != "none" || sty.TextDecorationSkipSpaces != textDecorationSkipStartEnd ||
		sty.TextDecorationSkipInk != "auto" {
		t.Fatalf("skip auto expanded to self=%q box=%q spaces=%q ink=%q",
			sty.TextDecorationSkipSelf, sty.TextDecorationSkipBox,
			sty.TextDecorationSkipSpaces, sty.TextDecorationSkipInk)
	}

	// CSS Text Decoration 3 legacy keywords each set the longhand they name
	// and reset the others to their initial values.
	legacy := map[string]struct {
		skipBox    string
		skipSpaces string
		skipInk    string
	}{
		"objects":        {columnSpanAll, textDecorationSkipStartEnd, overflowAuto},
		"box-decoration": {columnSpanAll, textDecorationSkipStartEnd, overflowAuto},
		"spaces":         {cssDisplayNone, columnSpanAll, overflowAuto},
		"edges":          {cssDisplayNone, textDecorationSkipStartEnd, overflowAuto},
		"ink":            {cssDisplayNone, textDecorationSkipStartEnd, overflowAuto},
	}
	for val, want := range legacy {
		sty, handled := applySupportProp("text-decoration-skip", val, 12)
		if !handled {
			t.Fatalf("text-decoration-skip %q: handled = false", val)
		}

		if sty.TextDecorationSkip != val || sty.TextDecorationSkipSelf != overflowAuto ||
			sty.TextDecorationSkipBox != want.skipBox ||
			sty.TextDecorationSkipSpaces != want.skipSpaces ||
			sty.TextDecorationSkipInk != want.skipInk {
			t.Fatalf("skip %q expanded to self=%q box=%q spaces=%q ink=%q",
				val, sty.TextDecorationSkipSelf, sty.TextDecorationSkipBox,
				sty.TextDecorationSkipSpaces, sty.TextDecorationSkipInk)
		}
	}

	sty, _ = applySupportProp("text-decoration-skip", "bogus", 12)
	if sty.TextDecorationSkip != overflowAuto {
		t.Fatalf("malformed shorthand changed value to %q, want initial auto", sty.TextDecorationSkip)
	}
}

func TestApplyTextSupportPropsTextDecorationSkipLonghands(t *testing.T) {
	t.Parallel()

	boxCases := map[string]string{"none": "none", "all": "all"}
	for val, want := range boxCases {
		sty, handled := applySupportProp("text-decoration-skip-box", val, 12)
		if !handled || sty.TextDecorationSkipBox != want {
			t.Fatalf("skip-box %q: handled=%v value=%q", val, handled, sty.TextDecorationSkipBox)
		}
	}

	sty, _ := applySupportProp("text-decoration-skip-box", "some", 12)
	if sty.TextDecorationSkipBox != "none" {
		t.Fatalf("malformed skip-box left %q, want initial none", sty.TextDecorationSkipBox)
	}

	selfCases := map[string]string{
		"auto":                             "auto",
		"skip-all":                         "skip-all",
		"no-skip":                          "no-skip",
		"skip-underline":                   "skip-underline",
		"skip-line-through skip-underline": "skip-underline skip-line-through",
		"skip-overline":                    "skip-overline",
	}
	for val, want := range selfCases {
		sty, handled := applySupportProp("text-decoration-skip-self", val, 12)
		if !handled || sty.TextDecorationSkipSelf != want {
			t.Fatalf("skip-self %q: handled=%v value=%q, want %q", val, handled, sty.TextDecorationSkipSelf, want)
		}
	}

	//nolint:dupword // duplicate tokens must be rejected
	sty, _ = applySupportProp("text-decoration-skip-self", "skip-underline skip-underline", 12)
	if sty.TextDecorationSkipSelf != "auto" {
		t.Fatalf("duplicate skip-self left %q, want initial auto", sty.TextDecorationSkipSelf)
	}

	spacesCases := map[string]string{
		"none":      "none",
		"all":       "all",
		"start":     "start",
		"end":       "end",
		"end start": textDecorationSkipStartEnd,
	}
	for val, want := range spacesCases {
		sty, handled := applySupportProp("text-decoration-skip-spaces", val, 12)
		if !handled || sty.TextDecorationSkipSpaces != want {
			t.Fatalf("skip-spaces %q: handled=%v value=%q, want %q", val, handled, sty.TextDecorationSkipSpaces, want)
		}
	}

	//nolint:dupword // duplicate tokens must be rejected
	sty, _ = applySupportProp("text-decoration-skip-spaces", "start start", 12)
	if sty.TextDecorationSkipSpaces != textDecorationSkipStartEnd {
		t.Fatalf("duplicate skip-spaces left %q, want initial start end", sty.TextDecorationSkipSpaces)
	}

	sty, _ = applySupportProp("text-decoration-skip-spaces", "none all", 12)
	if sty.TextDecorationSkipSpaces != textDecorationSkipStartEnd {
		t.Fatalf("mixed skip-spaces left %q, want initial start end", sty.TextDecorationSkipSpaces)
	}
}

// TestDecorationSkipPolicyTreatsPlumbingInitialsPerSpec pins the consumer side
// of the initialStyle deviation: the plumbing still stores the legacy
// "objects" keyword for the four skip fields, and the paint policy must read
// those as the spec initials (skip-self auto, skip-box none, skip-spaces
// start end). The style.go fix is reported separately; this test proves the
// consumers are correct either way.
func TestDecorationSkipPolicyTreatsPlumbingInitialsPerSpec(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	policy := resolveTextDecorationSkipPolicy(&sty)

	if !policy.skipSpacesStart || !policy.skipSpacesEnd {
		t.Fatalf("legacy initial objects: spaces start=%v end=%v, want start end",
			policy.skipSpacesStart, policy.skipSpacesEnd)
	}

	if policy.skipSpacesAll || policy.skipSelfAll || policy.skipBoxAll || policy.noSkipSelf {
		t.Fatalf("legacy initial objects produced active skips: %+v", policy)
	}

	sty.TextDecorationSkipSpaces = "none"
	policy = resolveTextDecorationSkipPolicy(&sty)

	if policy.skipSpacesStart || policy.skipSpacesEnd || policy.skipSpacesAll {
		t.Fatalf("skip-spaces none produced skips: %+v", policy)
	}
}
