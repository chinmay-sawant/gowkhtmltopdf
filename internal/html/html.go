// Package html implements a tokenizer and tree builder for the HTML subset
// gowkhtmltopdf accepts: tags, attributes, text, comments, doctype, CDATA,
// self-closing and void elements. Script/style contents are kept as raw text
// and stripped at the layout stage. No browser-grade error recovery: common
// malformed nesting degrades to a usable tree, not a crash.
package html

import (
	"errors"
	"strings"
)

// NodeType classifies a DOM node.
type NodeType int

const (
	ElementNode NodeType = iota
	TextNode
	CommentNode
	DoctypeNode
)

// Node is one DOM node.
type Node struct {
	Type      NodeType
	Name      string // element name (lowercased) for elements
	Attrs     map[string]string
	AttrOrder []string // preserves source order for deterministic output
	Text      string   // text/comment/doctype content
	Children  []*Node
	Parent    *Node
}

// Attribute returns an attribute value, or "".
func (n *Node) Attribute(name string) string { return n.Attrs[strings.ToLower(name)] }

// FirstChild returns the first element child with name, or nil.
func (n *Node) FirstChild(name string) *Node {
	for _, c := range n.Children {
		if c.Type == ElementNode && c.Name == name {
			return c
		}
	}
	return nil
}

// ElementChildren returns element children.
func (n *Node) ElementChildren() []*Node {
	out := make([]*Node, 0, len(n.Children))
	for _, c := range n.Children {
		if c.Type == ElementNode {
			out = append(out, c)
		}
	}
	return out
}

// TextContent concatenates all descendant text.
func (n *Node) TextContent() string {
	var b strings.Builder
	n.appendText(&b)
	return b.String()
}

func (n *Node) appendText(b *strings.Builder) {
	switch n.Type {
	case TextNode:
		b.WriteString(n.Text)
	case ElementNode:
		for _, c := range n.Children {
			c.appendText(b)
		}
	}
}

// voidElements never take content; rawTextElements consume everything until
// their closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

var rawTextElements = map[string]bool{
	"script": true, "style": true, "textarea": true, "title": true,
}

// autoClose[next] lists the open elements that a next start tag closes.
var autoClose = map[string][]string{
	"li":     {"li"},
	"p":      {"p"},
	"tr":     {"tr", "td", "th"},
	"td":     {"td", "th"},
	"th":     {"td", "th"},
	"option": {"option"},
	"dd":     {"dd", "dt"},
	"dt":     {"dd", "dt"},
	"thead":  {"thead", "tbody", "tfoot"},
	"tbody":  {"thead", "tbody", "tfoot"},
	"tfoot":  {"thead", "tbody", "tfoot"},
	"head":   {"body", "head"},
	"body":   {"head", "body"},
	"html":   {"html", "head", "body"},
}

// Parse turns HTML source into a tree with a synthetic root. The source is
// decoded UTF-8; callers handle charset detection beforehand.
func Parse(source string) (*Node, error) {
	tok, err := tokenize(source)
	if err != nil {
		return nil, err
	}
	root := &Node{Type: ElementNode, Name: "#document"}
	stack := []*Node{root}

	for _, t := range tok {
		top := stack[len(stack)-1]
		switch t.kind {
		case tokDoctype:
			top.Children = append(top.Children, &Node{Type: DoctypeNode, Text: t.data})
		case tokComment:
			top.Children = append(top.Children, &Node{Type: CommentNode, Text: t.data})
		case tokText:
			if len(t.data) == 0 {
				continue
			}
			if len(top.Children) > 0 {
				last := top.Children[len(top.Children)-1]
				if last.Type == TextNode {
					last.Text += t.data
					continue
				}
			}
			node := &Node{Type: TextNode, Text: t.data}
			node.Parent = top
			top.Children = append(top.Children, node)
		case tokStart:
			name := strings.ToLower(t.data)
			// html/head/body duplicates merge into the existing element
			// instead of nesting: drop the token if one is already open,
			// otherwise re-open a closed same-level sibling.
			if name == "html" || name == "head" || name == "body" {
				if openInStack(stack, name) {
					continue
				}
				if existing := findImplicit(top, name); existing != nil {
					stack = append(stack, existing)
					continue
				}
			}
			closedCell := false
			for len(stack) > 1 {
				openName := stack[len(stack)-1].Name
				if shouldAutoClose(openName, name) {
					if openName == "td" || openName == "th" {
						closedCell = true
					}
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			// close-a-cell: a new <td>/<th> after an open cell ends the row too
			if closedCell && (name == "td" || name == "th") && len(stack) > 1 {
				if stack[len(stack)-1].Name == "tr" {
					stack = stack[:len(stack)-1]
				}
			}
			top = stack[len(stack)-1]
			node := &Node{Type: ElementNode, Name: name, Attrs: map[string]string{}}
			for i := 0; i+1 < len(t.attrs); i += 2 {
				key := strings.ToLower(t.attrs[i])
				val := t.attrs[i+1]
				if _, dup := node.Attrs[key]; !dup {
					node.AttrOrder = append(node.AttrOrder, key)
					node.Attrs[key] = val
				}
			}
			top.Children = append(top.Children, node)
			node.Parent = top
			if t.selfClosing || voidElements[name] {
				continue // no child content
			}
			stack = append(stack, node)
		case tokEnd:
			name := strings.ToLower(t.data)
			for i := len(stack) - 1; i > 0; i-- {
				if stack[i].Name == name {
					stack = stack[:i]
					break
				}
			}
		}
	}
	return root, nil
}

// CharsetFromMeta scans the document for a meta element declaring a charset,
// either as <meta charset="..."> or as <meta http-equiv="content-type"
// content="...; charset=...">, and returns the charset name lowercased, or ""
// if none is declared.
func CharsetFromMeta(root *Node) string {
	for _, c := range root.Children {
		if cs := charsetFromNode(c); cs != "" {
			return cs
		}
		if c.Type == ElementNode {
			if cs := CharsetFromMeta(c); cs != "" {
				return cs
			}
		}
	}
	return ""
}

func charsetFromNode(n *Node) string {
	if n.Type != ElementNode || n.Name != "meta" {
		return ""
	}
	if cs := n.Attribute("charset"); cs != "" {
		return strings.ToLower(cs)
	}
	if !strings.EqualFold(strings.TrimSpace(n.Attribute("http-equiv")), "content-type") {
		return ""
	}
	content := strings.ToLower(n.Attribute("content"))
	i := strings.Index(content, "charset=")
	if i < 0 {
		return ""
	}
	cs := content[i+len("charset="):]
	if j := strings.IndexAny(cs, " \t;"); j >= 0 {
		cs = cs[:j]
	}
	return strings.Trim(cs, "\"'")
}

// openInStack reports whether an element with name is currently open.
func openInStack(stack []*Node, name string) bool {
	for i := len(stack) - 1; i > 0; i-- {
		if stack[i].Name == name {
			return true
		}
	}
	return false
}

// findImplicit looks for an existing same-name element child of top
// (browser-style html/head/body merging).
func findImplicit(top *Node, name string) *Node {
	for _, c := range top.Children {
		if c.Type == ElementNode && c.Name == name {
			return c
		}
	}
	return nil
}

// shouldAutoClose reports whether a start tag next closes the open element
// open.
func shouldAutoClose(open, next string) bool {
	for _, n := range autoClose[next] {
		if n == open {
			return true
		}
	}
	return false
}

// tokenKind discriminates token types.
type tokenKind int

const (
	tokDoctype tokenKind = iota
	tokStart
	tokEnd
	tokText
	tokComment
)

type token struct {
	kind        tokenKind
	data        string
	attrs       []string // interleaved name, value
	selfClosing bool
}

// tokenize scans raw HTML into tokens without parsing structure.
func tokenize(src string) ([]token, error) {
	var toks []token
	i := 0
	n := len(src)
	for i < n {
		if src[i] != '<' {
			j := strings.IndexByte(src[i:], '<')
			if j < 0 {
				j = n - i
			}
			toks = append(toks, token{kind: tokText, data: src[i : i+j]})
			i += j
			continue
		}
		if i+1 >= n {
			toks = append(toks, token{kind: tokText, data: "<"})
			break
		}
		switch {
		case src[i+1] == '!':
			if strings.HasPrefix(src[i:], "<!--") {
				end := strings.Index(src[i+4:], "-->")
				if end < 0 {
					return nil, errors.New("html: unterminated comment")
				}
				toks = append(toks, token{kind: tokComment, data: src[i+4 : i+4+end]})
				i += 4 + end + 3
				continue
			}
			if len(src)-i >= 9 && strings.EqualFold(src[i:i+9], "<!doctype") {
				end := strings.IndexByte(src[i:], '>')
				if end < 0 {
					return nil, errors.New("html: unterminated doctype")
				}
				toks = append(toks, token{kind: tokDoctype, data: src[i+2 : i+end]})
				i += end + 1
				continue
			}
			// other bogus declaration → skip to >
			end := strings.IndexByte(src[i:], '>')
			if end < 0 {
				return nil, errors.New("html: unterminated declaration")
			}
			i += end + 1
		case src[i+1] == '/':
			end := strings.IndexByte(src[i:], '>')
			if end < 0 {
				return nil, errors.New("html: unterminated end tag")
			}
			name := strings.TrimSpace(src[i+2 : i+end])
			toks = append(toks, token{kind: tokEnd, data: strings.ToLower(name)})
			i += end + 1
		case src[i+1] == '?':
			end := strings.Index(src[i:], "?>")
			if end < 0 {
				return nil, errors.New("html: unterminated processing instruction")
			}
			i += end + 2
		default:
			if !isASCIILetter(src[i+1]) {
				// bare '<' followed by no valid tag start becomes text
				toks = append(toks, token{kind: tokText, data: "<"})
				i++
				continue
			}
			end, err := tagEnd(src, i)
			if err != nil {
				return nil, err
			}
			if end < 0 {
				// no closing '>' — treat the rest as text
				toks = append(toks, token{kind: tokText, data: src[i:]})
				i = n
				continue
			}
			tag := src[i+1 : end]
			name, attrs, selfClose, err := parseTag(tag)
			if err != nil {
				return nil, err
			}
			if name == "" {
				i = end + 1
				continue
			}
			name = strings.ToLower(name)
			toks = append(toks, token{kind: tokStart, data: name, attrs: attrs, selfClosing: selfClose})
			i = end + 1
			// raw-text elements capture everything until their closing tag
			if rawTextElements[name] && !selfClose {
				s, e, ok := rawTextEnd(src, i, name)
				if !ok {
					toks = append(toks, token{kind: tokText, data: src[i:]})
					i = n
					continue
				}
				if s > i {
					toks = append(toks, token{kind: tokText, data: src[i:s]})
				}
				toks = append(toks, token{kind: tokEnd, data: name})
				i = e + 1
			}
		}
	}
	return toks, nil
}

// tagEnd returns the index of the '>' closing the start tag that begins at
// start, respecting quoted attribute values; -1 if the tag never closes.
func tagEnd(src string, start int) (int, error) {
	for j := start + 1; j < len(src); j++ {
		switch src[j] {
		case '"', '\'':
			q := src[j]
			k := strings.IndexByte(src[j+1:], q)
			if k < 0 {
				return 0, errors.New("html: unterminated attribute value")
			}
			j += k + 1
		case '>':
			return j, nil
		}
	}
	return -1, nil
}

// rawTextEnd finds the closing tag of a raw-text element whose content starts
// at from. It returns the span of the closing tag, or ok=false if the content
// runs to the end of the source.
func rawTextEnd(src string, from int, name string) (start, end int, ok bool) {
	low := strings.ToLower(src)
	needle := "</" + name
	for {
		k := strings.Index(low[from:], needle)
		if k < 0 {
			return 0, 0, false
		}
		k += from
		j := k + len(needle)
		for j < len(src) && isWhitespace(src[j]) {
			j++
		}
		if j < len(src) && src[j] == '>' {
			return k, j, true
		}
		from = k + 2
	}
}

// parseTag extracts the tag name and attribute pairs from a <...> body.
func parseTag(body string) (string, []string, bool, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", nil, false, nil
	}
	// name ends at first whitespace or '/'
	nameEnd := len(body)
	for j := 0; j < len(body); j++ {
		if isWhitespace(body[j]) || body[j] == '/' {
			nameEnd = j
			break
		}
	}
	name := body[:nameEnd]
	selfClose := strings.HasSuffix(body, "/")
	var attrs []string
	rest := strings.TrimSpace(body[nameEnd:])
	rest = strings.TrimSuffix(rest, "/")
	for rest != "" {
		rest = strings.TrimLeft(rest, " \t\n\r")
		if rest == "" {
			break
		}
		// attribute name: up to '=' or whitespace
		j := 0
		for j < len(rest) && rest[j] != '=' && !isWhitespace(rest[j]) {
			j++
		}
		key := rest[:j]
		rest = strings.TrimLeft(rest[j:], " \t\n\r")
		if !strings.HasPrefix(rest, "=") {
			attrs = append(attrs, key, "")
			continue
		}
		rest = strings.TrimLeft(rest[1:], " \t\n\r")
		var val string
		if rest == "" {
			attrs = append(attrs, key, "")
			continue
		}
		switch rest[0] {
		case '"', '\'':
			q := rest[0]
			end := strings.IndexByte(rest[1:], q)
			if end < 0 {
				return "", nil, false, errors.New("html: unterminated attribute value")
			}
			val = rest[1 : end+1]
			rest = rest[end+2:]
		default:
			end := strings.IndexAny(rest, " \t\n\r")
			if end < 0 {
				val = rest
				rest = ""
			} else {
				val = rest[:end]
				rest = rest[end:]
			}
		}
		attrs = append(attrs, key, val)
	}
	return name, attrs, selfClose, nil
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
