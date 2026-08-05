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
	return sizeContainer{}, false
}

// measureSizeContainers walks the tree top-down and records used content-box
// inline sizes for elements with container-type: inline-size|size. Widths are
// computed without children (size containment / as-if-empty for intrinsic
// contribution), matching buildBlock's definite-width rules.
func measureSizeContainers(root *html.Node, styles map[*html.Node]ResolvedStyle, viewportW float64) map[*html.Node]sizeContainer {
	out := map[*html.Node]sizeContainer{}
	var walk func(n *html.Node, availW float64)
	walk = func(n *html.Node, availW float64) {
		if n.Type != html.ElementNode {
			return
		}
		st := styles[n]
		if st.Display == "none" {
			return
		}
		borderW := contentInlineSize(st, availW)
		if st.ContainerType == "inline-size" || st.ContainerType == "size" {
			out[n] = sizeContainer{
				inlineSize: borderW,
				fontSize:   st.FontSize,
				names:      st.ContainerName,
			}
		}
		childAvail := borderW
		for _, c := range n.Children {
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
	ml, mr := st.MarginLeft, st.MarginRight
	if st.MarginLeftAuto {
		ml = 0
	}
	if st.MarginRightAuto {
		mr = 0
	}
	w := availW - ml - mr
	if w < 0 {
		w = 0
	}
	definiteW := st.Width >= 0 || st.WidthPercent >= 0
	if st.WidthPercent >= 0 {
		if availW > 0 && availW < 1e12 {
			w = availW * st.WidthPercent / 100
		} else {
			definiteW = false
		}
	} else if st.Width >= 0 {
		w = st.Width
	}
	if definiteW && st.BoxSizing != "border-box" {
		w += st.PaddingLeft + st.PaddingRight + st.BorderLeft.Width + st.BorderRight.Width
	}
	if st.MinWidth > 0 && w < st.MinWidth {
		w = st.MinWidth
	}
	if st.MaxWidth >= 0 && w > st.MaxWidth {
		w = st.MaxWidth
	}
	contentW := w - st.PaddingLeft - st.PaddingRight - st.BorderLeft.Width - st.BorderRight.Width
	if contentW < 0 {
		contentW = 0
	}
	return contentW
}

// isSizeContainer reports whether the style establishes a size query container.
func isSizeContainer(st ResolvedStyle) bool {
	return st.ContainerType == "inline-size" || st.ContainerType == "size"
}
