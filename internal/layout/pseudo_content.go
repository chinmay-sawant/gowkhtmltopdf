package layout

import (
	"fmt"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// pseudoContent cascades the CSS content property for ::before/::after on n.
// Supports string literals and none/normal (empty). Wiki hlist separators use
// li::after{content:"\a0 · "}.
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
	for _, sheet := range e.opts.Sheets {
		if sheet == nil {
			continue
		}
		for _, r := range sheet.Rules {
			if !css.MediaMatches(r.Media, media, e.opts.Width, e.opts.Height) {
				continue
			}
			for _, sel := range r.Selectors {
				if !css.MatchPseudo(sel, n, pe) {
					continue
				}
				a, b, c := css.Specificity(sel)
				for _, d := range r.Decls {
					if !strings.EqualFold(d.Prop, "content") {
						continue
					}
					h := hit{value: d.Value, a: a, b: b, c: c, order: r.Order, important: d.Important}
					if better(h) {
						hh := h
						best = &hh
					}
				}
			}
		}
	}
	if best == nil {
		return ""
	}
	return parseContentValue(best.value)
}

func parseContentValue(v string) string {
	v = strings.TrimSpace(v)
	low := strings.ToLower(v)
	if low == "none" || low == "normal" || v == "" {
		return ""
	}
	if len(v) >= 2 {
		q := v[0]
		if (q == '"' || q == '\'') && v[len(v)-1] == q {
			return decodeCSSString(v[1 : len(v)-1])
		}
	}
	return ""
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
