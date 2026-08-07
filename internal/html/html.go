// Package html implements a tokenizer and tree builder for the HTML subset
// gowkhtmltopdf accepts: tags, attributes, text, comments, doctype, CDATA,
// self-closing and void elements. Script/style contents are kept as raw text
// and stripped at the layout stage. No browser-grade error recovery: common
// malformed nesting degrades to a usable tree, not a crash.
//
// ponytail: custom Node tree (Parent/Attrs/void); migrate to x/net/html only if layout/css rewritten, not free delete.
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
	Type     NodeType
	Name     string // element name (lowercased) for elements
	Attrs    map[string]string
	Text     string // text/comment/doctype content
	Children []*Node
	Parent   *Node
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

// TextContent concatenates all descendant text.
func (n *Node) TextContent() string {
	var b strings.Builder

	n.appendText(&b)

	return b.String()
}

// Walk visits n and every descendant in pre-order (document order).
func (n *Node) Walk(f func(*Node)) {
	f(n)

	for _, c := range n.Children {
		c.Walk(f)
	}
}

// TextContentOf returns the text content of the first element descendant
// named name, or "" when there is none.
func (n *Node) TextContentOf(name string) string {
	var out string

	n.Walk(func(c *Node) {
		if out == "" && c.Type == ElementNode && c.Name == name {
			out = c.TextContent()
		}
	})

	return out
}

func (n *Node) appendText(buf *strings.Builder) {
	switch n.Type {
	case TextNode:
		buf.WriteString(n.Text)
	case ElementNode:
		for _, c := range n.Children {
			c.appendText(buf)
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
// decoded UTF-8; charset detection happens at the load seam (internal/load).
// Use ParseDocument for the bytes-to-tree path (it strips the BOM).
func Parse(source string) (*Node, error) {
	tok, err := tokenize(source)
	if err != nil {
		return nil, err
	}

	root := &Node{Type: ElementNode, Name: "#document"} //nolint:exhaustruct // intentional zero/partial fields
	stack := []*Node{root}

	for _, tokItem := range tok {
		top := stack[len(stack)-1]

		switch tokItem.kind {
		case tokDoctype:
			top.Children = append(top.Children, &Node{Type: DoctypeNode, Text: tokItem.data}) //nolint:exhaustruct // intentional zero/partial fields
		case tokComment:
			top.Children = append(top.Children, &Node{Type: CommentNode, Text: tokItem.data}) //nolint:exhaustruct // intentional zero/partial fields
		case tokText:
			if len(tokItem.data) == 0 {
				continue
			}

			data := UnescapeEntities(tokItem.data)

			if len(top.Children) > 0 {
				last := top.Children[len(top.Children)-1]
				if last.Type == TextNode {
					last.Text += data

					continue
				}
			}

			node := &Node{Type: TextNode, Text: data} //nolint:exhaustruct // intentional zero/partial fields
			node.Parent = top
			top.Children = append(top.Children, node)
		case tokStart:
			name := strings.ToLower(tokItem.data)
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
			node := &Node{Type: ElementNode, Name: name, Attrs: map[string]string{}} //nolint:exhaustruct // intentional zero/partial fields

			for i := 0; i+1 < len(tokItem.attrs); i += 2 {
				key := strings.ToLower(tokItem.attrs[i])
				val := UnescapeEntities(tokItem.attrs[i+1])

				if _, dup := node.Attrs[key]; !dup {
					node.Attrs[key] = val
				}
			}

			top.Children = append(top.Children, node)
			node.Parent = top

			if tokItem.selfClosing || voidElements[name] {
				continue // no child content
			}

			stack = append(stack, node)
		case tokEnd:
			name := strings.ToLower(tokItem.data)
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

// ParseDocument turns raw document bytes into a tree with a synthetic root,
// stripping a leading UTF-8 BOM (mirroring load.IsHTML). Only UTF-8/ASCII
// sources are supported; the charset rule is enforced at the load seam.
func ParseDocument(body []byte) (*Node, error) {
	s := string(body)
	if strings.HasPrefix(s, "\ufeff") { // BOM, mirroring load.IsHTML
		s = s[1:]
	}

	return Parse(s)
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

const (
	commentPrefixLen = 4 // len("<!--")
	commentSuffixLen = 3 // len("-->")
	piCloseLen       = 2 // len("?>")
	rawCloseMinSkip  = 2 // len("</")
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

	pos := 0
	srcLen := len(src)

	for pos < srcLen {
		if src[pos] != '<' {
			span := strings.IndexByte(src[pos:], '<')
			if span < 0 {
				span = srcLen - pos
			}

			toks = append(toks, token{kind: tokText, data: src[pos : pos+span]}) //nolint:exhaustruct // intentional zero/partial fields
			pos += span

			continue
		}

		if pos+1 >= srcLen {
			toks = append(toks, token{kind: tokText, data: "<"}) //nolint:exhaustruct // intentional zero/partial fields

			break
		}

		switch {
		case src[pos+1] == '!':
			if strings.HasPrefix(src[pos:], "<!--") {
				end := strings.Index(src[pos+commentPrefixLen:], "-->")
				if end < 0 {
					return nil, errors.New("html: unterminated comment")
				}

				toks = append(toks, token{kind: tokComment, data: src[pos+commentPrefixLen : pos+commentPrefixLen+end]}) //nolint:exhaustruct // intentional zero/partial fields
				pos += commentPrefixLen + end + commentSuffixLen

				continue
			}

			if len(src)-pos >= 9 && strings.EqualFold(src[pos:pos+9], "<!doctype") {
				end := strings.IndexByte(src[pos:], '>')
				if end < 0 {
					return nil, errors.New("html: unterminated doctype")
				}

				toks = append(toks, token{kind: tokDoctype, data: src[pos+2 : pos+end]}) //nolint:exhaustruct // intentional zero/partial fields
				pos += end + 1

				continue
			}
			// other bogus declaration → skip to >
			end := strings.IndexByte(src[pos:], '>')
			if end < 0 {
				return nil, errors.New("html: unterminated declaration")
			}

			pos += end + 1
		case src[pos+1] == '/':
			end := strings.IndexByte(src[pos:], '>')
			if end < 0 {
				return nil, errors.New("html: unterminated end tag")
			}

			name := strings.TrimSpace(src[pos+2 : pos+end])
			toks = append(toks, token{kind: tokEnd, data: strings.ToLower(name)}) //nolint:exhaustruct // intentional zero/partial fields
			pos += end + 1
		case src[pos+1] == '?':
			end := strings.Index(src[pos:], "?>")
			if end < 0 {
				return nil, errors.New("html: unterminated processing instruction")
			}

			pos += end + piCloseLen
		default:
			if !isASCIILetter(src[pos+1]) {
				// bare '<' followed by no valid tag start becomes text
				toks = append(toks, token{kind: tokText, data: "<"}) //nolint:exhaustruct // intentional zero/partial fields
				pos++

				continue
			}

			end, err := tagEnd(src, pos)
			if err != nil {
				return nil, err
			}

			if end < 0 {
				// no closing '>' - treat the rest as text
				toks = append(toks, token{kind: tokText, data: src[pos:]}) //nolint:exhaustruct // intentional zero/partial fields
				pos = srcLen

				continue
			}

			tag := src[pos+1 : end]

			name, attrs, selfClose, err := parseTag(tag)
			if err != nil {
				return nil, err
			}

			if name == "" {
				pos = end + 1

				continue
			}

			name = strings.ToLower(name)
			toks = append(toks, token{kind: tokStart, data: name, attrs: attrs, selfClosing: selfClose})
			pos = end + 1
			// raw-text elements capture everything until their closing tag
			if rawTextElements[name] && !selfClose {
				rawStart, rawEnd, ok := rawTextEnd(src, pos, name)
				if !ok {
					toks = append(toks, token{kind: tokText, data: src[pos:]}) //nolint:exhaustruct // intentional zero/partial fields
					pos = srcLen

					continue
				}

				if rawStart > pos {
					toks = append(toks, token{kind: tokText, data: src[pos:rawStart]}) //nolint:exhaustruct // intentional zero/partial fields
				}

				toks = append(toks, token{kind: tokEnd, data: name}) //nolint:exhaustruct // intentional zero/partial fields
				pos = rawEnd + 1
			}
		}
	}

	return toks, nil
}

// tagEnd returns the index of the '>' closing the start tag that begins at
// start, respecting quoted attribute values; -1 if the tag never closes.
func tagEnd(src string, start int) (int, error) {
	for idx := start + 1; idx < len(src); idx++ {
		switch src[idx] {
		case '"', '\'':
			q := src[idx]

			k := strings.IndexByte(src[idx+1:], q)
			if k < 0 {
				return 0, errors.New("html: unterminated attribute value")
			}

			idx += k + 1
		case '>':
			return idx, nil
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
		found := strings.Index(low[from:], needle)
		if found < 0 {
			return 0, 0, false
		}

		found += from

		after := found + len(needle)
		for after < len(src) && isWhitespace(src[after]) {
			after++
		}

		if after < len(src) && src[after] == '>' {
			return found, after, true
		}

		from = found + rawCloseMinSkip
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

	for idx := range len(body) {
		if isWhitespace(body[idx]) || body[idx] == '/' {
			nameEnd = idx

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
		idx := 0
		for idx < len(rest) && rest[idx] != '=' && !isWhitespace(rest[idx]) {
			idx++
		}

		key := rest[:idx]
		rest = strings.TrimLeft(rest[idx:], " \t\n\r")

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
