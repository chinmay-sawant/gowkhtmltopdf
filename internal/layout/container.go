package layout

import (
	"context"
	"fmt"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const (
	displayNone         = "none"
	borderBox           = "border-box"
	containerSize       = "size"
	containerInlineSize = "inline-size"
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
//
//nolint:unparam // production callers pass the viewport; tests use a fixed fixture viewport.
func measureSizeContainers(
	root *html.Node, styles map[*html.Node]*ResolvedStyle, viewportW float64,
) map[*html.Node]sizeContainer {
	containers, _ := measureSizeContainersContext(context.Background(), root, styles, viewportW)

	return containers
}

//nolint:cyclop,wsl // recursive container measurement keeps the cancellation gate local.
func measureSizeContainersContext(
	ctx context.Context, root *html.Node, styles map[*html.Node]*ResolvedStyle, viewportW float64,
) (map[*html.Node]sizeContainer, error) {
	out := map[*html.Node]sizeContainer{}
	visited := 0

	var walk func(n *html.Node, availW float64) error
	walk = func(node *html.Node, availW float64) error {
		visited++
		if visited&63 == 0 && ctx != nil {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("layout: container measurement: %w", err)
			}
		}

		if node.Type != html.ElementNode {
			return nil
		}

		sty := styles[node]
		if sty == nil || sty.Display == displayNone {
			return nil
		}

		borderW := contentInlineSize(*sty, availW)
		if sty.ContainerType == containerInlineSize || sty.ContainerType == containerSize {
			out[node] = sizeContainer{
				inlineSize: borderW,
				fontSize:   sty.FontSize,
				names:      sty.ContainerName,
			}
		}

		childAvail := borderW
		for _, c := range node.Children {
			if err := walk(c, childAvail); err != nil {
				return err
			}
		}

		return nil
	}
	if root == nil {
		return out, nil
	}
	if err := walk(root, viewportW); err != nil {
		return nil, err
	}

	return out, nil
}

// contentInlineSize returns the used content-box inline size (pt) for a block
// given containing-block availW, mirroring buildBlock without laying out
// children. Size-contained boxes use this definite width for @container.
func contentInlineSize(sty ResolvedStyle, availW float64) float64 {
	width, definite := contentBaseInlineSize(sty, availW)
	if definite && sty.BoxSizing != borderBox {
		width += sty.HorizChrome()
	}

	if sty.MinWidth > 0 && width < sty.MinWidth {
		width = sty.MinWidth
	}

	if sty.MaxWidth >= 0 && width > sty.MaxWidth {
		width = sty.MaxWidth
	}

	contentW := width - sty.HorizChrome()
	if contentW < 0 {
		contentW = 0
	}

	return contentW
}

// contentBaseInlineSize resolves the specified width (auto → containing-block
// width minus margins) and whether the width is definite.
func contentBaseInlineSize(sty ResolvedStyle, availW float64) (float64, bool) {
	margL, margR := sty.MarginLeft, sty.MarginRight
	if sty.MarginLeftAuto {
		margL = 0
	}

	if sty.MarginRightAuto {
		margR = 0
	}

	width := availW - margL - margR
	if width < 0 {
		width = 0
	}

	switch {
	case sty.WidthPercent >= 0:
		if availW > 0 && availW < 1e12 {
			return availW * sty.WidthPercent / cssPercent, true
		}
	case sty.Width >= 0:
		return sty.Width, true
	}

	return width, false
}

// isSizeContainer reports whether the style establishes a size query container.
func isSizeContainer(st ResolvedStyle) bool {
	return st.ContainerType == containerInlineSize || st.ContainerType == containerSize
}
