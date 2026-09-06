package layout

// namedPageEdgeEpsilon nudges box edges onto their flow page so a box that
// starts or ends exactly on a page boundary maps to the page it paints on.
const namedPageEdgeEpsilon = 1e-6

// applyNamedPageBreaks forces page-break-before:always on a box whose used
// CSS page name differs from the previous sibling. Parent/child name changes
// (body { page: chapter } wrapping the document) do not insert a blank first
// page. Interned styles are cloned before mutation.
func applyNamedPageBreaks(res *Result) {
	if res == nil || res.root == nil {
		return
	}

	walkNamedPageBreaks(res.root)
}

func walkNamedPageBreaks(boxNode *box) {
	if boxNode == nil {
		return
	}

	var prevName string

	var prevSet bool

	for _, child := range boxNode.children {
		name := boxPageName(child)
		if prevSet && name != prevName {
			forcePageBreakBefore(child)
		}

		prevName = name
		prevSet = true

		walkNamedPageBreaks(child)
	}
}

func boxPageName(boxNode *box) string {
	if boxNode == nil || boxNode.style == nil {
		return ""
	}

	return boxNode.style.PageName
}

func forcePageBreakBefore(boxNode *box) {
	if boxNode == nil || boxNode.style == nil {
		return
	}

	if boxNode.style.PageBreakBefore == pageBreakAlways {
		return
	}

	cloned := *boxNode.style
	cloned.PageBreakBefore = pageBreakAlways
	boxNode.style = &cloned
}

// namedPageNames maps each output page to the first in-flow box page name
// that overlaps that page. Empty means auto / unnamed. Lite: a continuation
// page with no overlapping named box keeps the unnamed side/:first cascade.
//
//nolint:cyclop // page-name extraction validates bounded pagination geometry
func namedPageNames(res *Result, contentH float64) []string {
	if res == nil || contentH <= 0 || len(res.Pages) == 0 {
		return nil
	}

	names := make([]string, len(res.Pages))

	for _, boxNode := range flowBoxList(res) {
		name := boxPageName(boxNode)
		if name == "" {
			continue
		}

		start, startOK := checkedFlowPageOfY(boxNode.y+namedPageEdgeEpsilon, contentH)
		if !startOK {
			continue
		}

		endY := boxNode.y + boxNode.height - namedPageEdgeEpsilon
		if endY < boxNode.y {
			endY = boxNode.y
		}

		end, endOK := checkedFlowPageOfY(endY, contentH)
		if !endOK {
			end = start
		}

		for pageIdx := start; pageIdx <= end && pageIdx < len(names); pageIdx++ {
			if names[pageIdx] == "" {
				names[pageIdx] = name
			}
		}
	}

	return names
}

// PageNames returns the page-name used by each paginated output page.
func PageNames(res *Result, contentH float64) []string {
	return namedPageNames(res, contentH)
}

func (opts PaintOptions) pageNameAt(pageIdx int) string {
	if pageIdx < 0 || pageIdx >= len(opts.pageNames) {
		return ""
	}

	return opts.pageNames[pageIdx]
}

func applyPaintMarginSet(opts PaintOptions, margins PageMargins) PaintOptions {
	opts.MarginTop = margins.Top
	opts.MarginRight = margins.Right
	opts.MarginBottom = margins.Bottom
	opts.MarginLeft = margins.Left

	return opts
}

// PageMarginsForPage resolves one page with the same cascade used by painting.
// pageNames contains the first in-flow page name for each output page.
func (opts PaintOptions) PageMarginsForPage(pageIdx int, pageNames []string) PageMargins {
	opts.pageNames = pageNames
	resolved := opts.forPage(pageIdx)

	return PageMargins{
		Top: resolved.MarginTop, Right: resolved.MarginRight,
		Bottom: resolved.MarginBottom, Left: resolved.MarginLeft,
	}
}

// forPage returns geometry for one output page.
// Cascade: unnamed, then named @page ident, then :left/:right by side
// (LTR page 1 is :right), then :first on page 0. Pagination still uses one
// contentH from the unnamed box.
func (opts PaintOptions) forPage(pageIdx int) PaintOptions {
	if name := opts.pageNameAt(pageIdx); name != "" {
		if margins, ok := opts.Named[name]; ok {
			opts = applyPaintMarginSet(opts, margins)
		}
	}

	pageNum := pageIdx + 1
	if pageNum%2 == 0 && opts.Left != nil {
		opts = applyPaintMarginSet(opts, *opts.Left)
	}

	if pageNum%2 == 1 && opts.Right != nil {
		opts = applyPaintMarginSet(opts, *opts.Right)
	}

	if pageIdx == 0 && opts.First != nil {
		opts = applyPaintMarginSet(opts, *opts.First)
	}

	return opts
}
