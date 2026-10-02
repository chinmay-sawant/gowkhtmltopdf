package css

import "strings"

// SupportsCondition is one parsed @supports prelude. Kind is "decl" for a
// declaration feature, "and"/"or" for a condition list, "not" for a negated
// condition, or "false" for a general-enclosed feature the engine cannot
// evaluate (selector(), font-tech(), ...). Decl carries Prop and Value;
// Children hold the and/or operands; Inner holds the negated condition.
type SupportsCondition struct {
	Kind     string
	Prop     string
	Value    string
	Children []SupportsCondition
	Inner    *SupportsCondition
}

// @supports condition kinds.
const (
	supportsDecl  = "decl"
	supportsAnd   = "and"
	supportsOr    = "or"
	supportsNot   = "not"
	supportsFalse = "false"
)

// parseSupportsRule consumes one @supports block. A syntactically invalid
// prelude drops the block (CSS treats the at-rule as invalid); a valid but
// unevaluable condition (general-enclosed) keeps the rules gated false.
func parseSupportsRule(src string, str *Stylesheet, order *int) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return skipAtRule(src)
	}

	prelude := strings.TrimSpace(src[len("@supports"):open])
	cond, ok := parseSupportsPrelude(prelude)

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	if !ok {
		return rest, nil
	}

	rules, err := parseRuleList(str, "all", nil, block, order, 0)
	if err != nil {
		return "", err
	}

	gateRulesSupports(rules, cond)
	str.Rules = append(str.Rules, rules...)

	return rest, nil
}

// gateRulesSupports sets cond on rules that carry no @supports gate and
// combines nested gates with and.
func gateRulesSupports(rules []Rule, cond *SupportsCondition) {
	for i := range rules {
		if rules[i].Supports == nil {
			rules[i].Supports = cond

			continue
		}

		rules[i].Supports = &SupportsCondition{
			Kind:     supportsAnd,
			Children: []SupportsCondition{*rules[i].Supports, *cond},
		}
	}
}

// parseSupportsPrelude parses one @supports prelude (the text between
// @supports and the block). It reports false for syntax the grammar rejects;
// general-enclosed features parse but evaluate false.
func parseSupportsPrelude(prelude string) (*SupportsCondition, bool) {
	cond, rest, ok := parseSupportsCond(strings.TrimSpace(prelude), 0)
	if !ok || strings.TrimSpace(rest) != "" {
		return nil, false
	}

	return &cond, true
}

// parseSupportsCond parses a supports-condition: `not <in-parens>` or one or
// more <in-parens> joined by a single operator (and/or cannot mix).
func parseSupportsCond(src string, depth int) (SupportsCondition, string, bool) {
	src = strings.TrimSpace(src)
	if src == "" || depth > maxParseDepth {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	if hasFoldPrefix(src, "not") && supportsBoundary(src, len("not")) {
		inner, rest, ok := parseSupportsInParens(strings.TrimSpace(src[len("not"):]), depth+1)
		if !ok {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		return SupportsCondition{Kind: supportsNot, Inner: &inner}, rest, true
	}

	first, rest, ok := parseSupportsInParens(src, depth+1)
	if !ok {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	op := ""
	children := []SupportsCondition{first}

	for {
		word, after, found := supportsOperator(strings.TrimSpace(rest))
		if !found {
			break
		}

		if op != "" && word != op {
			return SupportsCondition{}, "", false //nolint:exhaustruct // and/or cannot mix
		}

		op = word

		next, tail, ok := parseSupportsInParens(strings.TrimSpace(after), depth+1)
		if !ok {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		children = append(children, next)
		rest = tail
	}

	if op == "" {
		return first, rest, true
	}

	return SupportsCondition{Kind: op, Children: children}, rest, true
}

// parseSupportsInParens parses `( <condition> )`, `( <declaration> )`, or a
// general-enclosed feature (parenthesized text or a function token).
// General-enclosed consumes the feature and yields Kind "false".
func parseSupportsInParens(src string, depth int) (SupportsCondition, string, bool) {
	src = strings.TrimSpace(src)
	if src == "" {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	if src[0] == '(' {
		close := matchingSupportsParen(src)
		if close < 0 {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		inner := strings.TrimSpace(src[1:close])
		rest := src[close+1:]

		if decl, ok := parseSupportsDeclaration(inner); ok {
			return decl, rest, true
		}

		if cond, tail, ok := parseSupportsCond(inner, depth+1); ok && strings.TrimSpace(tail) == "" {
			return cond, rest, true
		}

		return SupportsCondition{Kind: supportsFalse}, rest, true
	}

	close := matchingFunctionParen(src)
	if close < 0 {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	return SupportsCondition{Kind: supportsFalse}, src[close+1:], true
}

// parseSupportsDeclaration parses `prop: value`. The property name is
// lowercased; an empty value is not a declaration feature.
func parseSupportsDeclaration(inner string) (SupportsCondition, bool) {
	colon := supportsTopLevelColon(inner)
	if colon <= 0 {
		return SupportsCondition{}, false //nolint:exhaustruct // not a declaration
	}

	prop := strings.ToLower(strings.TrimSpace(inner[:colon]))
	value := strings.TrimSpace(inner[colon+1:])
	if !validPropName(prop) || value == "" {
		return SupportsCondition{}, false //nolint:exhaustruct // not a declaration
	}

	return SupportsCondition{Kind: supportsDecl, Prop: prop, Value: value}, true
}

// SupportsMatches reports whether cond holds. supported decides one
// declaration feature and must report false for properties the engine has no
// apply arm for. A nil condition matches (no @supports gate).
func SupportsMatches(cond *SupportsCondition, supported func(prop, value string) bool) bool {
	if cond == nil {
		return true
	}

	switch cond.Kind {
	case supportsDecl:
		return supported(cond.Prop, cond.Value)
	case supportsAnd:
		for i := range cond.Children {
			if !SupportsMatches(&cond.Children[i], supported) {
				return false
			}
		}

		return true
	case supportsOr:
		for i := range cond.Children {
			if SupportsMatches(&cond.Children[i], supported) {
				return true
			}
		}

		return false
	case supportsNot:
		return !SupportsMatches(cond.Inner, supported)
	default:
		return false
	}
}

// supportsOperator reads a leading and/or token. The operator must be
// followed by whitespace or '(' so `(a)orange` is not read as `or`.
func supportsOperator(src string) (word, rest string, ok bool) {
	switch {
	case hasFoldPrefix(src, "and") && supportsBoundary(src, len("and")):
		return supportsAnd, src[len("and"):], true
	case hasFoldPrefix(src, "or") && supportsBoundary(src, len("or")):
		return supportsOr, src[len("or"):], true
	default:
		return "", src, false
	}
}

// supportsBoundary reports that position end in src starts a new token: end of
// input, whitespace, or an opening parenthesis.
func supportsBoundary(src string, end int) bool {
	if end >= len(src) {
		return true
	}

	switch src[end] {
	case ' ', '\t', '\r', '\n', '(':
		return true
	default:
		return false
	}
}

// matchingSupportsParen returns the index of the ')' matching the leading
// '(', or -1 when unbalanced.
func matchingSupportsParen(src string) int {
	depth := 0

	for i := range len(src) {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// matchingFunctionParen returns the index of the ')' closing a leading ident
// function token such as selector(...), or -1 when src is not one.
func matchingFunctionParen(src string) int {
	open := strings.IndexByte(src, '(')
	if open <= 0 {
		return -1
	}

	for i := range open {
		if !isIdentChar(src[i]) {
			return -1
		}
	}

	close := matchingSupportsParen(src[open:])
	if close < 0 {
		return -1
	}

	return open + close
}

// supportsTopLevelColon returns the first ':' at paren depth zero, or -1.
func supportsTopLevelColon(src string) int {
	depth := 0

	for i := range len(src) {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ':':
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}
