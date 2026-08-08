//nolint:testpackage // tokenizer/tree internals (tokenize, tokenKind) are tested from the same package
package html

import (
	"fmt"
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Node {
	t.Helper()

	root, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", src, err)
	}

	return root
}

// treeString renders the tree for failure messages.
func treeString(node *Node) string {
	var buf strings.Builder

	var rec func(*Node, int)
	rec = func(node *Node, depth int) {
		buf.WriteString(strings.Repeat("  ", depth))

		switch node.Type {
		case ElementNode:
			fmt.Fprintf(&buf, "<%s>", node.Name)
		case TextNode:
			fmt.Fprintf(&buf, "#text %q", node.Text)
		case CommentNode:
			fmt.Fprintf(&buf, "<!--%s-->", node.Text)
		case DoctypeNode:
			fmt.Fprintf(&buf, "<!%s>", node.Text)
		}

		buf.WriteByte('\n')

		for _, c := range node.Children {
			rec(c, depth+1)
		}
	}
	rec(node, 0)

	return buf.String()
}

func assertChildren(t *testing.T, node *Node, names ...string) {
	t.Helper()

	var got []*Node

	for _, c := range node.Children {
		if c.Type == ElementNode {
			got = append(got, c)
		}
	}

	if len(got) != len(names) {
		t.Errorf("children of <%s>: got %d elements, want %d\n%s", node.Name, len(got), len(names), treeString(node))

		return
	}

	for i, want := range names {
		if got[i].Name != want {
			t.Errorf("child %d of <%s>: got <%s>, want <%s>\n%s", i, node.Name, got[i].Name, want, treeString(node))
		}
	}
}

// --- tokenizer-level tests ---

func TestTokenizeAttributes(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<p id="a1" class='b2' data-x=unquoted hidden checked="yes">`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 {
		t.Fatalf("got %d tokens, want 1: %+v", len(toks), toks)
	}

	got := toks[0]
	if got.kind != tokStart || got.data != "p" {
		t.Fatalf("token = %+v, want start <p>", got)
	}

	want := []string{"id", "a1", "class", "b2", "data-x", "unquoted", "hidden", "", "checked", "yes"}
	if len(got.attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", got.attrs, want)
	}

	for i := range want {
		if got.attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, got.attrs[i], want[i])
		}
	}
}

func TestTokenizeWhitespaceAroundEquals(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<div a = "x" b= y>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 {
		t.Fatalf("got %d tokens, want 1: %+v", len(toks), toks)
	}

	attrs := toks[0].attrs
	want := []string{"a", "x", "b", "y"}

	if len(attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", attrs, want)
	}

	for i := range want {
		if attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, attrs[i], want[i])
		}
	}
}

func TestTokenizeGreaterThanInQuotedValue(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<p title="a > b" data-x='1>0'>x</p>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 3 {
		t.Fatalf("got %d tokens, want 3: %+v", len(toks), toks)
	}

	attrs := toks[0].attrs
	want := []string{"title", "a > b", "data-x", "1>0"}

	if len(attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", attrs, want)
	}

	for i := range want {
		if attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, attrs[i], want[i])
		}
	}
}

func TestTokenizeComments(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`a<!-- hello -->b<!---->c`)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		kind tokenKind
		data string
	}{
		{tokText, "a"},
		{tokComment, " hello "},
		{tokText, "b"},
		{tokComment, ""},
		{tokText, "c"},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}

	for i, wantTok := range want {
		if toks[i].kind != wantTok.kind || toks[i].data != wantTok.data {
			t.Errorf("token %d = %+v, want %+v", i, toks[i], wantTok)
		}
	}
}

func TestTokenizeDoctype(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<!DOCTYPE html><p>x</p>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 4 {
		t.Fatalf("got %d tokens, want 4: %+v", len(toks), toks)
	}

	if toks[0].kind != tokDoctype || toks[0].data != "DOCTYPE html" {
		t.Errorf("token 0 = %+v", toks[0])
	}

	for i, wantKind := range []tokenKind{tokDoctype, tokStart, tokText, tokEnd} {
		if toks[i].kind != wantKind {
			t.Errorf("token %d kind = %v, want %v", i, toks[i].kind, wantKind)
		}
	}

	toks, err = tokenize(`<!DoCtYpE html>ok`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 2 || toks[0].kind != tokDoctype {
		t.Fatalf("mixed-case doctype = %+v, want doctype token", toks)
	}
}

func TestTokenizeDeclarationsAndPI(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<?xml version="1.0"?><!bogus stuff><p>x</p>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 3 {
		t.Fatalf("got %d tokens, want 3: %+v", len(toks), toks)
	}

	for i, wantKind := range []tokenKind{tokStart, tokText, tokEnd} {
		if toks[i].kind != wantKind {
			t.Errorf("token %d kind = %v, want %v", i, toks[i].kind, wantKind)
		}
	}
}

func TestTokenizeRawText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src, content string
	}{
		{`<script>if (a < b) { x(); }</script>`, "if (a < b) { x(); }"},
		{`<style>p > b { color: red; }</style>`, "p > b { color: red; }"},
		{`<textarea><b>not bold</b></textarea>`, "<b>not bold</b>"},
		{`<title>My <Page></title>`, "My <Page>"},
		{`<SCRIPT>var a = 1;</script>`, "var a = 1;"},
		{`<script src="x.js"></script>`, ""},
		{`<script>a</SCRIPT>b`, "ab"},
		{`<script>var x = 1;`, "var x = 1;"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		var content string

		for _, tk := range toks {
			if tk.kind == tokText {
				content += tk.data
			}
		}

		if content != testCase.content {
			t.Errorf("tokenize(%q): raw text = %q, want %q (tokens %+v)", testCase.src, content, testCase.content, toks)
		}
	}
}

func TestTokenizeRawTextClosesOnlyRealEndTag(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<script>a</scriptx>b</script>`)
	if err != nil {
		t.Fatal(err)
	}

	var text string

	for _, tk := range toks {
		if tk.kind == tokText {
			text += tk.data
		}
	}

	if text != "a</scriptx>b" {
		t.Errorf("raw text = %q, want %q (tokens %+v)", text, "a</scriptx>b", toks)
	}
}

func TestTokenizeBareLessThanIsText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src, want string
	}{
		{"1 < 2", "1 < 2"},
		{"a < b > c", "a < b > c"},
		{"<3>", "<3>"},
		{"<>empty<>", "<>empty<>"},
		{"< -x-", "< -x-"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		var text string

		for _, tk := range toks {
			if tk.kind != tokText {
				t.Fatalf("tokenize(%q): unexpected non-text token %+v", testCase.src, tk)
			}

			text += tk.data
		}

		if text != testCase.want {
			t.Errorf("tokenize(%q) = %q, want %q", testCase.src, text, testCase.want)
		}
	}
}

func TestTokenizeUnterminated(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src, wantErr string
	}{
		{"<!-- unterminated", "unterminated comment"},
		{"</div", "unterminated end tag"},
		{`<div a="x`, "unterminated attribute value"},
		{`<div a='x`, "unterminated attribute value"},
		{`<div a="x>`, "unterminated attribute value"},
		{"<!DOCTYPE", "unterminated doctype"},
		{"<!bogus", "unterminated declaration"},
		{"<?pi", "unterminated processing instruction"},
	}
	for _, testCase := range cases {
		if _, err := Parse(testCase.src); err == nil {
			t.Errorf("Parse(%q): want error %q, got nil", testCase.src, testCase.wantErr)
		} else if !strings.Contains(err.Error(), testCase.wantErr) {
			t.Errorf("Parse(%q): got error %q, want it to contain %q", testCase.src, err, testCase.wantErr)
		}
	}
}

// --- tree-level tests ---

func TestUnescapeEntitiesInText(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><body><h2>Docs &amp; forms</h2><p>a &lt; b &#38; c</p></body></html>`)
	body := root.FirstChild("body")

	if body == nil {
		body = root.FirstChild("html").FirstChild("body")
	}

	h2 := body.FirstChild("h2")
	if h2 == nil || h2.TextContent() != "Docs & forms" {
		t.Fatalf("h2 text = %q, want Docs & forms", h2.TextContent())
	}

	p := body.FirstChild("p")
	if p == nil || p.TextContent() != "a < b & c" {
		t.Fatalf("p text = %q", p.TextContent())
	}
}

func TestParseNesting(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><head><title>t</title></head><body><div><p>hi</p></div></body></html>`)
	html := root.FirstChild("html")

	if html == nil {
		t.Fatalf("no <html>:\n%s", treeString(root))
	}

	assertChildren(t, html, "head", "body")
	assertChildren(t, html.FirstChild("head"), "title")
	assertChildren(t, html.FirstChild("body"), "div")
	assertChildren(t, html.FirstChild("body").FirstChild("div"), "p")

	if got := html.FirstChild("body").TextContent(); got != "hi" {
		t.Errorf("TextContent = %q, want %q", got, "hi")
	}
}

func TestParseParentPointers(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><body><div>x</div><p>y</p></body></html>`)

	var walk func(*Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			if c.Parent != n {
				t.Errorf("Parent of %s = %v, want %s", c.Name, c.Parent, n.Name)
			}

			walk(c)
		}
	}
	walk(root)
}

func TestParseVoidElements(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<p>a<br>x<img src="y.png" alt="y"><input type="text" disabled><hr></p>`)
	para := root.FirstChild("p")

	if para == nil {
		t.Fatalf("no <p>:\n%s", treeString(root))
	}
	// text a, br, text x, img, input, hr - br/img/input/hr must not consume the following content
	if len(para.Children) != 6 {
		t.Fatalf("<p> has %d children, want 6:\n%s", len(para.Children), treeString(para))
	}

	assertChildren(t, para, "br", "img", "input", "hr")

	if got := para.TextContent(); got != "ax" {
		t.Errorf("TextContent = %q, want %q", got, "ax")
	}

	img := para.FirstChild("img")
	if img.Attribute("src") != "y.png" || img.Attribute("alt") != "y" {
		t.Errorf("img attrs = %v", img.Attrs)
	}

	if len(img.Children) != 0 {
		t.Errorf("void <img> has children:\n%s", treeString(img))
	}
}

func TestParseAutoCloseTable(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<table><tr><td>a</td><td>b</td></tr><tr><td>c</td></tr></table>`)
	table := root.FirstChild("table")
	assertChildren(t, table, "tr", "tr")
	assertChildren(t, table.FirstChild("tr"), "td", "td")
	assertChildren(t, table.Children[1], "td")

	// <tr> closes an open <tr>/<td>/<th>
	root = mustParse(t, `<table><tr><td>a<tr><td>b</table>`)
	table = root.FirstChild("table")
	assertChildren(t, table, "tr", "tr")
	assertChildren(t, table.Children[0], "td")
	assertChildren(t, table.Children[1], "td")

	if got := table.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}

	// <td> closes an open <td>/<th>/<tr>
	root = mustParse(t, `<table><tr><td>a<td>b</table>`)
	table = root.FirstChild("table")
	assertChildren(t, table, "tr", "td")
}

func TestParseAutoCloseP(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<div><p>a<p>b</div>`)
	div := root.FirstChild("div")
	assertChildren(t, div, "p", "p")

	if got := div.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}
}

func TestParseAutoCloseList(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<ul><li>a<li>b</ul>`)
	ul := root.FirstChild("ul")
	assertChildren(t, ul, "li", "li")

	root = mustParse(t, `<select><option>a<option>b</select>`)
	sel := root.FirstChild("select")
	assertChildren(t, sel, "option", "option")

	root = mustParse(t, `<dl><dt>t<dd>d<dt>t2</dl>`)
	dl := root.FirstChild("dl")
	assertChildren(t, dl, "dt", "dd", "dt")
}

func TestParseAutoCloseTableSections(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<table><thead>h<tbody>b<tfoot>f</table>`)
	table := root.FirstChild("table")
	assertChildren(t, table, "thead", "tbody", "tfoot")

	root = mustParse(t, `<table><tbody>b<thead>h</table>`)
	table = root.FirstChild("table")
	assertChildren(t, table, "tbody", "thead")

	root = mustParse(t, `<table><tfoot>f<tbody>b</table>`)
	table = root.FirstChild("table")
	assertChildren(t, table, "tfoot", "tbody")
}

func TestParseHtmlHeadBodyMerge(t *testing.T) {
	t.Parallel()
	// second <head> merges into the existing one at the same level
	root := mustParse(t, `<html><head>a</head><head>b</head></html>`)
	html := root.FirstChild("html")

	if html == nil {
		t.Fatalf("no <html>:\n%s", treeString(root))
	}

	assertChildren(t, html, "head")

	if got := html.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}

	// second <body> merges into the existing one at the same level
	root = mustParse(t, `<body>x</body><body>y</body>`)
	assertChildren(t, root, "body")

	if got := root.TextContent(); got != "xy" {
		t.Errorf("TextContent = %q, want %q", got, "xy")
	}

	// second <html> merges into the existing one
	root = mustParse(t, `<html>a</html><html>b</html>`)
	assertChildren(t, root, "html")

	if got := root.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}

	// nested duplicate <body> is dropped, not nested and not closed
	root = mustParse(t, `<body><body>z</body>`)
	assertChildren(t, root, "body")

	if got := root.TextContent(); got != "z" {
		t.Errorf("TextContent = %q, want %q", got, "z")
	}

	// nested duplicate <body> with following content inside html
	root = mustParse(t, `<html><body>x<body>y</html>`)
	html = root.FirstChild("html")
	assertChildren(t, html, "body")

	if got := html.TextContent(); got != "xy" {
		t.Errorf("TextContent = %q, want %q", got, "xy")
	}
}

func TestParseHeadBodyTransition(t *testing.T) {
	t.Parallel()
	// <body> closes an open <head>
	root := mustParse(t, `<head><title>t</title></head><body>b</body>`)
	assertChildren(t, root, "head", "body")

	root = mustParse(t, `<html><head><title>t</title></head><body>b</body></html>`)
	html := root.FirstChild("html")
	assertChildren(t, html, "head", "body")
}

//nolint:cyclop // sequential scenario assertions, not branch logic
func TestParseTextMerging(t *testing.T) {
	t.Parallel()
	// adjacent text tokens merge into a single TextNode
	root := mustParse(t, `1 < 2`)
	if len(root.Children) != 1 {
		t.Fatalf("root has %d children, want 1:\n%s", len(root.Children), treeString(root))
	}

	txt := root.Children[0]
	if txt.Type != TextNode || txt.Text != "1 < 2" {
		t.Errorf("child = %+v, want single TextNode \"1 < 2\"", txt)
	}

	// text around comments stays as separate nodes
	root = mustParse(t, `a<!-- c -->b`)
	if len(root.Children) != 3 {
		t.Fatalf("root has %d children, want 3:\n%s", len(root.Children), treeString(root))
	}

	if root.Children[0].Type != TextNode || root.Children[0].Text != "a" {
		t.Errorf("child 0 = %+v", root.Children[0])
	}

	if root.Children[2].Type != TextNode || root.Children[2].Text != "b" {
		t.Errorf("child 2 = %+v", root.Children[2])
	}

	// bare '<' sequences inside an element merge
	root = mustParse(t, `<p>x <3>y</p>`)

	p := root.FirstChild("p")
	if len(p.Children) != 1 || p.Children[0].Type != TextNode || p.Children[0].Text != "x <3>y" {
		t.Fatalf("p children = %+v, want one TextNode \"x <3>y\"\n%s", p.Children, treeString(p))
	}
}

func TestParseAttrDuplicates(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<div ID="a" class="b" data-x="1" hidden id="dup">x</div>`)

	div := root.FirstChild("div")
	if len(div.Attrs) != 4 {
		t.Errorf("Attrs = %v, want 4 entries", div.Attrs)
	}
	// duplicate keeps the FIRST value
	if div.Attribute("id") != "a" {
		t.Errorf("id = %q, want first value %q", div.Attribute("id"), "a")
	}
	// attribute names are lowercased
	if div.Attribute("ID") != "a" || div.Attribute("Class") != "b" {
		t.Errorf("case-sensitive lookup failed: %v", div.Attrs)
	}

	if div.Attribute("hidden") != "" {
		t.Errorf("boolean attr hidden = %q, want \"\"", div.Attribute("hidden"))
	}

	if div.Attribute("data-x") != "1" {
		t.Errorf("data-x = %q, want %q", div.Attribute("data-x"), "1")
	}
}

func TestParseSelfClosing(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<div/><span>x</span>`)
	assertChildren(t, root, "div", "span")

	div := root.FirstChild("div")
	if len(div.Children) != 0 {
		t.Errorf("self-closing <div/> has children:\n%s", treeString(div))
	}

	// self-closing raw-text element takes no raw content
	root = mustParse(t, `<script src="x.js"/>ok`)
	assertChildren(t, root, "script")

	if got := root.TextContent(); got != "ok" {
		t.Errorf("TextContent = %q, want %q", got, "ok")
	}
}

func TestParseCommentsAndDoctype(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<!DOCTYPE html><html><body><!-- hello -->x</body></html>`)
	if len(root.Children) != 2 {
		t.Fatalf("root has %d children, want 2:\n%s", len(root.Children), treeString(root))
	}

	if root.Children[0].Type != DoctypeNode || root.Children[0].Text != "DOCTYPE html" {
		t.Errorf("child 0 = %+v", root.Children[0])
	}

	body := root.FirstChild("html").FirstChild("body")
	if len(body.Children) != 2 {
		t.Fatalf("body has %d children, want 2:\n%s", len(body.Children), treeString(body))
	}

	if body.Children[0].Type != CommentNode || body.Children[0].Text != " hello " {
		t.Errorf("child 0 = %+v", body.Children[0])
	}

	if body.Children[1].Type != TextNode || body.Children[1].Text != "x" {
		t.Errorf("child 1 = %+v", body.Children[1])
	}
}

func TestParseRawTextTree(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<script>if (a < b) { f(); }</script><p>ok</p>`)
	script := root.FirstChild("script")

	if len(script.Children) != 1 || script.Children[0].Type != TextNode {
		t.Fatalf("script children = %+v, want one TextNode\n%s", script.Children, treeString(root))
	}

	if script.Children[0].Text != "if (a < b) { f(); }" {
		t.Errorf("script text = %q", script.Children[0].Text)
	}

	assertChildren(t, root, "script", "p")

	if got := root.TextContent(); got != "if (a < b) { f(); }ok" {
		t.Errorf("TextContent = %q", got)
	}
}

//nolint:cyclop,funlen // table-driven scenario checks (closures are the scenario assertions)
func TestParseMalformed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src   string
		check func(t *testing.T, root *Node)
	}{
		{
			src: `<p><b>bold`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				p := root.FirstChild("p")
				assertChildren(t, p, "b")
				if got := root.TextContent(); got != "bold" {
					t.Errorf("TextContent = %q", got)
				}
			},
		},
		{
			src: `</div>text`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				for _, c := range root.Children {
					if c.Type == ElementNode {
						t.Errorf("stray end tag produced element:\n%s", treeString(root))

						return
					}
				}
				if root.TextContent() != "text" {
					t.Errorf("stray end tag:\n%s", treeString(root))
				}
			},
		},
		{
			src: `<div><span><p>x</div>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				div := root.FirstChild("div")
				assertChildren(t, div, "span")
				assertChildren(t, div.FirstChild("span"), "p")
			},
		},
		{
			src: `<table><tr><td>cell`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				table := root.FirstChild("table")
				assertChildren(t, table, "tr")
				assertChildren(t, table.FirstChild("tr"), "td")
				if got := root.TextContent(); got != "cell" {
					t.Errorf("TextContent = %q", got)
				}
			},
		},
		{
			src: `<>empty<>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				if got := root.TextContent(); got != "<>empty<>" {
					t.Errorf("TextContent = %q, want %q", got, "<>empty<>")
				}
				if len(root.Children) != 1 || root.Children[0].Type != TextNode {
					t.Errorf("children:\n%s", treeString(root))
				}
			},
		},
		{
			src: `<div a=b/ c="d" e>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				div := root.FirstChild("div")
				if div == nil {
					t.Fatalf("no <div>:\n%s", treeString(root))
				}
				if div.Attribute("c") != "d" || div.Attribute("e") != "" {
					t.Errorf("attrs = %v", div.Attrs)
				}
			},
		},
		{
			src: `<ul><li>a<li>b`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				assertChildren(t, root.FirstChild("ul"), "li", "li")
			},
		},
	}
	for _, testCase := range cases {
		root := mustParse(t, testCase.src)
		testCase.check(t, root)
	}
}

func TestParseUsableTreeNoPanic(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`<div><div><div><div>x`,
		`<b><i><u>deep</b>`,
		`<table><thead><tr><th>h<td>d`,
		`<p a=1><p a=2><p a=3>`,
		`<script><script></script>`,
		`<style>a{}</style>`,
		"",
		`plain text only`,
		`<html><head><meta charset="utf-8"><title>t</title></head><body><br><img></body></html>`,
	}
	for _, src := range inputs {
		if _, err := Parse(src); err != nil {
			t.Errorf("Parse(%q): %v", src, err)
		}
	}
}

func TestWalkPreOrder(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><head><title>t</title></head><body><h1>x</h1><p>y</p></body></html>`)

	var names []string

	root.Walk(func(n *Node) {
		if n.Type == ElementNode {
			names = append(names, n.Name)
		}
	})

	want := []string{"#document", "html", "head", "title", "body", "h1", "p"}
	if len(names) != len(want) {
		t.Fatalf("walk order = %v, want %v", names, want)
	}

	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("walk order = %v, want %v", names, want)
		}
	}
}

func TestTextContentOf(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><head><title>One</title><title>Two</title></head><body><p>body text</p></body></html>`)
	if got := root.TextContentOf("title"); got != "One" {
		t.Errorf("TextContentOf(title) = %q, want %q", got, "One")
	}

	if got := root.TextContentOf("section"); got != "" {
		t.Errorf("TextContentOf(section) = %q, want %q", got, "")
	}

	if got := root.TextContentOf("p"); got != "body text" {
		t.Errorf("TextContentOf(p) = %q, want %q", got, "body text")
	}
}

func TestParseDocument(t *testing.T) {
	t.Parallel()

	src := "<html><body>ok</body></html>"
	for _, body := range [][]byte{[]byte(src), append([]byte("\ufeff"), src...)} {
		root, err := ParseDocument(body)
		if err != nil {
			t.Fatalf("ParseDocument: %v", err)
		}

		if root.FirstChild("html") == nil {
			t.Errorf("ParseDocument(%q): no html element", body[:4])
		}
	}
}
