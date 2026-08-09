package layout

import (
	"fmt"
	"strings"

	"gowkhtmltopdf/internal/html"
)

const contentNormal = "normal"

// pseudoContent cascades the CSS content property for ::before/::after on n.
// Supports string literals, attr(name), and none/normal (empty). Wiki hlist
// separators use li::after{content:"\a0 · "}; print external links use
// a.external::after{content:' (' attr(href) ')'}.
func (e *engine) pseudoContent(node *html.Node, pseudoEl string) string {
	if e == nil || node == nil || (pseudoEl != "before" && pseudoEl != "after") {
		return ""
	}

	media := e.opts.Media
	if media == "" {
		media = "print"
	}
	// Shared cascade walk (media + @container gates). Containers are nil on
	// the engine's style map path; when styles were re-cascaded with size
	// containers, content under non-matching @container is already absent
	// from ResolvedStyle — but pseudo content re-walks sheets, so pass the
	// same gate via a styleContext without containers (pass-1 semantics:
	// skip @container rules) matching cascadeRaw's first pass. When
	// e.styles came from a container pass, non-matching container content
	// is suppressed here the same way colors are.
	ctx := &styleContext{ //nolint:exhaustruct // intentional zero fields
		sheets:     e.opts.Sheets,
		media:      media,
		viewportW:  e.opts.Width,
		viewportH:  e.opts.Height,
		containers: e.containers,
	}

	best := selectContentDecl(ctx, node, pseudoEl)
	if best == nil {
		return ""
	}

	return parseContentValue(best.value, node)
}

// contentHit is one content: declaration with its cascade priority.
type contentHit struct {
	value          string
	a, b, c, order int
	important      bool
}

// selectContentDecl picks the winning content declaration for the pseudo
// element pe on n.
func selectContentDecl(ctx *styleContext, n *html.Node, pseudoEl string) *contentHit {
	var best *contentHit

	for _, rowH := range ctx.matchedRules(n, pseudoEl) {
		for _, d := range rowH.r.Decls {
			if !strings.EqualFold(d.Prop, "content") {
				continue
			}

			h := contentHit{value: d.Value, a: rowH.a, b: rowH.b, c: rowH.c, order: rowH.r.Order, important: d.Important}
			if betterContentHit(h, best) {
				hh := h
				best = &hh
			}
		}
	}

	return best
}

// betterContentHit reports whether candidate outranks best by importance,
// then specificity, then source order.
func betterContentHit(candidate contentHit, best *contentHit) bool {
	if best == nil {
		return true
	}

	if candidate.important != best.important {
		return candidate.important
	}

	if candidate.a != best.a {
		return candidate.a > best.a
	}

	if candidate.b != best.b {
		return candidate.b > best.b
	}

	if candidate.c != best.c {
		return candidate.c > best.c
	}

	return candidate.order >= best.order
}

// parseContentValue evaluates a CSS content list: quoted strings, attr(name),
// and none/normal. Unsupported tokens (counter(), url(), …) are skipped so we
// never paint the literal source text "attr(href)".
func parseContentValue(value string, node *html.Node) string {
	value = strings.TrimSpace(value)
	low := strings.ToLower(value)

	if low == displayNone || low == contentNormal || value == "" {
		return ""
	}
	// Fast path: single quoted string.
	if content, ok := singleQuotedContent(value); ok {
		return decodeCSSString(content)
	}

	var boxNode strings.Builder

	idx := 0
	for idx < len(value) {
		next, ok := scanContentToken(value, idx, node, &boxNode)
		if !ok {
			break
		}

		idx = next
	}

	return boxNode.String()
}

// scanContentToken consumes one token of a content list at idx, appending any
// text to boxNode, and returns the index just past it. ok is false when
// scanning must stop (unbalanced quote).
func scanContentToken(value string, idx int, node *html.Node, boxNode *strings.Builder) (int, bool) {
	idx = skipCSSWhitespace(value, idx)
	if idx >= len(value) {
		return idx, false
	}

	child := value[idx]
	if child == '"' || child == '\'' {
		if end, ok := scanQuotedContent(value, idx+1, child); ok {
			boxNode.WriteString(decodeCSSString(value[idx+1 : end]))

			return end + 1, true
		}

		return idx, false
	}
	// attr(name) or attr(name, …) — only the attribute name is used.
	if strings.HasPrefix(strings.ToLower(value[idx:]), "attr(") {
		val, next := parseAttrToken(value, idx, node)
		boxNode.WriteString(val)

		return next, true
	}
	// Skip unknown function tokens: counter(...), counters(...), url(...).
	if j := strings.IndexByte(value[idx:], '('); j > 0 && isIdentStart(value[idx]) {
		return skipCSSFunction(value, idx+j+1), true
	}
	// Bare ident (open-quote, etc.) — skip one word.
	if isIdentStart(value[idx]) {
		return skipCSSIdent(value, idx), true
	}

	return idx + 1, true
}

// singleQuotedContent returns the inner text when value is exactly one quoted
// string with no inner unescaped quote.
func singleQuotedContent(value string) (string, bool) {
	if len(value) < two {
		return "", false
	}

	q := value[0]
	if (q != '"' && q != '\'') || value[len(value)-1] != q || strings.Contains(value[1:len(value)-1], string(q)) {
		return "", false
	}

	return value[1 : len(value)-1], true
}

// scanQuotedContent finds the closing quote of a string whose opening quote is
// value[open-1], honoring backslash escapes.
func scanQuotedContent(value string, open int, quote byte) (int, bool) {
	jdx := open
	for jdx < len(value) {
		if value[jdx] == '\\' && jdx+1 < len(value) {
			jdx += 2

			continue
		}

		if value[jdx] == quote {
			return jdx, true
		}

		jdx++
	}

	return 0, false
}

// parseAttrToken evaluates attr(...) starting at idx (value[idx:] begins with
// "attr("). Returns the attribute value ("" when absent) and the index just
// past the closing paren.
func parseAttrToken(value string, idx int, node *html.Node) (string, int) {
	arg, next := attrArg(value, idx)

	name := arg
	if sp := strings.IndexAny(arg, " \t,"); sp >= 0 {
		name = arg[:sp]
	}

	name = strings.Trim(name, `"'`)

	val := ""
	if node != nil && name != "" {
		val = node.Attribute(name)
	}

	if next < len(value) && value[next] == ')' {
		next++
	}

	return val, next
}

// attrArg returns the trimmed argument text of the attr(...) call starting at
// idx (value[idx:] begins with "attr(") plus the index just past the closing
// paren (len(value) when unbalanced).
func attrArg(value string, idx int) (string, int) {
	start := idx + len("attr(")

	end, depth := start, 1
	for end < len(value) && depth > 0 {
		if value[end] == '(' {
			depth++
		} else if value[end] == ')' {
			depth--
			if depth == 0 {
				break
			}
		}

		end++
	}

	return strings.TrimSpace(value[start:end]), end
}

// skipCSSFunction returns the index just past the closing paren of the
// function whose opening paren is at start (value[start-1] == '(').
func skipCSSFunction(value string, start int) int {
	depth := 1

	for start < len(value) && depth > 0 {
		switch value[start] {
		case '(':
			depth++
		case ')':
			depth--
		}

		start++
	}

	return start
}

// skipCSSIdent advances idx past a CSS identifier (one word).
func skipCSSIdent(v string, idx int) int {
	for idx < len(v) && isIdentCont(v[idx]) {
		idx++
	}

	return idx
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// isCSSWhitespace reports whether c is CSS whitespace.
func isCSSWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// skipCSSWhitespace advances idx past CSS whitespace.
func skipCSSWhitespace(v string, idx int) int {
	for idx < len(v) && isCSSWhitespace(v[idx]) {
		idx++
	}

	return idx
}

func decodeCSSString(value string) string {
	var boxNode strings.Builder

	for idx := 0; idx < len(value); idx++ {
		if value[idx] != '\\' || idx+1 >= len(value) {
			boxNode.WriteByte(value[idx])

			continue
		}

		idx++
		if isHex(value[idx]) {
			r, next := decodeHexEscape(value, idx)
			if r != 0 {
				boxNode.WriteRune(r)
			}

			idx = next

			continue
		}

		switch value[idx] {
		case 'n':
			boxNode.WriteByte('\n')
		case 'r':
			boxNode.WriteByte('\r')
		case 't':
			boxNode.WriteByte('\t')
		default:
			boxNode.WriteByte(value[idx])
		}
	}

	return boxNode.String()
}

// decodeHexEscape consumes up to six hex digits at value[start] (the digits
// after a backslash), honors the optional single-whitespace terminator, and
// returns the decoded rune plus the index to resume from.
func decodeHexEscape(value string, start int) (rune, int) {
	jdx := start
	for jdx < len(value) && jdx-start < 6 && isHex(value[jdx]) {
		jdx++
	}

	idx := jdx
	if idx >= len(value) || !isCSSWhitespace(value[idx]) {
		idx--
	}

	var code int

	if _, err := fmt.Sscanf(value[start:jdx], "%x", &code); err != nil {
		return 0, idx
	}

	return rune(code), idx
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
