package layout

import (
	"context"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const htmlRootName = "html"

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
