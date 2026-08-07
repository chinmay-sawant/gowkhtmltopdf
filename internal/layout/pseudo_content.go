package layout

import (
	"fmt"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// pseudoContent cascades the CSS content property for ::before/::after on n.
// Supports string literals, attr(name), and none/normal (empty). Wiki hlist
// separators use li::after{content:"\a0 · "}; print external links use
// a.external::after{content:' (' attr(href) ')'}.
func (e *engine) pseudoContent(n *html.Node, pe string) string {
	if e == nil || n == nil || (pe != "before" && pe != "after") {
		return ""
	}

	type hit struct {
		value          string
		a, b, c, order int
		important      bool
	}

	var best *hit

	better := func(h hit) bool {
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
	ctx := &styleContext{
		sheets:     e.opts.Sheets,
		media:      media,
		viewportW:  e.opts.Width,
		viewportH:  e.opts.Height,
		containers: e.containers,
	}
	for _, rh := range ctx.matchedRules(n, pe) {
		for _, d := range rh.r.Decls {
			if !strings.EqualFold(d.Prop, "content") {
				continue
			}

			h := hit{value: d.Value, a: rh.a, b: rh.b, c: rh.c, order: rh.r.Order, important: d.Important}
			if better(h) {
				hh := h
				best = &hh
			}
		}
	}

	if best == nil {
		return ""
	}

	return parseContentValue(best.value, n)
}

// parseContentValue evaluates a CSS content list: quoted strings, attr(name),
// and none/normal. Unsupported tokens (counter(), url(), …) are skipped so we
// never paint the literal source text "attr(href)".
func parseContentValue(v string, n *html.Node) string {
	v = strings.TrimSpace(v)
	low := strings.ToLower(v)

	if low == "none" || low == "normal" || v == "" {
		return ""
	}
	// Fast path: single quoted string.
	if len(v) >= 2 {
		q := v[0]
		if (q == '"' || q == '\'') && v[len(v)-1] == q && !strings.Contains(v[1:len(v)-1], string(q)) {
			return decodeCSSString(v[1 : len(v)-1])
		}
	}

	var b strings.Builder

	i := 0
	for i < len(v) {
		for i < len(v) && (v[i] == ' ' || v[i] == '\t' || v[i] == '\n' || v[i] == '\r') {
			i++
		}

		if i >= len(v) {
			break
		}

		c := v[i]
		if c == '"' || c == '\'' {
			j := i + 1
			for j < len(v) {
				if v[j] == '\\' && j+1 < len(v) {
					j += 2

					continue
				}

				if v[j] == c {
					break
				}

				j++
			}

			if j < len(v) {
				b.WriteString(decodeCSSString(v[i+1 : j]))
				i = j + 1

				continue
			}

			break
		}
		// attr(name) or attr(name, …) — only the attribute name is used.
		if strings.HasPrefix(strings.ToLower(v[i:]), "attr(") {
			start := i + len("attr(")
			depth := 1
			j := start

			for j < len(v) && depth > 0 {
				if v[j] == '(' {
					depth++
				} else if v[j] == ')' {
					depth--
					if depth == 0 {
						break
					}
				}

				j++
			}

			arg := strings.TrimSpace(v[start:j])
			// First token is the attribute name (ignore type/fallback args).
			name := arg
			if sp := strings.IndexAny(arg, " \t,"); sp >= 0 {
				name = arg[:sp]
			}

			name = strings.Trim(name, `"'`)
			if n != nil && name != "" {
				b.WriteString(n.Attribute(name))
			}

			if j < len(v) && v[j] == ')' {
				j++
			}

			i = j

			continue
		}
		// Skip unknown function tokens: counter(...), counters(...), url(...).
		if j := strings.IndexByte(v[i:], '('); j > 0 && isIdentStart(v[i]) {
			// function name
			k := i + j + 1
			depth := 1

			for k < len(v) && depth > 0 {
				if v[k] == '(' {
					depth++
				} else if v[k] == ')' {
					depth--
				}

				k++
			}

			i = k

			continue
		}
		// Bare ident (open-quote, etc.) — skip one word.
		if isIdentStart(v[i]) {
			j := i + 1
			for j < len(v) && isIdentCont(v[j]) {
				j++
			}

			i = j

			continue
		}

		i++
	}

	return b.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func decodeCSSString(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])

			continue
		}

		i++
		if isHex(s[i]) {
			j := i
			for j < len(s) && j-i < 6 && isHex(s[j]) {
				j++
			}

			var code int

			fmt.Sscanf(s[i:j], "%x", &code)

			if code != 0 {
				b.WriteRune(rune(code))
			}

			i = j
			if i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f') {
				// skip one whitespace terminator after hex escape
			} else {
				i--
			}

			continue
		}

		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte(s[i])
		}
	}

	return b.String()
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
