// Package outline builds the document outline (PDF bookmarks) from heading
// elements and their layout locations: collection, location lookup, sorting,
// level-stack tree construction, and the wkhtmltopdf-compatible --dump-outline
// XML. It is pure tree construction — converting canvas coordinates into PDF
// page space and wiring page object refs is the caller's job (internal/convert
// does it where the page geometry is known).
package outline

import (
	"sort"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
)

// Heading is one outline entry: an h1..h6 element plus its layout location.
// Page/X/Y/W/H are filled from an ElementLocation; X/Y/W/H are canvas
// coordinates (y grows downward from the top of the page content area), Page
// is the 0-based page index the element landed on (object-local; callers
// rebase it to document pages).
type Heading struct {
	Node   *html.Node
	Title  string
	Level  int // 1..6
	Page   int
	X, Y   float64
	W, H   float64
	Anchor string // synthetic __WKANCHOR_<base36>, stable across runs
}

// headingLevel maps an element name to its outline level, 0 when it is not a
// heading. h1..h6 are accepted (h7..h9 are not part of the HTML vocabulary).
func headingLevel(name string) int {
	switch name {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	}
	return 0
}

// CollectHeadings walks root in document order and returns one Heading per
// h1..h6 element with its whitespace-collapsed text title. Location fields
// are zero until Lookup runs.
func CollectHeadings(root *html.Node) []*Heading {
	var out []*Heading
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if lvl := headingLevel(n.Name); lvl > 0 {
			out = append(out, &Heading{
				Node:  n,
				Title: collapseWS(n.TextContent()),
				Level: lvl,
			})
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// Lookup fills Page/X/Y/W/H of each heading whose node appears in locs
// (matched by node pointer) and returns the headings that have a location.
// Headings without a location are skipped: they were laid out as display:none
// or never emitted a box.
func Lookup(headings []*Heading, locs []layout.ElementLocation) []*Heading {
	byNode := make(map[*html.Node]layout.ElementLocation, len(locs))
	for _, l := range locs {
		byNode[l.Node] = l
	}
	out := make([]*Heading, 0, len(headings))
	for _, h := range headings {
		l, ok := byNode[h.Node]
		if !ok {
			continue
		}
		h.Page, h.X, h.Y, h.W, h.H = l.Page, l.X, l.Y, l.W, l.H
		out = append(out, h)
	}
	return out
}

// AssignAnchors gives every heading a stable synthetic anchor of the form
// __WKANCHOR_<base36 index> in document order. The TOC uses these anchors for
// its forward links, so they must not depend on the outline depth or on any
// page-numbering iteration.
func AssignAnchors(headings []*Heading) {
	for i, h := range headings {
		h.Anchor = "__WKANCHOR_" + strconv.FormatInt(int64(i), 36)
	}
}

// Node is one node of the built outline tree.
type Node struct {
	Heading  *Heading
	Children []*Node
}

// Options controls tree construction.
type Options struct {
	// MaxDepth is the deepest heading level kept; headings (after clamping)
	// deeper than MaxDepth are dropped from the tree. 0 keeps everything.
	MaxDepth int
	// Exclude drops headings whose node matches any selector.
	Exclude []css.Selector
	// Include is an extra per-node gate (e.g. object-level outline flags);
	// nil includes every heading that survives Exclude.
	Include func(n *html.Node) bool
}

// BuildTree sorts headings by (page, y, x) — within a page, y-down order so a
// heading higher on the page comes first — and assembles them into a
// level-stack tree whose root children are the top-level headings.
//
// Heuristic for non-monotonic sequences: a heading at level L becomes a child
// of the nearest previous heading with level < L; when the level jumps up by
// more than one (e.g. an h4 directly under an h2, with no h3 in between) the
// heading's depth is clamped to previous+1 so the tree stays connected and
// never skips levels. Titles, anchors and sort order are unaffected by the
// clamp.
func BuildTree(headings []*Heading, opts Options) *Node {
	sel := make([]*Heading, 0, len(headings))
	for _, h := range headings {
		if opts.Include != nil && !opts.Include(h.Node) {
			continue
		}
		if matchAny(opts.Exclude, h.Node) {
			continue
		}
		sel = append(sel, h)
	}
	sort.SliceStable(sel, func(i, j int) bool {
		a, b := sel[i], sel[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})

	root := &Node{}
	stack := []*Node{root}
	stackLevel := []int{0}
	for _, h := range sel {
		lvl := h.Level
		if lvl > stackLevel[len(stackLevel)-1]+1 {
			lvl = stackLevel[len(stackLevel)-1] + 1 // clamp non-monotonic jump
		}
		if opts.MaxDepth > 0 && lvl > opts.MaxDepth {
			continue
		}
		for stackLevel[len(stackLevel)-1] >= lvl {
			stack = stack[:len(stack)-1]
			stackLevel = stackLevel[:len(stackLevel)-1]
		}
		n := &Node{Heading: h}
		stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, n)
		stack = append(stack, n)
		stackLevel = append(stackLevel, lvl)
	}
	return root
}

// Flatten returns the tree's nodes in depth-first document order — the order
// the TOC lists its entries in, with each node's clamped level.
func (n *Node) Flatten() []*Node {
	var out []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		out = append(out, n)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// DumpOutlineXML renders the tree in the wkhtmltopdf --dump-outline XML
// format, namespace http://wkhtmltopdf.org/outline. Pages are 1-based; link
// and backLink attributes carry the heading's synthetic anchor. Valid XML,
// no CDATA.
func DumpOutlineXML(root *Node) []byte {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<outline xmlns=\"http://wkhtmltopdf.org/outline\">\n")
	dumpNode(root, &b, 1)
	b.WriteString("</outline>\n")
	return []byte(b.String())
}

func dumpNode(n *Node, b *strings.Builder, depth int) {
	pad := strings.Repeat("  ", depth)
	for _, c := range n.Children {
		h := c.Heading
		if h == nil {
			continue
		}
		b.WriteString(pad)
		b.WriteString("<item title=\"")
		b.WriteString(xmlEscape(h.Title))
		b.WriteString("\" page=\"")
		b.WriteString(strconv.Itoa(h.Page + 1))
		b.WriteString("\" link=\"")
		b.WriteString(h.Anchor)
		b.WriteString("\" backLink=\"")
		b.WriteString(h.Anchor)
		if len(c.Children) == 0 {
			b.WriteString("\"/>\n")
			continue
		}
		b.WriteString("\">\n")
		dumpNode(c, b, depth+1)
		b.WriteString(pad)
		b.WriteString("</item>\n")
	}
}

func matchAny(sels []css.Selector, n *html.Node) bool {
	for _, s := range sels {
		if css.Match(s, n) {
			return true
		}
	}
	return false
}

// collapseWS collapses whitespace runs to a single space and trims the ends,
// mirroring the layout engine's inline text handling.
func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimRight(b.String(), " ")
}

// xmlEscape escapes the five XML special characters.
func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
