package css

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// matchingParen returns the index of the ')' that matches str[open]=='('.
// Quoted spans are skipped. ok is false if unbalanced or open is not '('.
// Shared by media features, container conditions, and :has()/:not() args.
func matchingParen(str string, open int) (int, bool) {
	if open >= len(str) || str[open] != '(' {
		return -1, false
	}

	depth := 0

	for idx := open; idx < len(str); idx++ {
		switch str[idx] {
		case '"', '\'':
			idx = skipQuoted(str, idx, str[idx])
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx, true
			}
		}
	}

	return -1, false
}

// takeParenArg parses a (...) argument whose '(' sits at open. Returns the
// inner text, the index just past the matching ')', and ok.
func takeParenArg(str string, open int) (string, int, bool) {
	end, ok := matchingParen(str, open)
	if !ok {
		return "", open, false
	}

	return str[open+1 : end], end + 1, true
}

// takeParen parses a leading (...) from str (after trim). Returns inner text
// and the remainder after the matching ')'.
func takeParen(str string) (string, string, bool) {
	str = strings.TrimSpace(str)
	if !strings.HasPrefix(str, "(") {
		return "", str, false
	}

	inner, end, ok := takeParenArg(str, 0)
	if !ok {
		return "", str, false
	}

	return inner, str[end:], true
}

func parseSelectorListStrict(s string, insideHas bool) ([]Selector, bool) {
	parts := splitTopLevel(s, ',')

	out := make([]Selector, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}

		sel, ok := parseSelectorCtx(part, insideHas)
		if !ok {
			return nil, false
		}

		out = append(out, sel)
	}

	return out, len(out) > 0
}

func parseSelectorCtx(str string, insideHas bool) (Selector, bool) {
	var sel Selector

	str = strings.TrimSpace(str)
	if str == "" {
		return sel, false
	}

	parts := splitSelectorChain(str)
	for idx, ch := range parts {
		if ch == ">" || ch == "+" || ch == "~" || ch == " " {
			continue
		}

		part, ok := parseCompoundCtx(ch, insideHas)
		if !ok {
			return sel, false
		}

		if len(sel.Parts) > 0 {
			part.Combinator = combinatorFor(parts[idx-1])
		}

		sel.Parts = append(sel.Parts, part)
	}

	return sel, len(sel.Parts) > 0
}

// combinatorFor maps a chain separator to the combinator stored on the part
// that follows it.
func combinatorFor(sep string) string {
	switch sep {
	case ">", "+", "~":
		return sep
	default:
		return " "
	}
}

func parseRelativeSelectorList(s string) ([]RelativeSelector, bool) {
	parts := splitTopLevel(s, ',')

	out := make([]RelativeSelector, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}

		rs, ok := parseRelativeSelector(part)
		if !ok {
			return nil, false
		}

		out = append(out, rs)
	}

	return out, len(out) > 0
}

func parseRelativeSelector(str string) (RelativeSelector, bool) {
	str = strings.TrimSpace(str)
	if str == "" {
		return RelativeSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	lead := " "

	switch {
	case strings.HasPrefix(str, ">"):
		lead = ">"
		str = strings.TrimSpace(str[1:])
	case strings.HasPrefix(str, "+"):
		lead = "+"
		str = strings.TrimSpace(str[1:])
	case strings.HasPrefix(str, "~"):
		lead = "~"
		str = strings.TrimSpace(str[1:])
	}

	if str == "" {
		return RelativeSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	sel, ok := parseSelectorCtx(str, true)
	if !ok {
		return RelativeSelector{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return RelativeSelector{Leading: lead, Parts: sel.Parts}, true
}

// matchRelative reports whether relative selector rel matches when anchored at subject.
func matchRelative(rel RelativeSelector, subject *html.Node) bool {
	if subject == nil || len(rel.Parts) == 0 {
		return false
	}

	sel := Selector{Parts: rel.Parts}

	switch rel.Leading {
	case "+":
		sib := nextElementSibling(subject)
		if sib == nil {
			return false
		}

		return matchRelativeFrom(sel, sib)
	case "~":
		for sib := nextElementSibling(subject); sib != nil; sib = nextElementSibling(sib) {
			if matchRelativeFrom(sel, sib) {
				return true
			}
		}

		return false
	case ">":
		return matchRelativeDescendant(sel, subject, true)
	default: // descendant
		return matchRelativeDescendant(sel, subject, false)
	}
}

// matchRelativeDescendant reports whether some descendant d of subject matches
// sel with its leftmost-match element anchored at subject (directly when
// direct is true, or anywhere beneath it otherwise).
func matchRelativeDescendant(sel Selector, subject *html.Node, direct bool) bool {
	for _, d := range elementDescendants(subject) {
		if !Match(sel, d) {
			continue
		}

		left := leftmostMatch(sel, d)
		if left == nil {
			continue
		}

		if (direct && left.Parent == subject) || (!direct && isElementDescendant(left, subject)) {
			return true
		}
	}

	return false
}

// matchRelativeFrom checks whether sel matches with its leftmost compound equal
// to anchor (the element reached via + / ~ from the subject).
func matchRelativeFrom(sel Selector, anchor *html.Node) bool {
	if len(sel.Parts) == 1 {
		return Match(sel, anchor)
	}

	if !matchPart(sel.Parts[0], anchor) {
		return false
	}

	cands := append([]*html.Node{anchor}, elementDescendants(anchor)...)
	for _, d := range cands {
		if Match(sel, d) && leftmostMatch(sel, d) == anchor {
			return true
		}
	}

	return false
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

func elementDescendants(count *html.Node) []*html.Node {
	var out []*html.Node

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		for _, cur := range node.Children {
			if cur.Type != html.ElementNode {
				continue
			}

			out = append(out, cur)
			walk(cur)
		}
	}
	if count != nil {
		walk(count)
	}

	return out
}

func isElementDescendant(n, ancestor *html.Node) bool {
	if n == nil || ancestor == nil || n == ancestor {
		return false
	}

	for p := n.Parent; p != nil; p = p.Parent {
		if p == ancestor {
			return true
		}
	}

	return false
}

func maxRelativeSpecificity(rels []RelativeSelector) (int, int, int) {
	maxA, maxB, maxC := 0, 0, 0

	for i, rs := range rels {
		sa, sb, sc := Specificity(Selector{Parts: rs.Parts})
		if i == 0 || betterSpec(sa, sb, sc, maxA, maxB, maxC) {
			maxA, maxB, maxC = sa, sb, sc
		}
	}

	return maxA, maxB, maxC
}

func maxSelectorSpecificity(sels []Selector) (int, int, int) {
	maxA, maxB, maxC := 0, 0, 0

	for i, sel := range sels {
		sa, sb, sc := Specificity(sel)
		if i == 0 || betterSpec(sa, sb, sc, maxA, maxB, maxC) {
			maxA, maxB, maxC = sa, sb, sc
		}
	}

	return maxA, maxB, maxC
}

// betterSpec reports whether (a1,b1,c1) is more specific than (a2,b2,c2).
func betterSpec(a1, b1, c1, a2, b2, c2 int) bool {
	return a1 > a2 || (a1 == a2 && b1 > b2) || (a1 == a2 && b1 == b2 && c1 > c2)
}
