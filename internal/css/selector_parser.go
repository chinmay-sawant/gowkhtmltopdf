package css

import (
	"strings"
)

// parseSelector parses one compound chain, e.g. "div.a#b > p.c" or
// "tr:nth-child(even)".
func parseSelector(s string) (Selector, bool) {
	return parseSelectorCtx(s, false)
}

// splitSelectorChain splits a selector into compounds and combinators, e.g.
// ["div.a", ">", "p", " ", "span"].
func splitSelectorChain(sel string) []string {
	var out []string

	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	for idx := 0; idx < len(sel); idx++ {
		cnt := sel[idx]

		switch cnt {
		case ' ', '\t', '\n', '\r':
			// skip whitespace; it becomes a descendant combinator only when
			// it sits between two compounds
			flush()

			idx = skipWhitespace(sel, idx)
			out = addDescendantCombinator(out, sel, idx)
		case '>', '+', '~':
			flush()

			out = append(out, string(cnt))
		case '[':
			idx = writeBracketLiteral(&cur, out, sel, idx)
		case ':':
			idx = writePseudoLiteral(&cur, out, sel, idx)
		case '\\':
			// escape: keep next char literally
			if idx+1 < len(sel) {
				cur.WriteByte(sel[idx+1])

				idx++
			}
		default:
			cur.WriteByte(cnt)
		}
	}

	flush()

	return out
}

func isSelBreak(b byte) bool {
	switch b {
	case '.', '#', '[', ':', '>', '+', '~', ' ', '\t', '\n', '\r':
		return true
	}

	return false
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// skipWhitespace advances idx to the end of a run of whitespace starting at
// idx (the character at idx is whitespace).
func skipWhitespace(s string, idx int) int {
	for idx+1 < len(s) && isWhitespace(s[idx+1]) {
		idx++
	}

	return idx
}

// addDescendantCombinator appends a " " token when the whitespace run ending
// at idx sits between two compounds (not after a combinator, not before '>').
func addDescendantCombinator(out []string, s string, idx int) []string {
	if len(out) > 0 && out[len(out)-1] != " " && out[len(out)-1] != ">" &&
		out[len(out)-1] != "+" && out[len(out)-1] != "~" && idx+1 < len(s) && s[idx+1] != '>' {
		return append(out, " ")
	}

	return out
}

// writeBracketLiteral keeps [attr] / [attr=value] inside the compound,
// prefixed with '*' when the compound starts with the bracket. Returns the
// index of the ']' (or the current idx when unterminated).
func writeBracketLiteral(cur *strings.Builder, out []string, sel string, idx int) int {
	jdx := strings.IndexByte(sel[idx:], ']')
	if jdx < 0 {
		cur.WriteByte('[')

		return idx
	}

	writeStarPrefix(cur, out)
	cur.WriteString(sel[idx : idx+jdx+1])

	return idx + jdx
}

// writeStarPrefix prefixes the current compound with '*' when it starts at a
// compound boundary (fresh compound after a combinator or list start).
func writeStarPrefix(cur *strings.Builder, out []string) {
	if cur.Len() == 0 && (len(out) == 0 || out[len(out)-1] == " " ||
		out[len(out)-1] == ">" || out[len(out)-1] == "+" || out[len(out)-1] == "~") {
		cur.WriteByte('*')
	}
}

// writePseudoLiteral keeps :pseudo / :nth-child(n) / :has(...) and
// ::pseudo-elements inside the compound so parseCompound can reject
// unsupported pseudo-elements. Never strip ::before/::after - that used to
// leave a bare host selector (Vector print `p::before{width:120pt}` became
// `p{width:120pt}` and crushed wiki body columns). Returns the index of the
// character after the pseudo (the caller's loop applies the final -1/+1).
func writePseudoLiteral(cur *strings.Builder, out []string, sel string, idx int) int {
	writeStarPrefix(cur, out)

	start := idx

	if idx+1 < len(sel) && sel[idx+1] == ':' {
		idx += 2 // ::pseudo-element
	} else {
		idx++ // :pseudo-class or CSS2 :before/:after
	}

	for idx < len(sel) && !isSelBreak(sel[idx]) {
		if sel[idx] == '(' {
			_, end, ok := takeParenArg(sel, idx)
			if ok {
				cur.WriteString(sel[start:end])

				return end - 1
			}

			cur.WriteString(sel[start:])

			return len(sel) - 1
		}

		idx++
	}

	cur.WriteString(sel[start:idx])

	return idx - 1
}

// parseCompoundCtx parses a compound ("tag#id.class[attr]:nth-child(even)").
// A tag of "*" or "" means universal. When insideHas is true, nested :has()
// and pseudo-elements are rejected as invalid.
func parseCompoundCtx(sel string, insideHas bool) (SelectorPart, bool) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return SelectorPart{Tag: "*"}, true //nolint:exhaustruct // intentional zero-value fields
	}

	tag, idx, valid := parseCompoundTag(sel)
	if !valid {
		return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	part := SelectorPart{Tag: tag} //nolint:exhaustruct // intentional zero-value fields

	for idx < len(sel) {
		switch sel[idx] {
		case '#':
			part, idx, valid = parseCompoundID(sel, part, idx)
		case '.':
			part, idx, valid = parseCompoundClass(sel, part, idx)
		case '[':
			part, idx, valid = parseCompoundAttr(sel, part, idx)
		case ':':
			part, idx, valid = parseCompoundPseudo(sel, part, insideHas, idx)
		default:
			return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		if !valid {
			return SelectorPart{}, false //nolint:exhaustruct // intentional zero-value fields
		}
	}

	return part, true
}

// parseCompoundTag reads the tag name at the start of sel. An empty tag
// ("*.cls", ".cls") means universal. Returns the tag and the scan index.
func parseCompoundTag(sel string) (string, int, bool) {
	idx := 0
	for idx < len(sel) && !isCompoundBreak(sel[idx]) {
		idx++
	}

	tag := sel[:idx]
	if tag == "" {
		return "*", 0, true
	}

	if tag != "*" && !validIdent(tag) {
		return "", 0, false
	}

	return tag, idx, true
}

func parseCompoundID(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	id := sel[idx+1 : jdx]
	if !validIdent(id) {
		return part, idx, false
	}

	part.ID = id

	return part, jdx, true
}

func parseCompoundClass(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	if jdx > idx+1 {
		cls := sel[idx+1 : jdx]
		if !validIdent(cls) {
			return part, idx, false
		}

		part.Classes = append(part.Classes, cls)
	}

	return part, jdx, true
}

func parseCompoundAttr(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := strings.IndexByte(sel[idx:], ']')
	if jdx < 0 {
		return part, idx, false
	}

	attr, ok := parseAttrSelector(sel[idx : idx+jdx+1])
	if !ok {
		return part, idx, false
	}

	part.Attrs = append(part.Attrs, attr)

	return part, idx + jdx + 1, true
}

// parseCompoundPseudo parses one :pseudo or ::pseudo-element starting at idx
// in sel and returns the updated part and the index of the next compound
// break.
func parseCompoundPseudo(sel string, part SelectorPart, insideHas bool, idx int) (SelectorPart, int, bool) {
	if idx+1 < len(sel) && sel[idx+1] == ':' {
		return parsePseudoElement(sel, part, idx)
	}

	return parsePseudoClass(sel, part, insideHas, idx)
}

// parsePseudoElement handles the ::pseudo-element form. Only ::before/::after
// are supported; others reject the selector so declarations do not apply to
// the host.
func parsePseudoElement(sel string, part SelectorPart, idx int) (SelectorPart, int, bool) {
	jdx := idx + doubleColonOffset
	for jdx < len(sel) && sel[jdx] != '(' && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	peVal := strings.ToLower(sel[idx+doubleColonOffset : jdx])
	if peVal != pseudoElemBefore && peVal != pseudoElemAfter {
		return part, idx, false
	}

	if jdx < len(sel) && sel[jdx] == '(' {
		return part, idx, false
	}

	part.PseudoElement = peVal

	return part, jdx, true
}

// parsePseudoClass handles the :pseudo-class form (including CSS2
// single-colon :before/:after).
func parsePseudoClass(sel string, part SelectorPart, insideHas bool, idx int) (SelectorPart, int, bool) {
	jdx := idx + 1
	for jdx < len(sel) && sel[jdx] != '(' && !isCompoundBreak(sel[jdx]) {
		jdx++
	}

	name := strings.ToLower(sel[idx+1 : jdx])
	arg := ""

	var argRaw string

	hasParen := jdx < len(sel) && sel[jdx] == '('
	if hasParen {
		raw, end, ok := takeParenArg(sel, jdx)
		if !ok {
			return part, idx, false
		}

		argRaw = raw
		arg = strings.ToLower(strings.TrimSpace(raw))
		jdx = end
	}

	part, ok := appendCompoundPseudo(part, name, arg, argRaw, hasParen, insideHas)

	return part, jdx, ok
}

// appendCompoundPseudo records the parsed pseudo on the part. When insideHas
// is true, nested :has() and pseudo-elements are rejected as invalid.
func appendCompoundPseudo(part SelectorPart, name, arg, argRaw string, hasParen, insideHas bool) (SelectorPart, bool) {
	if insideHas {
		switch name {
		case pseudoClassHas, pseudoElemBefore, pseudoElemAfter, "first-line", "first-letter":
			return part, false
		}
	}

	switch name {
	case pseudoClassHas, condKindNot, pseudoClassIs, pseudoClassWhere:
		return appendFunctionalPseudo(part, name, argRaw, hasParen, insideHas)
	default:
		return appendSimplePseudo(part, name, arg)
	}
}

// appendFunctionalPseudo handles :has(...), :not(...), :is(...), and :where(...).
func appendFunctionalPseudo(part SelectorPart, name, argRaw string, hasParen, insideHas bool) (SelectorPart, bool) {
	if !hasParen || strings.TrimSpace(argRaw) == "" {
		return part, false
	}

	switch name {
	case pseudoClassHas:
		return appendHasPseudo(part, argRaw)
	case condKindNot:
		return appendNotPseudo(part, argRaw, insideHas)
	case pseudoClassIs, pseudoClassWhere:
		return appendIsWherePseudo(part, name, argRaw, insideHas)
	default:
		return part, false
	}
}

func appendHasPseudo(part SelectorPart, argRaw string) (SelectorPart, bool) {
	lowArg := strings.ToLower(argRaw)
	if strings.Contains(lowArg, ":has(") || strings.Contains(argRaw, "::") {
		return part, false
	}

	rels, ok := parseRelativeSelectorList(argRaw)
	if !ok {
		return part, false
	}

	part.Pseudos = append(part.Pseudos, pseudoClass(pseudoClassHas, "", rels, nil))

	return part, true
}

func appendNotPseudo(part SelectorPart, argRaw string, insideHas bool) (SelectorPart, bool) {
	sels, ok := parseSelectorListStrict(argRaw, insideHas)
	if !ok {
		return part, false
	}

	part.Pseudos = append(part.Pseudos, pseudoClass(condKindNot, "", nil, sels))

	return part, true
}

func appendIsWherePseudo(part SelectorPart, name, argRaw string, insideHas bool) (SelectorPart, bool) {
	if strings.Contains(argRaw, "::") {
		return part, false
	}

	sels, ok := parseSelectorListStrict(argRaw, insideHas)
	if !ok {
		return part, false
	}

	part.Pseudos = append(part.Pseudos, isWherePseudo(name, sels))

	return part, true
}

// appendSimplePseudo handles the non-functional pseudo-classes and the
// CSS2 single-colon pseudo-elements.
func appendSimplePseudo(part SelectorPart, name, arg string) (SelectorPart, bool) {
	switch name {
	case firstChildPseudo, lastChildPseudo, nthChildPseudo,
		firstOfTypePseudo, lastOfTypePseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		part.Pseudos = append(part.Pseudos, pseudoClass(name, arg, nil, nil))
	case "link", "visited":
		// Print semantics: both mean "a[href]" (no browsing history).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, "", nil, nil))
	case "hover", "active", "focus", "target":
		// Accepted for parse/cascade structure but never match in print
		// (static PDF has no pointer/focus/:target fragment state).
		// Keeping them on the compound prevents li:target from
		// degrading to bare `li` (wiki reflist highlight blue).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, "", nil, nil))
	case pseudoElemBefore, pseudoElemAfter:
		// CSS2 single-colon pseudo-elements.
		part.PseudoElement = name
	case "first-line", "first-letter":
		return part, false
	default:
		// Keep unknown pseudos so they do not degrade to the host
		// selector (same class of bug as stripping :target / ::before).
		part.Pseudos = append(part.Pseudos, pseudoClass(name, arg, nil, nil))
	}

	return part, true
}

// pseudoClass builds a PseudoClass with explicit zero Has/Not/Is slices so the
// literal stays exhaustive without per-use nolint comments.
func pseudoClass(name, arg string, has []RelativeSelector, not []Selector) PseudoClass {
	pseudo := PseudoClass{Name: name, Arg: arg, Has: has, Not: not, Is: nil}

	if isNthArgPseudo(name) {
		pseudo.nth = parseNthArg(arg)
	}

	return pseudo
}

func isWherePseudo(name string, sels []Selector) PseudoClass {
	pseudo := pseudoClass(name, "", nil, nil)
	pseudo.Is = sels

	return pseudo
}

func parseAttrSelector(sel string) (AttrSelector, bool) {
	// sel includes brackets: [href], [href="x"], [typeof~='mw:File/Thumb'], [class*="noprint"]
	if len(sel) < 3 || sel[0] != '[' || sel[len(sel)-1] != ']' {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	inner := strings.TrimSpace(sel[1 : len(sel)-1])
	if inner == "" {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// Operator forms: ~= *= ^= $= |= =  (check multi-char before bare =)
	nameEnd, oper := findAttrOperator(inner)

	if nameEnd < 0 {
		if !validIdent(inner) {
			return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
		}

		return AttrSelector{Name: strings.ToLower(inner)}, true //nolint:exhaustruct // intentional zero-value fields
	}

	name := strings.TrimSpace(inner[:nameEnd])
	rawVal := strings.TrimSpace(inner[nameEnd+len(oper):])
	rawVal, ignoreCase := splitAttrIFlag(rawVal)
	val := stripAttrQuotes(rawVal)

	if !validIdent(name) {
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	switch oper {
	case "=", "~=", "*=", "^=", "$=", "|=":
		return AttrSelector{
			Name:       strings.ToLower(name),
			Op:         oper,
			Value:      val,
			IgnoreCase: ignoreCase,
		}, true
	default:
		return AttrSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}
}

// findAttrOperator locates the first operator in an attribute selector's
// inner text and returns its index and operator ("" when absent).
func findAttrOperator(inner string) (int, string) {
	for _, cand := range []string{"~=", "*=", "^=", "$=", "|="} {
		if i := strings.Index(inner, cand); i > 0 {
			return i, cand
		}
	}

	if i := strings.IndexByte(inner, '='); i >= 0 {
		return i, "="
	}

	return -1, ""
}

// stripAttrQuotes removes surrounding matching quotes from an attribute value.
func stripAttrQuotes(val string) string {
	if len(val) >= minQuotedLen {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}

	return val
}

// validIdent reports whether sel is a valid CSS identifier (letters, digits,
// '-', '_', and digits after the first character).
func validIdent(sel string) bool {
	if sel == "" {
		return false
	}

	for i := range len(sel) {
		if !isIdentChar(sel[i]) {
			return false
		}
	}

	if sel[0] >= '0' && sel[0] <= '9' {
		return false
	}

	return true
}

func isCompoundBreak(b byte) bool {
	return b == '#' || b == '.' || b == '[' || b == ':'
}

// splitAttrIFlag strips a trailing Selectors 4 ASCII i / I or s / S flag from
// the attribute-selector value. The s flag is the default exact comparison,
// so both flags return the same ignoreCase=false value.
func splitAttrIFlag(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, false
	}

	if raw[0] == '"' || raw[0] == '\'' {
		return splitQuotedAttrIFlag(raw)
	}

	return splitIdentAttrIFlag(raw)
}

func splitQuotedAttrIFlag(raw string) (string, bool) {
	quote := raw[0]
	closeAt := strings.LastIndexByte(raw, quote)

	if closeAt <= 0 {
		return raw, false
	}

	rest := strings.TrimSpace(raw[closeAt+1:])
	if isAttrIgnoreCaseFlag(rest) {
		return raw[:closeAt+1], true
	}

	if !isAttrCaseSensitiveFlag(rest) {
		return raw, false
	}

	return raw[:closeAt+1], false
}

func splitIdentAttrIFlag(raw string) (string, bool) {
	end := len(raw) - 1
	for end >= 0 && !isClassSpace(raw[end]) {
		end--
	}

	if end < 0 {
		return raw, false
	}

	flag := raw[end+1:]
	val := strings.TrimSpace(raw[:end])

	if val == "" {
		return raw, false
	}

	if isAttrIgnoreCaseFlag(flag) {
		return val, true
	}

	if isAttrCaseSensitiveFlag(flag) {
		return val, false
	}

	return raw, false
}

func isAttrIgnoreCaseFlag(s string) bool {
	return len(s) == 1 && (s[0] == 'i' || s[0] == 'I')
}

func isAttrCaseSensitiveFlag(s string) bool {
	return len(s) == 1 && (s[0] == 's' || s[0] == 'S')
}
