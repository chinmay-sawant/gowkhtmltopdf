package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/html"

// PlacedElement is one element border box in canvas points. Y grows downward
// from the top of the layout canvas. Layout fills these boxes before pagination.
type PlacedElement struct {
	ID     string
	Tag    string
	Action string
	Text   string
	X      float64
	Y      float64
	W      float64
	H      float64
}

// PlacedElements walks the laid-out element boxes in document order.
// A nil result returns nil. Paint and PDF pagination are not required.
func PlacedElements(res *Result) []PlacedElement {
	if res == nil || res.root == nil {
		return nil
	}

	out := make([]PlacedElement, 0, len(res.boxes))
	walkPlaced(res.root, &out)

	return out
}

func walkPlaced(boxNode *box, out *[]PlacedElement) {
	if boxNode == nil {
		return
	}

	if boxNode.node != nil && boxNode.node.Type == html.ElementNode {
		*out = append(*out, PlacedElement{
			ID:     boxNode.node.Attribute("id"),
			Tag:    boxNode.node.Name,
			Action: boxNode.node.Attribute("data-action"),
			Text:   boxNode.node.TextContent(),
			X:      boxNode.x,
			Y:      boxNode.y,
			W:      boxNode.w,
			H:      boxNode.height,
		})
	}

	for _, child := range boxNode.children {
		walkPlaced(child, out)
	}
}
