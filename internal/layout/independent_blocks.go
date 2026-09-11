package layout

import (
	"context"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const htmlRootName = "html"

// PaintMetadata is the Result projection convert needs after painting one
// independent block so Workspace.Release can drop the display list.
type PaintMetadata struct {
	PageNames        []string
	Locations        []ElementLocation
	HasIDs           bool
	HasFragmentLinks bool
	HasStructElems   bool
}

// CopyPaintMetadata copies page names, element locations, and compact
// navigation flags off res. Call this before Workspace.Release.
func CopyPaintMetadata(res *Result, contentH float64) PaintMetadata {
	if res == nil {
		return PaintMetadata{} //nolint:exhaustruct // zero metadata
	}

	locs := res.Locations
	if len(locs) == 0 {
		locs = locationsFromBoxes(res)
	}

	meta := PaintMetadata{ //nolint:exhaustruct // remaining flags filled below
		PageNames:      PageNames(res, contentH),
		Locations:      append([]ElementLocation(nil), locs...),
		HasStructElems: res.hasStructElems,
	}

	for i := range meta.Locations {
		node := meta.Locations[i].Node
		if node != nil && node.Attribute("id") != "" {
			meta.HasIDs = true

			break
		}
	}

	for i := range res.Ops {
		if res.Ops[i].Kind != OpLinkURI {
			continue
		}

		uri := res.Ops[i].URI
		if len(uri) > 0 && uri[0] == '#' {
			meta.HasFragmentLinks = true

			break
		}
	}

	return meta
}

func locationsFromBoxes(res *Result) []ElementLocation {
	if res == nil || len(res.boxes) == 0 {
		return nil
	}

	out := make([]ElementLocation, 0, len(res.boxes))

	for _, boxNode := range res.boxes {
		if boxNode == nil || boxNode.node == nil {
			continue
		}

		out = append(out, ElementLocation{ //nolint:exhaustruct // page filled at paint
			Node: boxNode.node,
			X:    boxNode.x,
			Y:    boxNode.y,
			W:    boxNode.w,
			H:    boxNode.height,
		})
	}

	return out
}

// IndependentBlocks reports an ordered, gap-free run of in-flow body children
// that can be laid out one at a time. It fails closed: any child that is
// floated, absolutely positioned, a flex/grid item, missing page-break-inside
// avoid, or missing page-break-before (except the first) rejects the document.
// It does not key on class names or HTML comments. contentH is the page
// content height convert will paint at; the CSS detector does not layout.
// IndependentBlocksForOptions resolves styles with opts then runs IndependentBlocks.
func IndependentBlocksForOptions(ctx context.Context, root *html.Node, opts Options) ([]*html.Node, bool) {
	if root == nil {
		return nil, false
	}

	styles, err := ResolveStyles(ctx, root, opts)
	if err != nil {
		return nil, false
	}

	return IndependentBlocks(root, styles, opts.Height)
}

// IndependentBlocks is the fail-closed CSS detector. See IndependentBlocksForOptions.
//
//nolint:cyclop // fail-closed detector is a flat checklist of disqualifiers
func IndependentBlocks(
	root *html.Node, styles map[*html.Node]*ResolvedStyle, contentH float64,
) ([]*html.Node, bool) {
	_ = contentH

	body := documentBody(root)
	if body == nil || styles == nil {
		return nil, false
	}

	bodyStyle := styles[body]
	if bodyStyle != nil && isFlexOrGridDisplay(bodyStyle.Display) {
		return nil, false
	}

	candidates := make([]*html.Node, 0, len(body.Children))

	for _, child := range body.Children {
		if child.Type != html.ElementNode {
			continue
		}

		sty := styles[child]
		if sty == nil {
			return nil, false
		}

		if sty.Display == cssDisplayNone {
			continue
		}

		if !independentCandidate(sty, len(candidates) == 0) {
			return nil, false
		}

		candidates = append(candidates, child)
	}

	if len(candidates) == 0 {
		return nil, false
	}

	return candidates, true
}

func documentBody(root *html.Node) *html.Node {
	if root == nil {
		return nil
	}

	if root.Type == html.ElementNode && root.Name == "body" {
		return root
	}

	htmlEl := root

	if root.Name != htmlRootName {
		if child := root.FirstChild(htmlRootName); child != nil {
			htmlEl = child
		}
	}

	if body := htmlEl.FirstChild("body"); body != nil {
		return body
	}

	return root.FirstChild("body")
}

func independentCandidate(sty *ResolvedStyle, isFirst bool) bool {
	if sty == nil {
		return false
	}

	switch sty.Position {
	case positionAbsolute, positionFixed:
		return false
	}

	if sty.Float != "" && sty.Float != cssDisplayNone {
		return false
	}

	if isFlexOrGridDisplay(sty.Display) {
		return false
	}

	if !isFirst && sty.PageBreakBefore != pageBreakAlways {
		return false
	}

	if sty.PageBreakInside != avoidKeyword && sty.PageBreakInside != avoidPageValue {
		return false
	}

	return isInFlowBlockDisplay(sty.Display)
}

func isFlexOrGridDisplay(display string) bool {
	switch display {
	case displayFlex, displayInlineFlex, displayGrid, displayInlineGrid:
		return true
	default:
		return false
	}
}

func isInFlowBlockDisplay(display string) bool {
	switch display {
	case displayBlock, displayListItem, displayTable, displayFlowRoot:
		return true
	default:
		return false
	}
}
