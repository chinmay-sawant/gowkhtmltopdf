package css

import (
	"strings"

	"gowkhtmltopdf/internal/html"
)

// matchingParen returns the index of the ')' that matches s[open]=='('.
// Quoted spans are skipped. ok is false if unbalanced or open is not '('.
// Shared by media features, container conditions, and :has()/:not() args.
func matchingParen(s string, open int) (close int, ok bool) {
	if open >= len(s) || s[open] != '(' {
		return -1, false
	}
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			q := s[i]
			i++
			for i < len(s) && s[i] != q {
				if s[i] == '\\' && i+1 < len(s) {
					i++
				}
				i++
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return -1, false
}

// takeParenArg parses a (...) argument whose '(' sits at open. Returns the
// inner text, the index just past the matching ')', and ok.
func takeParenArg(s string, open int) (inner string, end int, ok bool) {
	close, ok := matchingParen(s, open)
	if !ok {
		return "", open, false
	}
	return s[open+1 : close], close + 1, true
}

// takeParen parses a leading (...) from s (after trim). Returns inner text and
// the remainder after the matching ')'.
func takeParen(s string) (inner, rest string, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return "", s, false
	}
	inner, end, ok := takeParenArg(s, 0)
	if !ok {
		return "", s, false
	}
	return inner, s[end:], true
}

func parseSelectorListStrict(s string, insideHas bool) ([]Selector, bool) {
	var out []Selector
	for _, part := range splitTopLevel(s, ',') {
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

func parseSelectorCtx(s string, insideHas bool) (Selector, bool) {
	var sel Selector
	s = strings.TrimSpace(s)
	if s == "" {
		return sel, false
	}
	parts := splitSelectorChain(s)
	for i, ch := range parts {
		if ch == ">" || ch == "+" || ch == "~" || ch == " " {
			continue
		}
		part, ok := parseCompoundCtx(ch, insideHas)
		if !ok {
			return sel, false
		}
		if len(sel.Parts) > 0 {
			switch parts[i-1] {
			case ">", "+", "~":
				part.Combinator = parts[i-1]
			default:
				part.Combinator = " "
			}
		}
		sel.Parts = append(sel.Parts, part)
	}
	return sel, len(sel.Parts) > 0
}

func parseRelativeSelectorList(s string) ([]RelativeSelector, bool) {
	var out []RelativeSelector
	for _, part := range splitTopLevel(s, ',') {
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

func parseRelativeSelector(s string) (RelativeSelector, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return RelativeSelector{}, false
	}
	lead := " "
	switch {
	case strings.HasPrefix(s, ">"):
		lead = ">"
		s = strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "+"):
		lead = "+"
		s = strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "~"):
		lead = "~"
		s = strings.TrimSpace(s[1:])
	}
	if s == "" {
		return RelativeSelector{}, false
	}
	sel, ok := parseSelectorCtx(s, true)
	if !ok {
		return RelativeSelector{}, false
	}
	return RelativeSelector{Leading: lead, Parts: sel.Parts}, true
}

// matchRelative reports whether relative selector rs matches when anchored at subject.
func matchRelative(rs RelativeSelector, subject *html.Node) bool {
	if subject == nil || len(rs.Parts) == 0 {
		return false
	}
	sel := Selector{Parts: rs.Parts}
	switch rs.Leading {
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
		for _, d := range elementDescendants(subject) {
			if !Match(sel, d) {
				continue
			}
			left := leftmostMatch(sel, d)
			if left != nil && left.Parent == subject {
				return true
			}
		}
		return false
	default: // descendant
		for _, d := range elementDescendants(subject) {
			if !Match(sel, d) {
				continue
			}
			left := leftmostMatch(sel, d)
			if left != nil && isElementDescendant(left, subject) {
				return true
			}
		}
		return false
	}
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
// if the selector does not match n.
func leftmostMatch(sel Selector, n *html.Node) *html.Node {
	if n == nil || n.Type != html.ElementNode || len(sel.Parts) == 0 {
		return nil
	}
	if !matchPart(sel.Parts[len(sel.Parts)-1], n) {
		return nil
	}
	cur := n
	// Combinator is stored on the right-hand part of each pair (how that
	// part attaches to the previous). Walk left using Parts[i+1].Combinator.
	for i := len(sel.Parts) - 2; i >= 0; i-- {
		part := sel.Parts[i]
		switch sel.Parts[i+1].Combinator {
		case ">":
			cur = cur.Parent
			if cur == nil || cur.Type != html.ElementNode || !matchPart(part, cur) {
				return nil
			}
		case "+":
			prev := previousElementSibling(cur)
			if prev == nil || !matchPart(part, prev) {
				return nil
			}
			cur = prev
		case "~":
			found := false
			for sib := previousElementSibling(cur); sib != nil; sib = previousElementSibling(sib) {
				if matchPart(part, sib) {
					cur = sib
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		default: // descendant
			cur = cur.Parent
			for cur != nil && (cur.Type != html.ElementNode || !matchPart(part, cur)) {
				cur = cur.Parent
			}
			if cur == nil {
				return nil
			}
		}
	}
	return cur
}

func elementDescendants(n *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		for _, c := range node.Children {
			if c.Type != html.ElementNode {
				continue
			}
			out = append(out, c)
			walk(c)
		}
	}
	if n != nil {
		walk(n)
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

func maxRelativeSpecificity(rels []RelativeSelector) (a, b, c int) {
	for i, rs := range rels {
		sa, sb, sc := Specificity(Selector{Parts: rs.Parts})
		if i == 0 || betterSpec(sa, sb, sc, a, b, c) {
			a, b, c = sa, sb, sc
		}
	}
	return a, b, c
}

func maxSelectorSpecificity(sels []Selector) (a, b, c int) {
	for i, sel := range sels {
		sa, sb, sc := Specificity(sel)
		if i == 0 || betterSpec(sa, sb, sc, a, b, c) {
			a, b, c = sa, sb, sc
		}
	}
	return a, b, c
}

// betterSpec reports whether (a1,b1,c1) is more specific than (a2,b2,c2).
func betterSpec(a1, b1, c1, a2, b2, c2 int) bool {
	return a1 > a2 || (a1 == a2 && b1 > b2) || (a1 == a2 && b1 == b2 && c1 > c2)
}
