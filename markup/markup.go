package markup

import (
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Type classifies a parsed node.
type Type int

const (
	// TypeElement is a tag. Name is lowercase.
	TypeElement Type = iota + 1
	// TypeText is character data.
	TypeText
	// TypeComment is an HTML comment.
	TypeComment
	// TypeDoctype is the doctype declaration.
	TypeDoctype
)

// Node is one parsed HTML node. Children are in document order.
// There is no parent pointer. Attribute keys are lowercase.
type Node struct {
	Type     Type
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*Node
}

// Parse parses UTF-8 HTML and returns an owned copy of the tree.
// A nil source is parsed as an empty document.
func Parse(source []byte) (*Node, error) {
	root, err := html.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("markup: parse: %w", err)
	}

	return copyNode(root), nil
}

func copyNode(node *html.Node) *Node {
	if node == nil {
		return nil
	}

	kind, ok := copyType(node.Type)
	if !ok {
		return nil
	}

	children := make([]*Node, 0, len(node.Children))

	for _, child := range node.Children {
		copied := copyNode(child)
		if copied != nil {
			children = append(children, copied)
		}
	}

	return &Node{
		Type:     kind,
		Name:     node.Name,
		Attrs:    copyAttrs(node.Attrs),
		Text:     node.Text,
		Children: children,
	}
}

func copyType(kind html.NodeType) (Type, bool) {
	switch kind {
	case html.ElementNode:
		return TypeElement, true
	case html.TextNode:
		return TypeText, true
	case html.CommentNode:
		return TypeComment, true
	case html.DoctypeNode:
		return TypeDoctype, true
	case html.NodeUnknown:
		return 0, false
	default:
		return 0, false
	}
}

func copyAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}

	out := make(map[string]string, len(attrs))

	for key, value := range attrs {
		out[key] = value
	}

	return out
}
