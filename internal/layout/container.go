package layout

import (
	"strings"

	"gowkhtmltopdf/internal/html"
)

// findSizeContainer walks ancestors of n for the nearest size query container
// matching name (empty name = any size container). Elements without
// container-type size|inline-size are never registered, so size queries
// against name-only ancestors are rejected (no layout cycles).
func findSizeContainer(n *html.Node, name string, containers map[*html.Node]sizeContainer) (sizeContainer, bool) {
	for p := n.Parent; p != nil; p = p.Parent {
		info, ok := containers[p]
		if !ok {
			continue
		}

		if name == "" {
			return info, true
		}

		for _, nm := range strings.Fields(info.names) {
			if nm == name {
				return info, true
			}
		}
	}

	return sizeContainer{}, false //nolint:exhaustruct // intentional zero fields
}

// measureSizeContainers walks the tree top-down and records used content-box
// inline sizes for elements with container-type: inline-size|size. Widths are
// computed without children (size containment / as-if-empty for intrinsic
// contribution), matching buildBlock's definite-width rules.
func measureSizeContainers(root *html.Node, styles map[*html.Node]ResolvedStyle, viewportW float64) map[*html.Node]sizeContainer {
	out := map[*html.Node]sizeContainer{}

	var walk func(n *html.Node, availW float64)
	walk = func(node *html.Node, availW float64) {
		if node.Type != html.ElementNode {
			return
		}

		sty := styles[node]
		if sty.Display == "none" {
			return
		}

		borderW := contentInlineSize(sty, availW)
		if sty.ContainerType == "inline-size" || sty.ContainerType == "size" {
			out[node] = sizeContainer{
				inlineSize: borderW,
				fontSize:   sty.FontSize,
				names:      sty.ContainerName,
			}
		}

		childAvail := borderW
		for _, c := range node.Children {
			walk(c, childAvail)
		}
	}
	walk(root, viewportW)

	return out
}

// contentInlineSize returns the used content-box inline size (pt) for a block
// given containing-block availW, mirroring buildBlock without laying out
// children. Size-contained boxes use this definite width for @container.
func contentInlineSize(st ResolvedStyle, availW float64) float64 {
	margR, margR2 := st.MarginLeft, st.MarginRight
	if st.MarginLeftAuto {
		margR = 0
	}

	if st.MarginRightAuto {
		margR2 = 0
	}

	width := availW - margR - margR2
	if width < 0 {
		width = 0
	}

	definiteW := st.Width >= 0 || st.WidthPercent >= 0

	if st.WidthPercent >= 0 {
		if availW > 0 && availW < 1e12 {
			width = availW * st.WidthPercent / cssPercent
		} else {
			definiteW = false
		}
	} else if st.Width >= 0 {
		width = st.Width
	}

	if definiteW && st.BoxSizing != "border-box" {
		width += st.PaddingLeft + st.PaddingRight + st.BorderLeft.Width + st.BorderRight.Width
	}

	if st.MinWidth > 0 && width < st.MinWidth {
		width = st.MinWidth
	}

	if st.MaxWidth >= 0 && width > st.MaxWidth {
		width = st.MaxWidth
	}

	contentW := width - st.PaddingLeft - st.PaddingRight - st.BorderLeft.Width - st.BorderRight.Width
	if contentW < 0 {
		contentW = 0
	}

	return contentW
}

// isSizeContainer reports whether the style establishes a size query container.
func isSizeContainer(st ResolvedStyle) bool {
	return st.ContainerType == "inline-size" || st.ContainerType == "size"
}
