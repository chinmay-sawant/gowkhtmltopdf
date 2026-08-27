// Package outline builds the document outline (PDF bookmarks) from heading
// elements and their layout locations: collection, location lookup, sorting,
// level-stack tree construction, and the wkhtmltopdf-compatible --dump-outline
// XML. It is pure tree construction - converting canvas coordinates into PDF
// page space and wiring page object refs is the caller's job (internal/convert
// does it where the page geometry is known).
package outline

import (
	"sort"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Location is the small neutral projection of layout geometry needed by the
// outline builder. Keeping this value type here avoids making outline depend
// on the layout package's Result and ElementLocation implementation types.
type Location struct {
	Node       *html.Node
	Page       int
	X, Y, W, H float64
}

// LocationReader is implemented by layout metadata and by tests or other
// callers that already have a compatible location value.
type LocationReader interface {
	NodeRef() *html.Node
	PageIndex() int
	Bounds() (float64, float64, float64, float64)
}

// NodeRef returns the associated HTML node.
func (l Location) NodeRef() *html.Node { return l.Node }

// PageIndex returns the location's zero-based page.
func (l Location) PageIndex() int { return l.Page }

// Bounds returns the canvas rectangle.
func (l Location) Bounds() (float64, float64, float64, float64) {
	return l.X, l.Y, l.W, l.H
}

// Heading is one outline entry: an h1..h6 element plus its layout location.
// X/Y/W/H are canvas coordinates (y grows downward from the top of the page
// content area) filled from a Location projection. Page stays object-local
// forever; DocPage is set exactly once during assembly and never mutated
// afterwards.
type Heading struct {
	Node    *html.Node
	Title   string
	Level   int // 1..6
	Page    int // 0-based page within the calling object's layout
	DocPage int // document-global page, filled once during assembly, never mutated afterwards
	X, Y    float64
	W, H    float64
	Anchor  string // synthetic __WKANCHOR_<base36>, stable across runs
}

// PageOf is the explicit ordering contract used when headings from more than
// one page-producing scope are combined. It returns the page number used for
// ordering, section lookup, and outline serialization; it does not mutate the
// heading. Use LocalPage for an object's local layout or DocumentPage for a
// flattened document view.
type PageOf func(*Heading) int

// LocalPage orders a heading by the page in its object's layout.
func LocalPage(h *Heading) int {
	if h == nil {
		return 0
	}

	return h.Page
}

// DocumentPage orders a heading by the page assigned during document
// assembly. Unlike the old Page/DocPage view convention, this accessor keeps
// the object-local Page field unchanged.
func DocumentPage(h *Heading) int {
	if h == nil {
		return 0
	}

	return h.DocPage
}

const (
	headingLevelH2 = 2
	headingLevelH3 = 3
	headingLevelH4 = 4
	headingLevelH5 = 5
	headingLevelH6 = 6
)

// headingLevel maps an element name to its outline level, 0 when it is not a
// heading. h1..h6 are accepted (h7..h9 are not part of the HTML vocabulary).
func headingLevel(name string) int {
	switch name {
	case "h1":
		return 1
	case "h2":
		return headingLevelH2
	case "h3":
		return headingLevelH3
	case "h4":
		return headingLevelH4
	case "h5":
		return headingLevelH5
	case "h6":
		return headingLevelH6
	}

	return 0
}

// CollectHeadings walks root in document order and returns one Heading per
// h1..h6 element with its whitespace-collapsed text title. Location fields
// are zero until Lookup runs.
func CollectHeadings(root *html.Node) []*Heading {
	var out []*Heading

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		if lvl := headingLevel(node.Name); lvl > 0 {
			out = append(out, &Heading{ //nolint:exhaustruct // intentional zero/partial fields
				Node:  node,
				Title: CollapseWS(node.TextContent()),
				Level: lvl,
			})
		}
	})

	return out
}

// Lookup fills Page/X/Y/W/H of each heading whose node appears in locs
// (matched by node pointer) and returns the headings that have a location.
// Headings without a location are skipped: they were laid out as display:none
// or never emitted a box.
func Lookup[T LocationReader](headings []*Heading, locs []T) []*Heading {
	byNode := make(map[*html.Node]Location, len(locs))

	for _, l := range locs {
		x, y, w, h := l.Bounds()
		node := l.NodeRef()
		byNode[node] = Location{
			Node: node, Page: l.PageIndex(), X: x, Y: y, W: w, H: h,
		}
	}

	out := make([]*Heading, 0, len(headings))

	for _, heading := range headings {
		l, ok := byNode[heading.Node]
		if !ok {
			continue
		}

		heading.Page, heading.X, heading.Y, heading.W, heading.H = l.Page, l.X, l.Y, l.W, l.H
		out = append(out, heading)
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
	// Exclude drops headings whose node matches any selector (css.Match).
	// Callers pass --exclude-from-outline selectors here; object-level
	// gates (UseOutline / IncludeInOutline) filter before CollectHeadings.
	Exclude []css.Selector
}

// SortHeadings brings headings into the order used by the tree, the TOC and
// the section lookup: page, y-down within a page, then x.
func SortHeadings(hs []*Heading) {
	SortHeadingsBy(hs, LocalPage)
}

// SortHeadingsBy sorts headings using pageOf without changing any Heading
// fields. This is the preferred API for flattened multi-object outlines.
func SortHeadingsBy(hs []*Heading, pageOf PageOf) {
	pageOf = normalizePageOf(pageOf)

	sort.SliceStable(hs, func(i, j int) bool {
		leftH, rightH := hs[i], hs[j]
		if pageOf(leftH) != pageOf(rightH) {
			return pageOf(leftH) < pageOf(rightH)
		}

		if leftH.Y != rightH.Y {
			return leftH.Y < rightH.Y
		}

		return leftH.X < rightH.X
	})
}

// SectionOf mirrors the wkhtmltopdf outline cache: section = first heading at
// or before page, subsection = last. Headings must be in SortHeadings order.
func SectionOf(hs []*Heading, page int) (string, string) {
	return SectionOfBy(hs, page, LocalPage)
}

// SectionOfBy returns the section/subsection for page using the supplied
// explicit ordering accessor. The input must already be sorted with the same
// accessor.
func SectionOfBy(headings []*Heading, page int, pageOf PageOf) (string, string) {
	pageOf = normalizePageOf(pageOf)

	lastIndex := sort.Search(len(headings), func(index int) bool {
		return pageOf(headings[index]) > page
	})
	if lastIndex == 0 {
		return "", ""
	}

	first, last := headings[0], headings[lastIndex-1]

	if first != nil && last != nil {
		return first.Title, last.Title
	}

	if first != nil {
		return first.Title, ""
	}

	return "", ""
}

// BuildTree sorts headings by (page, y, x) - within a page, y-down order so a
// heading higher on the page comes first - and assembles them into a
// level-stack tree whose root children are the top-level headings.
//
// Heuristic for non-monotonic sequences: a heading at level L becomes a child
// of the nearest previous heading with level < L; when the level jumps up by
// more than one (e.g. an h4 directly under an h2, with no h3 in between) the
// heading's depth is clamped to previous+1 so the tree stays connected and
// never skips levels. Titles, anchors and sort order are unaffected by the
// clamp.
func BuildTree(headings []*Heading, opts Options) *Node {
	return BuildTreeBy(headings, opts, LocalPage)
}

// BuildTreeBy builds an outline tree using pageOf for ordering. The accessor
// is carried explicitly instead of requiring callers to copy document-global
// pages into Heading.Page.
func BuildTreeBy(headings []*Heading, opts Options, pageOf PageOf) *Node {
	pageOf = normalizePageOf(pageOf)
	sel := make([]*Heading, 0, len(headings))

	for _, heading := range headings {
		if matchAny(opts.Exclude, heading.Node) {
			continue
		}

		sel = append(sel, heading)
	}

	SortHeadingsBy(sel, pageOf)

	root := &Node{} //nolint:exhaustruct // intentional zero/partial fields
	stack := []*Node{root}
	stackLevel := []int{0}

	for _, heading := range sel {
		lvl := heading.Level
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

		child := &Node{Heading: heading} //nolint:exhaustruct // intentional zero/partial fields
		stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, child)
		stack = append(stack, child)
		stackLevel = append(stackLevel, lvl)
	}

	return root
}

// Flatten returns the tree's nodes in depth-first document order - the order
// the TOC lists its entries in, with each node's clamped level.
func (n *Node) Flatten() []*Node {
	var out []*Node

	var walk func(n *Node)
	walk = func(n *Node) {
		out = append(out, n)

		for _, child := range n.Children {
			walk(child)
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
	return DumpOutlineXMLBy(root, 0, LocalPage)
}

// DumpOutlineXMLOffset is DumpOutlineXML with a page offset (e.g. TOC page
// count) added to every item's page number so dump matches final PDF pages.
func DumpOutlineXMLOffset(root *Node, pageOffset int) []byte {
	return DumpOutlineXMLBy(root, pageOffset, LocalPage)
}

// DumpOutlineXMLBy serializes an outline using pageOf for the 1-based page
// attribute. It is the explicit counterpart to the legacy local-page helper.
func DumpOutlineXMLBy(root *Node, pageOffset int, pageOf PageOf) []byte {
	pageOf = normalizePageOf(pageOf)

	var buf strings.Builder

	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<outline xmlns=\"http://wkhtmltopdf.org/outline\">\n")
	dumpNode(root, &buf, 1, pageOffset, pageOf)
	buf.WriteString("</outline>\n")

	return []byte(buf.String())
}

func dumpNode(node *Node, buf *strings.Builder, depth, pageOffset int, pageOf PageOf) {
	pad := strings.Repeat("  ", depth)

	for _, child := range node.Children {
		heading := child.Heading
		if heading == nil {
			continue
		}

		buf.WriteString(pad)
		buf.WriteString("<item title=\"")
		buf.WriteString(xmlEscape(heading.Title))
		buf.WriteString("\" page=\"")
		buf.WriteString(strconv.Itoa(pageOf(heading) + 1 + pageOffset))
		buf.WriteString("\" link=\"")
		buf.WriteString(heading.Anchor)
		buf.WriteString("\" backLink=\"")
		buf.WriteString(heading.Anchor)

		if len(child.Children) == 0 {
			buf.WriteString("\"/>\n")

			continue
		}

		buf.WriteString("\">\n")
		dumpNode(child, buf, depth+1, pageOffset, pageOf)
		buf.WriteString(pad)
		buf.WriteString("</item>\n")
	}
}

func normalizePageOf(pageOf PageOf) PageOf {
	if pageOf == nil {
		return LocalPage
	}

	return pageOf
}

func matchAny(sels []css.Selector, n *html.Node) bool {
	for _, s := range sels {
		if css.Match(s, n) {
			return true
		}
	}

	return false
}

// CollapseWS collapses whitespace runs (space, tab, newline, CR, form feed)
// to a single space and trims leading/trailing spaces. Shared by outline
// title collection; convert may reuse for document titles and similar.
func CollapseWS(text string) string {
	var buf strings.Builder

	prevSpace := true

	for _, runeVal := range text {
		if runeVal == ' ' || runeVal == '\t' || runeVal == '\n' || runeVal == '\r' || runeVal == '\f' {
			if !prevSpace {
				buf.WriteByte(' ')

				prevSpace = true
			}

			continue
		}

		buf.WriteRune(runeVal)

		prevSpace = false
	}

	return strings.TrimRight(buf.String(), " ")
}

// xmlEscape escapes the five XML special characters.
func xmlEscape(text string) string {
	var buf strings.Builder

	for _, runeVal := range text {
		switch runeVal {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(runeVal)
		}
	}

	return buf.String()
}
