package css //nolint:testpackage // exercises unexported parseSelector/isImportant

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const colorRed = "red"

func mustSheet(t *testing.T, src string) *Stylesheet {
	t.Helper()

	s, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}

	return s
}

func TestParseBasic(t *testing.T) {
	t.Parallel()

	s := mustSheet(t, "p { color: red; font-size: 12pt }")
	if len(s.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(s.Rules))
	}

	rVal := s.Rules[0]
	if len(rVal.Selectors) != 1 || rVal.Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("selectors = %+v", rVal.Selectors)
	}

	if len(rVal.Decls) != 2 {
		t.Fatalf("decls = %+v", rVal.Decls)
	}

	if rVal.Decls[0].Prop != "color" || rVal.Decls[0].Value != colorRed {
		t.Errorf("decl 0 = %+v", rVal.Decls[0])
	}

	if rVal.Decls[1].Prop != "font-size" || rVal.Decls[1].Value != "12pt" {
		t.Errorf("decl 1 = %+v", rVal.Decls[1])
	}
}

func TestParseSelectorLists(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, "h1, h2, .title { font-weight: bold }")
	if len(str.Rules) != 1 || len(str.Rules[0].Selectors) != 3 {
		t.Fatalf("rules = %+v", str.Rules)
	}

	for i, tag := range []string{"h1", "h2", "*"} {
		got := str.Rules[0].Selectors[i].Parts[0].Tag
		if got != tag {
			t.Errorf("selector %d tag = %q, want %q", i, got, tag)
		}
	}
}

func TestParseCommentsAndGarbage(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `
		/* header comment */
		div { color: blue; } /* trailing */
		not-a-selector*&^ { x: 1 }
		garbage: without braces;
		p { border: 1px solid black /* inline */; }
	`)
	if len(str.Rules) != 2 {
		t.Fatalf("got %d rules, want 2: %+v", len(str.Rules), str.Rules)
	}

	if len(str.Rules[1].Decls) != 1 {
		t.Errorf("p decls = %+v", str.Rules[1].Decls)
	}
}

func TestParseImportant(t *testing.T) {
	t.Parallel()
	s := mustSheet(t, "p { color: red !important; margin: 0 ! IMPORTANT }")

	data := s.Rules[0].Decls
	if len(data) != 2 || !data[0].Important || !data[1].Important {
		t.Fatalf("decls = %+v", data)
	}

	if data[0].Value != colorRed || data[1].Value != "0" {
		t.Errorf("values not stripped: %+v", data)
	}

	if !isImportant("red !important") || isImportant(colorRed) {
		t.Errorf("isImportant broken")
	}
}

func TestParseMedia(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `
		@media print { .print-only { display: block } }
		@media screen { .screen-only { display: none } }
		@media screen and (max-width: 600px) { .narrow { color: red } }
		@media all { .any { color: black } }
		.all { color: gray }
	`)
	if len(str.Rules) != 5 {
		t.Fatalf("got %d rules: %+v", len(str.Rules), str.Rules)
	}

	want := []string{"print", "screen", "screen and (max-width: 600px)", "all", "all"}
	for i, w := range want {
		if str.Rules[i].Media != w {
			t.Errorf("rule %d media = %q, want %q", i, str.Rules[i].Media, w)
		}
	}
}

func TestParseAtRulesSkipped(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `
		@charset "utf-8";
		@import url("other.css");
		@page { margin: 2cm }
		@font-face { font-family: X; src: url(x.woff) }
		p { color: red }
	`)
	if len(str.Rules) != 1 || str.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("rules = %+v", str.Rules)
	}

	if len(str.FontFaces) != 1 || str.FontFaces[0].Family != "X" {
		t.Fatalf("font-faces = %+v", str.FontFaces)
	}

	urls := FontFaceURLs(str.FontFaces[0].Src)
	if len(urls) != 1 || urls[0] != "x.woff" {
		t.Fatalf("font-face urls = %v", urls)
	}

	if len(str.Imports) != 1 || str.Imports[0].URL != "other.css" || str.Imports[0].Media != "" {
		t.Fatalf("imports = %+v", str.Imports)
	}
}

func TestParseImport(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `
		@import url("a.css");
		@import url('b.css') print;
		@import "c.css" screen and (max-width: 600px);
		@import url(d.css);
		@import ;
		@import not-a-url;
		@import url("");
		p { color: red }
	`)
	if len(str.Rules) != 1 || str.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("rules = %+v", str.Rules)
	}

	want := []ImportRule{
		{URL: "a.css", Media: ""},
		{URL: "b.css", Media: "print"},
		{URL: "c.css", Media: "screen and (max-width: 600px)"},
		{URL: "d.css", Media: ""},
	}
	if len(str.Imports) != len(want) {
		t.Fatalf("imports = %+v, want %+v", str.Imports, want)
	}

	for i, w := range want {
		if str.Imports[i] != w {
			t.Errorf("import %d = %+v, want %+v", i, str.Imports[i], w)
		}
	}
}

//nolint:varnamelen // short local mirrors the stylesheet vocabulary
func TestParsePageStyle(t *testing.T) {
	t.Parallel()

	s := mustSheet(t, `@page { size: A4; margin: 0 2cm 4mm 6pt }`)
	if s.Page == nil {
		t.Fatal("page style = nil")
	}

	if s.Page.Size != "A4" || s.Page.Margin != "0 2cm 4mm 6pt" {
		t.Fatalf("page style = %+v", *s.Page)
	}
}

type pageSelectorCase struct {
	name    string
	src     string
	sel     string
	margin  string
	size    string
	unnamed bool
}

func pageSelectorCases() []pageSelectorCase {
	return []pageSelectorCase{
		pageSelCase("unnamed", `@page { size: A4; margin: 2cm }`, "", "2cm", "A4", true),
		pageSelCase("first", `@page :first { margin: 1cm }`, ":first", "1cm", "", false),
		pageSelCase("left", `@page :left { margin: 3cm }`, ":left", "3cm", "", false),
		pageSelCase("right", `@page :right { size: letter }`, ":right", "", "letter", false),
		pageSelCase("named landscape", `@page landscape { size: landscape }`, "landscape", "", "landscape", false),
		pageSelCase("first padded", `@page  :first  { margin: 1cm }`, ":first", "1cm", "", false),
		pageSelCase("left no space", `@page:left { margin: 4mm }`, ":left", "4mm", "", false),
		pageSelCase(
			"unnamed nested margin box",
			`@page { margin: 2cm; @top-center { content: "Header" } size: A4; }`,
			"", "2cm", "A4", true,
		),
	}
}

func pageSelCase(name, src, sel, margin, size string, unnamed bool) pageSelectorCase {
	return pageSelectorCase{
		name:    name,
		src:     src,
		sel:     sel,
		margin:  margin,
		size:    size,
		unnamed: unnamed,
	}
}

func TestParsePageSelectors(t *testing.T) {
	t.Parallel()

	for _, testCase := range pageSelectorCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			checkPageSelectorCase(t, testCase)
		})
	}

	t.Run("mixed unnamed last wins", func(t *testing.T) {
		t.Parallel()

		checkMixedPageSelectors(t)
	})
}

func TestParsePageSelectorRejectsTrailingTokens(t *testing.T) {
	t.Parallel()

	for _, src := range []string{
		`@page chapter:first { margin: 40mm }`,
		`@page chapter, appendix { margin: 40mm }`,
		`@page :first later { margin: 40mm }`,
	} {
		str := mustSheet(t, src)
		if len(str.Pages) != 0 || str.Page != nil {
			t.Fatalf("%q parsed invalid page selector: Pages=%+v Page=%+v", src, str.Pages, str.Page)
		}
	}
}

func checkPageSelectorCase(t *testing.T, testCase pageSelectorCase) {
	t.Helper()

	str := mustSheet(t, testCase.src)
	if len(str.Pages) != 1 {
		t.Errorf("%q: Pages = %+v, want 1", testCase.src, str.Pages)

		return
	}

	got := str.Pages[0]
	if got.Sel != testCase.sel || got.Margin != testCase.margin || got.Size != testCase.size {
		t.Errorf("%q: PageRule = %+v, want sel=%q margin=%q size=%q",
			testCase.src, got, testCase.sel, testCase.margin, testCase.size)
	}

	checkUnnamedPage(t, str, testCase)
}

func checkUnnamedPage(t *testing.T, str *Stylesheet, testCase pageSelectorCase) {
	t.Helper()

	if testCase.unnamed {
		if str.Page == nil {
			t.Errorf("%q: Page = nil, want unnamed style", testCase.src)

			return
		}

		if str.Page.Margin != testCase.margin || str.Page.Size != testCase.size {
			t.Errorf("%q: Page = %+v, want margin=%q size=%q",
				testCase.src, *str.Page, testCase.margin, testCase.size)
		}

		return
	}

	if str.Page != nil {
		t.Errorf("%q: Page = %+v, want nil for sel %q", testCase.src, *str.Page, testCase.sel)
	}
}

func checkMixedPageSelectors(t *testing.T) {
	t.Helper()

	mixed := mustSheet(t, `
		@page landscape { size: landscape }
		@page { margin: 12mm; size: A4 }
		@page :first { margin: 0 }
	`)
	if mixed.Page == nil || mixed.Page.Margin != "12mm" || mixed.Page.Size != "A4" {
		t.Fatalf("mixed unnamed Page = %+v", mixed.Page)
	}

	if len(mixed.Pages) != 3 {
		t.Fatalf("mixed Pages = %+v, want 3", mixed.Pages)
	}

	wantSel := []string{"landscape", "", ":first"}
	for idx, sel := range wantSel {
		if mixed.Pages[idx].Sel != sel {
			t.Errorf("mixed Pages[%d].Sel = %q, want %q", idx, mixed.Pages[idx].Sel, sel)
		}
	}
}

func TestParseOrderAndNestedMediaOrder(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `p { a: 1 } @media print { q { a: 2 } } r { a: 3 }`)
	if len(str.Rules) != 3 {
		t.Fatalf("rules = %+v", str.Rules)
	}

	for i, want := range []int{0, 1, 2} {
		if str.Rules[i].Order != want {
			t.Errorf("rule %d order = %d, want %d", i, str.Rules[i].Order, want)
		}
	}
}

func TestParseNeverPanics(t *testing.T) {
	t.Parallel()

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
		str, err := Parse(g)
		if err != nil {
			// unbalanced braces may error; that is fine as long as nothing panics
			continue
		}

		_ = str
	}
}

func TestParseUnbalancedErrors(t *testing.T) {
	t.Parallel()

	for _, g := range []string{"p {", "@media print { p {"} {
		if _, err := Parse(g); err == nil {
			t.Errorf("Parse(%q): want error, got nil", g)
		}
	}
}

func TestParseInline(t *testing.T) {
	t.Parallel()

	data := ParseInline("color: red; font-size: 12px !important; garbage; :bad")
	if len(data) != 2 {
		t.Fatalf("decls = %+v", data)
	}

	if data[0].Prop != "color" || data[1].Prop != "font-size" || !data[1].Important {
		t.Errorf("decls = %+v", data)
	}
}

func TestParseSelectorCompounds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src  string
		want []SelectorPart
	}{
		{"*", []SelectorPart{
			{Tag: "*"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div", []SelectorPart{
			{Tag: "div"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{".cls", []SelectorPart{
			{Tag: "*", Classes: []string{"cls"}}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"#id", []SelectorPart{
			{Tag: "*", ID: "id"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div.a.b", []SelectorPart{
			{Tag: "div", Classes: []string{"a", "b"}}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div#x.y", []SelectorPart{
			{Tag: "div", ID: "x", Classes: []string{"y"}}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"a:hover", []SelectorPart{
			{Tag: "a"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"a[href]", []SelectorPart{
			{Tag: "a"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"[disabled]", []SelectorPart{
			{Tag: "*"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div > p", []SelectorPart{
			{Tag: "div"},                //nolint:exhaustruct // intentional zero-value fields
			{Tag: "p", Combinator: ">"}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div p", []SelectorPart{
			{Tag: "div"},                //nolint:exhaustruct // intentional zero-value fields
			{Tag: "p", Combinator: " "}, //nolint:exhaustruct // intentional zero-value fields
		}},
		{"ul li a", []SelectorPart{
			{Tag: "ul"},                  //nolint:exhaustruct // intentional zero-value fields
			{Tag: "li", Combinator: " "}, //nolint:exhaustruct // intentional zero-value fields
			{Tag: "a", Combinator: " "},  //nolint:exhaustruct // intentional zero-value fields
		}},
		{"div.a > p.b i", []SelectorPart{
			{Tag: "div", Classes: []string{"a"}},                //nolint:exhaustruct // intentional zero-value fields
			{Tag: "p", Classes: []string{"b"}, Combinator: ">"}, //nolint:exhaustruct // intentional zero-value fields
			{Tag: "i", Combinator: " "},                         //nolint:exhaustruct // intentional zero-value fields
		}},
	}
	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.src)
		if !ok {
			t.Errorf("parseSelector(%q): !ok", testCase.src)

			continue
		}

		if len(sel.Parts) != len(testCase.want) {
			t.Errorf("parseSelector(%q): %d parts, want %d: %+v", testCase.src, len(sel.Parts), len(testCase.want), sel.Parts)

			continue
		}

		for i := range testCase.want {
			got, want := sel.Parts[i], testCase.want[i]
			if got.Tag != want.Tag || got.ID != want.ID || got.Combinator != want.Combinator ||
				strings.Join(got.Classes, ".") != strings.Join(want.Classes, ".") {
				t.Errorf("parseSelector(%q) part %d = %+v, want %+v", testCase.src, i, got, want)
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
	t.Parallel()
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

	var pNodes []*html.Node

	for _, c := range div.Children {
		if c.Type == html.ElementNode && c.Name == "p" {
			pNodes = append(pNodes, c)
		}
	}

	if len(pNodes) != 2 {
		t.Fatalf("want 2 <p> in div, got %d", len(pNodes))
	}

	note, plain := pNodes[0], pNodes[1]
	bold := div.FirstChild("span").FirstChild("b")
	second := body.FirstChild("p")
	checkMatchTable(t, div, note, plain, bold, second)
}

func checkMatchTable(t *testing.T, div, note, plain, bold, second *html.Node) {
	t.Helper()

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
	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		if got := Match(sel, testCase.node); got != testCase.want {
			t.Errorf("Match(%q) = %v, want %v", testCase.sel, got, testCase.want)
		}
	}
}

func TestLinkVisitedPseudos(t *testing.T) {
	t.Parallel()
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
	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		if got := Match(sel, byID[testCase.id]); got != testCase.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", testCase.sel, testCase.id, got, testCase.want)
		}
	}

	checkLinkVisitedSpecificity(t)
}

// checkLinkVisitedSpecificity: a:link must outrank bare a (pseudo counts as
// class-level specificity).
func checkLinkVisitedSpecificity(t *testing.T) {
	t.Helper()

	selA, found := parseSelector("a")
	if !found {
		t.Fatal("parse a")
	}

	slVal, found := parseSelector("a:link")
	if !found {
		t.Fatal("parse a:link")
	}

	_, ba, ca := Specificity(selA)
	_, blVal, cl := Specificity(slVal)

	if !(blVal > ba || (blVal == ba && cl >= ca)) {
		t.Fatalf("a:link specificity (%d,%d) should outrank a (%d,%d) on b-axis", blVal, cl, ba, ca)
	}

	if blVal < 1 {
		t.Fatalf("a:link b-specificity = %d, want >= 1", blVal)
	}
}

// TestRootPseudo: :root matches the document element (<html>), not body/descendants.
// Without this, Vector :root { --font-size-medium: … } never applies (1013e0f).
func TestRootPseudo(t *testing.T) {
	t.Parallel()
	doc := treeFor(t, `<html><body><p>x</p></body></html>`)
	htmlEl := doc.FirstChild("html")
	body := htmlEl.FirstChild("body")
	page := body.FirstChild("p")

	sel, found := parseSelector(":root")
	if !found {
		t.Fatal("parseSelector(:root) failed")
	}

	if Match(sel, doc) {
		t.Fatal(":root must not match synthetic #document")
	}

	if !Match(sel, htmlEl) {
		t.Fatal(":root must match <html>")
	}

	if Match(sel, body) || Match(sel, page) {
		t.Fatal(":root must not match body or p")
	}
	// Also accept html:root
	sel2, found := parseSelector("html:root")
	if !found {
		t.Fatal("parseSelector(html:root) failed")
	}

	if !Match(sel2, htmlEl) {
		t.Fatal("html:root must match <html>")
	}
}

func TestAttrWordAndSubstring(t *testing.T) {
	t.Parallel()
	root := treeFor(t, `<html><body>
		<figure typeof="mw:File/Thumb mw:Image" id="f1"></figure>
		<figure typeof="mw:File/Frame" id="f2"></figure>
	</body></html>`)
	body := root.FirstChild("html").FirstChild("body")
	f1Val := body.FirstChild("figure")
	f2Val := f1Val

	for _, c := range body.Children {
		if c.Type == html.ElementNode && c.Attribute("id") == "f2" {
			f2Val = c
		}
	}

	sel, found := parseSelector(`figure[typeof~="mw:File/Thumb"]`)
	if !found {
		t.Fatal("parse ~=")
	}

	if !Match(sel, f1Val) {
		t.Error("f1 should match typeof~=mw:File/Thumb")
	}

	if Match(sel, f2Val) {
		t.Error("f2 Frame should not match Thumb word")
	}

	sel2, found := parseSelector(`figure[typeof*="File/Fr"]`)
	if !found {
		t.Fatal("parse *=")
	}

	if !Match(sel2, f2Val) {
		t.Error("f2 should match typeof*=File/Fr")
	}
}

func TestAttrPrefixSuffixDash(t *testing.T) { //nolint:cyclop // attribute-operator matching has many independent checks
	t.Parallel()
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

	sel, found := parseSelector(`a[href$=".pdf"]`)
	if !found {
		t.Fatal("parse $=")
	}

	if !Match(sel, byID["pdf"]) {
		t.Error("pdf should match href$=.pdf")
	}

	if Match(sel, byID["png"]) || Match(sel, byID["PDF"]) {
		t.Error("$= is case-sensitive; png/PDF must not match .pdf")
	}

	sel2, found := parseSelector(`a[href^="/files/"]`)
	if !found {
		t.Fatal("parse ^=")
	}

	if !Match(sel2, byID["pdf"]) || Match(sel2, byID["en"]) {
		t.Error("^= /files/ should match pdf links only")
	}

	sel3, found := parseSelector(`[lang|="en"]`)
	if !found {
		t.Fatal("parse |=")
	}

	if !Match(sel3, byID["en"]) || !Match(sel3, byID["enus"]) {
		t.Error("|=en should match en and en-US")
	}

	if Match(sel3, byID["fr"]) {
		t.Error("fr should not match lang|=en")
	}
}

func TestSiblingCombinators(t *testing.T) { //nolint:cyclop // combinator checks across several sibling layouts
	t.Parallel()
	root := treeFor(t, `<html><body><div><p id="a">A</p><span>x</span><p id="b">B</p><p id="c">C</p></div></body></html>`)
	div := root.FirstChild("html").FirstChild("body").FirstChild("div")

	var nodeA, nodeB, cur *html.Node

	for _, chVal := range div.Children {
		if chVal.Type != html.ElementNode || chVal.Name != "p" {
			continue
		}

		switch chVal.Attribute("id") {
		case "a":
			nodeA = chVal
		case "b":
			nodeB = chVal
		case "c":
			cur = chVal
		}
	}

	if nodeA == nil || nodeB == nil || cur == nil {
		t.Fatal("missing siblings")
	}
	// p + p matches b (after span? no - + is next element sibling; a+span not p+p)
	// structure: p#a, span, p#b, p#c → p+p matches c (prev is p#b), and not b (prev is span)
	sel, found := parseSelector("p + p")
	if !found {
		t.Fatal("parse")
	}

	if Match(sel, nodeB) {
		t.Error("p+p should not match b (previous element is span)")
	}

	if !Match(sel, cur) {
		t.Error("p+p should match c")
	}

	sel, found = parseSelector("p ~ p")
	if !found {
		t.Fatal("parse ~")
	}

	if !Match(sel, nodeB) || !Match(sel, cur) {
		t.Error("p~p should match b and c")
	}

	if Match(sel, nodeA) {
		t.Error("p~p should not match first p")
	}
}

func TestSpecificity(t *testing.T) {
	t.Parallel()

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

func TestParseLength(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	if f, ok := ParseNumber("1.4"); !ok || f != 1.4 {
		t.Errorf("ParseNumber(1.4) = %v, %v", f, ok)
	}

	if _, ok := ParseNumber("bold"); ok {
		t.Error("ParseNumber(bold) should fail")
	}
}

func TestParseColor(t *testing.T) {
	t.Parallel()

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

func TestParseColorHsl(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src     string
		r, g, b int
		alpha   float64
		ok      bool
	}{
		{"hsl(0, 100%, 50%)", 255, 0, 0, 1, true},
		{"hsl(120, 100%, 50%)", 0, 255, 0, 1, true},
		{"hsl(240, 100%, 50%)", 0, 0, 255, 1, true},
		{"hsl(60, 100%, 50%)", 255, 255, 0, 1, true},
		{"hsl(0deg, 100%, 50%)", 255, 0, 0, 1, true},
		{"hsl(0, 0%, 0%)", 0, 0, 0, 1, true},
		{"hsl(0, 0%, 100%)", 255, 255, 255, 1, true},
		{"hsl(0, 0%, 50%)", 128, 128, 128, 1, true},
		{"hsla(0, 100%, 50%, 0.5)", 255, 0, 0, 0.5, true},
		{"hsla(0, 100%, 50%, 50%)", 255, 0, 0, 0.5, true},
		{"HSL(0, 100%, 50%)", 255, 0, 0, 1, true},
		{"hsl(480, 100%, 50%)", 0, 255, 0, 1, true},
		{"hsl(-120, 100%, 50%)", 0, 0, 255, 1, true},
		{"hsl(0, 100%)", 0, 0, 0, 0, false},
		{"hsl(0, 100, 50)", 0, 0, 0, 0, false},
		{"hsl()", 0, 0, 0, 0, false},
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
	t.Parallel()

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
	t.Parallel()

	str := mustSheet(t, "p {\n\tcolor: /* keep */ red;\n}")
	if len(str.Rules) != 1 || len(str.Rules[0].Decls) != 1 {
		t.Fatalf("rules = %+v", str.Rules)
	}

	if str.Rules[0].Decls[0].Value != colorRed {
		t.Errorf("decl = %+v", str.Rules[0].Decls[0])
	}
}

func TestLengthToPt(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()
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

func TestResolveVarsEmbeddedInCompoundValue(t *testing.T) {
	t.Parallel()

	got := ResolveVars("1px solid var(--line, #000)", func(name string) (string, bool) {
		if name == "--line" {
			return "#2563eb", true
		}

		return "", false
	})
	if got != "1px solid #2563eb" {
		t.Fatalf("ResolveVars compound value = %q, want %q", got, "1px solid #2563eb")
	}
}

func TestResolveCustomPropsSelfReferenceWithFallback(t *testing.T) {
	t.Parallel()
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

func TestParseSelectors(t *testing.T) { //nolint:cyclop // many independent strict-parsing checks
	t.Parallel()

	sels, found := ParseSelectors("h1, h2, .title")
	if !found || len(sels) != 3 {
		t.Fatalf("ParseSelectors(h1, h2, .title) = %d sels, ok=%v", len(sels), found)
	}

	for i, tag := range []string{"h1", "h2", "*"} {
		if got := sels[i].Parts[0].Tag; got != tag {
			t.Errorf("selector %d tag = %q, want %q", i, got, tag)
		}
	}
	// Single selector.
	single, found := ParseSelectors("div > p.a")
	if !found || len(single) != 1 || len(single[0].Parts) != 2 {
		t.Fatalf("ParseSelectors(div > p.a) = %+v, ok=%v", single, found)
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
