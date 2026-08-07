package css

import (
	"testing"

	"gowkhtmltopdf/internal/html"
)

func byID(root *html.Node, idVal string) *html.Node {
	var found *html.Node

	var walk func(*html.Node)
	walk = func(count *html.Node) {
		if found != nil || count == nil {
			return
		}

		if count.Type == html.ElementNode && count.Attribute("id") == idVal {
			found = count

			return
		}

		for _, c := range count.Children {
			walk(c)
		}
	}
	walk(root)

	return found
}

func TestHasParseAndMatch(t *testing.T) {
	t.Parallel()
	root := treeFor(t, `<html><body>
		<table>
			<tr id="warn"><td class="warning">x</td></tr>
			<tr id="ok"><td>y</td></tr>
		</table>
		<section id="sec"><h2>Title</h2><p>body</p></section>
		<section id="empty"><p>only</p></section>
		<div id="a">A</div><p id="p1">P</p><span id="s">S</span><p id="p2">P2</p>
		<article id="art"><span class="footnote">fn</span></article>
		<article id="plain">no fn</article>
		<div id="neg"><span class="x">1</span></div>
		<div id="pos"><span class="y">2</span></div>
	</body></html>`)

	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"tr:has(td.warning)", "warn", true},
		{"tr:has(td.warning)", "ok", false},
		{"section:has(> h2)", "sec", true},
		{"section:has(> h2)", "empty", false},
		{"div:has(+ p)", "a", true},
		{"div:has(+ p)", "neg", false},
		{"span:has(~ p)", "s", true},
		{"div:has(~ p)", "a", true},
		{"article:has(.footnote)", "art", true},
		{"article:has(.footnote)", "plain", false},
		{"div:has(span:not(.x))", "pos", true},
		{"div:has(span:not(.x))", "neg", false},
		{"tr:has(td.warning, td.missing)", "warn", true},
		{"tr:has(td.missing)", "warn", false},
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

func TestHasInvalid(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"div:has()",
		"div:has",
		"div:has(:has(span))",
		"div:has(::before)",
		"div:has(:before)",
		"div:has(span::after)",
	}
	for _, src := range invalid {
		if sel, ok := parseSelector(src); ok {
			t.Errorf("parseSelector(%q) ok unexpectedly: %+v", src, sel)
		}
	}
	// nested via stylesheet: whole rule dropped
	s := mustSheet(t, `div:has(:has(.x)) { color: red } p { color: blue }`)
	if len(s.Rules) != 1 || s.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("expected only p rule, got %+v", s.Rules)
	}
}

func TestHasSpecificity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		sel     string
		a, b, c int
	}{
		{"div:has(.warning)", 0, 1, 1},
		{"tr:has(td.warning)", 0, 1, 2},
		{"section:has(> h2)", 0, 0, 2},
		{"div:has(#x, .y)", 1, 0, 1},
		{"p:not(.a)", 0, 1, 1},
		{"div:has(span:not(.x))", 0, 1, 2},
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

func TestNotMatch(t *testing.T) {
	t.Parallel()
	root := treeFor(t, `<html><body><p class="note">a</p><p>b</p></body></html>`)
	body := root.FirstChild("html").FirstChild("body")
	note := body.FirstChild("p")
	plain := note

	for _, c := range body.Children {
		if c.Type == html.ElementNode && c.Name == "p" && c.Attribute("class") == "" {
			plain = c
		}
	}

	sel, ok := parseSelector("p:not(.note)")
	if !ok {
		t.Fatal("parse")
	}

	if Match(sel, note) {
		t.Error("p:not(.note) should not match .note")
	}

	if !Match(sel, plain) {
		t.Error("p:not(.note) should match plain p")
	}
}
