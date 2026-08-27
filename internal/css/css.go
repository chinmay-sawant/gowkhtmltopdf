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
	"strings"
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
	hexLetterBase     = 10 // 'a'/'A' -> 10 in hex
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
	PseudoElement string // "before" | "after" | "" - never matches the host element
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

	prelude := src[len("@page"):open]

	sel := parsePageSelector(prelude)
	if strings.TrimSpace(prelude) != "" && sel == "" {
		return rest, nil
	}

	boxes, declarations := parsePageBlock(block, sel == "")
	margin, size := pageDescriptors(declarations)

	str.Pages = append(str.Pages, PageRule{
		Sel:    sel,
		Margin: margin,
		Size:   size,
		Boxes:  boxes,
	})

	if sel == "" {
		str.Page = applyPageDescriptors(str.Page, margin, size)
	}

	return rest, nil
}

func pageDescriptors(declarations string) (string, string) {
	var margin, size string

	for _, decl := range parseDeclarations(declarations) {
		switch strings.ToLower(decl.Prop) {
		case "margin":
			margin = decl.Value
		case "size":
			size = decl.Value
		}
	}

	return margin, size
}

func applyPageDescriptors(page *PageStyle, margin, size string) *PageStyle {
	if page == nil {
		page = &PageStyle{} //nolint:exhaustruct // zero values represent omitted @page properties
	}

	if margin != "" {
		page.Margin = margin
	}

	if size != "" {
		page.Size = size
	}

	return page
}

// parsePageSelector reads the @page prelude. Empty is unnamed; :first, :left,
// and :right are the page pseudos (any ASCII case); any other single ident is
// a named page. A compound, list, or unknown pseudo is invalid.
func parsePageSelector(prelude string) string {
	prelude = strings.TrimSpace(prelude)
	if prelude == "" {
		return ""
	}

	if prelude[0] == ':' {
		pseudo := strings.TrimSpace(prelude[1:])
		ident := strings.ToLower(pageIdent(pseudo))

		if ident == "" || ident != pseudo {
			return ""
		}

		switch ident {
		case "first", "left", "right":
			return ":" + ident
		}

		return ""
	}

	if !IsIdentToken(prelude) {
		return ""
	}

	return prelude
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
