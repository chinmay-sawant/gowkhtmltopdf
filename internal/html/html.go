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

// Tokenizer failure modes, as sentinels so callers can match and wrap them.
var (
	errUnterminatedComment = errors.New("html: unterminated comment")
	errUnterminatedDoctype = errors.New("html: unterminated doctype")
	errUnterminatedDecl    = errors.New("html: unterminated declaration")
	errUnterminatedEndTag  = errors.New("html: unterminated end tag")
	errUnterminatedPI      = errors.New("html: unterminated processing instruction")
	errUnterminatedAttrVal = errors.New("html: unterminated attribute value")
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
	case CommentNode, DoctypeNode:
		// comments and doctypes contribute no text
		return
	}
}

// Parse turns HTML source into a tree with a synthetic root. The source is
// decoded UTF-8; charset detection happens at the load seam (internal/load).
// Use ParseDocument for the bytes-to-tree path (it strips the BOM).
func Parse(source string) (*Node, error) {
	tok, err := tokenize(source)
	if err != nil {
		return nil, err
	}

	root := &Node{Type: ElementNode, Name: "#document"} //nolint:exhaustruct
	stack := []*Node{root}

	for _, tokItem := range tok {
		switch tokItem.kind {
		case tokDoctype:
			appendDoctypeToken(&stack, tokItem.data)
		case tokComment:
			appendCommentToken(&stack, tokItem.data)
		case tokText:
			appendTextToken(&stack, tokItem.data)
		case tokStart:
			openElement(&stack, tokItem)
		case tokEnd:
			closeElement(&stack, tokItem.data)
		}
	}

	return root, nil
}

// appendDoctypeToken attaches a doctype node to the current top of stack.
func appendDoctypeToken(stack *[]*Node, data string) {
	top := (*stack)[len(*stack)-1]
	top.Children = append(top.Children, &Node{Type: DoctypeNode, Text: data}) //nolint:exhaustruct
}

// appendCommentToken attaches a comment node to the current top of stack.
func appendCommentToken(stack *[]*Node, data string) {
	top := (*stack)[len(*stack)-1]
	top.Children = append(top.Children, &Node{Type: CommentNode, Text: data}) //nolint:exhaustruct
}

// appendTextToken attaches decoded text to the current top of stack, merging
// into an adjacent text node when present.
func appendTextToken(stack *[]*Node, data string) {
	if data == "" {
		return
	}

	decoded := UnescapeEntities(data)
	top := (*stack)[len(*stack)-1]

	if len(top.Children) > 0 {
		last := top.Children[len(top.Children)-1]
		if last.Type == TextNode {
			last.Text += decoded

			return
		}
	}

	node := &Node{Type: TextNode, Text: decoded} //nolint:exhaustruct
	node.Parent = top
	top.Children = append(top.Children, node)
}

// openElement applies one start tag to the open-element stack. Token data is
// already lowercased by the tokenizer.
func openElement(stack *[]*Node, tokItem token) {
	name := tokItem.data
	if mergeRootElement(stack, name) {
		return
	}

	autoCloseOpen(stack, name)

	top := (*stack)[len(*stack)-1]

	node := &Node{Type: ElementNode, Name: name} //nolint:exhaustruct

	if len(tokItem.attrs) > 0 {
		const attrPairSize = 2 // attrs slice interleaves name and value

		node.Attrs = make(map[string]string, len(tokItem.attrs)/attrPairSize)
		applyAttributes(node, tokItem.attrs)
	}

	top.Children = append(top.Children, node)
	node.Parent = top

	if tokItem.selfClosing || isVoidElement(name) {
		return // no child content
	}

	*stack = append(*stack, node)
}

// mergeRootElement handles html/head/body duplicates, which merge into the
// existing element instead of nesting: the token is dropped when one is
// already open, otherwise a closed same-level sibling is re-opened. It
// reports whether the token was consumed.
func mergeRootElement(stack *[]*Node, name string) bool {
	if name != "html" && name != "head" && name != "body" {
		return false
	}

	if openInStack(*stack, name) {
		return true
	}

	if existing := findImplicit((*stack)[len(*stack)-1], name); existing != nil {
		*stack = append(*stack, existing)

		return true
	}

	return false
}

// autoCloseOpen pops every open element that the start tag closes, and ends
// the row when a new <td>/<th> follows an open cell.
func autoCloseOpen(stack *[]*Node, name string) {
	closedCell := false

	for len(*stack) > 1 {
		openName := (*stack)[len(*stack)-1].Name
		if !shouldAutoClose(openName, name) {
			break
		}

		if openName == "td" || openName == "th" {
			closedCell = true
		}

		*stack = (*stack)[:len(*stack)-1]
	}
	// close-a-cell: a new <td>/<th> after an open cell ends the row too
	if closedCell && (name == "td" || name == "th") && len(*stack) > 1 {
		if (*stack)[len(*stack)-1].Name == "tr" {
			*stack = (*stack)[:len(*stack)-1]
		}
	}
}

// applyAttributes stores the interleaved name/value pairs on node, keeping
// the first value of a duplicated attribute. Names are already lowercased
// by the tokenizer.
func applyAttributes(node *Node, attrs []string) {
	for i := 0; i+1 < len(attrs); i += 2 {
		val := UnescapeEntities(attrs[i+1])

		if _, dup := node.Attrs[attrs[i]]; !dup {
			node.Attrs[attrs[i]] = val
		}
	}
}

// closeElement pops the open-element stack back to (and including) the
// first element with name; a stray end tag is a no-op. Token data is already
// lowercased by the tokenizer.
func closeElement(stack *[]*Node, data string) {
	name := data

	for i := len(*stack) - 1; i > 0; i-- {
		if (*stack)[i].Name == name {
			*stack = (*stack)[:i]

			break
		}
	}
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

// isVoidElement reports whether name never takes content.
func isVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	}

	return false
}

// isRawTextElement reports whether name consumes everything up to its
// closing tag (script/style/textarea/title contents are raw text).
func isRawTextElement(name string) bool {
	switch name {
	case "script", "style", "textarea", "title":
		return true
	}

	return false
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

// autoClose lists, per start tag, the open elements that the tag closes.
var autoClose = map[string][]string{ //nolint:gochecknoglobals // immutable auto-close vocabulary
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
	toks := make([]token, 0, strings.Count(src, "<")+1)

	pos := 0
	srcLen := len(src)

	for pos < srcLen {
		if src[pos] != '<' {
			span := strings.IndexByte(src[pos:], '<')
			if span < 0 {
				span = srcLen - pos
			}

			toks = append(toks, token{kind: tokText, data: src[pos : pos+span]}) //nolint:exhaustruct
			pos += span

			continue
		}

		if pos+1 >= srcLen {
			toks = append(toks, token{kind: tokText, data: "<"}) //nolint:exhaustruct

			break
		}

		var next int

		var err error

		switch {
		case src[pos+1] == '!':
			next, err = scanBang(src, pos, &toks)
		case src[pos+1] == '/':
			next, err = scanEndTag(src, pos, &toks)
		case src[pos+1] == '?':
			next, err = scanPI(src, pos)
		default:
			next, err = scanStartTag(src, pos, &toks)
		}

		if err != nil {
			return nil, err
		}

		pos = next
	}

	return toks, nil
}

// scanBang tokenizes a '!' construct at pos: comment, doctype, or a bogus
// declaration that is skipped.
func scanBang(src string, pos int, toks *[]token) (int, error) {
	if strings.HasPrefix(src[pos:], "<!--") {
		end := strings.Index(src[pos+commentPrefixLen:], "-->")
		if end < 0 {
			return 0, errUnterminatedComment
		}

		data := src[pos+commentPrefixLen : pos+commentPrefixLen+end]
		next := pos + commentPrefixLen + end + commentSuffixLen

		*toks = append(*toks, token{kind: tokComment, data: data}) //nolint:exhaustruct

		return next, nil
	}

	if len(src)-pos >= 9 && strings.EqualFold(src[pos:pos+9], "<!doctype") {
		end := strings.IndexByte(src[pos:], '>')
		if end < 0 {
			return 0, errUnterminatedDoctype
		}

		*toks = append(*toks, token{kind: tokDoctype, data: src[pos+2 : pos+end]}) //nolint:exhaustruct

		return pos + end + 1, nil
	}

	// other bogus declaration → skip to >
	end := strings.IndexByte(src[pos:], '>')
	if end < 0 {
		return 0, errUnterminatedDecl
	}

	return pos + end + 1, nil
}

// scanEndTag tokenizes a closing tag at pos.
func scanEndTag(src string, pos int, toks *[]token) (int, error) {
	end := strings.IndexByte(src[pos:], '>')
	if end < 0 {
		return 0, errUnterminatedEndTag
	}

	name := strings.ToLower(strings.TrimSpace(src[pos+2 : pos+end]))
	*toks = append(*toks, token{kind: tokEnd, data: name}) //nolint:exhaustruct

	return pos + end + 1, nil
}

// scanPI skips a processing instruction at pos.
func scanPI(src string, pos int) (int, error) {
	end := strings.Index(src[pos:], "?>")
	if end < 0 {
		return 0, errUnterminatedPI
	}

	return pos + end + piCloseLen, nil
}

// scanStartTag tokenizes a start tag at pos, including the raw-text content
// of script/style/textarea/title elements up to their closing tag.
func scanStartTag(src string, pos int, toks *[]token) (int, error) {
	if !isASCIILetter(src[pos+1]) {
		// bare '<' followed by no valid tag start becomes text
		*toks = append(*toks, token{kind: tokText, data: "<"}) //nolint:exhaustruct

		return pos + 1, nil
	}

	end, err := tagEnd(src, pos)
	if err != nil {
		return 0, err
	}

	if end < 0 {
		// no closing '>' - treat the rest as text
		*toks = append(*toks, token{kind: tokText, data: src[pos:]}) //nolint:exhaustruct

		return len(src), nil
	}

	tag := src[pos+1 : end]

	name, attrs, selfClose, err := parseTag(tag)
	if err != nil {
		return 0, err
	}

	if name == "" {
		return end + 1, nil
	}

	name = strings.ToLower(name)
	*toks = append(*toks, token{kind: tokStart, data: name, attrs: attrs, selfClosing: selfClose})
	next := end + 1
	// raw-text elements capture everything until their closing tag
	if isRawTextElement(name) && !selfClose {
		rawStart, rawEnd, ok := rawTextEnd(src, next, name)
		if !ok {
			*toks = append(*toks, token{kind: tokText, data: src[next:]}) //nolint:exhaustruct

			return len(src), nil
		}

		if rawStart > next {
			*toks = append(*toks, token{kind: tokText, data: src[next:rawStart]}) //nolint:exhaustruct
		}

		*toks = append(*toks, token{kind: tokEnd, data: name}) //nolint:exhaustruct
		next = rawEnd + 1
	}

	return next, nil
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
				return 0, errUnterminatedAttrVal
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
func rawTextEnd(src string, from int, name string) (int, int, bool) {
	needle := "</" + name

	low := strings.ToLower(src[from:])
	offset := from

	for {
		found := strings.Index(low, needle)
		if found < 0 {
			return 0, 0, false
		}

		after := found + len(needle)
		for after < len(low) && isWhitespace(low[after]) {
			after++
		}

		if after < len(low) && low[after] == '>' {
			return offset + found, offset + after, true
		}

		low = low[found+rawCloseMinSkip:]
		offset += found + rawCloseMinSkip
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

	const attrPairSize = 2 // attrs slice interleaves name and value

	attrs := make([]string, 0, attrPairSize*strings.Count(body, "="))

	rest := strings.TrimSpace(body[nameEnd:])
	rest = strings.TrimSuffix(rest, "/")

	for rest != "" {
		key, val, after, err := nextAttr(rest)
		if err != nil {
			return "", nil, false, err
		}

		if key == "" && val == "" && after == "" {
			break
		}

		attrs = append(attrs, strings.ToLower(key), val)
		rest = after
	}

	return name, attrs, selfClose, nil
}

// nextAttr extracts one attribute (name, value) from the front of rest. The
// returned after is the remaining body; an all-empty result means there are
// no more attributes.
func nextAttr(rest string) (string, string, string, error) {
	rest = strings.TrimLeft(rest, " \t\n\r")
	if rest == "" {
		return "", "", "", nil
	}
	// attribute name: up to '=' or whitespace
	idx := 0
	for idx < len(rest) && rest[idx] != '=' && !isWhitespace(rest[idx]) {
		idx++
	}

	key := rest[:idx]
	rest = strings.TrimLeft(rest[idx:], " \t\n\r")

	if !strings.HasPrefix(rest, "=") {
		return key, "", rest, nil
	}

	rest = strings.TrimLeft(rest[1:], " \t\n\r")
	if rest == "" {
		return "", "", "", nil
	}

	val, after, err := attrTail(rest)
	if err != nil {
		return "", "", "", err
	}

	return key, val, after, nil
}

// attrTail extracts the value at the front of rest, which starts at the
// value position: a quoted value up to its closing quote, otherwise up to
// the next whitespace.
func attrTail(rest string) (string, string, error) {
	switch rest[0] {
	case '"', '\'':
		q := rest[0]

		end := strings.IndexByte(rest[1:], q)
		if end < 0 {
			return "", "", errUnterminatedAttrVal
		}

		return rest[1 : end+1], rest[end+2:], nil
	default:
		end := strings.IndexAny(rest, " \t\n\r")
		if end < 0 {
			return rest, "", nil
		}

		return rest[:end], rest[end:], nil
	}
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
