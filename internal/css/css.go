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
	"errors"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// Numeric constants used by the CSS parser and value helpers.
const (
	doubleColonOffset = 2 // bytes past ':' for "::pseudo"
	minQuotedLen      = 2 // opening + closing quote
	anPlusBSplitParts = 2 // SplitN(..., "n", 2)
	hexRGBLen         = 3
	hexRGBALen        = 4
	hexRRGGBBLen      = 6
	hexRRGGBBAALen    = 8
	maxRGBChannel     = 255
	percentScale      = 100
	rgbaChannelCount  = 4
	hexLetterBase     = 10 // 'a'/'A' → 10 in hex
	roundHalfUp       = 0.5
)

// Stylesheet is a parsed stylesheet. Rules keep their source order.
type Stylesheet struct {
	Rules     []Rule
	FontFaces []FontFace
}

// FontFace is one @font-face rule (local src subset).
// Family and Src are consumed by convert.MergeFontFaces; weight/style are ignored.
type FontFace struct {
	Family string
	Src    string // raw src value (may contain url(...) or local(...))
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
	Tag           string
	Classes       []string
	ID            string
	Attrs         []AttrSelector
	Pseudos       []PseudoClass
	PseudoElement string // "before" | "after" | "" — never matches the host element
	Combinator    string
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
// :not(...). :is()/:where() are not implemented (unknown, never match).
type PseudoClass struct {
	Name string // lower-case, without leading ':'
	Arg  string // nth-child argument, lower-case, trimmed
	Has  []RelativeSelector
	Not  []Selector
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
	str := &Stylesheet{} //nolint:exhaustruct // intentional zero-value fields
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

				str.Rules = append(str.Rules, rules...)
			case strings.HasPrefix(low, "@container"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					return nil, errUnbalanced
				}

				prelude := strings.TrimSpace(src[len("@container"):open])
				contQ, found := parseContainerPrelude(prelude)

				block, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}

				src = rest

				if !found {
					continue
				}

				rules, err := parseRuleList("all", &contQ, block, &order)
				if err != nil {
					return nil, err
				}

				str.Rules = append(str.Rules, rules...)
			case strings.HasPrefix(low, "@page"):
				rest, err := skipAtRule(src)
				if err != nil {
					return nil, err
				}

				src = rest
			case strings.HasPrefix(low, "@keyframes"), strings.HasPrefix(low, "@-webkit-keyframes"):
				// Animations are parse-ignored (static cascaded values only).
				rest, err := skipAtRule(src)
				if err != nil {
					return nil, err
				}

				src = rest
			case strings.HasPrefix(low, "@font-face"):
				open := strings.IndexByte(src, '{')
				if open < 0 {
					rest, err := skipAtRule(src)
					if err != nil {
						return nil, err
					}

					src = rest

					continue
				}

				block, rest, err := takeBlock(src, open)
				if err != nil {
					return nil, err
				}

				src = rest

				if ff := parseFontFace(block); ff.Family != "" || ff.Src != "" {
					str.FontFaces = append(str.FontFaces, ff)
				}
			default:
				rest, err := skipAtRule(src)
				if err != nil {
					return nil, err
				}

				src = rest
			}

			continue
		}
		// one top-level rule
		selEnd, err := findBlock(src)
		if errors.Is(err, errNoBlock) {
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

		if r, ok := parseOneRule(selText, block, "all", nil, &order); ok {
			str.Rules = append(str.Rules, r)
		}
	}

	return str, nil
}

// parseOneRule builds a Rule from selector text + declaration block, owning
// the order counter. Shared by Parse and parseRuleList.
func parseOneRule(selText, block, media string, contQ *ContainerQuery, order *int) (Rule, bool) {
	sel, ok := parseSelectorList(selText)
	if !ok || len(sel) == 0 {
		return Rule{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	rVal := Rule{ //nolint:exhaustruct // intentional zero-value fields
		Selectors: sel,
		Decls:     parseDeclarations(block),
		Media:     media,
		Order:     *order,
	}

	if contQ != nil {
		cp := *contQ
		rVal.Container = &cp
	}

	*order++

	return rVal, true
}

// skipAtRule consumes one at-rule from src: its braced block when present,
// otherwise a ';'-terminated statement (or everything when neither exists).
// Returns the remaining source.
func skipAtRule(src string) (rest string, err error) {
	if open := strings.IndexByte(src, '{'); open >= 0 {
		_, rest, err := takeBlock(src, open)
		if err != nil {
			return "", err
		}

		return rest, nil
	}

	if end := strings.IndexByte(src, ';'); end >= 0 {
		return src[end+1:], nil
	}

	return "", nil
}

func parseFontFace(block string) FontFace {
	var fontFace FontFace

	for _, data := range parseDeclarations(block) {
		switch strings.ToLower(data.Prop) {
		case "font-family":
			fams := ParseFontFamily(data.Value)
			if len(fams) > 0 {
				fontFace.Family = fams[0]
			} else {
				fontFace.Family = strings.Trim(data.Value, " \"'")
			}
		case "src":
			fontFace.Src = data.Value
		}
	}

	return fontFace
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
				innerCQ, found := parseContainerPrelude(prelude)

				innerBlock, rem, err := takeBlock(block, open)
				if err != nil {
					return nil, err
				}

				block = rem

				if !found {
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
			rest, err := skipAtRule(block)
			if err != nil {
				return nil, err
			}

			block = rest

			continue
		}

		selEnd, err := findBlock(block)
		if errors.Is(err, errNoBlock) {
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

		if r, ok := parseOneRule(selText, declBlock, media, cq, orderPtr); ok {
			rules = append(rules, r)
		}
	}

	return rules, nil
}

var (
	errUnbalanced = &parseError{"unbalanced braces in stylesheet"}
	errNoBlock    = &parseError{"missing '{' before ';'"}
)

type parseError struct{ msg string }

func (e *parseError) Error() string { return "css: " + e.msg }

// stripComments removes /* ... */ comments, preserving newlines so line
// numbers stay roughly stable.
func stripComments(src string) string {
	var buf strings.Builder

	for {
		idx := strings.Index(src, "/*")
		if idx < 0 {
			buf.WriteString(src)

			return buf.String()
		}

		buf.WriteString(src[:idx])
		rest := src[idx+2:]

		jdx := strings.Index(rest, "*/")
		if jdx < 0 {
			return buf.String()
		}

		for k := idx; k <= idx+1+jdx; k++ {
			if src[k] == '\n' {
				buf.WriteByte('\n')
			}
		}

		src = rest[jdx+2:]
	}
}

// findBlock returns the index of the '{' ending the selector list, tracking
// quotes and parens so braces inside them are ignored.
func findBlock(src string) (int, error) {
	depth := 0

	for idx := 0; idx < len(src); idx++ {
		switch src[idx] {
		case '"', '\'':
			q := src[idx]

			j := strings.IndexByte(src[idx+1:], q)
			if j < 0 {
				return -1, &parseError{"unterminated string in stylesheet"}
			}

			idx += j + 1
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
				return idx, nil
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

	for idx := open; idx < len(src); idx++ {
		switch src[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open+1 : idx], src[idx+1:], nil
			}
		case '"', '\'':
			q := src[idx]

			j := strings.IndexByte(src[idx+1:], q)
			if j < 0 {
				return "", "", &parseError{"unterminated string in stylesheet"}
			}

			idx += j + 1
		}
	}

	return "", "", &parseError{"unbalanced braces in stylesheet"}
}

// ParseSelectors parses a comma-separated selector list; every part must
// parse (strict). Callers needing a standalone selector no longer fabricate
// a whole stylesheet.
func ParseSelectors(s string) ([]Selector, bool) { return parseSelectorListStrict(s, false) }

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
func splitTopLevel(str string, sep byte) []string {
	var out []string

	depth := 0
	start := 0

	for idx := 0; idx < len(str); idx++ {
		switch str[idx] {
		case '"', '\'':
			q := str[idx]

			j := strings.IndexByte(str[idx+1:], q)
			if j < 0 {
				idx = len(str)
			} else {
				idx += j + 1
			}
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				out = append(out, strings.TrimSpace(str[start:idx]))
				start = idx + 1
			}
		}
	}

	out = append(out, strings.TrimSpace(str[start:]))

	return out
}

// parseSelector parses one compound chain, e.g. "div.a#b > p.c" or
// "tr:nth-child(even)".
func parseSelector(s string) (Selector, bool) {
	return parseSelectorCtx(s, false)
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

	for idx := 0; idx < len(s); idx++ {
		cnt := s[idx]

		switch {
		case cnt == ' ' || cnt == '\t' || cnt == '\n' || cnt == '\r':
			// skip whitespace; it becomes a descendant combinator only when
			// it sits between two compounds
			flush()

			for idx+1 < len(s) && (s[idx+1] == ' ' || s[idx+1] == '\t' || s[idx+1] == '\n' || s[idx+1] == '\r') {
				idx++
			}

			if len(out) > 0 && out[len(out)-1] != " " && out[len(out)-1] != ">" &&
				out[len(out)-1] != "+" && out[len(out)-1] != "~" && idx+1 < len(s) && s[idx+1] != '>' {
				out = append(out, " ")
			}
		case cnt == '>' || cnt == '+' || cnt == '~':
			flush()

			out = append(out, string(cnt))
		case cnt == '[':
			// keep [attr] / [attr=value] inside the compound
			jdx := strings.IndexByte(s[idx:], ']')
			if jdx < 0 {
				cur.WriteByte(cnt)
			} else {
				if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
					out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
					cur.WriteByte('*')
				}

				cur.WriteString(s[idx : idx+jdx+1])
				idx += jdx
			}
		case cnt == ':':
			// keep :pseudo / :nth-child(n) / :has(...) and ::pseudo-elements
			// inside the compound so parseCompound can reject unsupported
			// pseudo-elements. Never strip ::before/::after — that used to
			// leave a bare host selector (Vector print `p::before{width:120pt}`
			// became `p{width:120pt}` and crushed wiki body columns).
			if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
				out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
				cur.WriteByte('*')
			}

			start := idx

			if idx+1 < len(s) && s[idx+1] == ':' {
				idx += 2 // ::pseudo-element
			} else {
				idx++ // :pseudo-class or CSS2 :before/:after
			}

			for idx < len(s) && !isSelBreak(s[idx]) {
				if s[idx] == '(' {
					_, end, ok := takeParenArg(s, idx)
					if !ok {
						idx = len(s)

						break
					}

					idx = end

					break
				}

				idx++
			}

			cur.WriteString(s[start:idx])

			idx--
		case cnt == '\\':
			// escape: keep next char literally
			if idx+1 < len(s) {
				cur.WriteByte(s[idx+1])

				idx++
			}
		default:
			cur.WriteByte(cnt)
		}
	}

	flush()

	return out
}

func isSelBreak(b byte) bool {
	return b == '.' || b == '#' || b == '[' || b == ':' || b == '>' || b == '+' ||
		b == '~' || b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// parseCompoundCtx parses a compound ("tag#id.class[attr]:nth-child(even)").
// A tag of "*" or "" means universal. When insideHas is true, nested :has()
// and pseudo-elements are rejected as invalid.
func parseCompoundCtx(s string, insideHas bool) (SelectorPart, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return SelectorPart{Tag: "*"}, true //nolint:exhaustruct // intentional zero-value fields
	}

	var part SelectorPart

	idx := 0
	// tag name first
	for idx < len(s) && !isCompoundBreak(s[idx]) {
		idx++
	}

	part.Tag = s[:idx]
	if part.Tag == "" {
		part.Tag = "*"
	} else if part.Tag != "*" && !validIdent(part.Tag) {
		return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	for idx < len(s) {
		switch s[idx] {
		case '#':
			jdx := idx + 1
			for jdx < len(s) && !isCompoundBreak(s[jdx]) {
				jdx++
			}

			id := s[idx+1 : jdx]
			if !validIdent(id) {
				return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
			}

			part.ID = id
			idx = jdx
		case '.':
			jdx := idx + 1
			for jdx < len(s) && !isCompoundBreak(s[jdx]) {
				jdx++
			}

			if jdx > idx+1 {
				cls := s[idx+1 : jdx]
				if !validIdent(cls) {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				part.Classes = append(part.Classes, cls)
			}

			idx = jdx
		case '[':
			jdx := strings.IndexByte(s[idx:], ']')
			if jdx < 0 {
				return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
			}

			attr, ok := parseAttrSelector(s[idx : idx+jdx+1])
			if !ok {
				return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
			}

			part.Attrs = append(part.Attrs, attr)
			idx += jdx + 1
		case ':':
			if idx+1 < len(s) && s[idx+1] == ':' {
				// ::pseudo-element (Selectors 3+). Only ::before/::after are
				// supported; others reject the selector so declarations do
				// not apply to the host.
				jdx := idx + doubleColonOffset
				for jdx < len(s) && s[jdx] != '(' && !isCompoundBreak(s[jdx]) {
					jdx++
				}

				peVal := strings.ToLower(s[idx+doubleColonOffset : jdx])
				if peVal != "before" && peVal != "after" {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				if hasParen := jdx < len(s) && s[jdx] == '('; hasParen {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				part.PseudoElement = peVal
				idx = jdx

				continue
			}

			jdx := idx + 1
			for jdx < len(s) && s[jdx] != '(' && !isCompoundBreak(s[jdx]) {
				jdx++
			}

			name := strings.ToLower(s[idx+1 : jdx])
			arg := ""

			var argRaw string

			hasParen := jdx < len(s) && s[jdx] == '('
			if hasParen {
				raw, end, ok := takeParenArg(s, jdx)
				if !ok {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				argRaw = raw
				arg = strings.ToLower(strings.TrimSpace(raw))
				jdx = end
			}

			if insideHas {
				switch name {
				case "has", "before", "after", "first-line", "first-letter":
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}
			}

			switch name {
			case "first-child", "last-child", "nth-child":
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name, Arg: arg}) //nolint:exhaustruct // intentional zero-value fields
			case "has":
				if !hasParen || strings.TrimSpace(argRaw) == "" {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				lowArg := strings.ToLower(argRaw)
				if strings.Contains(lowArg, ":has(") || strings.Contains(argRaw, "::") {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				rels, ok := parseRelativeSelectorList(argRaw)
				if !ok {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				part.Pseudos = append(part.Pseudos, PseudoClass{Name: "has", Has: rels}) //nolint:exhaustruct // intentional zero-value fields
			case "not":
				if !hasParen || strings.TrimSpace(argRaw) == "" {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				sels, ok := parseSelectorListStrict(argRaw, insideHas)
				if !ok {
					return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
				}

				part.Pseudos = append(part.Pseudos, PseudoClass{Name: "not", Not: sels}) //nolint:exhaustruct // intentional zero-value fields
			case "link", "visited":
				// Print semantics: both mean "a[href]" (no browsing history).
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name}) //nolint:exhaustruct // intentional zero-value fields
			case "hover", "active", "focus", "target":
				// Accepted for parse/cascade structure but never match in print
				// (static PDF has no pointer/focus/:target fragment state).
				// Keeping them on the compound prevents li:target from
				// degrading to bare `li` (wiki reflist highlight blue).
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name}) //nolint:exhaustruct // intentional zero-value fields
			case "before", "after":
				// CSS2 single-colon pseudo-elements.
				part.PseudoElement = name
			case "first-line", "first-letter":
				return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
			default:
				// Keep unknown pseudos so they do not degrade to the host
				// selector (same class of bug as stripping :target / ::before).
				part.Pseudos = append(part.Pseudos, PseudoClass{Name: name, Arg: arg}) //nolint:exhaustruct // intentional zero-value fields
			}

			idx = jdx
		default:
			return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
		}
	}

	return part, true
}

func parseAttrSelector(s string) (AttrSelector, bool) {
	// s includes brackets: [href], [href="x"], [typeof~='mw:File/Thumb'], [class*="noprint"]
	if len(s) < 3 || s[0] != '[' || s[len(s)-1] != ']' {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// Operator forms: ~= *= ^= $= |= =  (check multi-char before bare =)
	oper := ""
	nameEnd := -1

	for _, cand := range []string{"~=", "*=", "^=", "$=", "|="} {
		if i := strings.Index(inner, cand); i > 0 {
			nameEnd = i
			oper = cand

			break
		}
	}

	if nameEnd < 0 {
		if i := strings.IndexByte(inner, '='); i >= 0 {
			nameEnd = i
			oper = "="
		}
	}

	if nameEnd < 0 {
		if !validIdent(inner) {
			return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		return AttrSelector{Name: strings.ToLower(inner)}, true //nolint:exhaustruct // intentional zero-value fields
	}

	name := strings.TrimSpace(inner[:nameEnd])
	val := strings.TrimSpace(inner[nameEnd+len(oper):])

	if !validIdent(name) {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	if len(val) >= minQuotedLen {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}

	switch oper {
	case "=", "~=", "*=", "^=", "$=", "|=":
		return AttrSelector{Name: strings.ToLower(name), Op: oper, Value: val}, true
	default:
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
}

// validIdent reports whether s is a valid CSS identifier (letters, digits,
// '-', '_', and digits after the first character).
func validIdent(s string) bool {
	if s == "" {
		return false
	}

	for i := range len(s) {
		c := s[i]
		found := c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')

		if i > 0 && (c >= '0' && c <= '9') {
			found = true
		}

		if !found {
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
// ancestors/siblings per their combinators. Implemented via leftmostMatch
// (same combinator walk; Match only needs success/failure).
func Match(s Selector, n *html.Node) bool {
	return leftmostMatch(s, n) != nil
}

// MatchPseudo reports whether s selects the ::before or ::after pseudo-element
// of n (pe is "before" or "after").
func MatchPseudo(sel Selector, count *html.Node, pseudo string) bool {
	if count == nil || pseudo == "" || len(sel.Parts) == 0 {
		return false
	}

	last := sel.Parts[len(sel.Parts)-1]
	if last.PseudoElement != pseudo {
		return false
	}

	parts := make([]SelectorPart, len(sel.Parts))
	copy(parts, sel.Parts)
	parts[len(parts)-1].PseudoElement = ""

	return Match(Selector{Parts: parts}, count)
}

// matchPart matches one compound against an element.
func matchPart(p SelectorPart, n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	// ::before/::after never match the host element (declarations apply to
	// generated pseudo boxes via MatchPseudo).
	if p.PseudoElement != "" {
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

	for _, arg := range p.Attrs {
		val, found := "", false
		if n.Attrs != nil {
			val, found = n.Attrs[arg.Name]
		}

		if arg.Op == "" {
			if !found {
				return false
			}

			continue
		}

		if !found {
			return false
		}

		switch arg.Op {
		case "=":
			if val != arg.Value {
				return false
			}
		case "~=":
			if arg.Value == "" || strings.Contains(arg.Value, " ") {
				return false
			}

			found := false

			for _, w := range strings.Fields(val) {
				if w == arg.Value {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		case "*=":
			if arg.Value == "" || !strings.Contains(val, arg.Value) {
				return false
			}
		case "^=":
			if arg.Value == "" || !strings.HasPrefix(val, arg.Value) {
				return false
			}
		case "$=":
			if arg.Value == "" || !strings.HasSuffix(val, arg.Value) {
				return false
			}
		case "|=":
			// Exact match or value is followed by a hyphen (HTML lang / BCP47-style).
			if arg.Value == "" {
				return false
			}

			if val != arg.Value && !strings.HasPrefix(val, arg.Value+"-") {
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
	case "link", "visited":
		// Print: no link history — both match any anchor with an href.
		return isLinkAnchor(n)
	case "root":
		// Document element (html in HTML). html.Parse wraps the tree in a
		// synthetic ElementNode named "#document" — that must not match, and
		// must not block <html> from matching.
		if n.Type != html.ElementNode || n.Name == "#document" || strings.HasPrefix(n.Name, "#") {
			return false
		}

		for p := n.Parent; p != nil; p = p.Parent {
			if p.Type == html.ElementNode && p.Name != "#document" && !strings.HasPrefix(p.Name, "#") {
				return false
			}
		}

		return true
	case "hover", "active", "focus", "target":
		return false
	default:
		// Unknown pseudo-classes never match in print (kept on the compound
		// so selectors do not degrade to the host).
		return false
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

func previousElementSibling(count *html.Node) *html.Node {
	if count == nil || count.Parent == nil {
		return nil
	}

	var prev *html.Node

	for _, cur := range count.Parent.Children {
		if cur == count {
			return prev
		}

		if cur.Type == html.ElementNode {
			prev = cur
		}
	}

	return nil
}

func nextElementSibling(count *html.Node) *html.Node {
	if count == nil || count.Parent == nil {
		return nil
	}

	seen := false

	for _, cur := range count.Parent.Children {
		if cur == count {
			seen = true

			continue
		}

		if seen && cur.Type == html.ElementNode {
			return cur
		}
	}

	return nil
}

// elementIndex is 1-based among element siblings.
func elementIndex(count *html.Node) int {
	if count == nil || count.Parent == nil {
		return 1
	}

	idx := 0

	for _, cur := range count.Parent.Children {
		if cur.Type != html.ElementNode {
			continue
		}

		idx++
		if cur == count {
			return idx
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
	// an+buf / n+buf / -n+buf / an
	var specA, buf int

	if strings.Contains(arg, "n") {
		parts := strings.SplitN(arg, "n", anPlusBSplitParts)
		asVal := strings.TrimSpace(parts[0])

		bsVal := ""
		if len(parts) == anPlusBSplitParts {
			bsVal = strings.TrimSpace(parts[1])
		}

		switch asVal {
		case "", "+":
			specA = 1
		case "-":
			specA = -1
		default:
			var err error

			specA, err = strconv.Atoi(asVal)
			if err != nil {
				return false
			}
		}

		if bsVal == "" {
			buf = 0
		} else {
			var err error

			buf, err = strconv.Atoi(bsVal)
			if err != nil {
				return false
			}
		}

		if specA == 0 {
			return index == buf
		}
		// index = a*k + b for integer k >= 0
		if (index-buf)%specA != 0 {
			return false
		}

		k := (index - buf) / specA

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
	for _, page := range s.Parts {
		if page.ID != "" {
			a++
		}

		b += len(page.Classes) + len(page.Attrs)

		if page.Tag != "*" {
			c++
		}

		if page.PseudoElement != "" {
			c++ // pseudo-elements count like type selectors
		}

		for _, pageSize := range page.Pseudos {
			switch pageSize.Name {
			case "has":
				a2, b2, c2 := maxRelativeSpecificity(pageSize.Has)
				a += a2
				b += b2
				c += c2
			case "not":
				a2, b2, c2 := maxSelectorSpecificity(pageSize.Not)
				a += a2
				b += b2
				c += c2
			default:
				b++
			}
		}
	}

	return a, b, c
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

		important := isImportant(value)
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

func validPropName(page string) bool {
	if page == "" {
		return false
	}

	for i := range len(page) {
		c := page[i]
		if !(c == '-' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}

	return true
}

// isImportant reports whether a declaration value carries !important
// (whitespace between ! and important is allowed).
func isImportant(val string) bool {
	val = strings.TrimSpace(val)

	const word = "important"

	if len(val) < len(word)+1 {
		return false
	}

	if !strings.EqualFold(val[len(val)-len(word):], word) {
		return false
	}

	rest := strings.TrimRight(val[:len(val)-len(word)], " \t")

	return strings.HasSuffix(rest, "!")
}

// stripImportant removes a trailing !important (any case, optional space)
// from a declaration value.
func stripImportant(val string) string {
	if !isImportant(val) {
		return val
	}

	val = strings.TrimRight(val, " \t")
	val = val[:len(val)-len("important")]
	val = strings.TrimRight(val, " \t")
	val = strings.TrimSuffix(val, "!")

	return strings.TrimSpace(val)
}

// ParseLength parses a CSS length: number + unit, where bare numbers are
// pixels. Units: px, pt, pc, in, cm, mm, em, rem, ex, ch, %, vw, vh.
func ParseLength(v string) (val float64, unit string, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, "", false
	}

	idx := 0
	if v[0] == '+' || v[0] == '-' {
		idx++
	}

	start := idx

	for idx < len(v) && (v[idx] >= '0' && v[idx] <= '9' || v[idx] == '.') {
		idx++
	}

	if idx == start {
		return 0, "", false
	}

	num, err := strconv.ParseFloat(v[:idx], 64)
	if err != nil {
		return 0, "", false
	}

	unit = strings.ToLower(strings.TrimSpace(v[idx:]))
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
func ParseNumber(val string) (float64, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, false
	}

	f, err := strconv.ParseFloat(val, 64)
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
	// ponytail: ParseColor accepts bare var() without a prop map (API is color-
	// string only). Layout resolves custom props via ResolveCustomProps before
	// color parse; upgrade if ParseColor gains a props argument.
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
		case hexRGBLen:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}

			return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), 1, true
		case hexRGBALen:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}

			return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), float64(hexNibble(hex[3])) / maxRGBChannel, true
		case hexRRGGBBLen:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}

			return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), 1, true
		case hexRRGGBBAALen:
			if !isHex(hex) {
				return 0, 0, 0, 0, false
			}

			return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), float64(hexByte(hex[6:8])) / maxRGBChannel, true
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

		channels := hexRGBLen
		if strings.HasPrefix(low, "rgba") {
			channels = rgbaChannelCount
		}

		if len(args) != channels {
			return 0, 0, 0, 0, false
		}

		var vals []float64

		for _, arg := range args {
			arg = strings.TrimSpace(arg)
			if strings.HasSuffix(arg, "%") {
				f, err := strconv.ParseFloat(strings.TrimSuffix(arg, "%"), 64)
				if err != nil {
					return 0, 0, 0, 0, false
				}

				vals = append(vals, f*maxRGBChannel/percentScale)
			} else {
				f, err := strconv.ParseFloat(arg, 64)
				if err != nil {
					return 0, 0, 0, 0, false
				}

				vals = append(vals, f)
			}
		}

		r = clampByte(vals[0])
		g = clampByte(vals[1])
		b = clampByte(vals[2])

		if channels == rgbaChannelCount {
			// alpha: 0..1 float, or percentage
			alpha = vals[3]
			if len(args) == rgbaChannelCount && strings.HasSuffix(strings.TrimSpace(args[3]), "%") {
				alpha /= maxRGBChannel
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
	_, fb, ok := parseVarFn(v)

	return fb, ok && fb != ""
}

// parseVarFn parses a top-level var(--name) or var(--name, fallback).
// ok is false when v is not a var() function.
func parseVarFn(v string) (name, fallback string, ok bool) {
	v = strings.TrimSpace(v)
	if len(v) < 6 || !strings.EqualFold(v[:4], "var(") {
		return "", "", false
	}

	inner := v[4:]
	if !strings.HasSuffix(inner, ")") {
		return "", "", false
	}

	inner = strings.TrimSpace(inner[:len(inner)-1])
	depth := 0

	for idx := range len(inner) {
		switch inner[idx] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				name = strings.ToLower(strings.TrimSpace(inner[:idx]))
				fallback = strings.TrimSpace(inner[idx+1:])

				return name, fallback, name != ""
			}
		}
	}

	name = strings.ToLower(strings.TrimSpace(inner))

	return name, "", name != ""
}

// ResolveVar expands CSS var() references in v using lookup(--name).
// Unresolved var() uses the CSS fallback when present; otherwise the empty
// string (caller treats as invalid / keeps the prior cascaded value).
// Nested var() expands up to a small depth.
func ResolveVar(val_2 string, lookup func(name string) (string, bool)) string {
	val_2 = strings.TrimSpace(val_2)
	for range 16 {
		if !strings.HasPrefix(strings.ToLower(val_2), "var(") {
			return val_2
		}

		name, fallback, ok := parseVarFn(val_2)
		if !ok {
			return val_2
		}

		if lookup != nil {
			if val, found := lookup(name); found && strings.TrimSpace(val) != "" {
				val_2 = strings.TrimSpace(val)

				continue
			}
		}

		if fallback != "" {
			val_2 = fallback

			continue
		}

		return ""
	}

	return val_2
}

// ResolveCustomProps walks a custom-property graph: the inherited overlay
// plus declared --* values, with var() chains expanded once using a memo and
// a cycle stack (cycles resolve to invalid → empty). The single place
// custom-property policy lives.
func ResolveCustomProps(declared, inherited map[string]string) map[string]string {
	work := make(map[string]string, len(inherited)+len(declared))
	for k, v := range inherited {
		work[k] = v
	}

	for k, v := range declared {
		work[k] = v
	}

	memo := map[string]string{}

	var eval func(name string, stack map[string]bool) string

	eval = func(name string, stack map[string]bool) string {
		if v, ok := memo[name]; ok {
			return v
		}

		raw, ok := work[name]
		if !ok || !strings.Contains(strings.ToLower(raw), "var(") {
			memo[name] = raw

			return raw
		}

		if stack[name] {
			return ""
		}

		stack[name] = true
		val := ResolveVar(raw, func(n string) (string, bool) {
			s := eval(n, stack)
			_, exists := work[n]

			return s, exists && strings.TrimSpace(s) != ""
		})

		delete(stack, name)

		memo[name] = val

		return val
	}
	for name := range work {
		eval(name, map[string]bool{})
	}

	return memo
}

func isHex(s string) bool {
	for i := range len(s) {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}

	return true
}

func hexNibble(buf byte) int {
	switch {
	case buf >= '0' && buf <= '9':
		n := int(buf - '0')

		return n*16 + n
	case buf >= 'a' && buf <= 'f':
		n := int(buf-'a') + hexLetterBase

		return n*16 + n
	case buf >= 'A' && buf <= 'F':
		n := int(buf-'A') + hexLetterBase

		return n*16 + n
	}

	return 0
}

func hexByte(s string) int {
	hi := hexVal(s[0])
	lo := hexVal(s[1])

	return hi*16 + lo
}

func hexVal(buf byte) int {
	switch {
	case buf >= '0' && buf <= '9':
		return int(buf - '0')
	case buf >= 'a' && buf <= 'f':
		return int(buf-'a') + hexLetterBase
	case buf >= 'A' && buf <= 'F':
		return int(buf-'A') + hexLetterBase
	}

	return 0
}

func clampByte(fVal float64) int {
	if fVal < 0 {
		return 0
	}

	if fVal > maxRGBChannel {
		return maxRGBChannel
	}

	return int(fVal + roundHalfUp)
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

// namedColors is CSS2 system colors plus greys/orange and common web names
// used by fixtures and layout tests (ponytail: not the full CSS Color 4 list).
var namedColors = map[string][3]int{
	// CSS2 core
	"black": {0, 0, 0}, "silver": {192, 192, 192}, "gray": {128, 128, 128},
	"grey": {128, 128, 128}, "white": {255, 255, 255}, "maroon": {128, 0, 0},
	"red": {255, 0, 0}, "purple": {128, 0, 128}, "fuchsia": {255, 0, 255},
	"green": {0, 128, 0}, "lime": {0, 255, 0}, "olive": {128, 128, 0},
	"yellow": {255, 255, 0}, "navy": {0, 0, 128}, "blue": {0, 0, 255},
	"teal": {0, 128, 128}, "aqua": {0, 255, 255},
	// Common aliases / CSS3 extras used in sheets
	"cyan": {0, 255, 255}, "magenta": {255, 0, 255}, "orange": {255, 165, 0},
	"brown": {165, 42, 42}, "pink": {255, 192, 203}, "gold": {255, 215, 0},
	"darkgray": {169, 169, 169}, "darkgrey": {169, 169, 169},
	"lightgray": {211, 211, 211}, "lightgrey": {211, 211, 211},
	"darkgreen": {0, 100, 0}, "darkblue": {0, 0, 139}, "darkred": {139, 0, 0},
	"darkorange": {255, 140, 0}, "lightblue": {173, 216, 230},
	"lightgreen": {144, 238, 144}, "lightyellow": {255, 255, 224},
	"coral": {255, 127, 80}, "crimson": {220, 20, 60}, "indigo": {75, 0, 130},
	"khaki": {240, 230, 140}, "lavender": {230, 230, 250}, "violet": {238, 130, 238},
	"tan": {210, 180, 140}, "salmon": {250, 128, 114}, "seagreen": {46, 139, 87},
	"steelblue": {70, 130, 180}, "turquoise": {64, 224, 208}, "wheat": {245, 222, 179},
	"orangered": {255, 69, 0}, "tomato": {255, 99, 71}, "whitesmoke": {245, 245, 245},
	"gainsboro": {220, 220, 220}, "rebeccapurple": {102, 51, 153},
}
