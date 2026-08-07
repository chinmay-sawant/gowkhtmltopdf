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

// Pseudo-class and pseudo-element names shared across selector parsing.
const (
	pseudoClassHas   = "has"
	pseudoElemBefore = "before"
	pseudoElemAfter  = "after"
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

		var err error

		if strings.HasPrefix(src, "@") {
			src, err = parseAtRule(src, str, &order)
		} else {
			src, err = parseTopLevelRule(src, str, &order)
		}

		if err != nil {
			return nil, err
		}
	}

	return str, nil
}

// parseAtRule consumes one at-rule at the start of src, appending any
// resulting rules or font faces to str, and returns the remaining source.
func parseAtRule(src string, str *Stylesheet, order *int) (string, error) {
	low := strings.ToLower(src)

	switch {
	case strings.HasPrefix(low, "@media"):
		return parseMediaRule(src, str, order)
	case strings.HasPrefix(low, "@container"):
		return parseContainerRule(src, str, order)
	case strings.HasPrefix(low, "@page"):
		return skipAtRule(src)
	case strings.HasPrefix(low, "@keyframes"), strings.HasPrefix(low, "@-webkit-keyframes"):
		// Animations are parse-ignored (static cascaded values only).
		return skipAtRule(src)
	case strings.HasPrefix(low, "@font-face"):
		return parseFontFaceRule(src, str)
	default:
		return skipAtRule(src)
	}
}

func parseMediaRule(src string, str *Stylesheet, order *int) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return "", errUnbalanced
	}

	media := strings.TrimSpace(src[len("@media"):open])
	if media == "" {
		media = "all"
	}

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	rules, err := parseRuleList(media, nil, block, order)
	if err != nil {
		return "", err
	}

	str.Rules = append(str.Rules, rules...)

	return rest, nil
}

func parseContainerRule(src string, str *Stylesheet, order *int) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return "", errUnbalanced
	}

	prelude := strings.TrimSpace(src[len("@container"):open])
	contQ, found := parseContainerPrelude(prelude)

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	if !found {
		return rest, nil
	}

	rules, err := parseRuleList("all", &contQ, block, order)
	if err != nil {
		return "", err
	}

	str.Rules = append(str.Rules, rules...)

	return rest, nil
}

func parseFontFaceRule(src string, str *Stylesheet) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return skipAtRule(src)
	}

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	if ff := parseFontFace(block); ff.Family != "" || ff.Src != "" {
		str.FontFaces = append(str.FontFaces, ff)
	}

	return rest, nil
}

// parseTopLevelRule consumes one rule set (or garbage prelude) at the start
// of src and returns the remaining source.
func parseTopLevelRule(src string, str *Stylesheet, order *int) (string, error) {
	selEnd, err := findBlock(src)
	if errors.Is(err, errNoBlock) {
		// garbage prelude without a block: discard up to ';'
		if end := strings.IndexByte(src, ';'); end >= 0 {
			return src[end+1:], nil
		}

		return "", nil
	}

	if err != nil {
		return "", err
	}

	selText := strings.TrimSpace(src[:selEnd])

	block, rest, err := takeBlock(src, selEnd)
	if err != nil {
		return "", err
	}

	if selText != "" {
		if r, ok := parseOneRule(selText, block, "all", nil, order); ok {
			str.Rules = append(str.Rules, r)
		}
	}

	return rest, nil
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
func skipAtRule(src string) (string, error) {
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
// When contQ is non-nil, every produced rule inherits that container query.
func parseRuleList(media string, contQ *ContainerQuery, block string, orderPtr *int) ([]Rule, error) {
	var rules []Rule

	for block != "" {
		block = strings.TrimLeft(block, " \t\r\n")
		if block == "" {
			break
		}
		// Nested @container inside @media (or another @container): flatten.
		if strings.HasPrefix(block, "@") {
			rest, nested, err := parseNestedAtRule(block, media, orderPtr)
			if err != nil {
				return nil, err
			}

			rules = append(rules, nested...)
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

		if r, ok := parseOneRule(selText, declBlock, media, contQ, orderPtr); ok {
			rules = append(rules, r)
		}
	}

	return rules, nil
}

// parseNestedAtRule consumes an at-rule inside a @media/@container body.
// Nested @container rules are flattened into the media context (the nested
// query replaces, not combines, the outer query); other at-rules are skipped.
func parseNestedAtRule(block, media string, orderPtr *int) (string, []Rule, error) {
	if !strings.HasPrefix(strings.ToLower(block), "@container") {
		rest, err := skipAtRule(block)
		if err != nil {
			return "", nil, err
		}

		return rest, nil, nil
	}

	open := strings.IndexByte(block, '{')
	if open < 0 {
		return "", nil, errUnbalanced
	}

	prelude := strings.TrimSpace(block[len("@container"):open])
	innerCQ, found := parseContainerPrelude(prelude)

	innerBlock, rem, err := takeBlock(block, open)
	if err != nil {
		return "", nil, err
	}

	if !found {
		return rem, nil, nil
	}
	// Nested @container replaces (does not combine) the query.
	use := &innerCQ

	nested, err := parseRuleList(media, use, innerBlock, orderPtr)
	if err != nil {
		return "", nil, err
	}

	return rem, nested, nil
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
			j := strings.IndexByte(src[idx+1:], src[idx])
			if j < 0 {
				return -1, &parseError{"unterminated string in stylesheet"}
			}

			idx += j + 1
		case '(', '[':
			depth++
		case ')', ']':
			depth = decDepth(depth)
		case '{', ';':
			if depth == 0 {
				if src[idx] == '{' {
					return idx, nil
				}

				return -1, errNoBlock // garbage prelude: discard up to ';'
			}
		}
	}

	return -1, &parseError{"missing '{' in stylesheet"}
}

// decDepth decrements a bracket depth, ignoring stray closers (depth never
// goes below zero).
func decDepth(depth int) int {
	if depth > 0 {
		return depth - 1
	}

	return depth
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
	parts := splitTopLevel(s, ',')
	out := make([]Selector, 0, len(parts))

	for _, part := range parts {
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
func splitSelectorChain(sel string) []string {
	var out []string

	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	for idx := 0; idx < len(sel); idx++ {
		cnt := sel[idx]

		switch cnt {
		case ' ', '\t', '\n', '\r':
			// skip whitespace; it becomes a descendant combinator only when
			// it sits between two compounds
			flush()

			idx = skipWhitespace(sel, idx)
			out = addDescendantCombinator(out, sel, idx)
		case '>', '+', '~':
			flush()

			out = append(out, string(cnt))
		case '[':
			idx = writeBracketLiteral(&cur, out, sel, idx)
		case ':':
			idx = writePseudoLiteral(&cur, out, sel, idx)
		case '\\':
			// escape: keep next char literally
			if idx+1 < len(sel) {
				cur.WriteByte(sel[idx+1])

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
	switch b {
	case '.', '#', '[', ':', '>', '+', '~', ' ', '\t', '\n', '\r':
		return true
	}

	return false
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// skipWhitespace advances idx to the end of a run of whitespace starting at
// idx (the character at idx is whitespace).
func skipWhitespace(s string, idx int) int {
	for idx+1 < len(s) && isWhitespace(s[idx+1]) {
		idx++
	}

	return idx
}

// addDescendantCombinator appends a " " token when the whitespace run ending
// at idx sits between two compounds (not after a combinator, not before '>').
func addDescendantCombinator(out []string, s string, idx int) []string {
	if len(out) > 0 && out[len(out)-1] != " " && out[len(out)-1] != ">" &&
		out[len(out)-1] != "+" && out[len(out)-1] != "~" && idx+1 < len(s) && s[idx+1] != '>' {
		return append(out, " ")
	}

	return out
}

// writeBracketLiteral keeps [attr] / [attr=value] inside the compound,
// prefixed with '*' when the compound starts with the bracket. Returns the
// index of the ']' (or the current idx when unterminated).
func writeBracketLiteral(cur *strings.Builder, out []string, sel string, idx int) int {
	jdx := strings.IndexByte(sel[idx:], ']')
	if jdx < 0 {
		cur.WriteByte('[')

		return idx
	}

	writeStarPrefix(cur, out)
	cur.WriteString(sel[idx : idx+jdx+1])

	return idx + jdx
}

// writeStarPrefix prefixes the current compound with '*' when it starts at a
// compound boundary (fresh compound after a combinator or list start).
func writeStarPrefix(cur *strings.Builder, out []string) {
	if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
		out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
		cur.WriteByte('*')
	}
}

// writePseudoLiteral keeps :pseudo / :nth-child(n) / :has(...) and
// ::pseudo-elements inside the compound so parseCompound can reject
// unsupported pseudo-elements. Never strip ::before/::after — that used to
// leave a bare host selector (Vector print `p::before{width:120pt}` became
// `p{width:120pt}` and crushed wiki body columns). Returns the index of the
// character after the pseudo (the caller's loop applies the final -1/+1).
func writePseudoLiteral(cur *strings.Builder, out []string, sel string, idx int) int {
	writeStarPrefix(cur, out)

	start := idx

	if idx+1 < len(sel) && sel[idx+1] == ':' {
		idx += 2 // ::pseudo-element
	} else {
		idx++ // :pseudo-class or CSS2 :before/:after
	}

	for idx < len(sel) && !isSelBreak(sel[idx]) {
		if sel[idx] == '(' {
			_, end, ok := takeParenArg(sel, idx)
			if ok {
				cur.WriteString(sel[start:end])

				return end - 1
			}

			cur.WriteString(sel[start:])

			return len(sel) - 1
		}

		idx++
	}

	cur.WriteString(sel[start:idx])

	return idx - 1
}

// parseCompoundCtx parses a compound ("tag#id.class[attr]:nth-child(even)").
// A tag of "*" or "" means universal. When insideHas is true, nested :has()
// and pseudo-elements are rejected as invalid.
func parseCompoundCtx(sel string, insideHas bool) (SelectorPart, bool) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return SelectorPart{Tag: "*"}, true //nolint:exhaustruct // intentional zero-value fields
	}

	tag, idx, valid := parseCompoundTag(sel)
	if !valid {
		return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	part := SelectorPart{Tag: tag} //nolint:exhaustruct // intentional zero-value fields

	for idx < len(sel) {
		switch sel[idx] {
		case '#':
			part, idx, valid = parseCompoundID(sel, part, idx)
		case '.':
			part, idx, valid = parseCompoundClass(sel, part, idx)
		case '[':
			part, idx, valid = parseCompoundAttr(sel, part, idx)
		case ':':
			part, idx, valid = parseCompoundPseudo(sel, part, insideHas, idx)
		default:
			return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		if !valid {
			return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
		}
	}

	return part, true
}

// parseCompoundTag reads the tag name at the start of sel. An empty tag
// ("*.cls", ".cls") means universal. Returns the tag and the scan index.
func parseCompoundTag(sel string) (string, int, bool) {
	idx := 0
	for idx < len(sel) && !isCompoundBreak(sel[idx]) {
		idx++
	}

	tag := sel[:idx]
	if tag == "" {
		return "*", 0, true
	}

	if tag != "*" && !validIdent(tag) {
		return "", 0, false
	}

	return tag, idx, true
}

func parseCompoundID(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	id := sel[idx+1 : jdx]
	if !validIdent(id) {
		return part, idx, false
	}

	part.ID = id

	return part, jdx, true
}

func parseCompoundClass(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	if jdx > idx+1 {
		cls := sel[idx+1 : jdx]
		if !validIdent(cls) {
			return part, idx, false
		}

		part.Classes = append(part.Classes, cls)
	}

	return part, jdx, true
}

func parseCompoundAttr(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := strings.IndexByte(sel[idx:], ']')
	if jdx < 0 {
		return part, idx, false
	}

	attr, ok := parseAttrSelector(sel[idx : idx+jdx+1])
	if !ok {
		return part, idx, false
	}

	part.Attrs = append(part.Attrs, attr)

	return part, idx + jdx + 1, true
}

// parseCompoundPseudo parses one :pseudo or ::pseudo-element starting at idx
// in sel and returns the updated part and the index of the next compound
// break.
func parseCompoundPseudo(sel string, part SelectorPart, insideHas bool, idx int) (SelectorPart, int, bool) {
	if idx+1 < len(sel) && sel[idx+1] == ':' {
		return parsePseudoElement(sel, part, idx)
	}

	return parsePseudoClass(sel, part, insideHas, idx)
}

// parsePseudoElement handles the ::pseudo-element form. Only ::before/::after
// are supported; others reject the selector so declarations do not apply to
// the host.
func parsePseudoElement(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + doubleColonOffset
	for jdx < len(sel) && sel[jdx] != '(' && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	peVal := strings.ToLower(sel[idx+doubleColonOffset : jdx])
	if peVal != pseudoElemBefore && peVal != pseudoElemAfter {
		return part, idx, false
	}

	if jdx < len(sel) && sel[jdx] == '(' {
		return part, idx, false
	}

	part.PseudoElement = peVal

	return part, jdx, true
}

// parsePseudoClass handles the :pseudo-class form (including CSS2
// single-colon :before/:after).
func parsePseudoClass(sel string, part SelectorPart, insideHas bool, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && sel[jdx] != '(' && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	name := strings.ToLower(sel[idx+1 : jdx])
	arg := ""

	var argRaw string

	hasParen := jdx < len(sel) && sel[jdx] == '('
	if hasParen {
		raw, end, ok := takeParenArg(sel, jdx)
		if !ok {
			return part, idx, false
		}

		argRaw = raw
		arg = strings.ToLower(strings.TrimSpace(raw))
		jdx = end
	}

	part, ok := appendCompoundPseudo(part, name, arg, argRaw, hasParen, insideHas)

	return part, jdx, ok
}

// appendCompoundPseudo records the parsed pseudo on the part. When insideHas
// is true, nested :has() and pseudo-elements are rejected as invalid.
func appendCompoundPseudo(part SelectorPart, name, arg, argRaw string, hasParen, insideHas bool) (SelectorPart, bool) {
	if insideHas {
		switch name {
		case pseudoClassHas, pseudoElemBefore, pseudoElemAfter, "first-line", "first-letter":
			return part, false
		}
	}

	if name == pseudoClassHas || name == condKindNot {
		return appendFunctionalPseudo(part, name, argRaw, hasParen, insideHas)
	}

	return appendSimplePseudo(part, name, arg)
}

// appendFunctionalPseudo handles :has(...) and :not(...).
func appendFunctionalPseudo(part SelectorPart, name, argRaw string, hasParen, insideHas bool) (SelectorPart, bool) {
	if !hasParen || strings.TrimSpace(argRaw) == "" {
		return part, false
	}

	if name == pseudoClassHas {
		lowArg := strings.ToLower(argRaw)
		if strings.Contains(lowArg, ":has(") || strings.Contains(argRaw, "::") {
			return part, false
		}

		rels, ok := parseRelativeSelectorList(argRaw)
		if !ok {
			return part, false
		}

		part.Pseudos = append(part.Pseudos, pseudoClass(pseudoClassHas, "", rels, nil))

		return part, true
	}

	if name == condKindNot {
		sels, ok := parseSelectorListStrict(argRaw, insideHas)
		if !ok {
			return part, false
		}

		part.Pseudos = append(part.Pseudos, pseudoClass(condKindNot, "", nil, sels))

		return part, true
	}

	return part, false
}

// appendSimplePseudo handles the non-functional pseudo-classes and the
// CSS2 single-colon pseudo-elements.
func appendSimplePseudo(part SelectorPart, name, arg string) (SelectorPart, bool) {
	switch name {
	case "first-child", "last-child", "nth-child":
		part.Pseudos = append(part.Pseudos, pseudoClass(name, arg, nil, nil))
	case "link", "visited":
		// Print semantics: both mean "a[href]" (no browsing history).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, "", nil, nil))
	case "hover", "active", "focus", "target":
		// Accepted for parse/cascade structure but never match in print
		// (static PDF has no pointer/focus/:target fragment state).
		// Keeping them on the compound prevents li:target from
		// degrading to bare `li` (wiki reflist highlight blue).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, "", nil, nil))
	case pseudoElemBefore, pseudoElemAfter:
		// CSS2 single-colon pseudo-elements.
		part.PseudoElement = name
	case "first-line", "first-letter":
		return part, false
	default:
		// Keep unknown pseudos so they do not degrade to the host
		// selector (same class of bug as stripping :target / ::before).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, arg, nil, nil))
	}

	return part, true
}

// pseudoClass builds a PseudoClass with explicit zero Has/Not slices so the
// literal stays exhaustive without per-use nolint comments.
func pseudoClass(name, arg string, has []RelativeSelector, not []Selector) PseudoClass {
	return PseudoClass{Name: name, Arg: arg, Has: has, Not: not}
}

func parseAttrSelector(sel string) (AttrSelector, bool) {
	// sel includes brackets: [href], [href="x"], [typeof~='mw:File/Thumb'], [class*="noprint"]
	if len(sel) < 3 || sel[0] != '[' || sel[len(sel)-1] != ']' {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	inner := strings.TrimSpace(sel[1 : len(sel)-1])
	if inner == "" {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// Operator forms: ~= *= ^= $= |= =  (check multi-char before bare =)
	nameEnd, oper := findAttrOperator(inner)

	if nameEnd < 0 {
		if !validIdent(inner) {
			return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		return AttrSelector{Name: strings.ToLower(inner)}, true //nolint:exhaustruct // intentional zero-value fields
	}

	name := strings.TrimSpace(inner[:nameEnd])
	val := stripAttrQuotes(strings.TrimSpace(inner[nameEnd+len(oper):]))

	if !validIdent(name) {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	switch oper {
	case "=", "~=", "*=", "^=", "$=", "|=":
		return AttrSelector{Name: strings.ToLower(name), Op: oper, Value: val}, true
	default:
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
}

// findAttrOperator locates the first operator in an attribute selector's
// inner text and returns its index and operator ("" when absent).
func findAttrOperator(inner string) (int, string) {
	for _, cand := range []string{"~=", "*=", "^=", "$=", "|="} {
		if i := strings.Index(inner, cand); i > 0 {
			return i, cand
		}
	}

	if i := strings.IndexByte(inner, '='); i >= 0 {
		return i, "="
	}

	return -1, ""
}

// stripAttrQuotes removes surrounding matching quotes from an attribute value.
func stripAttrQuotes(val string) string {
	if len(val) >= minQuotedLen {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}

	return val
}

// validIdent reports whether sel is a valid CSS identifier (letters, digits,
// '-', '_', and digits after the first character).
func validIdent(sel string) bool {
	if sel == "" {
		return false
	}

	for i := range len(sel) {
		if !isIdentChar(sel[i]) {
			return false
		}
	}

	if sel[0] >= '0' && sel[0] <= '9' {
		return false
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
func matchPart(part SelectorPart, node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	// ::before/::after never match the host element (declarations apply to
	// generated pseudo boxes via MatchPseudo).
	if part.PseudoElement != "" {
		return false
	}

	if part.Tag != "*" && !strings.EqualFold(part.Tag, node.Name) {
		return false
	}

	if part.ID != "" && node.Attribute("id") != part.ID {
		return false
	}

	if !hasClasses(part, node) {
		return false
	}

	if !matchAttrs(part, node) {
		return false
	}

	return matchPseudos(part.Pseudos, node)
}

// matchPseudos reports whether every pseudo-class of the part matches node.
func matchPseudos(pseudos []PseudoClass, node *html.Node) bool {
	for _, pseudo := range pseudos {
		if !matchPseudo(pseudo, node) {
			return false
		}
	}

	return true
}

// hasClasses reports whether the element carries every class of the part.
func hasClasses(part SelectorPart, node *html.Node) bool {
	if len(part.Classes) == 0 {
		return true
	}

	classes := classSet(node)
	for _, c := range part.Classes {
		if !classes[c] {
			return false
		}
	}

	return true
}

// matchAttrs matches every attribute selector of the part against node.
func matchAttrs(part SelectorPart, node *html.Node) bool {
	for _, arg := range part.Attrs {
		val, found := "", false
		if node.Attrs != nil {
			val, found = node.Attrs[arg.Name]
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

		if !attrValueMatches(arg.Op, val, arg.Value) {
			return false
		}
	}

	return true
}

// attrValueMatches evaluates one attribute operator against a value.
func attrValueMatches(oper, val, want string) bool {
	switch oper {
	case "=":
		return val == want
	case "~=":
		return containsWord(val, want)
	case "*=", "^=", "$=", "|=":
		if want == "" {
			return false
		}

		switch oper {
		case "*=":
			return strings.Contains(val, want)
		case "^=":
			return strings.HasPrefix(val, want)
		case "$=":
			return strings.HasSuffix(val, want)
		}
		// |= : exact match or value followed by a hyphen (HTML lang / BCP47-style).
		return val == want || strings.HasPrefix(val, want+"-")
	}

	return false
}

// containsWord reports whether want (a single space-free word) is one of the
// space-separated words of val.
func containsWord(val, want string) bool {
	if want == "" || strings.Contains(want, " ") {
		return false
	}

	for _, w := range strings.Fields(val) {
		if w == want {
			return true
		}
	}

	return false
}

func matchPseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case "first-child":
		return previousElementSibling(node) == nil
	case "last-child":
		return nextElementSibling(node) == nil
	case "nth-child":
		idx := elementIndex(node)

		return matchNth(pseudo.Arg, idx)
	case pseudoClassHas:
		return matchAnyRelative(pseudo.Has, node)
	case condKindNot:
		return matchNone(pseudo.Not, node)
	case "link", "visited":
		// Print: no link history — both match any anchor with an href.
		return isLinkAnchor(node)
	case "root":
		return isRootElement(node)
	case "hover", "active", "focus", "target":
		return false
	default:
		// Unknown pseudo-classes never match in print (kept on the compound
		// so selectors do not degrade to the host).
		return false
	}
}

// matchAnyRelative reports whether any relative selector applies to node.
func matchAnyRelative(rels []RelativeSelector, node *html.Node) bool {
	for _, rs := range rels {
		if matchRelative(rs, node) {
			return true
		}
	}

	return false
}

// matchNone reports whether no selector of the list matches node.
func matchNone(sels []Selector, node *html.Node) bool {
	for _, sel := range sels {
		if Match(sel, node) {
			return false
		}
	}

	return true
}

// isRootElement reports whether node is the document element (html in HTML).
// html.Parse wraps the tree in a synthetic ElementNode named "#document" —
// that must not match, and must not block <html> from matching.
func isRootElement(node *html.Node) bool {
	if node.Type != html.ElementNode || node.Name == "#document" || strings.HasPrefix(node.Name, "#") {
		return false
	}

	for p := node.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Name != "#document" && !strings.HasPrefix(p.Name, "#") {
			return false
		}
	}

	return true
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
	// an+b / n+b / -n+b / an
	if !strings.Contains(arg, "n") {
		return false
	}

	specA, buf, ok := parseAnPlusB(arg)
	if !ok {
		return false
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

// parseAnPlusB parses the "an+b" form of a nth-child argument: "an", "n+b",
// "-n+b", or "b". Returns ok=false for anything unrecognized.
func parseAnPlusB(arg string) (int, int, bool) {
	parts := strings.SplitN(arg, "n", anPlusBSplitParts)
	asVal := strings.TrimSpace(parts[0])

	specA := 1
	if asVal == "-" {
		specA = -1
	} else if asVal != "" && asVal != "+" {
		parsed, err := strconv.Atoi(asVal)
		if err != nil {
			return 0, 0, false
		}

		specA = parsed
	}

	if len(parts) != anPlusBSplitParts || strings.TrimSpace(parts[1]) == "" {
		return specA, 0, true
	}

	specB, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}

	return specA, specB, true
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
func Specificity(s Selector) (int, int, int) {
	idCount, classCount, typeCount := 0, 0, 0

	for _, page := range s.Parts {
		if page.ID != "" {
			idCount++
		}

		classCount += len(page.Classes) + len(page.Attrs)

		if page.Tag != "*" {
			typeCount++
		}

		if page.PseudoElement != "" {
			typeCount++ // pseudo-elements count like type selectors
		}

		for _, pageSize := range page.Pseudos {
			switch pageSize.Name {
			case "has":
				a2, b2, c2 := maxRelativeSpecificity(pageSize.Has)
				idCount += a2
				classCount += b2
				typeCount += c2
			case "not":
				a2, b2, c2 := maxSelectorSpecificity(pageSize.Not)
				idCount += a2
				classCount += b2
				typeCount += c2
			default:
				classCount++
			}
		}
	}

	return idCount, classCount, typeCount
}

// ParseInline parses a style="" attribute value into declarations.
func ParseInline(style string) []Declaration {
	return parseDeclarations(style)
}

// parseDeclarations splits a declaration block on top-level ';' and parses
// each "prop: value" pair. Garbage pairs are skipped.
func parseDeclarations(block string) []Declaration {
	parts := splitTopLevel(block, ';')
	decls := make([]Declaration, 0, len(parts))

	for _, part := range parts {
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
func ParseLength(val string) (float64, string, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, "", false
	}

	idx := scanLengthNumber(val)
	if idx == 0 {
		return 0, "", false
	}

	num, err := strconv.ParseFloat(val[:idx], 64)
	if err != nil {
		return 0, "", false
	}

	unit := strings.ToLower(strings.TrimSpace(val[idx:]))
	if unit == "" {
		unit = "px"
	}

	if !isLengthUnit(unit) {
		return 0, "", false
	}

	return num, unit, true
}

// scanLengthNumber returns the index of the end of the numeric prefix of val
// (optional sign, digits and one '.', per CSS lengths).
func scanLengthNumber(val string) int {
	idx := 0
	if val[0] == '+' || val[0] == '-' {
		idx++
	}

	for idx < len(val) && (val[idx] >= '0' && val[idx] <= '9' || val[idx] == '.') {
		idx++
	}

	return idx
}

// isLengthUnit reports whether unit is one of the CSS units ParseLength
// accepts.
func isLengthUnit(unit string) bool {
	switch unit {
	case "px", "pt", "pc", "in", "cm", "mm", "em", "rem", "ex", "ch", "%", "vw", "vh":
		return true
	}

	return false
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
func ParseColor(val string) (int, int, int, float64, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, 0, 0, 0, false
	}
	// CSS variables: var(--name, fallback) — resolve fallback only (no custom props).
	// ponytail: ParseColor accepts bare var() without a prop map (API is color-
	// string only). Layout resolves custom props via ResolveCustomProps before
	// color parse; upgrade if ParseColor gains a props argument.
	if strings.HasPrefix(strings.ToLower(val), "var(") {
		if fb, okFB := cssVarFallback(val); okFB {
			return ParseColor(fb)
		}

		return 0, 0, 0, 0, false
	}

	if val[0] == '#' {
		return parseHexColor(val[1:])
	}

	low := strings.ToLower(val)
	if low == "transparent" {
		return 0, 0, 0, 0, true
	}

	if name, found := namedColors()[low]; found {
		return name[0], name[1], name[2], 1, true
	}

	if strings.HasPrefix(low, "rgb") {
		return parseRGBColor(val, low)
	}

	return 0, 0, 0, 0, false
}

// parseHexColor parses #rgb, #rgba, #rrggbb and #rrggbbaa forms (hex is the
// content after '#').
func parseHexColor(hex string) (int, int, int, float64, bool) {
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

// parseRGBColor parses rgb()/rgba() with integer, float or percentage
// channels (low is the lower-cased original).
func parseRGBColor(val, low string) (int, int, int, float64, bool) {
	open := strings.IndexByte(val, '(')
	closeIdx := strings.LastIndexByte(val, ')')

	if open < 0 || closeIdx < open {
		return 0, 0, 0, 0, false
	}

	args := strings.Split(val[open+1:closeIdx], ",")

	channels := hexRGBLen
	if strings.HasPrefix(low, "rgba") {
		channels = rgbaChannelCount
	}

	if len(args) != channels {
		return 0, 0, 0, 0, false
	}

	vals, valid := parseColorChannels(args)
	if !valid {
		return 0, 0, 0, 0, false
	}

	red := clampByte(vals[0])
	green := clampByte(vals[1])
	blue := clampByte(vals[2])

	if channels != rgbaChannelCount {
		return red, green, blue, 1, true
	}

	alpha := vals[3]
	if strings.HasSuffix(strings.TrimSpace(args[3]), "%") {
		alpha /= maxRGBChannel
	}

	return red, green, blue, clampAlpha(alpha), true
}

// parseColorChannels parses comma-separated rgb()/rgba() channels, where
// percentages scale to 0..255.
func parseColorChannels(args []string) ([]float64, bool) {
	vals := make([]float64, 0, len(args))

	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if strings.HasSuffix(arg, "%") {
			f, err := strconv.ParseFloat(strings.TrimSuffix(arg, "%"), 64)
			if err != nil {
				return nil, false
			}

			vals = append(vals, f*maxRGBChannel/percentScale)

			continue
		}

		f, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return nil, false
		}

		vals = append(vals, f)
	}

	return vals, true
}

// clampAlpha clamps an alpha value to 0..1.
func clampAlpha(alpha float64) float64 {
	if alpha > 1 {
		return 1
	}

	if alpha < 0 {
		return 0
	}

	return alpha
}

// cssVarFallback extracts the fallback from var(--name, fallback). Nested
// var() in the fallback is not expanded further here.
func cssVarFallback(v string) (string, bool) {
	_, fb, ok := parseVarFn(v)

	return fb, ok && fb != ""
}

// parseVarFn parses a top-level var(--name) or var(--name, fallback).
// ok is false when v is not a var() function.
func parseVarFn(val string) (string, string, bool) {
	val = strings.TrimSpace(val)
	if len(val) < 6 || !strings.EqualFold(val[:4], "var(") {
		return "", "", false
	}

	inner := val[4:]
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
				name := strings.ToLower(strings.TrimSpace(inner[:idx]))
				fallback := strings.TrimSpace(inner[idx+1:])

				return name, fallback, name != ""
			}
		}
	}

	name := strings.ToLower(strings.TrimSpace(inner))

	return name, "", name != ""
}

// ResolveVar expands CSS var() references in v using lookup(--name).
// Unresolved var() uses the CSS fallback when present; otherwise the empty
// string (caller treats as invalid / keeps the prior cascaded value).
// Nested var() expands up to a small depth.
func ResolveVar(val2 string, lookup func(name string) (string, bool)) string {
	val2 = strings.TrimSpace(val2)
	for range 16 {
		if !strings.HasPrefix(strings.ToLower(val2), "var(") {
			return val2
		}

		name, fallback, ok := parseVarFn(val2)
		if !ok {
			return val2
		}

		if lookup != nil {
			if val, found := lookup(name); found && strings.TrimSpace(val) != "" {
				val2 = strings.TrimSpace(val)

				continue
			}
		}

		if fallback != "" {
			val2 = fallback

			continue
		}

		return ""
	}

	return val2
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

// namedColors returns the CSS2 system colors plus greys/orange and common web
// names used by fixtures and layout tests (ponytail: not the full CSS Color 4
// list). Function-scoped to keep the table out of package globals; the map is
// small enough that the per-call allocation is negligible.
func namedColors() map[string][3]int {
	return map[string][3]int{
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
}
