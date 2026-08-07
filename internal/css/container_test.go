package css

import "testing"

func TestParseContainerShorthand(t *testing.T) {
	t.Parallel()

	name, ctype := ParseContainerShorthand("card / inline-size")
	if name != "card" || ctype != "inline-size" {
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

	if ParseContainerNameValue("Card") != "card" {
		t.Fatalf("got %q", ParseContainerNameValue("Card"))
	}
}

func TestParseContainerRules(t *testing.T) {
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
	if r1Val.Container == nil || r1Val.Container.Name != "card" {
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

	contQ, found := parseContainerPrelude("card (inline-size > 20em)")
	if !found {
		t.Fatal("parse prelude")
	}
	// 20em at 12pt = 240pt
	if !contQ.Cond.Matches(241, 12) {
		t.Error("241pt should match > 20em@12pt")
	}

	if contQ.Cond.Matches(239, 12) {
		t.Error("239pt should not match > 20em@12pt")
	}

	cq2, found := parseContainerPrelude("(min-width: 100px)")
	if !found {
		t.Fatal("parse min-width")
	}
	// 100px = 75pt
	if !cq2.Cond.Matches(75, 12) {
		t.Error("75pt should match min-width:100px")
	}

	if cq2.Cond.Matches(74, 12) {
		t.Error("74pt should not match min-width:100px")
	}

	cq3, found := parseContainerPrelude("(width > 50pt) and (inline-size < 200pt)")
	if !found {
		t.Fatal("parse and")
	}

	if !cq3.Cond.Matches(100, 12) {
		t.Error("and should match mid")
	}

	if cq3.Cond.Matches(40, 12) {
		t.Error("and should fail low")
	}

	cq4, found := parseContainerPrelude("not (inline-size < 10pt)")
	if !found {
		t.Fatal("parse not")
	}

	if !cq4.Cond.Matches(10, 12) {
		t.Error("not (< 10) should match == 10")
	}

	if cq4.Cond.Matches(9, 12) {
		t.Error("not (< 10) should fail 9")
	}

	cq5, found := parseContainerPrelude("(width < 1pt) or (inline-size > 5pt)")
	if !found {
		t.Fatal("parse or")
	}

	if !cq5.Cond.Matches(6, 12) {
		t.Error("or should match via second")
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
