// Package css implements the CSS subset gowkhtmltopdf accepts: a
// declarations-and-rules parser, selector matching against the html tree,
// specificity ordering, and value helpers (lengths, colors, font families).
//
// Scope: `*`, type, `.class`, `#id`, attribute selectors (`[attr]`, `=`, `~=`,
// `*=`, `^=`, `$=`, `|=`),
// :first-child/:last-child/:nth-child/:has()/:not(), descendant/child/sibling
// combinators, `@media` type + size-feature matching (see MediaMatches),
// `@container` size queries (inline-size/width + and/or/not), `!important`,
// inline style attributes. Unsupported constructs degrade without panicking.
package css

import (
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// Stylesheet is a parsed stylesheet. Rules keep their source order.
type Stylesheet struct {
	Rules     []Rule
	FontFaces []FontFace
}

// FontFace is one @font-face rule (local src subset).
type FontFace struct {
	Family string
	Src    string // raw src value (may contain url(...) or local(...))
	Weight string
	Style  string
}

// Rule is one rule set: selectors plus a declaration block.
type Rule struct {
	Selectors []Selector
	Decls     []Declaration
	Media     string // raw @media prelude ("all", "print", "screen and (…)", …)
	Order     int    // source order within the sheet; callers rebase across sheets
	// Container is non-nil for rules nested under @container. The rule applies
	// only when the query matches the nearest eligible ancestor container.
	Container *ContainerQuery
}

// Selector is a chain of compound parts linked by combinators.
type Selector struct {
	Parts []SelectorPart
}

// SelectorPart is one compound selector. Combinator describes how it links to
// the following part: "" for the first part, ">" child, "+" next-sibling,
// "~" subsequent-sibling, " " descendant.
type SelectorPart struct {
	Tag        string
	Classes    []string
	ID         string
	Attrs      []AttrSelector
	Pseudos    []PseudoClass
	Combinator string
}

// AttrSelector is [name], [name=value] (exact), [name~=word] (space-separated
// word), [name*=substr] (substring), [name^=prefix], [name$=suffix], or
// [name|=ident] (exact or prefix-plus-hyphen).
type AttrSelector struct {
	Name  string
	Op    string // "", "=", "~=", "*=", "^=", "$=", "|="
	Value string
}

// RelativeSelector is a complex selector interpreted relative to a subject
// element (Selectors 4). Leading is " " (descendant), ">", "+", or "~".
type RelativeSelector struct {
	Leading string
	Parts   []SelectorPart
}

// PseudoClass is :first-child, :last-child, :nth-child(...), :has(...), or
// :not(...).
type PseudoClass struct {
	Name string // lower-case, without leading ':'
	Arg  string // nth-child argument, lower-case, trimmed
	Has  []RelativeSelector
	Not  []Selector
	// Where is the argument to :where() / :is() (selector list). Matching
	// uses OR semantics; :where contributes 0 specificity, :is uses the
	// most specific argument (Selectors 4).
	Where   []Selector
	WhereIs bool // true when parsed from :is() (specificity from args)
}

// Declaration is one property: value pair.
type Declaration struct {
	Prop      string
	Value     string
	Important bool
}

// Parse parses a stylesheet. Broken input never returns an error for recoverable
// garbage; only unbalanced blocks do. @media preambles are stored raw on
// Rule.Media and evaluated later via MediaMatches.
func Parse(src string) (*Stylesheet, error) {
	s := &Stylesheet{}
	src = stripComments(src)
	order := 0
	for src != "" {
		src = strings.TrimLeft(src, " \t\r\n")
		if src == "" {
			break
		}
		if strings.HasPrefix(src, "@") {
			low := strings.ToLower(src)
			switch {
			case strings.HasPrefix(low, "@media"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					return nil, errUnbalanced
				}
				media := strings.TrimSpace(src[len("@media"):open])
				if media == "" {
					media = "all"
				}
				block, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}
				src = rest
				rules, err := parseRuleList(media, nil, block, &order)
				if err != nil {
					return nil, err
				}
				s.Rules = append(s.Rules, rules...)
			case strings.HasPrefix(low, "@container"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					return nil, errUnbalanced
				}
				prelude := strings.TrimSpace(src[len("@container"):open])
				cq, ok := parseContainerPrelude(prelude)
				block, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}
				src = rest
				if !ok {
					continue
				}
				rules, err := parseRuleList("all", &cq, block, &order)
				if err != nil {
					return nil, err
				}
				s.Rules = append(s.Rules, rules...)
			case strings.HasPrefix(low, "@page"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					src = ""
					continue
				}
				_, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}
				src = rest
			case strings.HasPrefix(low, "@keyframes"), strings.HasPrefix(low, "@-webkit-keyframes"):
				// Animations are parse-ignored (static cascaded values only).
				open := strings.IndexByte(src, '{')
				if open < 0 {
					src = ""
					continue
				}
				_, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}
				src = rest
			case strings.HasPrefix(low, "@font-face"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					src = ""
					continue
				}
				block, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}
				src = rest
				if ff := parseFontFace(block); ff.Family != "" || ff.Src != "" {
					s.FontFaces = append(s.FontFaces, ff)
				}
			default:
				if end := strings.IndexByte(src, ';'); end >= 0 {
					src = src[end+1:]
				} else {
					src = ""
				}
			}
			continue
		}
		// one top-level rule
		selEnd, err := findBlock(src)
		if err == errNoBlock {
			// garbage prelude without a block: discard up to ';'
			if end := strings.IndexByte(src, ';'); end >= 0 {
				src = src[end+1:]
			} else {
				src = ""
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		selText := strings.TrimSpace(src[:selEnd])
		block, rest, err := takeBlock(src, selEnd)
		if err != nil {
			return nil, err
		}
		src = rest
		if selText == "" {
			continue
		}
		sel, ok := parseSelectorList(selText)
		if !ok || len(sel) == 0 {
			continue
		}
		s.Rules = append(s.Rules, Rule{
			Selectors: sel,
			Decls:     parseDeclarations(block),
			Media:     "all",
			Order:     order,
		})
		order++
	}
	return s, nil
}

func parseFontFace(block string) FontFace {
	var ff FontFace
	for _, d := range parseDeclarations(block) {
		switch strings.ToLower(d.Prop) {
		case "font-family":
			fams := ParseFontFamily(d.Value)
			if len(fams) > 0 {
				ff.Family = fams[0]
			} else {
				ff.Family = strings.Trim(d.Value, " \"'")
			}
		case "src":
			ff.Src = d.Value
		case "font-weight":
			ff.Weight = d.Value
		case "font-style":
			ff.Style = d.Value
		}
	}
	return ff
}

// FontFaceURLs extracts url(...) references from an @font-face src value.
func FontFaceURLs(src string) []string {
	var out []string
	low := src
	for {
		i := strings.Index(strings.ToLower(low), "url(")
		if i < 0 {
			break
		}
		rest := low[i+4:]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			break
		}
		raw := strings.TrimSpace(rest[:end])
		raw = strings.Trim(raw, `"'`)
		if raw != "" {
			out = append(out, raw)
		}
		low = rest[end+1:]
	}
	return out
}

// parseRuleList parses the rules inside a @media or @container block body.
// When cq is non-nil, every produced rule inherits that container query.
func parseRuleList(media string, cq *ContainerQuery, block string, orderPtr *int) ([]Rule, error) {
	var rules []Rule
	for block != "" {
		block = strings.TrimLeft(block, " \t\r\n")
		if block == "" {
			break
		}
		// Nested @container inside @media (or another @container): flatten.
		if strings.HasPrefix(block, "@") {
			low := strings.ToLower(block)
			if strings.HasPrefix(low, "@container") {
				open := strings.IndexByte(block, '{')
				if open < 0 {
					return nil, errUnbalanced
				}
				prelude := strings.TrimSpace(block[len("@container"):open])
				innerCQ, ok := parseContainerPrelude(prelude)
				innerBlock, rem, err := takeBlock(block, open)
				if err != nil {
					return nil, err
				}
				block = rem
				if !ok {
					continue
				}
				// Nested @container replaces (does not combine) the query.
				use := &innerCQ
				nested, err := parseRuleList(media, use, innerBlock, orderPtr)
				if err != nil {
					return nil, err
				}
				rules = append(rules, nested...)
				continue
			}
			// Other at-rules inside: skip their block or statement.
			if open := strings.IndexByte(block, '{'); open >= 0 {
				_, rem, err := takeBlock(block, open)
				if err != nil {
					return nil, err
				}
				block = rem
			} else if end := strings.IndexByte(block, ';'); end >= 0 {
				block = block[end+1:]
			} else {
				block = ""
			}
			continue
		}
		selEnd, err := findBlock(block)
		if err == errNoBlock {
			// garbage prelude inside a media block: discard up to ';'
			if end := strings.IndexByte(block, ';'); end >= 0 {
				block = block[end+1:]
			} else {
				block = ""
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		selText := strings.TrimSpace(block[:selEnd])
		declBlock, rem, err := takeBlock(block, selEnd)
		if err != nil {
			return nil, err
		}
		block = rem
		sel, ok := parseSelectorList(selText)
		if !ok || len(sel) == 0 {
			continue
		}
		r := Rule{
			Selectors: sel,
			Decls:     parseDeclarations(declBlock),
			Media:     media,
			Order:     *orderPtr,
		}
		if cq != nil {
			cp := *cq
			r.Container = &cp
		}
		rules = append(rules, r)
		*orderPtr++
	}
	return rules, nil
}

var errUnbalanced = &parseError{"unbalanced braces in stylesheet"}
var errNoBlock = &parseError{"missing '{' before ';'"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return "css: " + e.msg }

// stripComments removes /* ... */ comments, preserving newlines so line
// numbers stay roughly stable.
func stripComments(src string) string {
	var b strings.Builder
	for {
		i := strings.Index(src, "/*")
		if i < 0 {
			b.WriteString(src)
			return b.String()
		}
		b.WriteString(src[:i])
		rest := src[i+2:]
		j := strings.Index(rest, "*/")
		if j < 0 {
			return b.String()
		}
		for k := i; k <= i+1+j; k++ {
			if src[k] == '\n' {
				b.WriteByte('\n')
			}
		}
		src = rest[j+2:]
	}
}

// findBlock returns the index of the '{' ending the selector list, tracking
// quotes and parens so braces inside them are ignored.
func findBlock(src string) (int, error) {
	depth := 0
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '"', '\'':
			q := src[i]
			j := strings.IndexByte(src[i+1:], q)
			if j < 0 {
				return -1, &parseError{"unterminated string in stylesheet"}
			}
			i += j + 1
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '{':
			if depth == 0 {
				return i, nil
			}
		case ';':
			if depth == 0 {
				return -1, errNoBlock // garbage prelude: discard up to ';'
			}
		}
	}
	return -1, &parseError{"missing '{' in stylesheet"}
}

// takeBlock consumes the braced block whose '{' sits at open and returns the
// block body and the remaining source.
func takeBlock(src string, open int) (string, string, error) {
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open+1 : i], src[i+1:], nil
			}
		case '"', '\'':
			q := src[i]
			j := strings.IndexByte(src[i+1:], q)
			if j < 0 {
				return "", "", &parseError{"unterminated string in stylesheet"}
			}
			i += j + 1
		}
	}
	return "", "", &parseError{"unbalanced braces in stylesheet"}
}

// parseSelectorList splits a selector list on top-level commas.
func parseSelectorList(s string) ([]Selector, bool) {
	var out []Selector
	for _, part := range splitTopLevel(s, ',') {
		sel, ok := parseSelector(part)
		if !ok {
			continue
		}
		out = append(out, sel)
	}
	return out, len(out) > 0
}

// splitTopLevel splits on sep outside parens, brackets and quotes.
func splitTopLevel(s string, sep byte) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			q := s[i]
			j := strings.IndexByte(s[i+1:], q)
			if j < 0 {
				i = len(s)
			} else {
				i += j + 1
			}
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

// parseSelector parses one compound chain, e.g. "div.a#b > p.c" or
// "tr:nth-child(even)".
func parseSelector(s string) (Selector, bool) {
	var sel Selector
	s = strings.TrimSpace(s)
	if s == "" {
		return sel, false
	}
	parts := splitSelectorChain(s)
	for i, ch := range parts {
		if ch == ">" || ch == "+" || ch == "~" || ch == " " {
			continue // combinator marker, applied to the next compound
		}
		part, ok := parseCompound(ch)
		if !ok {
			return sel, false
		}
		if len(sel.Parts) > 0 {
			switch parts[i-1] {
			case ">", "+", "~":
				part.Combinator = parts[i-1]
			default:
				part.Combinator = " "
			}
		}
		sel.Parts = append(sel.Parts, part)
	}
	return sel, len(sel.Parts) > 0
}

// splitSelectorChain splits a selector into compounds and combinators, e.g.
// ["div.a", ">", "p", " ", "span"].
func splitSelectorChain(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			// skip whitespace; it becomes a descendant combinator only when
			// it sits between two compounds
			flush()
			for i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\n' || s[i+1] == '\r') {
				i++
			}
			if len(out) > 0 && out[len(out)-1] != " " && out[len(out)-1] != ">" &&
				out[len(out)-1] != "+" && out[len(out)-1] != "~" && i+1 < len(s) && s[i+1] != '>' {
				out = append(out, " ")
			}
		case c == '>' || c == '+' || c == '~':
			flush()
			out = append(out, string(c))
		case c == '[':
			// keep [attr] / [attr=value] inside the compound
			j := strings.IndexByte(s[i:], ']')
			if j < 0 {
				cur.WriteByte(c)
			} else {
				if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
					out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
					cur.WriteByte('*')
				}
				cur.WriteString(s[i : i+j+1])
				i += j
			}
		case c == ':':
			// keep :pseudo / :nth-child(n) / :has(...) inside the compound;
			// drop ::pseudo-elements
			if i+1 < len(s) && s[i+1] == ':' {
				// ::before etc. - skip
				i += 2
				for i < len(s) && !isSelBreak(s[i]) {
					if s[i] == '(' {
						_, end, ok := takeParenArg(s, i)
						if !ok {
							i = len(s)
							break
						}
						i = end
						break
					}
					i++
				}
				i--
				break
			}
			if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
				out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
				cur.WriteByte('*')
			}
			start := i
			i++
			for i < len(s) && !isSelBreak(s[i]) {
				if s[i] == '(' {
					_, end, ok := takeParenArg(s, i)
					if !ok {
						i = len(s)
						break
					}
					i = end
					break
				}
				i++
			}
			cur.WriteString(s[start:i])
			i--
		case c == '\\':
			// escape: keep next char literally
			if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
			}
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

func isSelBreak(b byte) bool {
	return b == '.' || b == '#' || b == '[' || b == ':' || b == '>' || b == '+' ||
		b == '~' || b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// parseCompound parses "tag#id.class1.class2[attr]:nth-child(even)" into a
// SelectorPart. A tag of "*" or "" means universal.
func parseCompound(s string) (SelectorPart, bool) {
	return parseCompoundCtx(s, false)
}

// parseCompoundCtx parses a compound. When insideHas is true, nested :has()
// and pseudo-elements are rejected as invalid.
func parseCompoundCtx(s string, insideHas bool) (SelectorPart, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return SelectorPart{Tag: "*"}, true
	}
	var part SelectorPart
	i := 0
	// tag name first
	for i < len(s) && !isCompoundBreak(s[i]) {
		i++
	}
	part.Tag = s[:i]
	if part.Tag == "" {
		part.Tag = "*"
	} else if part.Tag != "*" && !validIdent(part.Tag) {
		return SelectorPart{}, false
	}
	for i < len(s) {
		switch s[i] {
		case '#':
			j := i + 1
			for j < len(s) && !isCompoundBreak(s[j]) {
				j++
			}
			id := s[i+1 : j]
			if !validIdent(id) {
				return SelectorPart{}, false
			}
			part.ID = id
			i = j
		case '.':
			j := i + 1
			for j < len(s) && !isCompoundBreak(s[j]) {
				j++
			}
			if j > i+1 {
				cls := s[i+1 : j]
				if !validIdent(cls) {
					return SelectorPart{}, false
				}
				part.Classes = append(part.Classes, cls)
			}
			i = j
		case '[':
			j := strings.IndexByte(s[i:], ']')
			if j < 0 {
				return SelectorPart{}, false
			}
			attr, ok := parseAttrSelector(s[i : i+j+1])
			if !ok {
				return SelectorPart{}, false
			}
			part.Attrs = append(part.Attrs, attr)
			i += j + 1
		case ':':
			if i+1 < len(s) && s[i+1] == ':' {
				return SelectorPart{}, false // pseudo-elements not supported
			}
			j := i + 1
			for j < len(s) && s[j] != '(' && !isCompoundBreak(s[j]) {
				j++
			}
			name := strings.ToLower(s[i+1 : j])
			arg := ""
			var argRaw string
			hasParen := j < len(s) && s[j] == '('
			if hasParen {
				raw, end, ok := takeParenArg(s, j)
				if !ok {
					return SelectorPart{}, false
				}
				argRaw = raw
				arg = strings.ToLower(strings.TrimSpace(raw))
				j = end
			}
			if insideHas {
				switch name {
				case "has", "before", "after", "first-line", "first-letter":
					return SelectorPart{}, false
				}
			}
			switch name {
			case "first-child", "last-child", "nth-child":
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name, Arg: arg})
			case "has":
				if !hasParen || strings.TrimSpace(argRaw) == "" {
					return SelectorPart{}, false
				}
				lowArg := strings.ToLower(argRaw)
				if strings.Contains(lowArg, ":has(") || strings.Contains(argRaw, "::") {
					return SelectorPart{}, false
				}
				rels, ok := parseRelativeSelectorList(argRaw)
				if !ok {
					return SelectorPart{}, false
				}
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: "has", Has: rels})
			case "not":
				if !hasParen || strings.TrimSpace(argRaw) == "" {
					return SelectorPart{}, false
				}
				sels, ok := parseSelectorListStrict(argRaw, insideHas)
				if !ok {
					return SelectorPart{}, false
				}
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: "not", Not: sels})
			case "where", "is":
				if !hasParen || strings.TrimSpace(argRaw) == "" {
					return SelectorPart{}, false
				}
				sels, ok := parseSelectorListStrict(argRaw, insideHas)
				if !ok {
					return SelectorPart{}, false
				}
				part.Pseudos = append(part.Pseudos, PseudoClass{
					Name: "where", Where: sels, WhereIs: name == "is",
				})
			case "link", "visited":
				// Print semantics: both mean "a[href]" (no browsing history).
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name})
			case "hover", "active", "focus":
				// Accepted for parse/cascade structure but never match in print
				// (static PDF has no pointer/focus state). Keeping them on the
				// compound prevents a:hover from degrading to bare `a`.
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name})
			default:
				// unknown: ignore
			}
			i = j
		default:
			return SelectorPart{}, false
		}
	}
	return part, true
}

func parseAttrSelector(s string) (AttrSelector, bool) {
	// s includes brackets: [href], [href="x"], [typeof~='mw:File/Thumb'], [class*="noprint"]
	if len(s) < 3 || s[0] != '[' || s[len(s)-1] != ']' {
		return AttrSelector{}, false
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return AttrSelector{}, false
	}
	// Operator forms: ~= *= ^= $= |= =  (check multi-char before bare =)
	op := ""
	nameEnd := -1
	for _, cand := range []string{"~=", "*=", "^=", "$=", "|="} {
		if i := strings.Index(inner, cand); i > 0 {
			nameEnd = i
			op = cand
			break
		}
	}
	if nameEnd < 0 {
		if i := strings.IndexByte(inner, '='); i >= 0 {
			nameEnd = i
			op = "="
		}
	}
	if nameEnd < 0 {
		if !validIdent(inner) {
			return AttrSelector{}, false
		}
		return AttrSelector{Name: strings.ToLower(inner)}, true
	}
	name := strings.TrimSpace(inner[:nameEnd])
	val := strings.TrimSpace(inner[nameEnd+len(op):])
	if !validIdent(name) {
		return AttrSelector{}, false
	}
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	switch op {
	case "=", "~=", "*=", "^=", "$=", "|=":
		return AttrSelector{Name: strings.ToLower(name), Op: op, Value: val}, true
	default:
		return AttrSelector{}, false
	}
}

// validIdent reports whether s is a valid CSS identifier (letters, digits,
// '-', '_', and digits after the first character).
func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if i > 0 && (c >= '0' && c <= '9') {
			ok = true
		}
		if !ok {
			return false
		}
	}
	return true
}

func isCompoundBreak(b byte) bool {
	return b == '#' || b == '.' || b == '[' || b == ':'
}

// Match reports whether the selector matches the element node. Matching runs
// right to left: the last part must match n, earlier parts must match
// ancestors/siblings per their combinators.
func Match(s Selector, n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode || len(s.Parts) == 0 {
		return false
	}
	if !matchPart(s.Parts[len(s.Parts)-1], n) {
		return false
	}
	cur := n
	// Combinator is stored on the right-hand part of each pair (how that
	// part attaches to the previous). Walk left using Parts[i+1].Combinator.
	for i := len(s.Parts) - 2; i >= 0; i-- {
		part := s.Parts[i]
		switch s.Parts[i+1].Combinator {
		case ">":
			cur = cur.Parent
			if cur == nil || cur.Type != html.ElementNode || !matchPart(part, cur) {
				return false
			}
		case "+":
			prev := previousElementSibling(cur)
			if prev == nil || !matchPart(part, prev) {
				return false
			}
			cur = prev
		case "~":
			found := false
			for sib := previousElementSibling(cur); sib != nil; sib = previousElementSibling(sib) {
				if matchPart(part, sib) {
					cur = sib
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default: // descendant
			cur = cur.Parent
			for cur != nil && (cur.Type != html.ElementNode || !matchPart(part, cur)) {
				cur = cur.Parent
			}
			if cur == nil {
				return false
			}
		}
	}
	return true
}

// matchPart matches one compound against an element.
func matchPart(p SelectorPart, n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if p.Tag != "*" && !strings.EqualFold(p.Tag, n.Name) {
		return false
	}
	if p.ID != "" && n.Attribute("id") != p.ID {
		return false
	}
	if len(p.Classes) > 0 {
		classes := classSet(n)
		for _, c := range p.Classes {
			if !classes[c] {
				return false
			}
		}
	}
	for _, a := range p.Attrs {
		val, ok := "", false
		if n.Attrs != nil {
			val, ok = n.Attrs[a.Name]
		}
		if a.Op == "" {
			if !ok {
				return false
			}
			continue
		}
		if !ok {
			return false
		}
		switch a.Op {
		case "=":
			if val != a.Value {
				return false
			}
		case "~=":
			if a.Value == "" || strings.Contains(a.Value, " ") {
				return false
			}
			found := false
			for _, w := range strings.Fields(val) {
				if w == a.Value {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case "*=":
			if a.Value == "" || !strings.Contains(val, a.Value) {
				return false
			}
		case "^=":
			if a.Value == "" || !strings.HasPrefix(val, a.Value) {
				return false
			}
		case "$=":
			if a.Value == "" || !strings.HasSuffix(val, a.Value) {
				return false
			}
		case "|=":
			// Exact match or value is followed by a hyphen (HTML lang / BCP47-style).
			if a.Value == "" {
				return false
			}
			if val != a.Value && !strings.HasPrefix(val, a.Value+"-") {
				return false
			}
		default:
			return false
		}
	}
	for _, ps := range p.Pseudos {
		if !matchPseudo(ps, n) {
			return false
		}
	}
	return true
}

func matchPseudo(ps PseudoClass, n *html.Node) bool {
	switch ps.Name {
	case "first-child":
		return previousElementSibling(n) == nil
	case "last-child":
		return nextElementSibling(n) == nil
	case "nth-child":
		idx := elementIndex(n)
		return matchNth(ps.Arg, idx)
	case "has":
		for _, rs := range ps.Has {
			if matchRelative(rs, n) {
				return true
			}
		}
		return false
	case "not":
		for _, sel := range ps.Not {
			if Match(sel, n) {
				return false
			}
		}
		return true
	case "where":
		// :where() / :is() — match if any argument selector matches.
		for _, sel := range ps.Where {
			if Match(sel, n) {
				return true
			}
		}
		return false
	case "link", "visited":
		// Print: no link history — both match any anchor with an href.
		return isLinkAnchor(n)
	case "hover", "active", "focus":
		return false
	default:
		return true
	}
}

// isLinkAnchor reports whether n is an <a> element with a non-empty href
// (any scheme, including "#" fragments and relative paths).
func isLinkAnchor(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode || !strings.EqualFold(n.Name, "a") {
		return false
	}
	href := strings.TrimSpace(n.Attribute("href"))
	return href != ""
}

func previousElementSibling(n *html.Node) *html.Node {
	if n == nil || n.Parent == nil {
		return nil
	}
	var prev *html.Node
	for _, c := range n.Parent.Children {
		if c == n {
			return prev
		}
		if c.Type == html.ElementNode {
			prev = c
		}
	}
	return nil
}

func nextElementSibling(n *html.Node) *html.Node {
	if n == nil || n.Parent == nil {
		return nil
	}
	seen := false
	for _, c := range n.Parent.Children {
		if c == n {
			seen = true
			continue
		}
		if seen && c.Type == html.ElementNode {
			return c
		}
	}
	return nil
}

// elementIndex is 1-based among element siblings.
func elementIndex(n *html.Node) int {
	if n == nil || n.Parent == nil {
		return 1
	}
	i := 0
	for _, c := range n.Parent.Children {
		if c.Type != html.ElementNode {
			continue
		}
		i++
		if c == n {
			return i
		}
	}
	return 0
}

// matchNth implements :nth-child(an+b) / odd / even for 1-based index.
func matchNth(arg string, index int) bool {
	arg = strings.TrimSpace(strings.ToLower(arg))
	if arg == "" {
		return false
	}
	if arg == "odd" {
		return index%2 == 1
	}
	if arg == "even" {
		return index%2 == 0
	}
	// plain integer
	if n, err := strconv.Atoi(arg); err == nil {
		return index == n
	}
	// an+b / n+b / -n+b / an
	a, b := 0, 0
	if strings.Contains(arg, "n") {
		parts := strings.SplitN(arg, "n", 2)
		as := strings.TrimSpace(parts[0])
		bs := ""
		if len(parts) == 2 {
			bs = strings.TrimSpace(parts[1])
		}
		switch as {
		case "", "+":
			a = 1
		case "-":
			a = -1
		default:
			var err error
			a, err = strconv.Atoi(as)
			if err != nil {
				return false
			}
		}
		if bs == "" {
			b = 0
		} else {
			var err error
			b, err = strconv.Atoi(bs)
			if err != nil {
				return false
			}
		}
		if a == 0 {
			return index == b
		}
		// index = a*k + b for integer k >= 0
		if (index-b)%a != 0 {
			return false
		}
		k := (index - b) / a
		return k >= 0
	}
	return false
}

func classSet(n *html.Node) map[string]bool {
	set := map[string]bool{}
	for _, c := range strings.Fields(n.Attribute("class")) {
		set[c] = true
	}
	return set
}

// Specificity returns (a, b, c): ID count, class/attribute/pseudo count, type count.
// :has() / :not() contribute the specificity of their most specific argument
// (Selectors 4), not a flat class-level count for the pseudo itself.
func Specificity(s Selector) (a, b, c int) {
	for _, p := range s.Parts {
		if p.ID != "" {
			a++
		}
		b += len(p.Classes) + len(p.Attrs)
		if p.Tag != "*" {
			c++
		}
		for _, ps := range p.Pseudos {
			switch ps.Name {
			case "has":
				a2, b2, c2 := maxRelativeSpecificity(ps.Has)
				a += a2
				b += b2
				c += c2
			case "not":
				a2, b2, c2 := maxSelectorSpecificity(ps.Not)
				a += a2
				b += b2
				c += c2
			case "where":
				if ps.WhereIs {
					// :is() — most specific argument
					a2, b2, c2 := maxSelectorSpecificity(ps.Where)
					a += a2
					b += b2
					c += c2
				}
				// :where() — zero specificity
			default:
				b++
			}
		}
	}
	return a, b, c
}

// CompareSpecificity returns -1/0/1: lower specificity first; ties break on
// the given orders.
func CompareSpecificity(a, b Selector, orderA, orderB int) int {
	ia, ib, ic := Specificity(a)
	ja, jb, jc := Specificity(b)
	for _, t := range [][2]int{{ia, ja}, {ib, jb}, {ic, jc}} {
		if t[0] != t[1] {
			if t[0] < t[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case orderA < orderB:
		return -1
	case orderA > orderB:
		return 1
	default:
		return 0
	}
}

// ParseInline parses a style="" attribute value into declarations.
func ParseInline(style string) []Declaration {
	return parseDeclarations(style)
}

// parseDeclarations splits a declaration block on top-level ';' and parses
// each "prop: value" pair. Garbage pairs are skipped.
func parseDeclarations(block string) []Declaration {
	var decls []Declaration
	for _, part := range splitTopLevel(block, ';') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colon := strings.IndexByte(part, ':')
		if colon < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(part[:colon]))
		value := strings.TrimSpace(part[colon+1:])
		if prop == "" || value == "" {
			continue
		}
		important := IsImportant(value)
		if important {
			value = stripImportant(value)
		}
		if !validPropName(prop) {
			continue
		}
		decls = append(decls, Declaration{Prop: prop, Value: value, Important: important})
	}
	return decls
}

func validPropName(p string) bool {
	if p == "" {
		return false
	}
	for i := 0; i < len(p); i++ {
		c := p[i]
		if !(c == '-' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// IsImportant reports whether a declaration value carries !important
// (whitespace between ! and important is allowed).
func IsImportant(v string) bool {
	v = strings.TrimSpace(v)
	const word = "important"
	if len(v) < len(word)+1 {
		return false
	}
	if !strings.EqualFold(v[len(v)-len(word):], word) {
		return false
	}
	rest := strings.TrimRight(v[:len(v)-len(word)], " \t")
	return strings.HasSuffix(rest, "!")
}

// stripImportant removes a trailing !important (any case, optional space)
// from a declaration value.
func stripImportant(v string) string {
	if !IsImportant(v) {
		return v
	}
	v = strings.TrimRight(v, " \t")
	v = v[:len(v)-len("important")]
	v = strings.TrimRight(v, " \t")
	v = strings.TrimSuffix(v, "!")
	return strings.TrimSpace(v)
}

// IsInherited reports whether the property inherits from its parent.
func IsInherited(prop string) bool {
	switch prop {
	case "color", "font", "font-family", "font-size", "font-style", "font-variant",
		"font-weight", "letter-spacing", "line-height", "text-align", "text-indent",
		"text-transform", "visibility", "white-space", "word-spacing",
		"border-collapse", "border-spacing", "caption-side", "empty-cells",
		"list-style", "list-style-image", "list-style-position", "list-style-type",
		"quotes", "cursor", "direction", "unicode-bidi":
		return true
	}
	return false
}

// ParseLength parses a CSS length: number + unit, where bare numbers are
// pixels. Units: px, pt, pc, in, cm, mm, em, rem, ex, ch, %, vw, vh.
func ParseLength(v string) (val float64, unit string, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, "", false
	}
	i := 0
	if v[0] == '+' || v[0] == '-' {
		i++
	}
	start := i
	for i < len(v) && (v[i] >= '0' && v[i] <= '9' || v[i] == '.') {
		i++
	}
	if i == start {
		return 0, "", false
	}
	num, err := strconv.ParseFloat(v[:i], 64)
	if err != nil {
		return 0, "", false
	}
	unit = strings.ToLower(strings.TrimSpace(v[i:]))
	if unit == "" {
		unit = "px"
	}
	switch unit {
	case "px", "pt", "pc", "in", "cm", "mm", "em", "rem", "ex", "ch", "%", "vw", "vh":
		return num, unit, true
	default:
		return 0, "", false
	}
}

// ParseNumber parses a bare number, e.g. line-height or font-weight.
func ParseNumber(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ParseColor parses #rgb, #rrggbb, #rrggbbaa, rgb()/rgba() with integer,
// float or percentage channels, and a named-color subset. It returns RGB in
// 0..255 and alpha in 0..1; ok=false for anything unrecognized.
func ParseColor(v string) (r, g, b int, alpha float64, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, 0, 0, 0, false
	}
	// CSS variables: var(--name, fallback) — resolve fallback only (no custom props).
	if strings.HasPrefix(strings.ToLower(v), "var(") {
		if fb, okFB := cssVarFallback(v); okFB {
			return ParseColor(fb)
		}
		return 0, 0, 0, 0, false
	}
	alpha = 1
	if v[0] == '#' {
		hex := v[1:]
		switch len(hex) {
		case 3:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}
			return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), 1, true
		case 4:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}
			return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), float64(hexNibble(hex[3])) / 255, true
		case 6:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}
			return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), 1, true
		case 8:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}
			return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), float64(hexByte(hex[6:8])) / 255, true
		}
		return 0, 0, 0, 0, false
	}
	low := strings.ToLower(v)
	if low == "transparent" {
		return 0, 0, 0, 0, true
	}
	if name, found := namedColors[low]; found {
		return name[0], name[1], name[2], 1, true
	}
	if strings.HasPrefix(low, "rgb") {
		open := strings.IndexByte(v, '(')
		closeIdx := strings.LastIndexByte(v, ')')
		if open < 0 || closeIdx < open {
			return 0, 0, 0, 0, false
		}
		args := strings.Split(v[open+1:closeIdx], ",")
		channels := 3
		if strings.HasPrefix(low, "rgba") {
			channels = 4
		}
		if len(args) != channels {
			return 0, 0, 0, 0, false
		}
		var vals []float64
		for _, a := range args {
			a = strings.TrimSpace(a)
			if strings.HasSuffix(a, "%") {
				f, err := strconv.ParseFloat(strings.TrimSuffix(a, "%"), 64)
				if err != nil {
					return 0, 0, 0, 0, false
				}
				vals = append(vals, f*255/100)
			} else {
				f, err := strconv.ParseFloat(a, 64)
				if err != nil {
					return 0, 0, 0, 0, false
				}
				vals = append(vals, f)
			}
		}
		r = clampByte(vals[0])
		g = clampByte(vals[1])
		b = clampByte(vals[2])
		if channels == 4 {
			// alpha: 0..1 float, or percentage
			alpha = vals[3]
			if len(args) == 4 && strings.HasSuffix(strings.TrimSpace(args[3]), "%") {
				alpha /= 255
			}
			if alpha > 1 {
				alpha = 1
			}
			if alpha < 0 {
				alpha = 0
			}
		}
		return r, g, b, alpha, true
	}
	return 0, 0, 0, 0, false
}

// cssVarFallback extracts the fallback from var(--name, fallback). Nested
// var() in the fallback is not expanded further here.
func cssVarFallback(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if len(v) < 6 || !strings.EqualFold(v[:4], "var(") {
		return "", false
	}
	inner := v[4:]
	if !strings.HasSuffix(inner, ")") {
		return "", false
	}
	inner = strings.TrimSpace(inner[:len(inner)-1])
	// Split on top-level comma.
	depth := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				fb := strings.TrimSpace(inner[i+1:])
				return fb, fb != ""
			}
		}
	}
	return "", false
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func hexNibble(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		n := int(b - '0')
		return n*16 + n
	case b >= 'a' && b <= 'f':
		n := int(b-'a') + 10
		return n*16 + n
	case b >= 'A' && b <= 'F':
		n := int(b-'A') + 10
		return n*16 + n
	}
	return 0
}

func hexByte(s string) int {
	hi := hexVal(s[0])
	lo := hexVal(s[1])
	return hi*16 + lo
}

func hexVal(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	}
	return 0
}

func clampByte(f float64) int {
	if f < 0 {
		return 0
	}
	if f > 255 {
		return 255
	}
	return int(f + 0.5)
}

// ParseFontFamily splits a font-family value on commas and trims quotes.
func ParseFontFamily(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"'")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// namedColors is the named-color subset: the CSS2 set plus common extras.
var namedColors = map[string][3]int{
	"black": {0, 0, 0}, "silver": {192, 192, 192}, "gray": {128, 128, 128},
	"grey": {128, 128, 128}, "white": {255, 255, 255}, "maroon": {128, 0, 0},
	"red": {255, 0, 0}, "purple": {128, 0, 128}, "fuchsia": {255, 0, 255},
	"magenta": {255, 0, 255}, "green": {0, 128, 0}, "lime": {0, 255, 0},
	"olive": {128, 128, 0}, "yellow": {255, 255, 0}, "navy": {0, 0, 128},
	"blue": {0, 0, 255}, "teal": {0, 128, 128}, "aqua": {0, 255, 255},
	"cyan": {0, 255, 255}, "orange": {255, 165, 0}, "brown": {165, 42, 42},
	"pink": {255, 192, 203}, "gold": {255, 215, 0}, "darkgray": {169, 169, 169},
	"darkgrey": {169, 169, 169}, "lightgray": {211, 211, 211}, "lightgrey": {211, 211, 211},
	"darkgreen": {0, 100, 0}, "darkblue": {0, 0, 139}, "darkred": {139, 0, 0},
	"darkorange": {255, 140, 0}, "lightblue": {173, 216, 230}, "lightgreen": {144, 238, 144},
	"lightyellow": {255, 255, 224}, "coral": {255, 127, 80}, "crimson": {220, 20, 60},
	"khaki": {240, 230, 140}, "indigo": {75, 0, 130}, "ivory": {255, 255, 240},
	"lavender": {230, 230, 250}, "violet": {238, 130, 238}, "tan": {210, 180, 140},
	"salmon": {250, 128, 114}, "seagreen": {46, 139, 87}, "steelblue": {70, 130, 180},
	"turquoise": {64, 224, 208}, "wheat": {245, 222, 179}, "aliceblue": {240, 248, 255},
	"antiquewhite": {250, 235, 215}, "azure": {240, 255, 255}, "beige": {245, 245, 220},
	"bisque": {255, 228, 196}, "blanchedalmond": {255, 235, 205}, "burlywood": {222, 184, 135},
	"cadetblue": {95, 158, 160}, "chocolate": {210, 105, 30}, "darkslategray": {47, 79, 79},
	"deepskyblue": {0, 191, 255}, "dodgerblue": {30, 144, 255}, "firebrick": {178, 34, 34},
	"forestgreen": {34, 139, 34}, "gainsboro": {220, 220, 220}, "ghostwhite": {248, 248, 255},
	"goldenrod": {218, 165, 32}, "honeydew": {240, 255, 240}, "hotpink": {255, 105, 180},
	"indianred": {205, 92, 92}, "lightcoral": {240, 128, 128}, "lightcyan": {224, 255, 255},
	"lightpink": {255, 182, 193}, "lightsalmon": {255, 160, 122}, "lightseagreen": {32, 178, 170},
	"lightskyblue": {135, 206, 250}, "lightslategray": {119, 136, 153}, "lightsteelblue": {176, 196, 222},
	"mediumblue": {0, 0, 205}, "mediumpurple": {147, 112, 219}, "mediumseagreen": {60, 179, 113},
	"mediumslateblue": {123, 104, 238}, "mediumturquoise": {72, 209, 204}, "mediumvioletred": {199, 21, 133},
	"midnightblue": {25, 25, 112}, "mintcream": {245, 255, 250}, "mistyrose": {255, 228, 225},
	"moccasin": {255, 228, 181}, "navajowhite": {255, 222, 173}, "oldlace": {253, 245, 230},
	"olivedrab": {107, 142, 35}, "orangered": {255, 69, 0}, "orchid": {218, 112, 214},
	"palegoldenrod": {238, 232, 170}, "palegreen": {152, 251, 152}, "paleturquoise": {175, 238, 238},
	"palevioletred": {219, 112, 147}, "papayawhip": {255, 239, 213}, "peachpuff": {255, 218, 185},
	"peru": {205, 133, 63}, "plum": {221, 160, 221}, "powderblue": {176, 224, 230},
	"rebeccapurple": {102, 51, 153}, "rosybrown": {188, 143, 143}, "royalblue": {65, 105, 225},
	"saddlebrown": {139, 69, 19}, "sandybrown": {244, 164, 96}, "sienna": {160, 82, 45},
	"skyblue": {135, 206, 235}, "slateblue": {106, 90, 205}, "slategray": {112, 128, 144},
	"snow": {255, 250, 250}, "springgreen": {0, 255, 127}, "thistle": {216, 191, 216},
	"tomato": {255, 99, 71}, "violetred": {208, 32, 144}, "whitesmoke": {245, 245, 245},
	"yellowgreen": {154, 205, 50},
}
