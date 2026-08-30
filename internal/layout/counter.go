package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const (
	cssPropCounterReset     = "counter-reset"
	cssPropCounterIncrement = "counter-increment"
	cssPropQuotes           = "quotes"
	cssContentOpenQuote     = "open-quote"
	cssContentCloseQuote    = "close-quote"
	cssContentNoOpenQuote   = "no-open-quote"
	cssContentNoCloseQuote  = "no-close-quote"
	defaultQuoteOpen        = "\u201c"
	defaultQuoteClose       = "\u201d"
	counterResetDefault     = 0
	counterIncrementDefault = 1
	cssKeywordInitial       = "initial"
	cssKeywordUnset         = "unset"
	cssKeywordRevert        = "revert"
)

// counterOp is one name plus integer from counter-reset or counter-increment.
type counterOp struct {
	name  string
	value int
}

// counterMap is the nested CSS counter set: byName[name] is outermost-first.
type counterMap struct {
	byName map[string][]int
}

func newCounterMap() *counterMap {
	return &counterMap{byName: make(map[string][]int)}
}

func (m *counterMap) applyReset(spec string) []string {
	ops := parseCounterList(spec, counterResetDefault)
	if len(ops) == 0 {
		return nil
	}

	names := make([]string, 0, len(ops))

	for _, item := range ops {
		m.byName[item.name] = append(m.byName[item.name], item.value)
		names = append(names, item.name)
	}

	return names
}

func (m *counterMap) applyIncrement(spec string) {
	for _, item := range parseCounterList(spec, counterIncrementDefault) {
		stack := m.byName[item.name]
		if len(stack) == 0 {
			// CSS 2.1: increment of an unseen counter behaves as a 0 reset
			// on the root, so later siblings share the same values.
			m.byName[item.name] = []int{item.value}

			continue
		}

		stack[len(stack)-1] += item.value
		m.byName[item.name] = stack
	}
}

func (m *counterMap) pop(names []string) {
	for idx := len(names) - 1; idx >= 0; idx-- {
		name := names[idx]
		stack := m.byName[name]

		if len(stack) == 0 {
			continue
		}

		m.byName[name] = stack[:len(stack)-1]
	}
}

func (m *counterMap) value(name string) int {
	stack := m.byName[name]
	if len(stack) == 0 {
		return counterResetDefault
	}

	return stack[len(stack)-1]
}

func (m *counterMap) values(name string) []int {
	stack := m.byName[name]
	if len(stack) == 0 {
		return []int{counterResetDefault}
	}

	out := make([]int, len(stack))
	copy(out, stack)

	return out
}

// parseCounterList parses `none | [ <ident> <integer>? ]+`.
// defaultValue is 0 for reset and 1 for increment.
func parseCounterList(spec string, defaultValue int) []counterOp {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.EqualFold(spec, cssDisplayNone) || isCSSWideKeyword(spec) {
		return nil
	}

	tokens := strings.Fields(spec)
	ops := make([]counterOp, 0, len(tokens))

	for idx := 0; idx < len(tokens); idx++ {
		tok := tokens[idx]
		if strings.EqualFold(tok, cssDisplayNone) || strings.ContainsRune(tok, '(') {
			continue
		}

		item := counterOp{name: tok, value: defaultValue}

		if idx+1 < len(tokens) && isIntegerToken(tokens[idx+1]) {
			val, err := strconv.Atoi(tokens[idx+1])
			if err == nil {
				item.value = val
				idx++
			}
		}

		ops = append(ops, item)
	}

	return ops
}

func isIntegerToken(tok string) bool {
	if tok == "" {
		return false
	}

	start := 0

	if tok[0] == '+' || tok[0] == '-' {
		if len(tok) == 1 {
			return false
		}

		start = 1
	}

	for idx := start; idx < len(tok); idx++ {
		if tok[idx] < '0' || tok[idx] > '9' {
			return false
		}
	}

	return true
}

func isCSSWideKeyword(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword, cssKeywordInitial, cssKeywordUnset, cssKeywordRevert:
		return true
	default:
		return false
	}
}

// quoteStyle is the used quotes list. none emits no glyphs but still nests.
type quoteStyle struct {
	opens  []string
	closes []string
	none   bool
}

func defaultQuotes() quoteStyle {
	return quoteStyle{
		opens:  []string{defaultQuoteOpen},
		closes: []string{defaultQuoteClose},
		none:   false,
	}
}

func (qs quoteStyle) glyph(depth int, open bool) string {
	if qs.none || depth < 0 {
		return ""
	}

	list := qs.closes
	if open {
		list = qs.opens
	}

	if len(list) == 0 {
		return ""
	}

	if depth >= len(list) {
		depth = len(list) - 1
	}

	return list[depth]
}

func parseQuotes(value string) quoteStyle {
	value = strings.TrimSpace(value)
	low := strings.ToLower(value)

	if low == cssDisplayNone {
		return quoteStyle{opens: nil, closes: nil, none: true}
	}

	if value == "" || isCSSWideKeyword(low) || low == overflowAuto {
		return defaultQuotes()
	}

	parts := collectQuotedStrings(value)
	if len(parts) < 2 {
		return defaultQuotes()
	}

	count := len(parts) / 2
	style := quoteStyle{
		opens:  make([]string, 0, count),
		closes: make([]string, 0, count),
		none:   false,
	}

	for idx := 0; idx+1 < len(parts); idx += 2 {
		style.opens = append(style.opens, parts[idx])
		style.closes = append(style.closes, parts[idx+1])
	}

	return style
}

func collectQuotedStrings(value string) []string {
	parts := make([]string, 0, 2)
	idx := 0

	for {
		part, next, ok := nextQuotedCSSString(value, idx)
		if !ok {
			break
		}

		parts = append(parts, part)
		idx = next
	}

	return parts
}

// contentEnv is the generated-content evaluation context for one ::before/::after.
type contentEnv struct {
	counters   *counterMap
	quotes     quoteStyle
	quoteDepth int
}

func defaultContentEnv() *contentEnv {
	return &contentEnv{
		counters:   newCounterMap(),
		quotes:     defaultQuotes(),
		quoteDepth: 0,
	}
}

func (env *contentEnv) takeOpenQuote() string {
	if env == nil {
		return defaultQuoteOpen
	}

	text := env.quotes.glyph(env.quoteDepth, true)
	env.quoteDepth++

	return text
}

func (env *contentEnv) takeCloseQuote() string {
	if env == nil {
		return defaultQuoteClose
	}

	if env.quoteDepth > 0 {
		env.quoteDepth--
	}

	return env.quotes.glyph(env.quoteDepth, false)
}

func (env *contentEnv) skipOpenQuote() {
	if env != nil {
		env.quoteDepth++
	}
}

func (env *contentEnv) skipCloseQuote() {
	if env != nil && env.quoteDepth > 0 {
		env.quoteDepth--
	}
}

func documentRoot(node *html.Node) *html.Node {
	for node != nil && node.Parent != nil {
		node = node.Parent
	}

	return node
}

func (e *engine) contentEnvAt(target *html.Node, pseudo string) *contentEnv {
	env := defaultContentEnv()
	if e == nil || target == nil {
		return env
	}

	ctx := e.pseudoStyleContext()
	env.quotes = e.quotesFor(ctx, target)
	e.walkContentEnv(ctx, documentRoot(target), target, pseudo, env)

	return env
}

func (e *engine) quotesFor(ctx *styleContext, node *html.Node) quoteStyle {
	if e != nil {
		for cur := node; cur != nil; cur = cur.Parent {
			if sty := e.stylePtr(cur); sty != nil && sty.QuotesRaw != "" {
				return parseQuotes(sty.QuotesRaw)
			}
		}
	} else {
		for cur := node; cur != nil; cur = cur.Parent {
			raw := cascadedProp(ctx, cur, cssPropQuotes)
			if raw != "" && !isCSSWideKeyword(raw) {
				return parseQuotes(raw)
			}
		}
	}

	return defaultQuotes()
}

func (e *engine) counterResetSpec(ctx *styleContext, node *html.Node) string {
	if e != nil {
		if sty := e.stylePtr(node); sty != nil {
			return sty.CounterReset
		}
	}

	return cascadedProp(ctx, node, cssPropCounterReset)
}

func (e *engine) counterIncrementSpec(ctx *styleContext, node *html.Node) string {
	if e != nil {
		if sty := e.stylePtr(node); sty != nil {
			return sty.CounterIncrement
		}
	}

	return cascadedProp(ctx, node, cssPropCounterIncrement)
}

func cascadedProp(ctx *styleContext, node *html.Node, prop string) string {
	if node == nil || prop == "" {
		return ""
	}

	var best *contentHit

	if ctx != nil {
		best = winningPropHit(best, ctx.matchedRules(node, ""), prop)
	}

	best = winningInlinePropHit(best, node, prop)
	if best == nil {
		return ""
	}

	return best.value
}

func winningPropHit(best *contentHit, hits []ruleHit, prop string) *contentHit {
	for _, rowH := range hits {
		for _, decl := range rowH.r.Decls {
			if !strings.EqualFold(decl.Prop, prop) {
				continue
			}

			hit := contentHit{
				value:     decl.Value,
				a:         rowH.a,
				b:         rowH.b,
				c:         rowH.c,
				order:     rowH.r.Order,
				important: decl.Important,
			}
			if betterContentHit(hit, best) {
				copyHit := hit
				best = &copyHit
			}
		}
	}

	return best
}

func winningInlinePropHit(best *contentHit, node *html.Node, prop string) *contentHit {
	for _, decl := range css.ParseInline(node.Attribute("style")) {
		if !strings.EqualFold(decl.Prop, prop) {
			continue
		}

		hit := contentHit{
			value:     decl.Value,
			a:         1 << 30,
			b:         0,
			c:         0,
			order:     1 << 30,
			important: decl.Important,
		}
		if betterContentHit(hit, best) {
			copyHit := hit
			best = &copyHit
		}
	}

	return best
}

func (e *engine) walkContentEnv(
	ctx *styleContext, node, target *html.Node, pseudo string, env *contentEnv,
) bool {
	if node == nil {
		return false
	}

	if node.Type != html.ElementNode {
		return e.walkContentChildren(ctx, node, target, pseudo, env)
	}

	if e.stylePtr(node).Display == cssDisplayNone {
		return false
	}

	pushes := env.counters.applyReset(e.counterResetSpec(ctx, node))
	env.counters.applyIncrement(e.counterIncrementSpec(ctx, node))

	if node == target && pseudo == pseudoBefore {
		return true
	}

	applyGeneratedQuotes(ctx, node, pseudoBefore, env)

	if e.walkContentChildren(ctx, node, target, pseudo, env) {
		return true
	}

	if node == target && pseudo == pseudoAfter {
		return true
	}

	applyGeneratedQuotes(ctx, node, pseudoAfter, env)
	env.counters.pop(pushes)

	return false
}

func (e *engine) walkContentChildren(
	ctx *styleContext, node, target *html.Node, pseudo string, env *contentEnv,
) bool {
	for _, child := range node.Children {
		if e.walkContentEnv(ctx, child, target, pseudo, env) {
			return true
		}
	}

	return false
}

func applyGeneratedQuotes(ctx *styleContext, node *html.Node, pseudo string, env *contentEnv) {
	best := selectContentDecl(ctx, node, pseudo)
	if best == nil {
		return
	}

	// Quote tokens mutate env.quoteDepth; the generated string is discarded.
	evalContentValue(best.value, node, env)
}

func evalCounterFn(value string, idx int, env *contentEnv) (string, int) {
	inner, next := cssFunctionInner(value, idx)
	args := splitCSSArgs(inner)
	name := ""

	if len(args) > 0 {
		name = args[0]
	}

	number := counterResetDefault
	if env != nil && env.counters != nil && name != "" {
		number = env.counters.value(name)
	}

	return strconv.Itoa(number), next
}

func evalCountersFn(value string, idx int, env *contentEnv) (string, int) {
	inner, next := cssFunctionInner(value, idx)
	args := splitCSSArgs(inner)

	if len(args) == 0 || args[0] == "" {
		return "0", next
	}

	sep := ""
	if len(args) >= 2 {
		sep = unquoteCSSArg(args[1])
	}

	vals := []int{counterResetDefault}
	if env != nil && env.counters != nil {
		vals = env.counters.values(args[0])
	}

	parts := make([]string, len(vals))
	for pos, val := range vals {
		parts[pos] = strconv.Itoa(val)
	}

	return strings.Join(parts, sep), next
}

func cssFunctionInner(value string, idx int) (string, int) {
	paren := strings.IndexByte(value[idx:], '(')
	if paren < 0 {
		return "", skipCSSIdent(value, idx)
	}

	start := idx + paren + 1
	next := skipCSSFunction(value, start)
	end := next

	if end > start && value[end-1] == ')' {
		end--
	}

	return strings.TrimSpace(value[start:end]), next
}

func splitCSSArgs(inner string) []string {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return nil
	}

	args := make([]string, 0, 2)
	start, depth := 0, 0
	inQuote := byte(0)

	for idx := 0; idx < len(inner); idx++ {
		idx, start, depth, inQuote = consumeCSSArgByte(inner, idx, start, depth, inQuote, &args)
	}

	return append(args, strings.TrimSpace(inner[start:]))
}

func consumeCSSArgByte(
	inner string, idx, start, depth int, inQuote byte, args *[]string,
) (int, int, int, byte) {
	char := inner[idx]
	if inQuote != 0 {
		return consumeQuotedCSSArg(inner, idx, start, depth, inQuote, char)
	}

	switch char {
	case '"', '\'':
		return idx, start, depth, char
	case '(':
		return idx, start, depth + 1, inQuote
	case ')':
		if depth > 0 {
			depth--
		}

		return idx, start, depth, inQuote
	case ',':
		if depth == 0 {
			*args = append(*args, strings.TrimSpace(inner[start:idx]))
			start = idx + 1
		}
	}

	return idx, start, depth, inQuote
}

func consumeQuotedCSSArg(
	inner string, idx, start, depth int, inQuote, char byte,
) (int, int, int, byte) {
	if char == '\\' && idx+1 < len(inner) {
		return idx + 1, start, depth, inQuote
	}

	if char == inQuote {
		return idx, start, depth, 0
	}

	return idx, start, depth, inQuote
}

func unquoteCSSArg(arg string) string {
	arg = strings.TrimSpace(arg)
	if len(arg) < 2 {
		return arg
	}

	quote := arg[0]
	if (quote == '"' || quote == '\'') && arg[len(arg)-1] == quote {
		return decodeCSSString(arg[1 : len(arg)-1])
	}

	return arg
}
