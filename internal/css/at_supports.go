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
	cond, parsed := parseSupportsPrelude(prelude)

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	if !parsed {
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
	for ruleIndex := range rules {
		if rules[ruleIndex].Supports == nil {
			rules[ruleIndex].Supports = cond

			continue
		}

		rules[ruleIndex].Supports = &SupportsCondition{
			Kind:     supportsAnd,
			Prop:     "",
			Value:    "",
			Children: []SupportsCondition{*rules[ruleIndex].Supports, *cond},
			Inner:    nil,
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
		inner, rest, parsed := parseSupportsInParens(strings.TrimSpace(src[len("not"):]), depth+1)
		if !parsed {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		return SupportsCondition{
			Kind:     supportsNot,
			Prop:     "",
			Value:    "",
			Children: nil,
			Inner:    &inner,
		}, rest, true
	}

	first, rest, matched := parseSupportsInParens(src, depth+1)
	if !matched {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	return parseSupportsChain(first, rest, depth)
}

// parseSupportsChain parses the and/or continuation after a condition's first
// operand. Mixing the two operators rejects the whole condition.
func parseSupportsChain(first SupportsCondition, rest string, depth int) (SupportsCondition, string, bool) {
	operator := ""
	children := []SupportsCondition{first}

	for {
		word, after, found := supportsOperator(strings.TrimSpace(rest))
		if !found {
			break
		}

		if operator != "" && word != operator {
			return SupportsCondition{}, "", false //nolint:exhaustruct // and/or cannot mix
		}

		operator = word

		next, tail, matched := parseSupportsInParens(strings.TrimSpace(after), depth+1)
		if !matched {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		children = append(children, next)
		rest = tail
	}

	if operator == "" {
		return first, rest, true
	}

	return SupportsCondition{
		Kind:     operator,
		Prop:     "",
		Value:    "",
		Children: children,
		Inner:    nil,
	}, rest, true
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
		closeIdx := matchingSupportsParen(src)
		if closeIdx < 0 {
			return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
		}

		inner := strings.TrimSpace(src[1:closeIdx])
		rest := src[closeIdx+1:]

		if decl, matched := parseSupportsDeclaration(inner); matched {
			return decl, rest, true
		}

		if cond, tail, matched := parseSupportsCond(inner, depth+1); matched && strings.TrimSpace(tail) == "" {
			return cond, rest, true
		}

		return SupportsCondition{
			Kind:     supportsFalse,
			Prop:     "",
			Value:    "",
			Children: nil,
			Inner:    nil,
		}, rest, true
	}

	closeIdx := matchingFunctionParen(src)
	if closeIdx < 0 {
		return SupportsCondition{}, "", false //nolint:exhaustruct // parse failure sentinel
	}

	return SupportsCondition{
		Kind:     supportsFalse,
		Prop:     "",
		Value:    "",
		Children: nil,
		Inner:    nil,
	}, src[closeIdx+1:], true
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

	return SupportsCondition{
		Kind:     supportsDecl,
		Prop:     prop,
		Value:    value,
		Children: nil,
		Inner:    nil,
	}, true
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
		return supportsAll(cond.Children, supported)
	case supportsOr:
		return supportsAny(cond.Children, supported)
	case supportsNot:
		return !SupportsMatches(cond.Inner, supported)
	default:
		return false
	}
}

// supportsAll reports whether supported holds for every child condition.
func supportsAll(children []SupportsCondition, supported func(prop, value string) bool) bool {
	for index := range children {
		if !SupportsMatches(&children[index], supported) {
			return false
		}
	}

	return true
}

// supportsAny reports whether supported holds for at least one child condition.
func supportsAny(children []SupportsCondition, supported func(prop, value string) bool) bool {
	for index := range children {
		if SupportsMatches(&children[index], supported) {
			return true
		}
	}

	return false
}

// supportsOperator reads a leading and/or token. The operator must be
// followed by whitespace or '(' so `(a)orange` is not read as `or`.
func supportsOperator(src string) (string, string, bool) {
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

	for index := range len(src) {
		switch src[index] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
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

	closeIdx := matchingSupportsParen(src[open:])
	if closeIdx < 0 {
		return -1
	}

	return open + closeIdx
}

// supportsTopLevelColon returns the first ':' at paren depth zero, or -1.
func supportsTopLevelColon(src string) int {
	depth := 0

	for index := range len(src) {
		switch src[index] {
		case '(':
			depth++
		case ')':
			depth--
		case ':':
			if depth == 0 {
				return index
			}
		}
	}

	return -1
}
