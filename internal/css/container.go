package css

import (
	"strconv"
	"strings"
)

const (
	condKindFeat   = "feat"
	condKindAnd    = "and"
	condKindOr     = "or"
	condKindNot    = "not"
	featInlineSize = "inline-size"
	featWidth      = "width"
	unitRem        = "rem"
)

// ContainerQuery is the prelude of an @container rule: optional name plus a
// size condition (CSS Conditional 5 subset).
type ContainerQuery struct {
	Name string // empty = unnamed (nearest size container)
	Cond ContainerCond
}

// ContainerCond is a boolean tree of size features (and / or / not).
//
// ponytail: full Cond tree, simplify if no nested and/or in real sheets
// (fixture-42 only needs single feat comparisons; layout tests still cover
// and/or/not).
type ContainerCond struct {
	Kind string // "feat", "and", "or", "not"
	Feat *SizeFeature
	Kids []ContainerCond
}

// SizeFeature is one container size query feature comparison.
// Name is "width" or "inline-size". Op is "<", ">", "<=", ">=", or "=".
// For min-/max- forms, Op is ">=" or "<=" respectively.
type SizeFeature struct {
	Name  string
	Op    string
	Value float64
	Unit  string
}

// Matches reports whether cond holds for a container whose content-box
// inline size is inlineSizePt and whose used font-size is fontSizePt (for em).
func (c ContainerCond) Matches(inlineSizePt, fontSizePt float64) bool {
	switch c.Kind {
	case condKindFeat:
		return c.Feat != nil && c.Feat.matches(inlineSizePt, fontSizePt)
	case condKindNot:
		return len(c.Kids) > 0 && !c.Kids[0].Matches(inlineSizePt, fontSizePt)
	case condKindAnd:
		return matchAllKids(c.Kids, inlineSizePt, fontSizePt)
	case condKindOr:
		return matchAnyKid(c.Kids, inlineSizePt, fontSizePt)
	default:
		return false
	}
}

// matchAllKids reports whether every kid matches (false when there are none).
func matchAllKids(kids []ContainerCond, inlineSizePt, fontSizePt float64) bool {
	for _, k := range kids {
		if !k.Matches(inlineSizePt, fontSizePt) {
			return false
		}
	}

	return len(kids) > 0
}

// matchAnyKid reports whether any kid matches.
func matchAnyKid(kids []ContainerCond, inlineSizePt, fontSizePt float64) bool {
	for _, k := range kids {
		if k.Matches(inlineSizePt, fontSizePt) {
			return true
		}
	}

	return false
}

func (f SizeFeature) matches(inlineSizePt, fontSizePt float64) bool {
	switch f.Name {
	case featWidth, featInlineSize:
		return f.matchesAxis(inlineSizePt, fontSizePt)
	default:
		return false
	}
}

// CSS length conversion constants (96 CSS px/in, 72 pt/in).
const (
	pxToPt         = 0.75 // 72/96
	pointsPerInch  = 72.0
	cmPerInch      = 2.54
	mmPerInch      = 25.4
	pointsPerPica  = 12.0
	rootFontSizePx = 16.0 // CSS initial font-size for rem
	exChToEmFactor = 0.5  // approximate ex/ch as half an em
)

// LengthToPt converts a parsed length to points. basePt is the em/rem base
// (the element's font size). Same conversions as the former internal helper;
// % and viewport units are unsupported (return false). Unknown units return
// false so callers can apply their own policy (e.g. line-height inherits).
//
// Physical units use multiply-then-divide (val * 72 / 25.4) so values like
// 25.4mm cancel cleanly to 72pt in IEEE float arithmetic. Precomputing
// (72/25.4) and multiplying loses that cancellation (25.4mm → 71.999…).
func LengthToPt(val float64, unit string, basePt float64) (float64, bool) {
	low := strings.ToLower(unit)
	if pt, ok := absoluteLengthToPt(val, low); ok {
		return pt, true
	}

	switch low {
	case "em":
		return val * basePt, true
	case unitRem:
		return val * rootFontSizePx * pxToPt, true
	case "ex", "ch":
		return val * basePt * exChToEmFactor, true
	default:
		return 0, false
	}
}

// absoluteLengthToPt converts non-relative CSS length units to points.
func absoluteLengthToPt(val float64, unit string) (float64, bool) {
	switch unit {
	case "px":
		return val * pxToPt, true
	case "pt":
		return val, true
	case "in":
		return val * pointsPerInch, true
	case "cm":
		return val * pointsPerInch / cmPerInch, true
	case "mm":
		return val * pointsPerInch / mmPerInch, true
	case "pc":
		return val * pointsPerPica, true
	default:
		return 0, false
	}
}

// ParseContainerNameValue parses container-name: none | <custom-ident>+.
// Returns the space-joined names (empty for none / invalid).
// ponytail: space-joined string is the wire form layout re-splits in two
// places; upgrade to typed ContainerNames with Matches(name) when those
// call sites change together.
func ParseContainerNameValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return ""
	}

	fields := strings.Fields(value)

	names := make([]string, 0, len(fields))

	for _, tok := range fields {
		low := strings.ToLower(tok)
		if low == "none" || low == condKindAnd || low == condKindOr || low == condKindNot || low == "default" {
			continue
		}

		names = append(names, low)
	}

	return strings.Join(names, " ")
}

// ParseContainerShorthand parses container: <name>+ [ / <type> ]?.
// Returns name string and type ("", "normal", "size", or "inline-size").
func ParseContainerShorthand(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	namePart, typePart, hasSlash := strings.Cut(value, "/")
	name := ParseContainerNameValue(strings.TrimSpace(namePart))

	ctype := ""

	if hasSlash {
		t := strings.ToLower(strings.TrimSpace(typePart))
		switch t {
		case "normal", "size", featInlineSize:
			ctype = t
		}
	}

	return name, ctype
}

// parseContainerPrelude parses the text between @container and `{`.
func parseContainerPrelude(prelude string) (ContainerQuery, bool) {
	prelude = strings.TrimSpace(prelude)
	if prelude == "" {
		return ContainerQuery{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// Optional name: leading ident that is not not/and/or and not starting with '('.
	name := ""

	rest := prelude
	if !strings.HasPrefix(rest, "(") && !strings.HasPrefix(strings.ToLower(rest), condKindNot) {
		ident, rem, ok := readIdent(rest)
		if ok {
			low := strings.ToLower(ident)
			if low != condKindAnd && low != condKindOr && low != condKindNot && low != "none" {
				name = low
				rest = strings.TrimSpace(rem)
			}
		}
	}

	cond, ok := parseContainerCond(rest)
	if !ok {
		return ContainerQuery{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return ContainerQuery{Name: name, Cond: cond}, true
}

func readIdent(str string) (string, string, bool) {
	str = strings.TrimLeft(str, " \t\r\n")
	if str == "" {
		return "", str, false
	}

	end := identEnd(str)

	if end == 0 {
		return "", str, false
	}

	return str[:end], str[end:], true
}

// identEnd returns the length of the leading CSS identifier in str.
func identEnd(str string) int {
	idx := 0
	for idx < len(str) {
		if idx > 0 && str[idx] >= '0' && str[idx] <= '9' {
			idx++

			continue
		}

		if !isIdentStart(str[idx]) {
			break
		}

		idx++
	}

	return idx
}

// isIdentStart reports whether c can begin a CSS identifier.
func isIdentStart(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// parseContainerCond parses a container condition with or < and < not precedence.
func parseContainerCond(str string) (ContainerCond, bool) {
	str = strings.TrimSpace(str)
	if str == "" {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return parseOrCond(str)
}

func parseOrCond(s string) (ContainerCond, bool) {
	parts, ok := splitCondKeyword(s, "or")
	if !ok {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	if len(parts) == 1 {
		return parseAndCond(parts[0])
	}

	kids := make([]ContainerCond, 0, len(parts))

	for _, p := range parts {
		c, ok := parseAndCond(p)
		if !ok {
			return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		kids = append(kids, c)
	}

	return ContainerCond{Kind: "or", Kids: kids}, true //nolint:exhaustruct // intentional zero-value fields
}

func parseAndCond(s string) (ContainerCond, bool) {
	parts, ok := splitCondKeyword(s, condKindAnd)
	if !ok {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	if len(parts) == 1 {
		return parseNotCond(parts[0])
	}

	kids := make([]ContainerCond, 0, len(parts))

	for _, p := range parts {
		c, ok := parseNotCond(p)
		if !ok {
			return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		kids = append(kids, c)
	}

	return ContainerCond{Kind: condKindAnd, Kids: kids}, true //nolint:exhaustruct // intentional zero-value fields
}

func parseNotCond(str string) (ContainerCond, bool) {
	str = strings.TrimSpace(str)
	low := strings.ToLower(str)

	if strings.HasPrefix(low, condKindNot) {
		rest := strings.TrimSpace(str[3:])
		if rest == "" {
			return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
		}
		// "not (...)" or "not <cond>"
		inner, ok := parseNotCond(rest)
		if !ok {
			return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		kids := []ContainerCond{inner}
		cond := ContainerCond{Kind: condKindNot, Kids: kids} //nolint:exhaustruct // intentional zero-value fields

		return cond, true
	}

	return parseParenOrFeat(str)
}

func parseParenOrFeat(str string) (ContainerCond, bool) {
	str = strings.TrimSpace(str)
	if !strings.HasPrefix(str, "(") {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	found, rest, okVal := takeParen(str)
	if !okVal || strings.TrimSpace(rest) != "" {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	found = strings.TrimSpace(found)
	// Nested condition vs feature: if it looks like and/or/not/(, recurse.
	low := strings.ToLower(found)
	if strings.HasPrefix(found, "(") || strings.HasPrefix(low, condKindNot) ||
		containsTopLevelKeyword(found, condKindAnd) || containsTopLevelKeyword(found, condKindOr) {
		return parseContainerCond(found)
	}

	feat, okVal := parseSizeFeature(found)
	if !okVal {
		return ContainerCond{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return ContainerCond{Kind: condKindFeat, Feat: &feat}, true //nolint:exhaustruct // intentional zero-value fields
}

// splitCondKeyword splits on top-level `and`/`or` keywords (not inside parens).
func splitCondKeyword(str, keyword string) ([]string, bool) {
	str = strings.TrimSpace(str)

	var parts []string

	depth := 0
	start := 0
	low := strings.ToLower(str)

	for idx := 0; idx < len(str); {
		c := str[idx]
		switch c {
		case '"', '\'':
			idx = skipQuoted(str, idx, c)

			continue
		case '(':
			depth++
			idx++

			continue
		case ')':
			depth--
			idx++

			continue
		}

		if depth == 0 && hasKeywordAt(low, idx, keyword) {
			parts = append(parts, strings.TrimSpace(str[start:idx]))
			idx += len(keyword)
			start = idx

			continue
		}

		idx++
	}

	parts = append(parts, strings.TrimSpace(str[start:]))
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
	}

	return parts, true
}

// skipQuoted advances idx past the quoted string starting at value[idx] == q,
// honoring backslash escapes, and returns the index just past the closing
// quote (len(value) when unbalanced).
func skipQuoted(value string, idx int, q byte) int {
	idx++
	for idx < len(value) && value[idx] != q {
		if value[idx] == '\\' && idx+1 < len(value) {
			idx++
		}

		idx++
	}

	if idx < len(value) {
		idx++
	}

	return idx
}

func hasKeywordAt(low string, idx int, kwVal string) bool {
	if idx+len(kwVal) > len(low) {
		return false
	}

	if low[idx:idx+len(kwVal)] != kwVal {
		return false
	}
	// word boundaries
	if idx > 0 {
		prev := low[idx-1]
		if isIdentChar(prev) {
			return false
		}
	}

	if idx+len(kwVal) < len(low) {
		next := low[idx+len(kwVal)]
		if isIdentChar(next) {
			return false
		}
	}

	return true
}

func isIdentChar(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func containsTopLevelKeyword(s, kw string) bool {
	parts, ok := splitCondKeyword(s, kw)

	return ok && len(parts) > 1
}

// parseSizeFeature parses the inside of (...): range or min-/max- forms.
func parseSizeFeature(inner string) (SizeFeature, bool) {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// min-width: 400px / max-inline-size: 20em
	if colon := strings.IndexByte(inner, ':'); colon >= 0 {
		name := strings.ToLower(strings.TrimSpace(inner[:colon]))
		valStr := strings.TrimSpace(inner[colon+1:])

		val, unit, ok := ParseLength(valStr)
		if !ok {
			return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		switch name {
		case "min-width", "min-inline-size":
			feat := featWidth
			if strings.HasSuffix(name, featInlineSize) {
				feat = featInlineSize
			}

			return SizeFeature{Name: feat, Op: ">=", Value: val, Unit: unit}, true
		case "max-width", "max-inline-size":
			feat := featWidth
			if strings.HasSuffix(name, featInlineSize) {
				feat = featInlineSize
			}

			return SizeFeature{Name: feat, Op: "<=", Value: val, Unit: unit}, true
		case featWidth, featInlineSize:
			return SizeFeature{Name: name, Op: "=", Value: val, Unit: unit}, true
		default:
			return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
		}
	}
	// Range: inline-size > 20em  |  20em < inline-size < 40em (single compare only)
	return parseRangeFeature(inner)
}

// rangeTok is one token of a range-feature comparison.
type rangeTok struct {
	kind string // "ident", "num", "op"
	val  string
	num  float64
	unit string
}

func parseRangeFeature(inner string) (SizeFeature, bool) {
	toks, ok := tokenizeRangeFeature(inner)
	if !ok {
		return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	if len(toks) != 3 || toks[1].kind != "op" {
		return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return rangeFeatureFromTokens(toks)
}

// tokenizeRangeFeature splits a range-feature body into ident/num/op tokens.
func tokenizeRangeFeature(inner string) ([]rangeTok, bool) {
	var toks []rangeTok

	str := strings.TrimSpace(inner)
	for str != "" {
		str = strings.TrimLeft(str, " \t\r\n")
		if str == "" {
			break
		}

		if op, ok := rangeOpPrefix(str); ok {
			toks = append(toks, rangeTok{kind: "op", val: op}) //nolint:exhaustruct // intentional zero-value fields
			str = str[len(op):]

			continue
		}

		if isRangeNumberStart(str[0]) {
			tok, rem, ok := scanRangeLength(str)
			if !ok {
				return nil, false
			}

			toks = append(toks, tok)
			str = rem

			continue
		}

		ident, rem, ok := readIdent(str)
		if !ok {
			return nil, false
		}

		tok := rangeTok{kind: "ident", val: strings.ToLower(ident)} //nolint:exhaustruct // intentional zero-value fields

		toks = append(toks, tok)
		str = rem
	}

	return toks, true
}

// rangeOpPrefix returns the comparison operator at the start of str, if any.
func rangeOpPrefix(str string) (string, bool) {
	if strings.HasPrefix(str, "<=") || strings.HasPrefix(str, ">=") {
		return str[:2], true
	}

	if str[0] == '<' || str[0] == '>' || str[0] == '=' {
		return str[:1], true
	}

	return "", false
}

// isRangeNumberStart reports whether c can begin a numeric length token.
func isRangeNumberStart(c byte) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == '+' || c == '-'
}

// scanRangeLength parses a number-with-optional-unit token at the start of
// str (which begins a numeric token). Returns the token and the remainder.
func scanRangeLength(str string) (rangeTok, string, bool) {
	idx := numberEnd(str)
	jdx := unitEnd(str, idx)

	val, unit, ok := ParseLength(str[:jdx])
	if !ok {
		// bare number?
		if n, err := strconv.ParseFloat(str[:idx], 64); err == nil && idx == jdx {
			return rangeTok{kind: "num", num: n, unit: "px", val: str[:idx]}, str[idx:], true
		}

		return rangeTok{}, str, false //nolint:exhaustruct // intentional zero-value fields
	}

	return rangeTok{kind: "num", num: val, unit: unit, val: str[:jdx]}, str[jdx:], true
}

// numberEnd returns the length of the leading numeric part of str.
func numberEnd(str string) int {
	idx := 0
	if str[0] == '+' || str[0] == '-' {
		idx++
	}

	for idx < len(str) && (str[idx] >= '0' && str[idx] <= '9' || str[idx] == '.') {
		idx++
	}

	return idx
}

// unitEnd returns the length of the leading unit part of str starting at idx.
func unitEnd(str string, idx int) int {
	for idx < len(str) && isRangeUnitChar(str[idx]) {
		idx++
	}

	return idx
}

// isRangeUnitChar reports whether c may continue a length unit token.
func isRangeUnitChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '%'
}

// rangeFeatureFromTokens matches the two supported 3-token range forms:
// name op num  OR  num op name (flipped to name op num).
func rangeFeatureFromTokens(toks []rangeTok) (SizeFeature, bool) {
	if toks[0].kind == "ident" && toks[2].kind == "num" {
		name := toks[0].val
		if name == featWidth || name == featInlineSize {
			return SizeFeature{Name: name, Op: toks[1].val, Value: toks[2].num, Unit: toks[2].unit}, true
		}
	}

	if toks[0].kind == "num" && toks[2].kind == "ident" {
		name := toks[2].val
		if name == featWidth || name == featInlineSize {
			// flip: 20em < width  →  width > 20em
			op := flipOp(toks[1].val)
			if op != "" {
				return SizeFeature{Name: name, Op: op, Value: toks[0].num, Unit: toks[0].unit}, true
			}
		}
	}

	return SizeFeature{}, false //nolint:exhaustruct // intentional zero-value fields
}

func flipOp(op string) string {
	switch op {
	case "<":
		return ">"
	case ">":
		return "<"
	case "<=":
		return ">="
	case ">=":
		return "<="
	case "=":
		return "="
	default:
		return ""
	}
}

// HasContainerRules reports whether any rule in the sheets carries a
// container query (used to skip the second style pass).
func HasContainerRules(sheets []*Stylesheet) bool {
	for _, s := range sheets {
		if s == nil {
			continue
		}

		for _, r := range s.Rules {
			if r.Container != nil {
				return true
			}
		}
	}

	return false
}
