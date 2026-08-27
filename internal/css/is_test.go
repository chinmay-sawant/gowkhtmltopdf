package css //nolint:testpackage // exercises unexported parseSelector and helpers

import (
	"testing"
)

func TestParseIs(t *testing.T) {
	t.Parallel()

	t.Run("simple list", func(t *testing.T) {
		t.Parallel()

		checkParseIsSimple(t)
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		checkParseIsNested(t)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		checkParseIsInvalid(t)
	})

	t.Run("invalid rule dropped", func(t *testing.T) {
		t.Parallel()

		checkParseIsInvalidDropped(t)
	})
}

func checkParseIsSimple(t *testing.T) {
	t.Helper()

	sel, parsed := parseSelector(":is(h1, h2, h3)")
	if !parsed {
		t.Fatal("parseSelector(:is(h1, h2, h3)) failed")
	}

	if len(sel.Parts) != 1 || len(sel.Parts[0].Pseudos) != 1 {
		t.Fatalf("parts = %+v", sel.Parts)
	}

	pseudo := sel.Parts[0].Pseudos[0]
	if pseudo.Name != pseudoClassIs || len(pseudo.Is) != 3 {
		t.Fatalf("pseudo = %+v", pseudo)
	}

	for idx, tag := range []string{"h1", "h2", "h3"} {
		if got := pseudo.Is[idx].Parts[0].Tag; got != tag {
			t.Errorf("arg %d tag = %q, want %q", idx, got, tag)
		}
	}
}

func checkParseIsNested(t *testing.T) {
	t.Helper()

	nested, parsed := parseSelector(":is(:is(.a), p)")
	if !parsed {
		t.Fatal("nested :is failed")
	}

	inner := nested.Parts[0].Pseudos[0]
	if inner.Name != pseudoClassIs || len(inner.Is) != 2 {
		t.Fatalf("nested args = %+v", inner.Is)
	}

	if inner.Is[0].Parts[0].Pseudos[0].Name != pseudoClassIs {
		t.Fatalf("inner :is missing: %+v", inner.Is[0])
	}
}

func checkParseIsInvalid(t *testing.T) {
	t.Helper()

	invalid := []string{
		":is()",
		":is",
		":is(span::after)",
		":is(::before)",
		":is(h1, :::)",
	}
	for _, src := range invalid {
		if got, parsed := parseSelector(src); parsed {
			t.Errorf("parseSelector(%q) ok unexpectedly: %+v", src, got)
		}
	}
}

func checkParseIsInvalidDropped(t *testing.T) {
	t.Helper()

	str := mustSheet(t, `:is(span::after) { color: red } p { color: blue }`)
	if len(str.Rules) != 1 || str.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("expected only p rule, got %+v", str.Rules)
	}
}

func TestIsPseudo(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<html><body>
		<h1 id="h1">H</h1>
		<h2 id="h2">H2</h2>
		<p id="note" class="note">n</p>
		<p id="plain">p</p>
		<div id="box" class="box"><p id="child">c</p></div>
	</body></html>`)

	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{":is(h1, h2)", "h1", true},
		{":is(h1, h2)", "h2", true},
		{":is(h1, h2)", "note", false},
		{":is(.note, #h1)", "note", true},
		{":is(.note, #h1)", "h1", true},
		{":is(.note, #h1)", "plain", false},
		{"div:is(.box, p)", "box", true},
		{"div:is(.box, p)", "plain", false},
		{":is(div > p)", "child", true},
		{":is(div > p)", "box", false},
		{":is(:is(.note), h2)", "note", true},
		{":is(:is(.note), h2)", "h2", true},
		{":is(:is(.note), h2)", "plain", false},
		{":is(:hover, h1)", "h1", true},
		{":is(:hover)", "h1", false},
		{"p:is(:hover)", "note", false},
	}
	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		n := byID(root, testCase.id)
		if n == nil {
			t.Fatalf("missing id %q", testCase.id)
		}

		if got := Match(sel, n); got != testCase.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", testCase.sel, testCase.id, got, testCase.want)
		}
	}
}

func TestIsSpecificity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		sel     string
		a, b, c int
	}{
		{":is(h1, h2)", 0, 0, 1},
		{":is(#x, .y)", 1, 0, 0},
		{":is(.a, p)", 0, 1, 0},
		{"div:is(.a)", 0, 1, 1},
		{"div:is(#x)", 1, 0, 1},
		{":is(:is(#a, .b), p)", 1, 0, 0},
	}
	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		a, b, c := Specificity(sel)
		if a != testCase.a || b != testCase.b || c != testCase.c {
			t.Errorf("Specificity(%q) = (%d,%d,%d), want (%d,%d,%d)", testCase.sel, a, b, c, testCase.a, testCase.b, testCase.c)
		}
	}
}

func TestWherePseudo(t *testing.T) {
	t.Parallel()

	t.Run("match", func(t *testing.T) {
		t.Parallel()

		checkWhereMatch(t)
	})

	t.Run("specificity", func(t *testing.T) {
		t.Parallel()

		checkWhereSpecificity(t)
	})

	t.Run("type beats where id", func(t *testing.T) {
		t.Parallel()

		checkTypeBeatsWhereID(t)
	})
}

func checkWhereMatch(t *testing.T) {
	t.Helper()

	root := treeFor(t, `<html><body>
		<h1 id="h1">H</h1>
		<p id="note" class="note">n</p>
		<p id="plain">p</p>
	</body></html>`)

	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{":where(h1, p)", "h1", true},
		{":where(h1, p)", "note", true},
		{":where(h1, .note)", "plain", false},
		{":where(#h1, .missing)", "h1", true},
	}
	for _, testCase := range cases {
		sel, parsed := parseSelector(testCase.sel)
		if !parsed {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		node := byID(root, testCase.id)
		if node == nil {
			t.Fatalf("missing id %q", testCase.id)
		}

		if got := Match(sel, node); got != testCase.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", testCase.sel, testCase.id, got, testCase.want)
		}
	}
}

func checkWhereSpecificity(t *testing.T) {
	t.Helper()

	specCases := []struct {
		sel     string
		a, b, c int
	}{
		{":where(#x, .y)", 0, 0, 0},
		{"div:where(#x)", 0, 0, 1},
		{":where(h1)", 0, 0, 0},
		{"p", 0, 0, 1},
	}
	for _, testCase := range specCases {
		sel, parsed := parseSelector(testCase.sel)
		if !parsed {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		a, b, c := Specificity(sel)
		if a != testCase.a || b != testCase.b || c != testCase.c {
			t.Errorf("Specificity(%q) = (%d,%d,%d), want (%d,%d,%d)", testCase.sel, a, b, c, testCase.a, testCase.b, testCase.c)
		}
	}
}

func checkTypeBeatsWhereID(t *testing.T) {
	t.Helper()

	whereID, whereOK := parseSelector(":where(#h1)")
	if !whereOK {
		t.Fatal("parse :where(#h1)")
	}

	typeSel, typeOK := parseSelector("p")
	if !typeOK {
		t.Fatal("parse p")
	}

	wa, wb, wc := Specificity(whereID)
	ta, tb, tc := Specificity(typeSel)

	if !betterSpec(ta, tb, tc, wa, wb, wc) {
		t.Fatalf("type p %+v should beat :where(#h1) %+v", []int{ta, tb, tc}, []int{wa, wb, wc})
	}
}
