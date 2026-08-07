package css

import (
	"strconv"
	"strings"
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
	case "feat":
		if c.Feat == nil {
			return false
		}
		return c.Feat.matches(inlineSizePt, fontSizePt)
	case "not":
		if len(c.Kids) == 0 {
			return false
		}
		return !c.Kids[0].Matches(inlineSizePt, fontSizePt)
	case "and":
		for _, k := range c.Kids {
			if !k.Matches(inlineSizePt, fontSizePt) {
				return false
			}
		}
		return len(c.Kids) > 0
	case "or":
		for _, k := range c.Kids {
			if k.Matches(inlineSizePt, fontSizePt) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (f SizeFeature) matches(inlineSizePt, fontSizePt float64) bool {
	switch f.Name {
	case "width", "inline-size":
		return f.matchesAxis(inlineSizePt, fontSizePt)
	default:
		return false
	}
}

// LengthToPt converts a parsed length to points. basePt is the em/rem base
// (the element's font size). Same conversions as the former internal helper;
// % and viewport units are unsupported (return false). Unknown units return
// false so callers can apply their own policy (e.g. line-height inherits).
func LengthToPt(val float64, unit string, basePt float64) (float64, bool) {
	switch strings.ToLower(unit) {
	case "px":
		return val * 0.75, true
	case "pt":
		return val, true
	case "in":
		return val * 72, true
	case "cm":
		return val * 72 / 2.54, true
	case "mm":
		return val * 72 / 25.4, true
	case "pc":
		return val * 12, true
	case "em", "rem":
		if unit == "rem" {
			return val * 16 * 0.75, true // 16px root
		}
		return val * basePt, true
	case "ex", "ch":
		return val * basePt * 0.5, true
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
	var names []string
	for _, tok := range strings.Fields(value) {
		low := strings.ToLower(tok)
		if low == "none" || low == "and" || low == "or" || low == "not" || low == "default" {
			continue
		}
		names = append(names, low)
	}
	return strings.Join(names, " ")
}

// ParseContainerShorthand parses container: <name>+ [ / <type> ]?.
// Returns name string and type ("", "normal", "size", or "inline-size").
func ParseContainerShorthand(value string) (name, ctype string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	namePart, typePart, hasSlash := strings.Cut(value, "/")
	name = ParseContainerNameValue(strings.TrimSpace(namePart))
	if hasSlash {
		t := strings.ToLower(strings.TrimSpace(typePart))
		switch t {
		case "normal", "size", "inline-size":
			ctype = t
		}
	}
	return name, ctype
}

// parseContainerPrelude parses the text between @container and `{`.
func parseContainerPrelude(prelude string) (ContainerQuery, bool) {
	prelude = strings.TrimSpace(prelude)
	if prelude == "" {
		return ContainerQuery{}, false
	}
	// Optional name: leading ident that is not not/and/or and not starting with '('.
	name := ""
	rest := prelude
	if !strings.HasPrefix(rest, "(") && !strings.HasPrefix(strings.ToLower(rest), "not") {
		ident, rem, ok := readIdent(rest)
		if ok {
			low := strings.ToLower(ident)
			if low != "and" && low != "or" && low != "not" && low != "none" {
				name = low
				rest = strings.TrimSpace(rem)
			}
		}
	}
	cond, ok := parseContainerCond(rest)
	if !ok {
		return ContainerQuery{}, false
	}
	return ContainerQuery{Name: name, Cond: cond}, true
}

func readIdent(s string) (ident, rest string, ok bool) {
	s = strings.TrimLeft(s, " \t\r\n")
	if s == "" {
		return "", s, false
	}
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(i > 0 && c >= '0' && c <= '9') {
			i++
			continue
		}
		break
	}
	if i == 0 {
		return "", s, false
	}
	return s[:i], s[i:], true
}

// parseContainerCond parses a container condition with or < and < not precedence.
func parseContainerCond(s string) (ContainerCond, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ContainerCond{}, false
	}
	return parseOrCond(s)
}

func parseOrCond(s string) (ContainerCond, bool) {
	parts, ok := splitCondKeyword(s, "or")
	if !ok {
		return ContainerCond{}, false
	}
	if len(parts) == 1 {
		return parseAndCond(parts[0])
	}
	kids := make([]ContainerCond, 0, len(parts))
	for _, p := range parts {
		c, ok := parseAndCond(p)
		if !ok {
			return ContainerCond{}, false
		}
		kids = append(kids, c)
	}
	return ContainerCond{Kind: "or", Kids: kids}, true
}

func parseAndCond(s string) (ContainerCond, bool) {
	parts, ok := splitCondKeyword(s, "and")
	if !ok {
		return ContainerCond{}, false
	}
	if len(parts) == 1 {
		return parseNotCond(parts[0])
	}
	kids := make([]ContainerCond, 0, len(parts))
	for _, p := range parts {
		c, ok := parseNotCond(p)
		if !ok {
			return ContainerCond{}, false
		}
		kids = append(kids, c)
	}
	return ContainerCond{Kind: "and", Kids: kids}, true
}

func parseNotCond(s string) (ContainerCond, bool) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "not") {
		rest := strings.TrimSpace(s[3:])
		if rest == "" {
			return ContainerCond{}, false
		}
		// "not (...)" or "not <cond>"
		inner, ok := parseNotCond(rest)
		if !ok {
			return ContainerCond{}, false
		}
		return ContainerCond{Kind: "not", Kids: []ContainerCond{inner}}, true
	}
	return parseParenOrFeat(s)
}

func parseParenOrFeat(s string) (ContainerCond, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return ContainerCond{}, false
	}
	inner, rest, ok := takeParen(s)
	if !ok || strings.TrimSpace(rest) != "" {
		return ContainerCond{}, false
	}
	inner = strings.TrimSpace(inner)
	// Nested condition vs feature: if it looks like and/or/not/(, recurse.
	low := strings.ToLower(inner)
	if strings.HasPrefix(inner, "(") || strings.HasPrefix(low, "not") ||
		containsTopLevelKeyword(inner, "and") || containsTopLevelKeyword(inner, "or") {
		return parseContainerCond(inner)
	}
	feat, ok := parseSizeFeature(inner)
	if !ok {
		return ContainerCond{}, false
	}
	return ContainerCond{Kind: "feat", Feat: &feat}, true
}

// splitCondKeyword splits on top-level `and`/`or` keywords (not inside parens).
func splitCondKeyword(s, kw string) ([]string, bool) {
	s = strings.TrimSpace(s)
	var parts []string
	depth := 0
	start := 0
	low := strings.ToLower(s)
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '"', '\'':
			q := c
			i++
			for i < len(s) && s[i] != q {
				if s[i] == '\\' && i+1 < len(s) {
					i++
				}
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		case '(':
			depth++
			i++
			continue
		case ')':
			depth--
			i++
			continue
		}
		if depth == 0 && hasKeywordAt(low, i, kw) {
			parts = append(parts, strings.TrimSpace(s[start:i]))
			i += len(kw)
			start = i
			continue
		}
		i++
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
	}
	return parts, true
}

func hasKeywordAt(low string, i int, kw string) bool {
	if i+len(kw) > len(low) {
		return false
	}
	if low[i:i+len(kw)] != kw {
		return false
	}
	// word boundaries
	if i > 0 {
		prev := low[i-1]
		if isIdentChar(prev) {
			return false
		}
	}
	if i+len(kw) < len(low) {
		next := low[i+len(kw)]
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
		return SizeFeature{}, false
	}
	// min-width: 400px / max-inline-size: 20em
	if colon := strings.IndexByte(inner, ':'); colon >= 0 {
		name := strings.ToLower(strings.TrimSpace(inner[:colon]))
		valStr := strings.TrimSpace(inner[colon+1:])
		val, unit, ok := ParseLength(valStr)
		if !ok {
			return SizeFeature{}, false
		}
		switch name {
		case "min-width", "min-inline-size":
			feat := "width"
			if strings.HasSuffix(name, "inline-size") {
				feat = "inline-size"
			}
			return SizeFeature{Name: feat, Op: ">=", Value: val, Unit: unit}, true
		case "max-width", "max-inline-size":
			feat := "width"
			if strings.HasSuffix(name, "inline-size") {
				feat = "inline-size"
			}
			return SizeFeature{Name: feat, Op: "<=", Value: val, Unit: unit}, true
		case "width", "inline-size":
			return SizeFeature{Name: name, Op: "=", Value: val, Unit: unit}, true
		default:
			return SizeFeature{}, false
		}
	}
	// Range: inline-size > 20em  |  20em < inline-size < 40em (single compare only)
	return parseRangeFeature(inner)
}

func parseRangeFeature(inner string) (SizeFeature, bool) {
	// Tokenize roughly: ident/number/op
	type tok struct {
		kind string // "ident", "num", "op"
		val  string
		num  float64
		unit string
	}
	var toks []tok
	s := strings.TrimSpace(inner)
	for s != "" {
		s = strings.TrimLeft(s, " \t\r\n")
		if s == "" {
			break
		}
		if strings.HasPrefix(s, "<=") || strings.HasPrefix(s, ">=") {
			toks = append(toks, tok{kind: "op", val: s[:2]})
			s = s[2:]
			continue
		}
		if s[0] == '<' || s[0] == '>' || s[0] == '=' {
			toks = append(toks, tok{kind: "op", val: s[:1]})
			s = s[1:]
			continue
		}
		// length or ident
		if (s[0] >= '0' && s[0] <= '9') || s[0] == '.' || s[0] == '+' || s[0] == '-' {
			// find end of length token
			i := 0
			if s[0] == '+' || s[0] == '-' {
				i++
			}
			for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
				i++
			}
			j := i
			for j < len(s) && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z') || s[j] == '%') {
				j++
			}
			val, unit, ok := ParseLength(s[:j])
			if !ok {
				// bare number?
				if n, err := strconv.ParseFloat(s[:i], 64); err == nil && i == j {
					toks = append(toks, tok{kind: "num", num: n, unit: "px", val: s[:i]})
					s = s[i:]
					continue
				}
				return SizeFeature{}, false
			}
			toks = append(toks, tok{kind: "num", num: val, unit: unit, val: s[:j]})
			s = s[j:]
			continue
		}
		ident, rem, ok := readIdent(s)
		if !ok {
			return SizeFeature{}, false
		}
		toks = append(toks, tok{kind: "ident", val: strings.ToLower(ident)})
		s = rem
	}
	// Patterns: name op num  OR  num op name
	if len(toks) == 3 && toks[1].kind == "op" {
		if toks[0].kind == "ident" && toks[2].kind == "num" {
			name := toks[0].val
			if name != "width" && name != "inline-size" {
				return SizeFeature{}, false
			}
			return SizeFeature{Name: name, Op: toks[1].val, Value: toks[2].num, Unit: toks[2].unit}, true
		}
		if toks[0].kind == "num" && toks[2].kind == "ident" {
			name := toks[2].val
			if name != "width" && name != "inline-size" {
				return SizeFeature{}, false
			}
			// flip: 20em < width  →  width > 20em
			op := flipOp(toks[1].val)
			if op == "" {
				return SizeFeature{}, false
			}
			return SizeFeature{Name: name, Op: op, Value: toks[0].num, Unit: toks[0].unit}, true
		}
	}
	return SizeFeature{}, false
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
