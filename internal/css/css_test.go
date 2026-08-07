package css

import (
	"fmt"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/html"
)

func mustSheet(t *testing.T, src string) *Stylesheet {
	t.Helper()
	s, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return s
}

func TestParseBasic(t *testing.T) {
	s := mustSheet(t, "p { color: red; font-size: 12pt }")
	if len(s.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(s.Rules))
	}
	r := s.Rules[0]
	if len(r.Selectors) != 1 || r.Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("selectors = %+v", r.Selectors)
	}
	if len(r.Decls) != 2 {
		t.Fatalf("decls = %+v", r.Decls)
	}
	if r.Decls[0].Prop != "color" || r.Decls[0].Value != "red" {
		t.Errorf("decl 0 = %+v", r.Decls[0])
	}
	if r.Decls[1].Prop != "font-size" || r.Decls[1].Value != "12pt" {
		t.Errorf("decl 1 = %+v", r.Decls[1])
	}
}

func TestParseSelectorLists(t *testing.T) {
	s := mustSheet(t, "h1, h2, .title { font-weight: bold }")
	if len(s.Rules) != 1 || len(s.Rules[0].Selectors) != 3 {
		t.Fatalf("rules = %+v", s.Rules)
	}
	for i, tag := range []string{"h1", "h2", "*"} {
		got := s.Rules[0].Selectors[i].Parts[0].Tag
		if got != tag {
			t.Errorf("selector %d tag = %q, want %q", i, got, tag)
		}
	}
}

func TestParseCommentsAndGarbage(t *testing.T) {
	s := mustSheet(t, `
		/* header comment */
		div { color: blue; } /* trailing */
		not-a-selector*&^ { x: 1 }
		garbage: without braces;
		p { border: 1px solid black /* inline */; }
	`)
	if len(s.Rules) != 2 {
		t.Fatalf("got %d rules, want 2: %+v", len(s.Rules), s.Rules)
	}
	if len(s.Rules[1].Decls) != 1 {
		t.Errorf("p decls = %+v", s.Rules[1].Decls)
	}
}

func TestParseImportant(t *testing.T) {
	s := mustSheet(t, "p { color: red !important; margin: 0 ! IMPORTANT }")
	d := s.Rules[0].Decls
	if len(d) != 2 || !d[0].Important || !d[1].Important {
		t.Fatalf("decls = %+v", d)
	}
	if d[0].Value != "red" || d[1].Value != "0" {
		t.Errorf("values not stripped: %+v", d)
	}
	if !isImportant("red !important") || isImportant("red") {
		t.Errorf("isImportant broken")
	}
}

func TestParseMedia(t *testing.T) {
	s := mustSheet(t, `
		@media print { .print-only { display: block } }
		@media screen { .screen-only { display: none } }
		@media screen and (max-width: 600px) { .narrow { color: red } }
		@media all { .any { color: black } }
		.all { color: gray }
	`)
	if len(s.Rules) != 5 {
		t.Fatalf("got %d rules: %+v", len(s.Rules), s.Rules)
	}
	want := []string{"print", "screen", "screen and (max-width: 600px)", "all", "all"}
	for i, w := range want {
		if s.Rules[i].Media != w {
			t.Errorf("rule %d media = %q, want %q", i, s.Rules[i].Media, w)
		}
	}
}

func TestParseAtRulesSkipped(t *testing.T) {
	s := mustSheet(t, `
		@charset "utf-8";
		@import url("other.css");
		@page { margin: 2cm }
		@font-face { font-family: X; src: url(x.woff) }
		p { color: red }
	`)
	if len(s.Rules) != 1 || s.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("rules = %+v", s.Rules)
	}
	if len(s.FontFaces) != 1 || s.FontFaces[0].Family != "X" {
		t.Fatalf("font-faces = %+v", s.FontFaces)
	}
	urls := FontFaceURLs(s.FontFaces[0].Src)
	if len(urls) != 1 || urls[0] != "x.woff" {
		t.Fatalf("font-face urls = %v", urls)
	}
}

func TestParseOrderAndNestedMediaOrder(t *testing.T) {
	s := mustSheet(t, `p { a: 1 } @media print { q { a: 2 } } r { a: 3 }`)
	if len(s.Rules) != 3 {
		t.Fatalf("rules = %+v", s.Rules)
	}
	for i, want := range []int{0, 1, 2} {
		if s.Rules[i].Order != want {
			t.Errorf("rule %d order = %d, want %d", i, s.Rules[i].Order, want)
		}
	}
}

func TestParseNeverPanics(t *testing.T) {
	garbage := []string{
		"",
		"p {",
		"p }",
		"} {",
		"@media print { p {",
		"p { color: }",
		"p { : }",
		"p { color red }",
		"{ }",
		"p { x: url(a;b) }",
		"p { content: \"a;b\" }",
		"@",
		"/* unterminated",
		"\x00\x01\x02",
	}
	for _, g := range garbage {
		s, err := Parse(g)
		if err != nil {
			// unbalanced braces may error; that is fine as long as nothing panics
			continue
		}
		_ = s
	}
}

func TestParseUnbalancedErrors(t *testing.T) {
	for _, g := range []string{"p {", "@media print { p {"} {
		if _, err := Parse(g); err == nil {
			t.Errorf("Parse(%q): want error, got nil", g)
		}
	}
}

func TestParseInline(t *testing.T) {
	d := ParseInline("color: red; font-size: 12px !important; garbage; :bad")
	if len(d) != 2 {
		t.Fatalf("decls = %+v", d)
	}
	if d[0].Prop != "color" || d[1].Prop != "font-size" || !d[1].Important {
		t.Errorf("decls = %+v", d)
	}
}

func TestParseSelectorCompounds(t *testing.T) {
	cases := []struct {
		src  string
		want []SelectorPart
	}{
		{"*", []SelectorPart{{Tag: "*"}}},
		{"div", []SelectorPart{{Tag: "div"}}},
		{".cls", []SelectorPart{{Tag: "*", Classes: []string{"cls"}}}},
		{"#id", []SelectorPart{{Tag: "*", ID: "id"}}},
		{"div.a.b", []SelectorPart{{Tag: "div", Classes: []string{"a", "b"}}}},
		{"div#x.y", []SelectorPart{{Tag: "div", ID: "x", Classes: []string{"y"}}}},
		{"a:hover", []SelectorPart{{Tag: "a"}}},
		{"a[href]", []SelectorPart{{Tag: "a"}}},
		{"[disabled]", []SelectorPart{{Tag: "*"}}},
		{"div > p", []SelectorPart{{Tag: "div"}, {Tag: "p", Combinator: ">"}}},
		{"div p", []SelectorPart{{Tag: "div"}, {Tag: "p", Combinator: " "}}},
		{"ul li a", []SelectorPart{{Tag: "ul"}, {Tag: "li", Combinator: " "}, {Tag: "a", Combinator: " "}}},
		{"div.a > p.b i", []SelectorPart{
			{Tag: "div", Classes: []string{"a"}},
			{Tag: "p", Classes: []string{"b"}, Combinator: ">"},
			{Tag: "i", Combinator: " "},
		}},
	}
	for _, tc := range cases {
		sel, ok := parseSelector(tc.src)
		if !ok {
			t.Errorf("parseSelector(%q): !ok", tc.src)
			continue
		}
		if len(sel.Parts) != len(tc.want) {
			t.Errorf("parseSelector(%q): %d parts, want %d: %+v", tc.src, len(sel.Parts), len(tc.want), sel.Parts)
			continue
		}
		for i := range tc.want {
			got, want := sel.Parts[i], tc.want[i]
			if got.Tag != want.Tag || got.ID != want.ID || got.Combinator != want.Combinator ||
				strings.Join(got.Classes, ".") != strings.Join(want.Classes, ".") {
				t.Errorf("parseSelector(%q) part %d = %+v, want %+v", tc.src, i, got, want)
			}
		}
	}
}

func treeFor(t *testing.T, src string) *html.Node {
	t.Helper()
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestMatch(t *testing.T) {
	root := treeFor(t, `<html><body>
		<div id="main" class="box big">
			<p class="note">a</p>
			<p>plain</p>
			<span><b>x</b></span>
		</div>
		<p id="second">b</p>
	</body></html>`)
	body := root.FirstChild("html").FirstChild("body")
	div := body.FirstChild("div")
	ps := []*html.Node{}
	for _, c := range div.Children {
		if c.Type == html.ElementNode && c.Name == "p" {
			ps = append(ps, c)
		}
	}
	if len(ps) != 2 {
		t.Fatalf("want 2 <p> in div, got %d", len(ps))
	}
	note, plain := ps[0], ps[1]
	bold := div.FirstChild("span").FirstChild("b")
	second := body.FirstChild("p")

	cases := []struct {
		sel  string
		node *html.Node
		want bool
	}{
		{"p", note, true},
		{"p", div, false},
		{"div", div, true},
		{"*", bold, true},
		{".note", note, true},
		{".note", plain, false},
		{".box", div, true},
		{".box.big", div, true},
		{".big.box", div, true},
		{"div.box", div, true},
		{"div#main", div, true},
		{"div#nope", div, false},
		{"#second", second, true},
		{"#main p", note, true},
		{"#main p", second, false},
		{"div > p", note, true},
		{"div > p", second, false},
		{"div > span", bold, false},
		{"div span b", bold, true},
		{"div > span > b", bold, true},
		{"body div p", plain, true},
		{"html body > div p.note", note, true},
		{"b", div, false},
		{"a:hover", note, false}, // type a does not match <p>
		{"[x]", note, false},     // attribute required; note has no x
		{"[id]", second, true},
		{"[id=second]", second, true},
		{"[id=nope]", second, false},
		{"[id*=eco]", second, true},
		{"[id*=zzz]", second, false},
		{"p:first-child", note, true},
		{"p:first-child", plain, false},
		{"p:last-child", plain, false}, // last element child of div is span
		{"span:last-child", div.FirstChild("span"), true},
		{"p:nth-child(even)", plain, true},
		{"p:nth-child(odd)", note, true},
		{"p:nth-child(2)", plain, true},
	}
	for _, tc := range cases {
		sel, ok := parseSelector(tc.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", tc.sel)
		}
		if got := Match(sel, tc.node); got != tc.want {
			t.Errorf("Match(%q) = %v, want %v", tc.sel, got, tc.want)
		}
	}
}

func TestLinkVisitedPseudos(t *testing.T) {
	root := treeFor(t, `<html><body>
		<p><a id="ext" href="https://example.com/">ext</a>
		<a id="frag" href="#x">frag</a>
		<a id="empty" href="">empty</a>
		<a id="bare">bare</a></p>
	</body></html>`)
	p := root.FirstChild("html").FirstChild("body").FirstChild("p")
	byID := map[string]*html.Node{}
	for _, c := range p.Children {
		if c.Type == html.ElementNode && c.Name == "a" {
			byID[c.Attribute("id")] = c
		}
	}
	for _, id := range []string{"ext", "frag", "empty", "bare"} {
		if byID[id] == nil {
			t.Fatalf("missing #%s", id)
		}
	}
	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"a:link", "ext", true},
		{"a:visited", "ext", true},
		{":link", "ext", true},
		{"a:link", "frag", true},
		{"a:link", "empty", false},
		{"a:link", "bare", false},
		{"a:hover", "ext", false},
		{"a:focus", "ext", false},
		{"a:active", "ext", false},
	}
	for _, tc := range cases {
		sel, ok := parseSelector(tc.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", tc.sel)
		}
		if got := Match(sel, byID[tc.id]); got != tc.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", tc.sel, tc.id, got, tc.want)
		}
	}
	// Specificity: a:link beats bare a (pseudo counts as class-level).
	sa, ok := parseSelector("a")
	if !ok {
		t.Fatal("parse a")
	}
	sl, ok := parseSelector("a:link")
	if !ok {
		t.Fatal("parse a:link")
	}
	_, ba, ca := Specificity(sa)
	_, bl, cl := Specificity(sl)
	if !(bl > ba || (bl == ba && cl >= ca)) {
		t.Fatalf("a:link specificity (%d,%d) should outrank a (%d,%d) on b-axis", bl, cl, ba, ca)
	}
	if bl < 1 {
		t.Fatalf("a:link b-specificity = %d, want >= 1", bl)
	}
}

// TestRootPseudo: :root matches the document element (<html>), not body/descendants.
// Without this, Vector :root { --font-size-medium: … } never applies (1013e0f).
func TestRootPseudo(t *testing.T) {
	doc := treeFor(t, `<html><body><p>x</p></body></html>`)
	htmlEl := doc.FirstChild("html")
	body := htmlEl.FirstChild("body")
	p := body.FirstChild("p")
	sel, ok := parseSelector(":root")
	if !ok {
		t.Fatal("parseSelector(:root) failed")
	}
	if Match(sel, doc) {
		t.Fatal(":root must not match synthetic #document")
	}
	if !Match(sel, htmlEl) {
		t.Fatal(":root must match <html>")
	}
	if Match(sel, body) || Match(sel, p) {
		t.Fatal(":root must not match body or p")
	}
	// Also accept html:root
	sel2, ok := parseSelector("html:root")
	if !ok {
		t.Fatal("parseSelector(html:root) failed")
	}
	if !Match(sel2, htmlEl) {
		t.Fatal("html:root must match <html>")
	}
}

func TestAttrWordAndSubstring(t *testing.T) {
	root := treeFor(t, `<html><body>
		<figure typeof="mw:File/Thumb mw:Image" id="f1"></figure>
		<figure typeof="mw:File/Frame" id="f2"></figure>
	</body></html>`)
	body := root.FirstChild("html").FirstChild("body")
	f1 := body.FirstChild("figure")
	f2 := f1
	for _, c := range body.Children {
		if c.Type == html.ElementNode && c.Attribute("id") == "f2" {
			f2 = c
		}
	}
	sel, ok := parseSelector(`figure[typeof~="mw:File/Thumb"]`)
	if !ok {
		t.Fatal("parse ~=")
	}
	if !Match(sel, f1) {
		t.Error("f1 should match typeof~=mw:File/Thumb")
	}
	if Match(sel, f2) {
		t.Error("f2 Frame should not match Thumb word")
	}
	sel2, ok := parseSelector(`figure[typeof*="File/Fr"]`)
	if !ok {
		t.Fatal("parse *=")
	}
	if !Match(sel2, f2) {
		t.Error("f2 should match typeof*=File/Fr")
	}
}

func TestAttrPrefixSuffixDash(t *testing.T) {
	root := treeFor(t, `<html><body>
		<a id="pdf" href="/files/report.pdf">PDF</a>
		<a id="png" href="/files/report.png">PNG</a>
		<a id="PDF" href="/files/report.PDF">PDF2</a>
		<span id="en" lang="en"></span>
		<span id="enus" lang="en-US"></span>
		<span id="fr" lang="fr"></span>
	</body></html>`)
	body := root.FirstChild("html").FirstChild("body")
	byID := map[string]*html.Node{}
	for _, c := range body.Children {
		if c.Type == html.ElementNode {
			byID[c.Attribute("id")] = c
		}
	}
	sel, ok := parseSelector(`a[href$=".pdf"]`)
	if !ok {
		t.Fatal("parse $=")
	}
	if !Match(sel, byID["pdf"]) {
		t.Error("pdf should match href$=.pdf")
	}
	if Match(sel, byID["png"]) || Match(sel, byID["PDF"]) {
		t.Error("$= is case-sensitive; png/PDF must not match .pdf")
	}
	sel2, ok := parseSelector(`a[href^="/files/"]`)
	if !ok {
		t.Fatal("parse ^=")
	}
	if !Match(sel2, byID["pdf"]) || Match(sel2, byID["en"]) {
		t.Error("^= /files/ should match pdf links only")
	}
	sel3, ok := parseSelector(`[lang|="en"]`)
	if !ok {
		t.Fatal("parse |=")
	}
	if !Match(sel3, byID["en"]) || !Match(sel3, byID["enus"]) {
		t.Error("|=en should match en and en-US")
	}
	if Match(sel3, byID["fr"]) {
		t.Error("fr should not match lang|=en")
	}
}

func TestSiblingCombinators(t *testing.T) {
	root := treeFor(t, `<html><body><div><p id="a">A</p><span>x</span><p id="b">B</p><p id="c">C</p></div></body></html>`)
	div := root.FirstChild("html").FirstChild("body").FirstChild("div")
	var a, b, c *html.Node
	for _, ch := range div.Children {
		if ch.Type != html.ElementNode || ch.Name != "p" {
			continue
		}
		switch ch.Attribute("id") {
		case "a":
			a = ch
		case "b":
			b = ch
		case "c":
			c = ch
		}
	}
	if a == nil || b == nil || c == nil {
		t.Fatal("missing siblings")
	}
	// p + p matches b (after span? no - + is next element sibling; a+span not p+p)
	// structure: p#a, span, p#b, p#c → p+p matches c (prev is p#b), and not b (prev is span)
	sel, ok := parseSelector("p + p")
	if !ok {
		t.Fatal("parse")
	}
	if Match(sel, b) {
		t.Error("p+p should not match b (previous element is span)")
	}
	if !Match(sel, c) {
		t.Error("p+p should match c")
	}
	sel, ok = parseSelector("p ~ p")
	if !ok {
		t.Fatal("parse ~")
	}
	if !Match(sel, b) || !Match(sel, c) {
		t.Error("p~p should match b and c")
	}
	if Match(sel, a) {
		t.Error("p~p should not match first p")
	}
}

func TestSpecificity(t *testing.T) {
	cases := []struct {
		sel     string
		a, b, c int
	}{
		{"p", 0, 0, 1},
		{"*", 0, 0, 0},
		{".a", 0, 1, 0},
		{"#x", 1, 0, 0},
		{"#x .y p", 1, 1, 1},
		{"div > p.a", 0, 1, 2},
		{"ul li", 0, 0, 2},
	}
	for _, tc := range cases {
		sel, ok := parseSelector(tc.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", tc.sel)
		}
		a, b, c := Specificity(sel)
		if a != tc.a || b != tc.b || c != tc.c {
			t.Errorf("Specificity(%q) = (%d,%d,%d), want (%d,%d,%d)", tc.sel, a, b, c, tc.a, tc.b, tc.c)
		}
	}
}

func TestParseLength(t *testing.T) {
	cases := []struct {
		src  string
		val  float64
		unit string
		ok   bool
	}{
		{"12px", 12, "px", true},
		{"12.5pt", 12.5, "pt", true},
		{"2em", 2, "em", true},
		{"50%", 50, "%", true},
		{"-4px", -4, "px", true},
		{"+3mm", 3, "mm", true},
		{"0", 0, "px", true},
		{"1.5", 1.5, "px", true},
		{"auto", 0, "", false},
		{"", 0, "", false},
		{"12q", 0, "", false},
		{"px", 0, "", false},
		{"1..2", 0, "", false},
	}
	for _, tc := range cases {
		val, unit, ok := ParseLength(tc.src)
		if ok != tc.ok || (ok && (val != tc.val || unit != tc.unit)) {
			t.Errorf("ParseLength(%q) = (%v, %q, %v), want (%v, %q, %v)", tc.src, val, unit, ok, tc.val, tc.unit, tc.ok)
		}
	}
}

func TestParseNumber(t *testing.T) {
	if f, ok := ParseNumber("1.4"); !ok || f != 1.4 {
		t.Errorf("ParseNumber(1.4) = %v, %v", f, ok)
	}
	if _, ok := ParseNumber("bold"); ok {
		t.Error("ParseNumber(bold) should fail")
	}
}

func TestParseColor(t *testing.T) {
	cases := []struct {
		src     string
		r, g, b int
		alpha   float64
		ok      bool
	}{
		{"#f00", 255, 0, 0, 1, true},
		{"#ff0000", 255, 0, 0, 1, true},
		{"#abcdef", 171, 205, 239, 1, true},
		{"#f00a", 255, 0, 0, 170.0 / 255, true},
		{"red", 255, 0, 0, 1, true},
		{"Blue", 0, 0, 255, 1, true},
		{"transparent", 0, 0, 0, 0, true},
		{"rgb(0, 128, 255)", 0, 128, 255, 1, true},
		{"rgb(100%, 0%, 50%)", 255, 0, 128, 1, true},
		{"rgba(0, 0, 0, 0.5)", 0, 0, 0, 0.5, true},
		{"rgba(255, 0, 0, 50%)", 255, 0, 0, 0.5, true},
		{"rgba(10, 20, 30, 2)", 10, 20, 30, 1, true},
		{"notacolor", 0, 0, 0, 0, false},
		{"#12", 0, 0, 0, 0, false},
		{"rgb(1,2)", 0, 0, 0, 0, false},
		{"", 0, 0, 0, 0, false},
	}
	for _, tc := range cases {
		r, g, b, a, ok := ParseColor(tc.src)
		if ok != tc.ok || (ok && (r != tc.r || g != tc.g || b != tc.b || a != tc.alpha)) {
			t.Errorf("ParseColor(%q) = (%d,%d,%d,%v,%v), want (%d,%d,%d,%v,%v)",
				tc.src, r, g, b, a, ok, tc.r, tc.g, tc.b, tc.alpha, tc.ok)
		}
	}
}

func TestParseFontFamily(t *testing.T) {
	got := ParseFontFamily(`"Helvetica Neue", Arial, sans-serif`)
	want := []string{"Helvetica Neue", "Arial", "sans-serif"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("family %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseStripCommentsPreservesNewlines(t *testing.T) {
	s := mustSheet(t, "p {\n\tcolor: /* keep */ red;\n}")
	if len(s.Rules) != 1 || len(s.Rules[0].Decls) != 1 {
		t.Fatalf("rules = %+v", s.Rules)
	}
	if s.Rules[0].Decls[0].Value != "red" {
		t.Errorf("decl = %+v", s.Rules[0].Decls[0])
	}
}

func TestLengthToPt(t *testing.T) {
	cases := []struct {
		val    float64
		unit   string
		basePt float64
		want   float64
		ok     bool
	}{
		{12, "px", 12, 9, true},        // 0.75
		{10, "pt", 12, 10, true},       // 1:1
		{1, "in", 12, 72, true},        // 72
		{1, "cm", 12, 72 / 2.54, true}, // 72/2.54
		{25.4, "mm", 12, 72, true},     // 72/25.4
		{2, "pc", 12, 24, true},        // * 12
		{2, "em", 12, 24, true},        // * basePt
		{2, "EM", 12, 24, true},        // case-insensitive
		{2, "rem", 12, 24, true},       // 16px root = 12pt
		{2, "ex", 12, 12, true},        // * basePt * 0.5
		{2, "ch", 12, 12, true},        // * basePt * 0.5
		{5, "px", 12, 3.75, true},      // fractional
		{0, "em", 12, 0, true},         // zero em
		{50, "%", 12, 0, false},        // percentages unsupported
		{2, "vw", 12, 0, false},        // viewport units unsupported
		{2, "q", 12, 0, false},         // unknown unit
		{2, "", 12, 0, false},          // empty unit
	}
	for _, tc := range cases {
		got, ok := LengthToPt(tc.val, tc.unit, tc.basePt)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("LengthToPt(%v, %q, %v) = (%v, %v), want (%v, %v)",
				tc.val, tc.unit, tc.basePt, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveCustomProps(t *testing.T) {
	declared := map[string]string{
		"--a": "10px",
		"--b": "var(--a)",
		"--c": "var(--b)",
		"--d": "var(--missing, 5px)",
	}
	got := ResolveCustomProps(declared, nil)
	want := map[string]string{
		"--a": "10px",
		"--b": "10px",
		"--c": "10px",
		"--d": "5px",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("ResolveCustomProps(%q) = %q, want %q", k, got[k], v)
		}
	}
}

func TestResolveCustomPropsInheritedOverlay(t *testing.T) {
	inherited := map[string]string{"--size": "12px", "--color": "red"}
	declared := map[string]string{"--size": "14px", "--double": "var(--size)"}
	got := ResolveCustomProps(declared, inherited)
	if got["--size"] != "14px" {
		t.Errorf("declared must win over inherited: %q", got["--size"])
	}
	if got["--color"] != "red" {
		t.Errorf("inherited-only prop missing: %q", got["--color"])
	}
	if got["--double"] != "14px" {
		t.Errorf("chain into overlaid prop: %q", got["--double"])
	}
}

func TestResolveCustomPropsCycle(t *testing.T) {
	declared := map[string]string{
		"--x": "var(--y)",
		"--y": "var(--x)",
	}
	got := ResolveCustomProps(declared, nil)
	// Cycle → invalid → empty, for both members.
	if got["--x"] != "" {
		t.Errorf("--x cycle = %q, want empty", got["--x"])
	}
	if got["--y"] != "" {
		t.Errorf("--y cycle = %q, want empty", got["--y"])
	}
}

func TestResolveCustomPropsDeepChain(t *testing.T) {
	// A chain longer than ResolveVar's 16-deep bound must still resolve fully
	// (the stack-walk is the stricter policy; this is what the memo walk
	// guarantees at the css seam).
	declared := map[string]string{"--v0": "final"}
	// build --v1 = var(--v0) ... --v20 = var(--v19)
	for i := 1; i <= 20; i++ {
		declared[fmt.Sprintf("--v%d", i)] = fmt.Sprintf("var(--v%d)", i-1)
	}
	got := ResolveCustomProps(declared, nil)
	if got["--v20"] != "final" {
		t.Errorf("deep chain --v20 = %q, want final", got["--v20"])
	}
}

func TestResolveCustomPropsSelfReferenceWithFallback(t *testing.T) {
	// Self-reference (cycle) but with a fallback: spec says the fallback is
	// used only when the variable is invalid at computed-value time; the
	// cycle makes the reference invalid, so the fallback applies.
	declared := map[string]string{
		"--a": "var(--a, 9px)",
		"--b": "var(--a)",
	}
	got := ResolveCustomProps(declared, nil)
	if got["--b"] != "9px" {
		t.Errorf("--b through cyclic --a with fallback = %q, want 9px", got["--b"])
	}
}

func TestParseSelectors(t *testing.T) {
	sels, ok := ParseSelectors("h1, h2, .title")
	if !ok || len(sels) != 3 {
		t.Fatalf("ParseSelectors(h1, h2, .title) = %d sels, ok=%v", len(sels), ok)
	}
	for i, tag := range []string{"h1", "h2", "*"} {
		if got := sels[i].Parts[0].Tag; got != tag {
			t.Errorf("selector %d tag = %q, want %q", i, got, tag)
		}
	}
	// Single selector.
	single, ok := ParseSelectors("div > p.a")
	if !ok || len(single) != 1 || len(single[0].Parts) != 2 {
		t.Fatalf("ParseSelectors(div > p.a) = %+v, ok=%v", single, ok)
	}
	// Invalid part fails the whole list (strict), even with valid parts.
	if _, ok := ParseSelectors("p, &^*%"); ok {
		t.Error("ParseSelectors with invalid part must fail (strict)")
	}
	// Empty / whitespace-only.
	if _, ok := ParseSelectors(""); ok {
		t.Error("ParseSelectors(\"\") must fail")
	}
	if _, ok := ParseSelectors("  , p"); ok {
		t.Error("ParseSelectors with empty list item must fail")
	}
	// Unsupported pseudo-element fails.
	if _, ok := ParseSelectors("p::first-line"); ok {
		t.Error("ParseSelectors with ::first-line must fail")
	}
}
