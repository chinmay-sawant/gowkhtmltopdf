// Package css implements the CSS subset gowkhtmltopdf accepts: a
// declarations-and-rules parser, selector matching against the html tree,
// specificity ordering, and value helpers (lengths, colors, font families).
//
// Scope: `*`, type, `.class`, `#id`, attribute selectors (`[attr]`, `=`, `~=`,
// `*=`, `^=`, `$=`, `|=`, ASCII `i` flag),
// :first-child/:last-child/:nth-child/:first-of-type/:last-of-type/
// :nth-of-type/:nth-last-of-type/:has()/:not()/:is()/:where(),
// descendant/child/sibling combinators, `@media` type + size-feature matching
// (see MediaMatches),
// `@container` size queries (inline-size/width + and/or/not), `!important`,
// inline style attributes. Unsupported constructs degrade without panicking.
package css

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
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
	nonASCIIStart     = 0x80
	hslChannelCount   = 3
	hslCircleDeg      = 360
	hslSectorDeg      = 60
	hslEvenPeriod     = 2 // H' mod 2 in the second-largest HSL component
	hslChromaHalf     = 2 // m = L - C/2
	hslSectorYG       = 2 // yellow-green, H' in [1, 2)
	hslSectorGC       = 3 // green-cyan
	hslSectorCB       = 4 // cyan-blue
	hslSectorBM       = 5 // blue-magenta
)

// Pseudo-class and pseudo-element names shared across selector parsing.
const (
	pseudoClassHas      = "has"
	pseudoClassIs       = "is"
	pseudoClassWhere    = "where"
	pseudoElemBefore    = "before"
	pseudoElemAfter     = "after"
	nthChildPseudo      = "nth-child"
	firstChildPseudo    = "first-child"
	lastChildPseudo     = "last-child"
	nthOfTypePseudo     = "nth-of-type"
	nthLastOfTypePseudo = "nth-last-of-type"
	firstOfTypePseudo   = "first-of-type"
	lastOfTypePseudo    = "last-of-type"
)

// Stylesheet is a parsed stylesheet. Rules keep their source order.
type Stylesheet struct {
	Rules     []Rule
	FontFaces []FontFace
	Page      *PageStyle
	// Pages holds every @page rule in source order, including :first/:left/:right
	// and named pages. Page remains the last unnamed @page for older callers.
	Pages []PageRule
	// Imports are @import url/media pairs in source order. Parse fills this;
	// convert.prepare fetches each under the same ACL as <link rel=stylesheet>.
	Imports []ImportRule
}

// ImportRule is one @import. URL is the raw url("...") or unquoted path.
// Media is the optional prelude after the URL (empty means the sheet's media).
type ImportRule struct {
	URL   string
	Media string
}

// PageStyle stores the unnamed @page declarations that affect the print
// viewport. The converter currently consumes the margin shorthand; keeping
// the raw values here lets page geometry resolve physical units at the PDF
// boundary instead of pretending they are element styles.
type PageStyle struct {
	Margin string
	Size   string
}

// PageRule is one @page block. Sel is "" (unnamed), ":first", ":left",
// ":right", or a page name ident. Boxes holds lite margin-box content
// strings (@top-center and friends); empty slots were not declared.
type PageRule struct {
	Sel    string
	Margin string
	Size   string
	Boxes  PageMarginBoxes
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
	// spec caches the specificity of parsed selectors (see Specificity).
	// The zero value means "not yet computed": Specificity falls back to a
	// walk so hand-built selectors keep working unchanged.
	spec      [3]int `exhaustruct:"optional"`
	specValid bool   `exhaustruct:"optional"`
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
// [name|=ident] (exact or prefix-plus-hyphen). IgnoreCase is the Selectors 4
// ASCII i flag on valued selectors ([attr=value i]).
type AttrSelector struct {
	Name       string
	Op         string // "", "=", "~=", "*=", "^=", "$=", "|="
	Value      string
	IgnoreCase bool
}

// RelativeSelector is a complex selector interpreted relative to a subject
// element (Selectors 4). Leading is " " (descendant), ">", "+", or "~".
type RelativeSelector struct {
	Leading string
	Parts   []SelectorPart
}

// PseudoClass is :first-child, :last-child, :nth-child(...), :nth-of-type(...),
// :has(...), :not(...), :is(...), or :where(...).
type PseudoClass struct {
	Name string // lower-case, without leading ':'
	Arg  string // nth-child / nth-of-type argument, lower-case, trimmed
	Has  []RelativeSelector
	Not  []Selector
	Is   []Selector // :is() / :where() argument list
	// nth caches the parsed :nth-child / :nth-of-type argument (see nthForm)
	// so matching is pure integer arithmetic; kind zero (unparseable) never matches.
	nth nthForm `exhaustruct:"optional"`
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
		return parsePageRule(src, str)
	case strings.HasPrefix(low, "@keyframes"), strings.HasPrefix(low, "@-webkit-keyframes"):
		// Animations are parse-ignored (static cascaded values only).
		return skipAtRule(src)
	case strings.HasPrefix(low, "@font-face"):
		return parseFontFaceRule(src, str)
	case strings.HasPrefix(low, "@import"):
		return parseImportRule(src, str)
	default:
		return skipAtRule(src)
	}
}

func parsePageRule(src string, str *Stylesheet) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return skipAtRule(src)
	}

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	boxes := extractPageMarginBoxes(block)
	block = stripNestedAtRules(block)
	sel := parsePageSelector(src[len("@page"):open])

	var margin, size string

	for _, decl := range parseDeclarations(block) {
		switch strings.ToLower(decl.Prop) {
		case "margin":
			margin = decl.Value
		case "size":
			size = decl.Value
		}
	}

	str.Pages = append(str.Pages, PageRule{
		Sel:    sel,
		Margin: margin,
		Size:   size,
		Boxes:  boxes,
	})

	if sel == "" {
		page := str.Page
		if page == nil {
			page = &PageStyle{} //nolint:exhaustruct // zero values represent omitted @page properties
			str.Page = page
		}

		if margin != "" {
			page.Margin = margin
		}

		if size != "" {
			page.Size = size
		}
	}

	return rest, nil
}

// parsePageSelector reads the @page prelude. Empty is unnamed; :first, :left,
// and :right are the page pseudos (any ASCII case); any other ident is a named
// page. Nested @margin-box rules live in the block and are not part of Sel.
func parsePageSelector(prelude string) string {
	prelude = strings.TrimSpace(prelude)
	if prelude == "" {
		return ""
	}

	if prelude[0] == ':' {
		ident := strings.ToLower(pageIdent(strings.TrimSpace(prelude[1:])))
		switch ident {
		case "first", "left", "right":
			return ":" + ident
		}

		if ident != "" {
			return ":" + ident
		}

		return ""
	}

	return pageIdent(prelude)
}

// IsIdentToken reports that s is a single CSS ident with no leftover tokens.
func IsIdentToken(s string) bool {
	s = strings.TrimSpace(s)

	return s != "" && pageIdent(s) == s
}

// pageIdent returns the leading CSS ident in src, or "" if src does not start with one.
func pageIdent(src string) string {
	if src == "" || !isIdentStart(src[0]) {
		return ""
	}

	end := 1
	for end < len(src) && isIdentChar(src[end]) {
		end++
	}

	return src[:end]
}

// stripNestedAtRules drops at-rules inside a declaration block (margin boxes
// such as @top-center) so following descriptors are still parsed.
func stripNestedAtRules(block string) string {
	var out strings.Builder

	out.Grow(len(block))

	for idx := 0; idx < len(block); {
		char := block[idx]
		if char == '"' || char == '\'' {
			relEnd := strings.IndexByte(block[idx+1:], char)
			if relEnd < 0 {
				out.WriteString(block[idx:])

				return out.String()
			}

			end := idx + minQuotedLen + relEnd
			out.WriteString(block[idx:end])
			idx = end

			continue
		}

		if char == '@' {
			rest, err := skipAtRule(block[idx:])
			if err != nil {
				return out.String()
			}

			idx += len(block[idx:]) - len(rest)

			continue
		}

		out.WriteByte(char)

		idx++
	}

	return out.String()
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

// skipAtRule consumes one at-rule from src. A ';' before the first '{' is a
// statement at-rule (@charset, malformed @import); otherwise a braced block
// is consumed. Returns the remaining source.
func skipAtRule(src string) (string, error) {
	semi := strings.IndexByte(src, ';')
	open := strings.IndexByte(src, '{')

	if open >= 0 && (semi < 0 || open < semi) {
		_, rest, err := takeBlock(src, open)
		if err != nil {
			return "", err
		}

		return rest, nil
	}

	if semi >= 0 {
		return src[semi+1:], nil
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

	// Search the lowercased copy for "url(" but extract from the original
	// case: the URL itself must keep its case (file paths are case-sensitive).
	// ToLower is 1:1 in byte length for the ASCII "url(" marker, so both
	// remainders advance by the same offsets.
	low := src
	search := strings.ToLower(src)

	for {
		urlIdx := strings.Index(search, "url(")
		if urlIdx < 0 {
			break
		}

		rest := low[urlIdx+4:]
		search = search[urlIdx+4:]

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
		search = search[end+1:]
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
	if !strings.Contains(src, "/*") {
		return src
	}

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

		a, b, c := computeSpecificity(sel)
		sel.spec = [3]int{a, b, c}
		sel.specValid = true

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

	switch name {
	case pseudoClassHas, condKindNot, pseudoClassIs, pseudoClassWhere:
		return appendFunctionalPseudo(part, name, argRaw, hasParen, insideHas)
	default:
		return appendSimplePseudo(part, name, arg)
	}
}

// appendFunctionalPseudo handles :has(...), :not(...), :is(...), and :where(...).
func appendFunctionalPseudo(part SelectorPart, name, argRaw string, hasParen, insideHas bool) (SelectorPart, bool) {
	if !hasParen || strings.TrimSpace(argRaw) == "" {
		return part, false
	}

	switch name {
	case pseudoClassHas:
		return appendHasPseudo(part, argRaw)
	case condKindNot:
		return appendNotPseudo(part, argRaw, insideHas)
	case pseudoClassIs, pseudoClassWhere:
		return appendIsWherePseudo(part, name, argRaw, insideHas)
	default:
		return part, false
	}
}

func appendHasPseudo(part SelectorPart, argRaw string) (SelectorPart, bool) {
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

func appendNotPseudo(part SelectorPart, argRaw string, insideHas bool) (SelectorPart, bool) {
	sels, ok := parseSelectorListStrict(argRaw, insideHas)
	if !ok {
		return part, false
	}

	part.Pseudos = append(part.Pseudos, pseudoClass(condKindNot, "", nil, sels))

	return part, true
}

func appendIsWherePseudo(part SelectorPart, name, argRaw string, insideHas bool) (SelectorPart, bool) {
	if strings.Contains(argRaw, "::") {
		return part, false
	}

	sels, ok := parseSelectorListStrict(argRaw, insideHas)
	if !ok {
		return part, false
	}

	part.Pseudos = append(part.Pseudos, isWherePseudo(name, sels))

	return part, true
}

// appendSimplePseudo handles the non-functional pseudo-classes and the
// CSS2 single-colon pseudo-elements.
func appendSimplePseudo(part SelectorPart, name, arg string) (SelectorPart, bool) {
	switch name {
	case firstChildPseudo, lastChildPseudo, nthChildPseudo,
		firstOfTypePseudo, lastOfTypePseudo, nthOfTypePseudo, nthLastOfTypePseudo:
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

// pseudoClass builds a PseudoClass with explicit zero Has/Not/Is slices so the
// literal stays exhaustive without per-use nolint comments.
func pseudoClass(name, arg string, has []RelativeSelector, not []Selector) PseudoClass {
	pseudo := PseudoClass{Name: name, Arg: arg, Has: has, Not: not, Is: nil}

	if isNthArgPseudo(name) {
		pseudo.nth = parseNthArg(arg)
	}

	return pseudo
}

func isWherePseudo(name string, sels []Selector) PseudoClass {
	pseudo := pseudoClass(name, "", nil, nil)
	pseudo.Is = sels

	return pseudo
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
	rawVal := strings.TrimSpace(inner[nameEnd+len(oper):])
	rawVal, ignoreCase := splitAttrIFlag(rawVal)
	val := stripAttrQuotes(rawVal)

	if !validIdent(name) {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	switch oper {
	case "=", "~=", "*=", "^=", "$=", "|=":
		return AttrSelector{
			Name:       strings.ToLower(name),
			Op:         oper,
			Value:      val,
			IgnoreCase: ignoreCase,
		}, true
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

	if sel.Parts[len(sel.Parts)-1].PseudoElement != pseudo {
		return false
	}

	return matchPseudoWalk(sel, count)
}

// matchPseudoWalk mirrors leftmostMatch with the final part's PseudoElement
// treated as cleared, without copying the parts slice (the host element must
// match the pseudo's compound, not the pseudo itself).
func matchPseudoWalk(sel Selector, node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode || len(sel.Parts) == 0 {
		return false
	}

	last := sel.Parts[len(sel.Parts)-1]
	last.PseudoElement = ""

	if !matchPart(last, node) {
		return false
	}

	cur := node

	const prevPartOffset = 2 // walk left: last part is host, start at len-2

	for i := len(sel.Parts) - prevPartOffset; i >= 0; i-- {
		next := leftmostStep(sel.Parts[i+1].Combinator, sel.Parts[i], cur)
		if next == nil {
			return false
		}

		cur = next
	}

	return true
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

	for _, c := range part.Classes {
		if !hasClassToken(node.Attribute("class"), c) {
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

		if !attrValueMatches(arg.Op, val, arg.Value, arg.IgnoreCase) {
			return false
		}
	}

	return true
}

// attrValueMatches evaluates one attribute operator against a value.
// ignoreCase is the Selectors 4 ASCII i flag; comparison uses ToLower.
func attrValueMatches(oper, val, want string, ignoreCase bool) bool {
	if ignoreCase {
		val = strings.ToLower(val)
		want = strings.ToLower(want)
	}

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
// space-separated words of val. Tokenizes val without allocating a fields
// slice (same whitespace definition and Unicode fallback as hasClassToken).
//
//nolint:cyclop // two-zone token walk (ASCII, then Unicode fallback) stays linear
func containsWord(val, want string) bool {
	if want == "" || strings.Contains(want, " ") {
		return false
	}

	for start := 0; start < len(val); {
		for start < len(val) && isClassSpace(val[start]) {
			start++
		}

		end := start
		for end < len(val) && !isClassSpace(val[end]) {
			if val[end] >= nonASCIIStart {
				return hasUnicodeClassToken(val, want)
			}

			end++
		}

		if start < end && val[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func matchPseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstChildPseudo, lastChildPseudo, nthChildPseudo, "root",
		firstOfTypePseudo, lastOfTypePseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return matchTreePseudo(pseudo, node)
	case pseudoClassHas:
		return matchAnyRelative(pseudo.Has, node)
	case condKindNot:
		return matchNone(pseudo.Not, node)
	case pseudoClassIs, pseudoClassWhere:
		return matchAnySelector(pseudo.Is, node)
	case "link", "visited":
		// Print has no link history, so both match any anchor with an href.
		return isLinkAnchor(node)
	default:
		// :hover/:active/:focus/:target never match in print. Unknown
		// pseudo-classes never match either (kept on the compound so
		// selectors do not degrade to the host).
		return false
	}
}

// matchTreePseudo handles tree-structural pseudo-classes.
func matchTreePseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstChildPseudo:
		return previousElementSibling(node) == nil
	case lastChildPseudo:
		return nextElementSibling(node) == nil
	case nthChildPseudo:
		return matchNth(pseudo.nth, elementIndex(node))
	case "root":
		return isRootElement(node)
	default:
		return matchOfTypePseudo(pseudo, node)
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

// matchAnySelector reports whether any selector of the list matches node.
func matchAnySelector(sels []Selector, node *html.Node) bool {
	for _, sel := range sels {
		if Match(sel, node) {
			return true
		}
	}

	return false
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

// nthKind discriminates the pre-parsed :nth-child() argument forms.
type nthKind int

const (
	nthInvalid nthKind = iota // unparseable, or not an nth-* pseudo
	nthOdd
	nthEven
	nthInt
	nthAnB
)

// nthForm is a :nth-child() argument parsed at selector-parse time so that
// matching is pure integer arithmetic (see matchNth).
type nthForm struct {
	kind nthKind `exhaustruct:"optional"` // nthInt: exact index; nthAnB: coefficient
	a    int     `exhaustruct:"optional"` // nthAnB: the constant
	b    int     `exhaustruct:"optional"`
}

// parseNthArg pre-parses a :nth-child() argument into the form matchNth
// evaluates. The argument is already lower-cased and trimmed at parse time;
// normalizing again here keeps the acceptance rules identical to the former
// string-based parser, including the never-match fallback.
func parseNthArg(arg string) nthForm {
	arg = strings.TrimSpace(strings.ToLower(arg))
	if arg == "" {
		return nthForm{}
	}

	if arg == "odd" {
		return nthForm{kind: nthOdd}
	}

	if arg == "even" {
		return nthForm{kind: nthEven}
	}
	// plain integer
	if n, err := strconv.Atoi(arg); err == nil {
		return nthForm{kind: nthInt, a: n}
	}
	// an+b / n+b / -n+b / an
	if !strings.Contains(arg, "n") {
		return nthForm{}
	}

	a, b, ok := parseAnPlusB(arg)
	if !ok {
		return nthForm{}
	}

	return nthForm{kind: nthAnB, a: a, b: b}
}

// matchNth implements :nth-child(an+b) / odd / even for 1-based index,
// evaluating the pre-parsed argument form.
func matchNth(nth nthForm, index int) bool {
	switch nth.kind {
	case nthOdd:
		return index%2 == 1
	case nthEven:
		return index%2 == 0
	case nthInt:
		return index == nth.a
	case nthAnB:
		if nth.a == 0 {
			return index == nth.b
		}

		if (index-nth.b)%nth.a != 0 {
			return false
		}

		k := (index - nth.b) / nth.a

		return k >= 0
	case nthInvalid:
		return false
	}

	return false
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

// hasClassToken reports whether want is one whitespace-separated class token.
// The ASCII path covers HTML/CSS's normal class syntax without allocating a
// token slice or map. Non-ASCII whitespace falls back to the same Unicode
// whitespace behavior previously provided by strings.Fields.
func hasClassToken(value, want string) bool {
	if want == "" {
		return false
	}

	for start := 0; start < len(value); {
		for start < len(value) && isClassSpace(value[start]) {
			start++
		}

		end := start
		for end < len(value) && !isClassSpace(value[end]) {
			if value[end] >= nonASCIIStart {
				return hasUnicodeClassToken(value, want)
			}

			end++
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func hasUnicodeClassToken(value, want string) bool {
	for start := 0; start < len(value); {
		for start < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[start:])
			if !unicode.IsSpace(runeValue) {
				break
			}

			start += size
		}

		end := start
		for end < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[end:])
			if unicode.IsSpace(runeValue) {
				break
			}

			end += size
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func isClassSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\v' || value == '\f' || value == '\r'
}

// Specificity returns (a, b, c): ID count, class/attribute/pseudo count, type count.
// :has() / :not() / :is() contribute the specificity of their most specific
// argument (Selectors 4); :where() contributes 0. Parsed selectors return
// their cached triple; selectors built by hand (or by wrappers that recombine
// parts) compute it on the fly.
func Specificity(s Selector) (int, int, int) {
	if s.specValid {
		return s.spec[0], s.spec[1], s.spec[2]
	}

	return computeSpecificity(s)
}

// computeSpecificity walks the selector parts; see Specificity.
func computeSpecificity(s Selector) (int, int, int) {
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
			aDelta, bDelta, cDelta := pseudoSpecificityDelta(pageSize)
			idCount += aDelta
			classCount += bDelta
			typeCount += cDelta
		}
	}

	return idCount, classCount, typeCount
}
