package css //nolint:testpackage // exercises unexported parseContainerPrelude

import "testing"

const cardName = "card"

func TestParseContainerShorthand(t *testing.T) {
	t.Parallel()

	name, ctype := ParseContainerShorthand("card / inline-size")
	if name != cardName || ctype != "inline-size" {
		t.Fatalf("got name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("sidebar / size")
	if name != "sidebar" || ctype != "size" {
		t.Fatalf("got name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("none")
	if name != "" || ctype != "" {
		t.Fatalf("none: name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("a b / normal")
	if name != "a b" || ctype != "normal" {
		t.Fatalf("multi name: name=%q type=%q", name, ctype)
	}
}

func TestParseContainerNameValue(t *testing.T) {
	t.Parallel()

	if ParseContainerNameValue("none") != "" {
		t.Fatal("none should clear")
	}

	if ParseContainerNameValue("Card") != cardName {
		t.Fatalf("got %q", ParseContainerNameValue("Card"))
	}
}

func TestParseContainerRules(t *testing.T) { //nolint:cyclop,funlen // per-variant structural checks
	t.Parallel()
	sty := mustSheet(t, `
		.card { container: card / inline-size; width: 400px }
		@container card (inline-size > 20em) {
			.title { font-size: 2em }
		}
		@container (width > 300px) {
			.wide { color: red }
		}
		@container card (min-width: 10em) and (max-width: 1000px) {
			.mid { color: blue }
		}
		@container not (inline-size < 5em) {
			.ok { display: block }
		}
		@container card (inline-size > 10em) or (width < 1px) {
			.or { color: green }
		}
	`)
	// 1 normal + 5 container rules
	if len(sty.Rules) != 6 {
		t.Fatalf("rules = %d: %+v", len(sty.Rules), sty.Rules)
	}

	if sty.Rules[0].Container != nil {
		t.Fatal("first rule should not be container-conditional")
	}

	r1Val := sty.Rules[1]
	if r1Val.Container == nil || r1Val.Container.Name != cardName {
		t.Fatalf("rule1 container = %+v", r1Val.Container)
	}

	if r1Val.Container.Cond.Kind != "feat" || r1Val.Container.Cond.Feat == nil {
		t.Fatalf("rule1 cond = %+v", r1Val.Container.Cond)
	}

	if r1Val.Container.Cond.Feat.Name != "inline-size" || r1Val.Container.Cond.Feat.Op != ">" {
		t.Fatalf("feat = %+v", r1Val.Container.Cond.Feat)
	}

	if len(r1Val.Selectors) != 1 || r1Val.Selectors[0].Parts[0].Classes[0] != "title" {
		t.Fatalf("selectors = %+v", r1Val.Selectors)
	}

	r2 := sty.Rules[2]
	if r2.Container == nil || r2.Container.Name != "" {
		t.Fatalf("unnamed container = %+v", r2.Container)
	}

	r3 := sty.Rules[3]
	if r3.Container == nil || r3.Container.Cond.Kind != "and" || len(r3.Container.Cond.Kids) != 2 {
		t.Fatalf("and cond = %+v", r3.Container)
	}

	r4 := sty.Rules[4]
	if r4.Container == nil || r4.Container.Cond.Kind != "not" {
		t.Fatalf("not cond = %+v", r4.Container)
	}

	r5 := sty.Rules[5]
	if r5.Container == nil || r5.Container.Cond.Kind != "or" {
		t.Fatalf("or cond = %+v", r5.Container)
	}
}

func TestContainerCondMatches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prelude string
		size    float64
		want    bool
	}{
		{"card (inline-size > 20em)", 241, true},                // 20em at 12pt = 240pt
		{"card (inline-size > 20em)", 239, false},               // 20em at 12pt = 240pt
		{"(min-width: 100px)", 75, true},                        // 100px = 75pt
		{"(min-width: 100px)", 74, false},                       // 100px = 75pt
		{"(width > 50pt) and (inline-size < 200pt)", 100, true}, // mid-range matches both
		{"(width > 50pt) and (inline-size < 200pt)", 40, false}, // low fails the > 50pt arm
		{"not (inline-size < 10pt)", 10, true},                  // == 10 is not < 10
		{"not (inline-size < 10pt)", 9, false},                  // 9 is < 10
		{"(width < 1pt) or (inline-size > 5pt)", 6, true},       // second arm matches
	}
	for _, testCase := range cases {
		cq, found := parseContainerPrelude(testCase.prelude)
		if !found {
			t.Fatalf("parse prelude %q", testCase.prelude)
		}

		if got := cq.Cond.Matches(testCase.size, 12); got != testCase.want {
			t.Errorf("Matches(%q, %g) = %v, want %v", testCase.prelude, testCase.size, got, testCase.want)
		}
	}
}

func TestContainerWithoutSizeRejectedAtEval(t *testing.T) {
	t.Parallel()
	// Parsing succeeds; layout refuses to treat non-size-containers as
	// query containers. Here we only prove unknown features don't match.
	cq, ok := parseContainerPrelude("(height > 10px)")
	if ok && cq.Cond.Kind == "feat" {
		if cq.Cond.Matches(1000, 12) {
			t.Error("height feature must not match size-lite evaluator")
		}
	}
}

func TestHasContainerRules(t *testing.T) {
	t.Parallel()

	s := mustSheet(t, `p { color: red }`)
	if HasContainerRules([]*Stylesheet{s}) {
		t.Fatal("expected no container rules")
	}

	s2 := mustSheet(t, `@container (width > 1px) { p { color: red } }`)
	if !HasContainerRules([]*Stylesheet{s2}) {
		t.Fatal("expected container rules")
	}
}

func TestInvalidContainerPreludeSkipped(t *testing.T) {
	t.Parallel()
	sty := mustSheet(t, `
		@container { .x { color: red } }
		@container !!! { .y { color: blue } }
		p { color: green }
	`)
	// invalid @container bodies skipped; p remains
	found := false

	for _, r := range sty.Rules {
		if r.Container != nil {
			t.Fatalf("unexpected container rule: %+v", r)
		}

		if len(r.Selectors) > 0 && r.Selectors[0].Parts[0].Tag == "p" {
			found = true
		}
	}

	if !found {
		t.Fatalf("rules = %+v", sty.Rules)
	}
}
