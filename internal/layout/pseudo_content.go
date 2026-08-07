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
func (e *engine) pseudoContent(n *html.Node, pe string) string {
	if e == nil || n == nil || (pe != "before" && pe != "after") {
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
	best := selectContentDecl(ctx, n, pe)
	if best == nil {
		return ""
	}

	return parseContentValue(best.value, n)
}

// contentHit is one content: declaration with its cascade priority.
type contentHit struct {
	value          string
	a, b, c, order int
	important      bool
}

// selectContentDecl picks the winning content declaration for the pseudo
// element pe on n.
func selectContentDecl(ctx *styleContext, n *html.Node, pe string) *contentHit {
	var best *contentHit

	for _, rowH := range ctx.matchedRules(n, pe) {
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

// betterContentHit reports whether candidate h outranks best by importance,
// then specificity, then source order.
func betterContentHit(h contentHit, best *contentHit) bool {
	if best == nil {
		return true
	}

	if h.important != best.important {
		return h.important
	}

	if h.a != best.a {
		return h.a > best.a
	}

	if h.b != best.b {
		return h.b > best.b
	}

	if h.c != best.c {
		return h.c > best.c
	}

	return h.order >= best.order
}

// parseContentValue evaluates a CSS content list: quoted strings, attr(name),
// and none/normal. Unsupported tokens (counter(), url(), …) are skipped so we
// never paint the literal source text "attr(href)".
func parseContentValue(v string, n *html.Node) string {
	v = strings.TrimSpace(v)
	low := strings.ToLower(v)

	if low == displayNone || low == contentNormal || v == "" {
		return ""
	}
	// Fast path: single quoted string.
	if content, ok := singleQuotedContent(v); ok {
		return decodeCSSString(content)
	}

	var boxNode strings.Builder

	idx := 0
	for idx < len(v) {
		idx = skipCSSWhitespace(v, idx)
		if idx >= len(v) {
			break
		}

		child := v[idx]
		if child == '"' || child == '\'' {
			if end, ok := scanQuotedContent(v, idx+1, child); ok {
				boxNode.WriteString(decodeCSSString(v[idx+1 : end]))
				idx = end + 1

				continue
			}

			break
		}
		// attr(name) or attr(name, …) — only the attribute name is used.
		if strings.HasPrefix(strings.ToLower(v[idx:]), "attr(") {
			val, next := parseAttrToken(v, idx, n)
			boxNode.WriteString(val)
			idx = next

			continue
		}
		// Skip unknown function tokens: counter(...), counters(...), url(...).
		if j := strings.IndexByte(v[idx:], '('); j > 0 && isIdentStart(v[idx]) {
			idx = skipCSSFunction(v, idx+j+1)

			continue
		}
		// Bare ident (open-quote, etc.) — skip one word.
		if isIdentStart(v[idx]) {
			idx = skipCSSIdent(v, idx)

			continue
		}

		idx++
	}

	return boxNode.String()
}

// singleQuotedContent returns the inner text when v is exactly one quoted
// string with no inner unescaped quote.
func singleQuotedContent(v string) (string, bool) {
	if len(v) < two {
		return "", false
	}

	q := v[0]
	if (q != '"' && q != '\'') || v[len(v)-1] != q || strings.Contains(v[1:len(v)-1], string(q)) {
		return "", false
	}

	return v[1 : len(v)-1], true
}

// scanQuotedContent finds the closing quote of a string whose opening quote is
// v[open-1], honoring backslash escapes.
func scanQuotedContent(v string, open int, q byte) (int, bool) {
	jdx := open
	for jdx < len(v) {
		if v[jdx] == '\\' && jdx+1 < len(v) {
			jdx += 2

			continue
		}

		if v[jdx] == q {
			return jdx, true
		}

		jdx++
	}

	return 0, false
}

// parseAttrToken evaluates attr(...) starting at idx (v[idx:] begins with
// "attr("). Returns the attribute value ("" when absent) and the index just
// past the closing paren.
func parseAttrToken(v string, idx int, n *html.Node) (string, int) {
	start := idx + len("attr(")
	depth := 1
	jdx := start

	for jdx < len(v) && depth > 0 {
		if v[jdx] == '(' {
			depth++
		} else if v[jdx] == ')' {
			depth--
			if depth == 0 {
				break
			}
		}

		jdx++
	}

	arg := strings.TrimSpace(v[start:jdx])
	// First token is the attribute name (ignore type/fallback args).
	name := arg
	if sp := strings.IndexAny(arg, " \t,"); sp >= 0 {
		name = arg[:sp]
	}

	name = strings.Trim(name, `"'`)
	val := ""
	if n != nil && name != "" {
		val = n.Attribute(name)
	}

	next := jdx
	if next < len(v) && v[next] == ')' {
		next++
	}

	return val, next
}

// skipCSSFunction returns the index just past the closing paren of the
// function whose opening paren is at start (v[start-1] == '(').
func skipCSSFunction(v string, start int) int {
	depth := 1

	for start < len(v) && depth > 0 {
		if v[start] == '(' {
			depth++
		} else if v[start] == ')' {
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

func decodeCSSString(s string) string {
	var boxNode strings.Builder

	for idx := 0; idx < len(s); idx++ {
		if s[idx] != '\\' || idx+1 >= len(s) {
			boxNode.WriteByte(s[idx])

			continue
		}

		idx++
		if isHex(s[idx]) {
			r, next := decodeHexEscape(s, idx)
			if r != 0 {
				boxNode.WriteRune(r)
			}

			idx = next

			continue
		}

		switch s[idx] {
		case 'n':
			boxNode.WriteByte('\n')
		case 'r':
			boxNode.WriteByte('\r')
		case 't':
			boxNode.WriteByte('\t')
		default:
			boxNode.WriteByte(s[idx])
		}
	}

	return boxNode.String()
}

// decodeHexEscape consumes up to six hex digits at s[at] (the digits after a
// backslash), honors the optional single-whitespace terminator, and returns
// the decoded rune plus the index to resume from.
func decodeHexEscape(s string, at int) (rune, int) {
	jdx := at
	for jdx < len(s) && jdx-at < 6 && isHex(s[jdx]) {
		jdx++
	}

	idx := jdx
	if idx >= len(s) || !isCSSWhitespace(s[idx]) {
		idx--
	}

	var code int

	if _, err := fmt.Sscanf(s[at:jdx], "%x", &code); err != nil {
		return 0, idx
	}

	return rune(code), idx
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
