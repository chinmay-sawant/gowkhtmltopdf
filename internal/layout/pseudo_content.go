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

	better := func(height hit) bool {
		if best == nil {
			return true
		}

		if height.important != best.important {
			return height.important
		}

		if height.a != best.a {
			return height.a > best.a
		}

		if height.b != best.b {
			return height.b > best.b
		}

		if height.c != best.c {
			return height.c > best.c
		}

		return height.order >= best.order
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
	for _, rowH := range ctx.matchedRules(n, pe) {
		for _, d := range rowH.r.Decls {
			if !strings.EqualFold(d.Prop, "content") {
				continue
			}

			h := hit{value: d.Value, a: rowH.a, b: rowH.b, c: rowH.c, order: rowH.r.Order, important: d.Important}
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
	if len(v) >= two {
		q := v[0]
		if (q == '"' || q == '\'') && v[len(v)-1] == q && !strings.Contains(v[1:len(v)-1], string(q)) {
			return decodeCSSString(v[1 : len(v)-1])
		}
	}

	var boxNode strings.Builder

	idx := 0
	for idx < len(v) {
		for idx < len(v) && (v[idx] == ' ' || v[idx] == '\t' || v[idx] == '\n' || v[idx] == '\r') {
			idx++
		}

		if idx >= len(v) {
			break
		}

		child := v[idx]
		if child == '"' || child == '\'' {
			jdx := idx + 1
			for jdx < len(v) {
				if v[jdx] == '\\' && jdx+1 < len(v) {
					jdx += 2

					continue
				}

				if v[jdx] == child {
					break
				}

				jdx++
			}

			if jdx < len(v) {
				boxNode.WriteString(decodeCSSString(v[idx+1 : jdx]))
				idx = jdx + 1

				continue
			}

			break
		}
		// attr(name) or attr(name, …) — only the attribute name is used.
		if strings.HasPrefix(strings.ToLower(v[idx:]), "attr(") {
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
			if n != nil && name != "" {
				boxNode.WriteString(n.Attribute(name))
			}

			if jdx < len(v) && v[jdx] == ')' {
				jdx++
			}

			idx = jdx

			continue
		}
		// Skip unknown function tokens: counter(...), counters(...), url(...).
		if j := strings.IndexByte(v[idx:], '('); j > 0 && isIdentStart(v[idx]) {
			// function name
			key := idx + j + 1
			depth := 1

			for key < len(v) && depth > 0 {
				if v[key] == '(' {
					depth++
				} else if v[key] == ')' {
					depth--
				}

				key++
			}

			idx = key

			continue
		}
		// Bare ident (open-quote, etc.) — skip one word.
		if isIdentStart(v[idx]) {
			j := idx + 1
			for j < len(v) && isIdentCont(v[j]) {
				j++
			}

			idx = j

			continue
		}

		idx++
	}

	return boxNode.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
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
			jdx := idx
			for jdx < len(s) && jdx-idx < 6 && isHex(s[jdx]) {
				jdx++
			}

			var code int

			fmt.Sscanf(s[idx:jdx], "%x", &code)

			if code != 0 {
				boxNode.WriteRune(rune(code))
			}

			idx = jdx
			if idx < len(s) && (s[idx] == ' ' || s[idx] == '\t' || s[idx] == '\n' || s[idx] == '\r' || s[idx] == '\f') {
				// skip one whitespace terminator after hex escape
			} else {
				idx--
			}

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

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
