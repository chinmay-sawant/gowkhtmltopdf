package css

import (
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// siblingInfo aggregates all sibling metrics for a node in one cache lookup.
type siblingInfo struct {
	prev          *html.Node
	next          *html.Node
	index         int
	total         int
	typeIndex     int
	typeLastIndex int
}

type parentSibCache struct {
	elemIdx     map[*html.Node]int
	prev        map[*html.Node]*html.Node
	next        map[*html.Node]*html.Node
	typeIdx     map[*html.Node]int
	typeLastIdx map[*html.Node]int
	total       int
}

var (
	sibMu    sync.RWMutex                           //nolint:gochecknoglobals // guards the sibling cache below
	sibCache = make(map[*html.Node]*parentSibCache) //nolint:gochecknoglobals // process-wide sibling index cache
)

func getParentCache(parent *html.Node) *parentSibCache {
	if parent == nil {
		return nil
	}

	sibMu.RLock()
	cached := sibCache[parent]
	sibMu.RUnlock()

	if cached != nil {
		return cached
	}

	cached = buildParentCache(parent)

	sibMu.Lock()
	if existing := sibCache[parent]; existing != nil {
		sibMu.Unlock()

		return existing
	}

	sibCache[parent] = cached
	sibMu.Unlock()

	return cached
}

func buildParentCache(parent *html.Node) *parentSibCache {
	elems := make([]*html.Node, 0, len(parent.Children))

	for _, child := range parent.Children {
		if child.Type == html.ElementNode {
			elems = append(elems, child)
		}
	}

	total := len(elems)
	elemIdx := make(map[*html.Node]int, total)
	prev := make(map[*html.Node]*html.Node, total)
	next := make(map[*html.Node]*html.Node, total)

	for idx, elem := range elems {
		elemIdx[elem] = idx + 1

		if idx > 0 {
			prev[elem] = elems[idx-1]
		}

		if idx+1 < total {
			next[elem] = elems[idx+1]
		}
	}

	tagTotals := make(map[string]int, total)

	for _, elem := range elems {
		key := strings.ToLower(elem.Name)
		tagTotals[key]++
	}

	typeIdx := make(map[*html.Node]int, total)
	typeLastIdx := make(map[*html.Node]int, total)
	seen := make(map[string]int, len(tagTotals))

	for _, elem := range elems {
		key := strings.ToLower(elem.Name)
		seen[key]++
		idx := seen[key]
		typeIdx[elem] = idx
		typeLastIdx[elem] = tagTotals[key] - idx + 1
	}

	return &parentSibCache{
		elemIdx:     elemIdx,
		prev:        prev,
		next:        next,
		typeIdx:     typeIdx,
		typeLastIdx: typeLastIdx,
		total:       total,
	}
}

func getSiblingInfo(node *html.Node) siblingInfo {
	if node == nil || node.Parent == nil {
		// No parent means no sibling edges; only the 1-based indices
		// carry meaning here.
		return siblingInfo{ //nolint:exhaustruct // nil prev/next means uncached edges
			index: 1, total: 1, typeIndex: 1, typeLastIndex: 1,
		}
	}

	cached := getParentCache(node.Parent)
	if cached == nil {
		return siblingInfo{} //nolint:exhaustruct // zero info for uncached parent
	}

	return siblingInfo{
		prev:          cached.prev[node],
		next:          cached.next[node],
		index:         cached.elemIdx[node],
		total:         cached.total,
		typeIndex:     cached.typeIdx[node],
		typeLastIndex: cached.typeLastIdx[node],
	}
}

// Match reports whether the selector matches the element node. Matching runs
// right to left: the last part must match n, earlier parts must match
// ancestors/siblings per their combinators. Implemented via leftmostMatch
// (same combinator walk; Match only needs success/failure).
func Match(s Selector, n *html.Node) bool {
	return leftmostMatch(s, n) != nil
}

// MatchPseudo reports whether s selects the ::before or ::after pseudo-element
// of n (pe is "before" or "after").
func MatchPseudo(sel Selector, count *html.Node, pseudo string) bool {
	if count == nil || pseudo == "" || len(sel.Parts) == 0 {
		return false
	}

	if sel.Parts[len(sel.Parts)-1].PseudoElement != pseudo {
		return false
	}

	return matchPseudoWalk(sel, count)
}

// matchPseudoWalk mirrors leftmostMatch with the final part's PseudoElement
// treated as cleared, without copying the parts slice (the host element must
// match the pseudo's compound, not the pseudo itself).
func matchPseudoWalk(sel Selector, node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode || len(sel.Parts) == 0 {
		return false
	}

	last := sel.Parts[len(sel.Parts)-1]
	last.PseudoElement = ""

	if !matchPart(last, node) {
		return false
	}

	cur := node

	const prevPartOffset = 2 // walk left: last part is host, start at len-2

	for i := len(sel.Parts) - prevPartOffset; i >= 0; i-- {
		next := leftmostStep(sel.Parts[i+1].Combinator, sel.Parts[i], cur)
		if next == nil {
			return false
		}

		cur = next
	}

	return true
}

// leftmostMatch walks the selector combinator chain right-to-left (same walk
// as Match) and returns the element that matched the leftmost compound, or nil
// if the selector does not match node.
func leftmostMatch(sel Selector, node *html.Node) *html.Node {
	if node == nil || node.Type != html.ElementNode || len(sel.Parts) == 0 {
		return nil
	}

	if !matchPart(sel.Parts[len(sel.Parts)-1], node) {
		return nil
	}

	cur := node
	// Combinator is stored on the right-hand part of each pair (how that
	// part attaches to the previous). Walk left using Parts[i+1].Combinator.
	const prevPartOffset = 2 // walk left: last part is host, start at len-2

	for i := len(sel.Parts) - prevPartOffset; i >= 0; i-- {
		next := leftmostStep(sel.Parts[i+1].Combinator, sel.Parts[i], cur)
		if next == nil {
			return nil
		}

		cur = next
	}

	return cur
}

// leftmostStep advances cur one step left through the combinator chain: it
// returns the element that must match part, or nil when none exists.
func leftmostStep(combinator string, part SelectorPart, cur *html.Node) *html.Node {
	switch combinator {
	case ">":
		return matchLeftChild(part, cur)
	case "+":
		return matchLeftAdjacent(part, cur)
	case "~":
		return matchLeftSibling(part, cur)
	default: // descendant
		return matchLeftAncestor(part, cur)
	}
}

// matchLeftChild returns cur's element parent when it matches part, else nil.
func matchLeftChild(part SelectorPart, cur *html.Node) *html.Node {
	cur = cur.Parent
	if cur == nil || cur.Type != html.ElementNode || !matchPart(part, cur) {
		return nil
	}

	return cur
}

// matchLeftAdjacent returns cur's previous element sibling when it matches
// part, else nil.
func matchLeftAdjacent(part SelectorPart, cur *html.Node) *html.Node {
	prev := previousElementSibling(cur)
	if prev == nil || !matchPart(part, prev) {
		return nil
	}

	return prev
}

// matchLeftSibling returns the nearest previous element sibling of cur that
// matches part, else nil.
func matchLeftSibling(part SelectorPart, cur *html.Node) *html.Node {
	for sib := previousElementSibling(cur); sib != nil; sib = previousElementSibling(sib) {
		if matchPart(part, sib) {
			return sib
		}
	}

	return nil
}

// matchLeftAncestor returns the nearest element ancestor of cur that matches
// part, else nil.
func matchLeftAncestor(part SelectorPart, cur *html.Node) *html.Node {
	cur = cur.Parent
	for cur != nil && (cur.Type != html.ElementNode || !matchPart(part, cur)) {
		cur = cur.Parent
	}

	return cur
}

// matchPart matches one compound against an element.
func matchPart(part SelectorPart, node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	// ::before/::after never match the host element (declarations apply to
	// generated pseudo boxes via MatchPseudo).
	if part.PseudoElement != "" {
		return false
	}

	if part.Tag != "*" && !strings.EqualFold(part.Tag, node.Name) {
		return false
	}

	if part.ID != "" && node.Attribute("id") != part.ID {
		return false
	}

	if !hasClasses(part, node) {
		return false
	}

	if !matchAttrs(part, node) {
		return false
	}

	return matchPseudos(part.Pseudos, node)
}

// matchPseudos reports whether every pseudo-class of the part matches node.
func matchPseudos(pseudos []PseudoClass, node *html.Node) bool {
	for _, pseudo := range pseudos {
		if !matchPseudo(pseudo, node) {
			return false
		}
	}

	return true
}

// hasClasses reports whether the element carries every class of the part.
func hasClasses(part SelectorPart, node *html.Node) bool {
	if len(part.Classes) == 0 {
		return true
	}

	for _, c := range part.Classes {
		if !hasClassToken(node.Attribute("class"), c) {
			return false
		}
	}

	return true
}

// matchAttrs matches every attribute selector of the part against node.
func matchAttrs(part SelectorPart, node *html.Node) bool {
	for _, arg := range part.Attrs {
		val, found := "", false
		if node.Attrs != nil {
			val, found = node.Attrs[arg.Name]
		}

		if arg.Op == "" {
			if !found {
				return false
			}

			continue
		}

		if !found {
			return false
		}

		if !attrValueMatches(arg, val) {
			return false
		}
	}

	return true
}

// attrValueMatches evaluates one attribute operator against a value.
// ignoreCase is the Selectors 4 ASCII i flag; the selector side is pre-lowered
// at parse time (AttrSelector.valueLower) and only the element side is lowered
// per call.
func attrValueMatches(arg AttrSelector, val string) bool {
	if arg.IgnoreCase {
		val = strings.ToLower(val)
	}

	want := attrWantValue(arg)

	switch arg.Op {
	case "=":
		return val == want
	case "~=":
		return containsWord(val, want)
	case "*=", "^=", "$=", "|=":
		if want == "" {
			return false
		}

		return matchSubstringOp(arg.Op, val, want)
	}

	return false
}

// attrWantValue resolves the comparison value, using the parse-time lowered
// cache when available and lowering on the fly for hand-built selectors.
func attrWantValue(arg AttrSelector) string {
	if !arg.IgnoreCase {
		return arg.Value
	}

	if arg.valueLower != "" || arg.Value == "" {
		return arg.valueLower
	}

	// Hand-built selector without the parse-time cache.
	return strings.ToLower(arg.Value)
}

// matchSubstringOp evaluates one substring attribute operator against val.
// want is non-empty; the caller rejects the empty case (which matches
// nothing) before dispatching here.
func matchSubstringOp(op, val, want string) bool {
	switch op {
	case "*=":
		return strings.Contains(val, want)
	case "^=":
		return strings.HasPrefix(val, want)
	case "$=":
		return strings.HasSuffix(val, want)
	}
	// |= : exact match or value followed by a hyphen (HTML lang / BCP47-style).
	return val == want || strings.HasPrefix(val, want+"-")
}

// containsWord reports whether want (a single space-free word) is one of the
// space-separated words of val. Tokenizes val without allocating a fields
// slice (same whitespace definition and Unicode fallback as hasClassToken).
//
//nolint:cyclop // two-zone token walk (ASCII, then Unicode fallback) stays linear
func containsWord(val, want string) bool {
	if want == "" || strings.Contains(want, " ") {
		return false
	}

	for start := 0; start < len(val); {
		for start < len(val) && isClassSpace(val[start]) {
			start++
		}

		end := start
		for end < len(val) && !isClassSpace(val[end]) {
			if val[end] >= nonASCIIStart {
				return hasUnicodeClassToken(val, want)
			}

			end++
		}

		if start < end && val[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func matchPseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstChildPseudo, lastChildPseudo, nthChildPseudo, "root",
		firstOfTypePseudo, lastOfTypePseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return matchTreePseudo(pseudo, node)
	case pseudoClassHas:
		return matchAnyRelative(pseudo.Has, node)
	case condKindNot:
		return matchNone(pseudo.Not, node)
	case pseudoClassIs, pseudoClassWhere:
		return matchAnySelector(pseudo.Is, node)
	case "link", "visited":
		// Print has no link history, so both match any anchor with an href.
		return isLinkAnchor(node)
	default:
		// :hover/:active/:focus/:target never match in print. Unknown
		// pseudo-classes never match either (kept on the compound so
		// selectors do not degrade to the host).
		return false
	}
}

// matchTreePseudo handles tree-structural pseudo-classes.
func matchTreePseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstChildPseudo:
		return getSiblingInfo(node).prev == nil
	case lastChildPseudo:
		return getSiblingInfo(node).next == nil
	case nthChildPseudo:
		return matchNth(pseudo.nth, getSiblingInfo(node).index)
	case "root":
		return isRootElement(node)
	default:
		return matchOfTypePseudo(pseudo, node)
	}
}

// matchAnyRelative reports whether any relative selector applies to node.
func matchAnyRelative(rels []RelativeSelector, node *html.Node) bool {
	for _, rs := range rels {
		if matchRelative(rs, node) {
			return true
		}
	}

	return false
}

// matchNone reports whether no selector of the list matches node.
func matchNone(sels []Selector, node *html.Node) bool {
	for _, sel := range sels {
		if Match(sel, node) {
			return false
		}
	}

	return true
}

// matchAnySelector reports whether any selector of the list matches node.
func matchAnySelector(sels []Selector, node *html.Node) bool {
	for _, sel := range sels {
		if Match(sel, node) {
			return true
		}
	}

	return false
}

// isRootElement reports whether node is the document element (html in HTML).
// html.Parse wraps the tree in a synthetic ElementNode named "#document" -
// that must not match, and must not block <html> from matching.
func isRootElement(node *html.Node) bool {
	if node.Type != html.ElementNode || node.Name == "#document" || strings.HasPrefix(node.Name, "#") {
		return false
	}

	for p := node.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Name != "#document" && !strings.HasPrefix(p.Name, "#") {
			return false
		}
	}

	return true
}

// isLinkAnchor reports whether n is an <a> element with a non-empty href
// (any scheme, including "#" fragments and relative paths).
func isLinkAnchor(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode || !strings.EqualFold(n.Name, "a") {
		return false
	}

	href := strings.TrimSpace(n.Attribute("href"))

	return href != ""
}

func previousElementSibling(count *html.Node) *html.Node {
	if count == nil || count.Parent == nil {
		return nil
	}

	if cached := getParentCache(count.Parent); cached != nil {
		return cached.prev[count]
	}

	return nil
}

func nextElementSibling(count *html.Node) *html.Node {
	if count == nil || count.Parent == nil {
		return nil
	}

	if cached := getParentCache(count.Parent); cached != nil {
		return cached.next[count]
	}

	return nil
}

// nthKind discriminates the pre-parsed :nth-child() argument forms.
type nthKind int

const (
	nthInvalid nthKind = iota // unparseable, or not an nth-* pseudo
	nthOdd
	nthEven
	nthInt
	nthAnB
)

// nthForm is a :nth-child() argument parsed at selector-parse time so that
// matching is pure integer arithmetic (see matchNth).
type nthForm struct {
	kind nthKind `exhaustruct:"optional"` // nthInt: exact index; nthAnB: coefficient
	a    int     `exhaustruct:"optional"` // nthAnB: the constant
	b    int     `exhaustruct:"optional"`
}

// parseNthArg pre-parses a :nth-child() argument into the form matchNth
// evaluates. The argument is already lower-cased and trimmed at parse time;
// normalizing again here keeps the acceptance rules identical to the former
// string-based parser, including the never-match fallback.
func parseNthArg(arg string) nthForm {
	arg = strings.TrimSpace(strings.ToLower(arg))
	if arg == "" {
		return nthForm{}
	}

	if arg == "odd" {
		return nthForm{kind: nthOdd}
	}

	if arg == "even" {
		return nthForm{kind: nthEven}
	}
	// plain integer
	if n, err := strconv.Atoi(arg); err == nil {
		return nthForm{kind: nthInt, a: n}
	}
	// an+b / n+b / -n+b / an
	if !strings.Contains(arg, "n") {
		return nthForm{}
	}

	a, b, ok := parseAnPlusB(arg)
	if !ok {
		return nthForm{}
	}

	return nthForm{kind: nthAnB, a: a, b: b}
}

// matchNth implements :nth-child(an+b) / odd / even for 1-based index,
// evaluating the pre-parsed argument form.
func matchNth(nth nthForm, index int) bool {
	switch nth.kind {
	case nthOdd:
		return index%2 == 1
	case nthEven:
		return index%2 == 0
	case nthInt:
		return index == nth.a
	case nthAnB:
		if nth.a == 0 {
			return index == nth.b
		}

		if (index-nth.b)%nth.a != 0 {
			return false
		}

		k := (index - nth.b) / nth.a

		return k >= 0
	case nthInvalid:
		return false
	}

	return false
}

// parseAnPlusB parses the "an+b" form of a nth-child argument: "an", "n+b",
// "-n+b", or "b". Returns ok=false for anything unrecognized.
func parseAnPlusB(arg string) (int, int, bool) {
	parts := strings.SplitN(arg, "n", anPlusBSplitParts)
	asVal := strings.TrimSpace(parts[0])

	specA := 1
	if asVal == "-" {
		specA = -1
	} else if asVal != "" && asVal != "+" {
		parsed, err := strconv.Atoi(asVal)
		if err != nil {
			return 0, 0, false
		}

		specA = parsed
	}

	if len(parts) != anPlusBSplitParts || strings.TrimSpace(parts[1]) == "" {
		return specA, 0, true
	}

	specB, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}

	return specA, specB, true
}

// hasClassToken reports whether want is one whitespace-separated class token.
// The ASCII path covers HTML/CSS's normal class syntax without allocating a
// token slice or map. Non-ASCII whitespace falls back to the same Unicode
// whitespace behavior previously provided by strings.Fields.
func hasClassToken(value, want string) bool {
	if want == "" {
		return false
	}

	for start := 0; start < len(value); {
		for start < len(value) && isClassSpace(value[start]) {
			start++
		}

		end := start
		for end < len(value) && !isClassSpace(value[end]) {
			if value[end] >= nonASCIIStart {
				return hasUnicodeClassToken(value, want)
			}

			end++
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func hasUnicodeClassToken(value, want string) bool {
	for start := 0; start < len(value); {
		for start < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[start:])
			if !unicode.IsSpace(runeValue) {
				break
			}

			start += size
		}

		end := start
		for end < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[end:])
			if unicode.IsSpace(runeValue) {
				break
			}

			end += size
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func isClassSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\v' || value == '\f' || value == '\r'
}

func isNthArgPseudo(name string) bool {
	switch name {
	case nthChildPseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return true
	default:
		return false
	}
}

// matchOfTypePseudo handles :first-of-type, :last-of-type, :nth-of-type(),
// and :nth-last-of-type(). Index is 1-based among element siblings with the
// same tag (HTML tag names compare case-insensitively).
func matchOfTypePseudo(pseudo PseudoClass, node *html.Node) bool {
	info := getSiblingInfo(node)

	switch pseudo.Name {
	case firstOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, info.typeIndex)
	case lastOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, info.typeLastIndex)
	case nthOfTypePseudo:
		return matchNth(pseudo.nth, info.typeIndex)
	case nthLastOfTypePseudo:
		return matchNth(pseudo.nth, info.typeLastIndex)
	default:
		return false
	}
}
