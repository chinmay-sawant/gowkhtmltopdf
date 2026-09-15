package css

import (
	"net/url"
	"strings"
)

// ResolveURLs records base as the stylesheet's source URL and rewrites the
// url() references in its declarations and @font-face src values into
// absolute URLs, so later fetch stages do not need the sheet URL.
//
// References whose meaning does not depend on a base are left untouched:
// data: URIs, fragment-only refs such as url(#clip), and already-absolute
// URLs. @import URLs are also left untouched because the stylesheet collector
// resolves them against the importer's base when it fetches them.
//
// An empty or unusable base (relative URL, opaque scheme) records Base and
// rewrites nothing. Calling ResolveURLs twice with the same base is harmless
// because rewritten references are absolute.
//
// <base href> is not consulted: inline <style> sheets resolve against the
// document URL and fetched sheets against their own URL, never against the
// document's <base> element.
func (s *Stylesheet) ResolveURLs(base string) {
	if s == nil {
		return
	}

	s.Base = base

	baseURL := absoluteBase(base)
	if baseURL == nil {
		return
	}

	for i := range s.Rules {
		decls := s.Rules[i].Decls
		for j := range decls {
			decls[j].Value = resolveURLRefs(decls[j].Value, baseURL)
		}
	}

	for i := range s.FontFaces {
		s.FontFaces[i].Src = resolveURLRefs(s.FontFaces[i].Src, baseURL)
	}
}

// absoluteBase parses base and reports it only when references can resolve
// against it: absolute with a hierarchical (non-opaque) scheme.
func absoluteBase(base string) *url.URL {
	base = strings.TrimSpace(base)
	if base == "" {
		return nil
	}

	parsed, err := url.Parse(base)
	if err != nil || !parsed.IsAbs() || parsed.Opaque != "" {
		return nil
	}

	return parsed
}

// resolveURLRefs rewrites every url() reference in one declaration value
// against base. Quoted spans are skipped, so url() text inside a string
// literal is not a reference. The original string is returned when nothing
// changed.
func resolveURLRefs(value string, base *url.URL) string {
	if !hasURLFunc(value) {
		return value
	}

	var out strings.Builder

	changed := false

	for idx := 0; idx < len(value); {
		switch {
		case value[idx] == '"' || value[idx] == '\'':
			end := skipQuoted(value, idx, value[idx])
			out.WriteString(value[idx:end])

			idx = end
		case hasFoldPrefix(value[idx:], "url("):
			end, ref, tokenFound := urlFuncRef(value[idx:])
			if !tokenFound {
				out.WriteString(value[idx:])

				idx = len(value)

				continue
			}

			resolvedURL, hasBase := resolveRef(base, ref)
			if hasBase {
				out.WriteString(`url("`)
				out.WriteString(resolvedURL)
				out.WriteString(`")`)

				changed = true
			} else {
				out.WriteString(value[idx : idx+end])
			}

			idx += end
		default:
			out.WriteByte(value[idx])

			idx++
		}
	}

	if !changed {
		return value
	}

	return out.String()
}

// hasURLFunc reports whether value contains a url( marker in any ASCII case.
func hasURLFunc(value string) bool {
	const marker = "url("

	for idx := 0; idx+len(marker) <= len(value); idx++ {
		if hasFoldPrefix(value[idx:], marker) {
			return true
		}
	}

	return false
}

// urlFuncRef reads one url(...) token at the start of src and returns the
// index just past ')' plus the raw reference (quotes and surrounding space
// trimmed). Quoted spans are skipped so a ')' inside the reference does not
// end the token. ok is false for an unterminated token; the caller then keeps
// the remaining value verbatim.
func urlFuncRef(src string) (int, string, bool) {
	const markerLen = len("url(")

	idx := markerLen

	for idx < len(src) {
		switch src[idx] {
		case '"', '\'':
			idx = skipQuoted(src, idx, src[idx])

			continue
		case ')':
			ref := strings.TrimSpace(src[markerLen:idx])
			ref = strings.Trim(ref, `"'`)

			return idx + 1, ref, true
		}

		idx++
	}

	return 0, "", false
}

// resolveRef resolves one url() reference against base. ok is false for
// references that keep their meaning without a base: empty refs,
// fragment-only refs, and absolute URLs (including data:).
func resolveRef(base *url.URL, ref string) (string, bool) {
	if ref == "" || strings.HasPrefix(ref, "#") {
		return "", false
	}

	parsed, err := url.Parse(ref)
	if err != nil || parsed.IsAbs() {
		return "", false
	}

	resolved := base.ResolveReference(parsed).String()

	// A raw double quote can survive in a query string; escape it so the
	// emitted token stays a valid quoted url().
	return strings.ReplaceAll(resolved, `"`, "%22"), true
}
