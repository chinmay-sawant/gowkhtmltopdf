package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/svg"
)

func (e *engine) contentBox(posX, boxW float64, style ResolvedStyle) (float64, float64) {
	contentW := boxW - e.scalePt(style.PaddingLeft) - e.scalePt(style.PaddingRight) -
		e.scalePt(style.BorderLeft.Width) - e.scalePt(style.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}

	return posX + e.scalePt(style.BorderLeft.Width) + e.scalePt(style.PaddingLeft), contentW
}

// imageRef is the resolved form of one <img src>: bytes + intrinsic size,
// resolved at most once per Layout run (measure and build share it).
type imageRef struct {
	src    string
	data   []byte
	w, h   int
	isJPEG bool
}

// resolveImage fetches (once) and decodes (once) src; nil on any failure.
func (e *engine) resolveImage(src string) *imageRef {
	if src == "" || e.opts.Images == nil {
		return nil
	}

	if e.imgCache == nil {
		e.imgCache = map[string]*imageRef{}
	}

	if ref, ok := e.imgCache[src]; ok {
		return ref
	}

	data, err := e.opts.Images(src)
	if err != nil {
		// Cache nil-miss? Store a sentinel empty ref so we do not re-fetch.
		e.imgCache[src] = nil

		return nil
	}

	ref := &imageRef{src: src, data: data} //nolint:exhaustruct // intentional zero fields
	if png, pw, ph, err := svg.Rasterize(data, svgRasterMax); err == nil {
		ref.data, ref.w, ref.h = png, pw, ph
	} else if w, h, jpeg, ok := imageDims(data); ok {
		ref.w, ref.h, ref.isJPEG = w, h, jpeg
	}

	e.imgCache[src] = ref

	return ref
}

// isInlineChild reports whether n participates in an inline formatting context.
func (e *engine) isInlineChild(node *html.Node) bool {
	if node.Type == html.TextNode {
		return true
	}

	if node.Type != html.ElementNode {
		return false
	}

	cstate := e.styles[node]
	if cstate.Display == cssDisplayNone || cstate.Float != cssDisplayNone ||
		cstate.Position == positionAbsolute || cstate.Position == positionFixed {
		return false
	}
	// <img> is replaced and UA-default inline-block, but author CSS may set
	// display:block (wiki .mw-logo-wordmark / .mw-logo-tagline stack).
	if node.Name == cssTagImg {
		return !blockishDisplay(cstate.Display)
	}

	return cstate.Display == cssDisplayInline || cstate.Display == cssDisplayInlineBlock ||
		cstate.Display == displayInlineFlex
}

// blockishDisplay reports display values that force a block formatting
// context for <img> and inline-level replaced elements.
func blockishDisplay(display string) bool {
	switch display {
	case displayBlock, displayFlex, displayGrid, displayTable, displayListItem, displayFlowRoot:
		return true
	default:
		return false
	}
}

// onlyCollapsibleWS reports whether every node is a text node of only
// whitespace (collapses between blocks and must not kill margin collapse).
func onlyCollapsibleWS(nodes []*html.Node) bool {
	if len(nodes) == 0 {
		return true
	}

	for _, n := range nodes {
		if n.Type != html.TextNode || strings.TrimSpace(n.Text) != "" {
			return false
		}
	}

	return true
}

// flowChildren lays out children in document order: runs of inlines, then
// block boxes, alternating as they appear. Floated children are positioned
// out of flow with a lite exclusion model; clear advances past them.
// Returns the advanced content height (cy end − cy start contribution is
// encoded as the final cy relative to start; callers pass starting cy).
// Float enclosure (extentCy) is the caller's job when it owns a BFC.
//
//nolint:cyclop,wsl,varnamelen,funlen // document-order flow keeps its state machine explicit
func (e *engine) flowChildren(
	parent *box, children []*html.Node, sty ResolvedStyle,
	contentW, contentX, posY, curY float64,
) float64 {
	prevBottom := 0.0

	var local floatState

	floats := e.bfcFloats
	if floats == nil {
		// Defensive: callers should pushBFCFloats first. Keep a local state
		// so isolated measure passes (layoutCell) still work.
		local = newFloatState(contentX, contentW)
		floats = &local
	}

	var deferred []*html.Node
	// Absolute/fixed containing-block origin is the content edge at entry.
	// Do not use the post-flow cy or deferred boxes sit below in-flow siblings.
	absOriginY := posY + curY

	absCBX, absCBW := contentX, contentW
	paddingBoxCB := sty.HasTransform
	if sty.Position == positionRelative {
		for _, child := range children {
			childStyle := e.stylePtr(child)
			if childStyle.Position == positionAbsolute &&
				(!childStyle.BottomAuto || (childStyle.Height < 0 && childStyle.HeightPercent < 0)) {
				paddingBoxCB = true

				break
			}
		}
	}

	if paddingBoxCB {
		// A positioned or transformed element uses its padding box as the
		// containing block for absolute descendants.
		absCBX = contentX - e.scalePt(sty.PaddingLeft)
		absOriginY -= e.scalePt(sty.PaddingTop)
		absCBW = contentW + e.scalePt(sty.PaddingLeft) + e.scalePt(sty.PaddingRight)
	}

	idx := 0
	for idx < len(children) {
		if e.checkContext() {
			return curY
		}

		curY, prevBottom, idx, deferred = e.flowOneChild(parent, children, idx, sty,
			contentW, contentX, posY, curY, prevBottom, floats, deferred)
	}

	parentHeight := e.applyHeightConstraints(sty, curY+e.scalePt(sty.PaddingBottom))
	cbHeight := parentHeight - (absOriginY - posY)
	if cbHeight < 0 {
		cbHeight = 0
	}

	for _, n := range deferred {
		if e.absCBHeights == nil {
			e.absCBHeights = make(map[*html.Node]float64, len(deferred))
		}
		e.absCBHeights[n] = cbHeight
		ab := e.build(n, absCBW, absCBX, absOriginY)
		delete(e.absCBHeights, n)
		if ab != nil && parent != nil {
			parent.children = append(parent.children, ab)
		}
	}
	// A final child margin is inside a parent that has bottom padding or a
	// bottom border. Without this, the margin disappears from the parent's
	// used height, making padded cards and diagram boxes shorter than HTML.
	if sty.PaddingBottom > 0 || sty.BorderBottom.Width > 0 {
		curY += prevBottom
	}

	return curY
}

// flowOneChild advances one flow child (inline run, block, float or
// out-of-flow deferral), returning the updated cy, prevBottom, loop index and
// deferred list (extracted from flowChildren to keep each piece focused).
func (e *engine) flowOneChild(
	parent *box, children []*html.Node, idx int, sty ResolvedStyle,
	contentW, contentX, posY, curY, prevBottom float64, floats *floatState, deferred []*html.Node,
) (float64, float64, int, []*html.Node) {
	node := children[idx]
	// Fetch the child's resolved style once; all flow-child predicates and
	// the float branch reuse it (was four e.styles map lookups per child).
	cst := e.styles[node]

	switch {
	case isSkippableFlowNode(node, cst):
		idx++
	case isOutOfFlowNode(node, cst):
		// Defer out-of-flow boxes so they paint above in-flow content
		// (absolute overlays sit on top of later siblings' text).
		deferred = append(deferred, node)
		idx++
	case isFlowFloat(node, cst):
		var childStyle ResolvedStyle
		if cst != nil {
			childStyle = *cst
		}

		curY = floats.clearFloats(childStyle.Clear, posY, curY)
		attachFlowBox(parent, e.placeFloat(node, childStyle, floats, contentW, contentX, posY, curY), e)

		prevBottom = 0
		idx++
	case e.isInlineChild(node):
		run, next := collectInlineRun(children, idx, e)
		idx = next
		curY, prevBottom = e.layoutInlineRun(parent, sty, run, contentW, contentX, posY, curY, floats, prevBottom)
	case node.Type == html.ElementNode:
		var cblock *box
		curY, prevBottom, cblock = e.layoutBlockChild(node, floats, contentW, contentX, posY, curY, prevBottom)
		attachFlowBox(parent, cblock, e)

		idx++
	default:
		idx++
	}

	return curY, prevBottom, idx, deferred
}

// layoutInlineRun lays out one maximal inline run, returning the advanced cy
// and the margin accumulator.
func (e *engine) layoutInlineRun(
	parent *box, _ ResolvedStyle, run []*html.Node, contentW, contentX, posY, curY float64,
	floats *floatState, prevBottom float64,
) (float64, float64) {
	if onlyCollapsibleWS(run) {
		return curY, prevBottom
	}

	if len(run) > 0 {
		// Pass the real parent only. Synthetic measure-only boxes used to
		// allocate a full ResolvedStyle per inline run (hot on table cells);
		// emitLine tolerates a nil box (skips firstBaseline / text-align).
		h := e.layoutInlineFloats(parent, run, contentW, contentX, posY+curY, floats)
		curY += h

		if h > 0 {
			prevBottom = 0
		}
	}

	return curY, prevBottom
}

// isSkippableFlowNode reports nodes that are dropped from flow: display:none
// elements and pure-whitespace text (so margin collapse between block
// siblings is not interrupted — fixture-19 margin-bottom between divs).
// st is the node's resolved style (nil when the element has none).
func isSkippableFlowNode(node *html.Node, st *ResolvedStyle) bool {
	if node.Type == html.ElementNode {
		if st != nil && st.Display == cssDisplayNone {
			return true
		}
	}

	return node.Type == html.TextNode && strings.TrimSpace(node.Text) == ""
}

// isOutOfFlowNode reports absolute/fixed children (deferred to paint above
// the in-flow content of the current box).
func isOutOfFlowNode(node *html.Node, stylePtr *ResolvedStyle) bool {
	if node.Type != html.ElementNode {
		return false
	}

	if stylePtr == nil {
		return false
	}

	return stylePtr.Position == positionAbsolute || stylePtr.Position == positionFixed
}

// isFlowFloat reports floated element children.
func isFlowFloat(node *html.Node, st *ResolvedStyle) bool {
	if node.Type != html.ElementNode {
		return false
	}

	return st != nil && st.Float != cssDisplayNone
}

// collectInlineRun gathers a maximal run of inline children starting at idx,
// skipping display:none elements and keeping interior whitespace.
//
//nolint:cyclop // hot-path run scanner; splitting adds indirection for no clarity
func collectInlineRun(children []*html.Node, idx int, engine *engine) ([]*html.Node, int) {
	start := idx
	hasDisplayNone := false

	for idx < len(children) {
		child := children[idx]
		if child.Type == html.ElementNode && engine.styles[child].Display == cssDisplayNone {
			hasDisplayNone = true
			idx++

			continue
		}

		if child.Type == html.ElementNode && engine.styles[child].Float != cssDisplayNone {
			break
		}

		if child.Type == html.TextNode && isAllWhitespace(child.Text) {
			// keep interior whitespace inside an inline run, but a run that
			// is only WS is dropped below.
			idx++

			continue
		}

		if !engine.isInlineChild(child) {
			break
		}

		idx++
	}

	if !hasDisplayNone {
		return children[start:idx], idx
	}

	run := make([]*html.Node, 0, idx-start)

	for _, child := range children[start:idx] {
		if child.Type == html.ElementNode && engine.styles[child].Display == cssDisplayNone {
			continue
		}

		run = append(run, child)
	}

	return run, idx
}

// attachFlowBox appends a built child to its parent and draws the debug
// outline when DebugBoxes is on.
func attachFlowBox(parent *box, child *box, engine *engine) {
	if child == nil || parent == nil {
		return
	}

	parent.children = append(parent.children, child)

	if engine.opts.DebugBoxes {
		engine.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: child.x, Y: child.y, W: child.w, H: child.height, R: 1, G: 0, B: 0,
		})
	}
}

// layoutBlockChild builds one block-level child: it clears floats, collapses
// margins with the previous sibling, applies the BFC float exclusion, and
// returns the advanced cy, the next margin accumulator, and the built box.
func (e *engine) layoutBlockChild(
	node *html.Node, floats *floatState, contentW, contentX, posY, curY, prevBottom float64,
) (float64, float64, *box) {
	cstate := e.styleVal(node)
	// In-flow tables always clear below preceding floats (deterministic
	// report policy). Shrink-to-fit / squeeze-beside is unsupported.
	clearVal := cstate.Clear
	if cstate.Display == displayTable {
		clearVal = clearBoth
	}

	curY = floats.clearFloats(clearVal, posY, curY)
	curY += collapseMargins(prevBottom, e.scalePt(cstate.MarginTop))
	// CSS2.1 §9.5: line boxes next to floats are shortened, not the
	// block box — so normal paragraphs get full content width and
	// re-query exclusion per line (wiki "Leading roles" reclaim).
	// §9.5 / BFC: flow-root, overflow≠visible, flex, etc. must not
	// overlap float margin boxes — otherwise heading border-bottom
	// paints through the infobox (wiki .mw-heading{display:flow-root}).
	boxX, boxW := contentX, contentW
	if establishesBFC(cstate) {
		boxX, boxW = floats.exclusion(contentX, contentW, posY, curY)
	}

	if node.Name == cssTagImg {
		marginL := e.scalePt(cstate.MarginLeft)
		marginR := e.scalePt(cstate.MarginRight)
		boxX += marginL
		boxW -= marginL + marginR

		if boxW < 0 {
			boxW = 0
		}
	}

	cblock := e.build(node, boxW, boxX, posY+curY)
	if cblock == nil {
		return curY, 0, nil
	}

	return curY + cblock.height, e.scalePt(cstate.MarginBottom), cblock
}

// pushBFCFloats installs a floatState for the current box. When the box
// establishes a BFC (or is the root), a fresh state is used and enclose is
// true so the caller should extend height with extentCy. Otherwise the
// parent BFC's state is reused and floats may protrude.
//
// Pair every push with popBFCFloats(enclose). No per-call closure is allocated.
func (e *engine) pushBFCFloats(style ResolvedStyle, contentX, contentW float64) bool {
	if e.bfcFloats != nil && !establishesBFC(style) {
		return false
	}

	e.bfcStack = append(e.bfcStack, e.bfcFloats)

	var state *floatState
	if n := len(e.bfcPool); n > 0 {
		state = e.bfcPool[n-1]
		e.bfcPool = e.bfcPool[:n-1]
	} else {
		state = new(floatState)
	}

	*state = newFloatState(contentX, contentW)
	e.bfcFloats = state

	return true
}

// popBFCFloats restores the BFC float state installed by a matching
// pushBFCFloats that returned enclose=true.
func (e *engine) popBFCFloats(enclose bool) {
	if !enclose {
		return
	}

	if cur := e.bfcFloats; cur != nil {
		*cur = floatState{} //nolint:exhaustruct // clear before pool reuse
		e.bfcPool = append(e.bfcPool, cur)
	}

	stackLen := len(e.bfcStack)
	if stackLen == 0 {
		e.bfcFloats = nil

		return
	}

	e.bfcFloats = e.bfcStack[stackLen-1]
	e.bfcStack = e.bfcStack[:stackLen-1]
}

// emitListMarker paints the list marker for an <li>.
// list-style-image, when it resolves, replaces the type glyph. Missing images
// fall back to list-style-type. list-style-position:inside places the marker
// at contentX (start of the first line box). outside and empty hang in the
// gutter at contentX - gap - marker width.
func (e *engine) emitListMarker(node *html.Node, style ResolvedStyle, contentX, baseline float64) {
	size := e.scalePt(style.FontSize)
	if e.emitListStyleImageMarker(style, contentX, baseline, size) {
		return
	}

	typ := style.ListStyleType
	if typ == "" {
		typ = listStyleDisc
	}

	if typ == cssDisplayNone {
		return
	}

	face := e.faceFor(style)

	text := markerText(node, typ)

	minW := 0.0

	if face != nil {
		for _, r := range text {
			minW += face.AdvanceInPoints(r, size)
		}
	}

	if minW <= 0 {
		minW = size * float64(len([]rune(text))) * halfRatio
	}

	posX := listMarkerX(style.ListStylePosition, contentX, size, minW)

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpBullet, X: posX, Y: baseline, Text: text, Font: face, Size: size,
		InkDescent: e.fontDescentFace(face, size),
		R:          style.Color[0], G: style.Color[1], B: style.Color[2],
	})
}

// emitListStyleImageMarker paints a list-style-image via resolveImage. false
// means the type marker should be used instead (no image, or fetch failed).
func (e *engine) emitListStyleImageMarker(
	style ResolvedStyle, contentX, baseline, size float64,
) bool {
	if style.ListStyleImage == "" {
		return false
	}

	src := backgroundImageSrc(style.ListStyleImage)
	if src == "" {
		return false
	}

	ref := e.resolveImage(src)
	if ref == nil || ref.data == nil {
		return false
	}

	imgW := e.scalePt(pxToPt(float64(ref.w)))
	imgH := e.scalePt(pxToPt(float64(ref.h)))

	if imgW <= 0 {
		imgW = size
	}

	if imgH <= 0 {
		imgH = size
	}

	posX := listMarkerX(style.ListStylePosition, contentX, size, imgW)

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpImage,
		X:      posX,
		Y:      baseline - imgH,
		W:      imgW,
		H:      imgH,
		Image:  ref.data,
		ImgW:   ref.w,
		ImgH:   ref.h,
		IsJPEG: ref.isJPEG,
	})

	return true
}

// listMarkerX is the left edge of a list marker of width markerW. inside sits
// at the content edge; outside hangs in the gutter, clamped at 0.
func listMarkerX(position string, contentX, emSize, markerW float64) float64 {
	if position == listPosInside {
		return contentX
	}

	posX := contentX - emSize*bulletGapRatio - markerW
	if posX < 0 {
		return 0
	}

	return posX
}

// markerText returns the glyph/string for a list-style-type keyword.
func markerText(node *html.Node, typ string) string {
	switch typ {
	case listStyleDisc:
		return bulletDisc
	case "circle":
		return "\u25E6"
	case listStyleSquare:
		return "\u25AA"
	case listStyleDecimal, "decimal-leading-zero":
		return strconv.Itoa(listItemIndex(node)) + "."
	case "lower-alpha", "lower-latin":
		return alphaMarker(listItemIndex(node), false) + "."
	case "upper-alpha", "upper-latin":
		return alphaMarker(listItemIndex(node), true) + "."
	case "lower-roman":
		return romanMarker(listItemIndex(node), false) + "."
	case "upper-roman":
		return romanMarker(listItemIndex(node), true) + "."
	default:
		return bulletDisc
	}
}

// listItemIndex is the 1-based index among element siblings that are list items.
func listItemIndex(node *html.Node) int {
	if node == nil || node.Parent == nil {
		return 1
	}

	idx := 0

	for _, child := range node.Parent.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if !strings.EqualFold(child.Name, "li") {
			continue
		}

		idx++
		if child == node {
			return idx
		}
	}

	return 1
}

func alphaMarker(node int, upper bool) string {
	if node < 1 {
		node = 1
	}

	var chars []byte

	for node > 0 {
		node--

		ch := byte('a' + (node % alphabetLen))
		if upper {
			ch = byte('A' + (node % alphabetLen))
		}

		chars = append(chars, ch)
		node /= 26
	}

	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars)
}

func romanMarker(node int, upper bool) string {
	if node < 1 {
		node = 1
	}

	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}

	var boxNode strings.Builder

	for i, v := range vals {
		for node >= v {
			boxNode.WriteString(syms[i])

			node -= v
		}
	}

	s := boxNode.String()
	if upper {
		return strings.ToUpper(s)
	}

	return s
}

// placeFloat lays out n as a float:left|right box and records it in floats.
// Consecutive same-side floats pack horizontally when width remains;
// otherwise they stack below the previous float bottom.
func (e *engine) placeFloat(
	node *html.Node, cstate ResolvedStyle, floats *floatState, contentW, contentX, posY, curY float64,
) *box {
	avail := contentW
	if cstate.Width < 0 && cstate.WidthPercent < 0 {
		avail = e.floatIntrinsicAvail(node, cstate, avail)
	}

	flowY := posY + curY

	fixX, fromY := contentX, flowY

	switch cstate.Float {
	case floatLeft, floatRight:
		fixX, fromY, avail = packFloatPosition(floats, contentX, contentW, flowY, avail, cstate.Float == floatLeft)
	}

	oldMax := e.imgMaxW
	e.setFloatImgMaxW(cstate, contentW, avail)

	fbox := e.build(node, avail, fixX, fromY)
	e.imgMaxW = oldMax

	if fbox == nil {
		return nil
	}

	if cstate.Float == floatLeft && floats.hasLeft && fbox.x+fbox.w > contentX+contentW {
		// Overflowed the pack attempt — stack below.
		fromY = maxY(floats.leftBottom, flowY)
		dx, dy := contentX-fbox.x, fromY-fbox.y
		fbox.x, fbox.y = contentX, fromY
		e.shiftBoxOps(fbox, dx, dy)
	}

	margL := e.scalePt(cstate.MarginLeft)
	margR := e.scalePt(cstate.MarginRight)

	if cstate.Float == floatRight {
		wantX := contentX + contentW - fbox.w - margR
		dx := wantX - fbox.x
		fbox.x = wantX
		e.shiftBoxOps(fbox, dx, 0)
	}

	floats.place(cstate.Float, fbox, margL, margR)

	return fbox
}

// setFloatImgMaxW clamps replaced images inside the float to its used width
// (extracted from placeFloat for clarity).
func (e *engine) setFloatImgMaxW(cs ResolvedStyle, contentW, avail float64) {
	switch {
	case cs.Width >= 0:
		e.imgMaxW = e.scalePt(cs.Width)
	case cs.WidthPercent >= 0 && contentW > 0:
		e.imgMaxW = contentW * cs.WidthPercent / cssPercent
	case avail > 0 && avail < contentW:
		e.imgMaxW = avail
	}
}

// packFloatPosition resolves where a new float starts: beside the existing
// same-side floats when there is room, otherwise below their bottom edge.
func packFloatPosition(
	floats *floatState, contentX, contentW, flowY, avail float64, isLeft bool,
) (float64, float64, float64) {
	fixX := contentX
	fromY := flowY
	packedAvail := avail

	if isLeft {
		if !floats.hasLeft {
			return fixX, fromY, packedAvail
		}

		room := floatsPackRoom(floats, contentX, contentW, true)
		if room >= avail*halfRatio { // enough room to attempt side-by-side
			fixX = floats.leftEdge
			fromY = maxY(floats.leftTop, flowY)
			packedAvail = minY(avail, room)

			return fixX, fromY, packedAvail
		}

		fromY = maxY(floats.leftBottom, flowY)

		return fixX, fromY, packedAvail
	}

	if !floats.hasRight {
		return fixX, fromY, packedAvail
	}

	room := floatsPackRoom(floats, contentX, contentW, false)
	if room >= avail*halfRatio {
		fromY = maxY(floats.rightTop, flowY)
		packedAvail = minY(avail, room)

		return fixX, fromY, packedAvail
	}

	fromY = maxY(floats.rightBottom, flowY)

	return fixX, fromY, packedAvail
}

// floatIntrinsicAvail measures the shrink-to-fit width of a float without a
// definite width: size containment, the widest descendant image, or the
// cell content max-content (plus chrome and margins).
func (e *engine) floatIntrinsicAvail(node *html.Node, style ResolvedStyle, avail float64) float64 {
	var intr float64
	if isSizeContainer(style) {
		// Size containment: intrinsic inline size as-if-empty (padding+border
		// only) so used size does not depend on descendants.
		intr = e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	} else if imgW := e.measureLargestImageWidth(node); imgW > 0 {
		// Wiki thumbs: size the float to the image, not the unwrapped
		// figcaption max-content (which letterboxed images in a wide frame).
		intr = imgW +
			e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	} else {
		intr = e.measureCellContent(node, style)
		intr += e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	}

	if intr > 0 && intr < avail {
		return intr
	}

	return avail
}

// floatsPackRoom is the horizontal room left beside existing floats for a
// new float on side isLeft.
func floatsPackRoom(floats *floatState, contentX, contentW float64, isLeft bool) float64 {
	room := contentX + contentW - floats.leftEdge
	if !isLeft {
		room = floats.rightEdge - contentX
	}

	if other := floats.rightEdge - floats.leftEdge; other < room {
		room = other
	}

	return room
}

func maxY(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

func minY(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

// shiftBoxOps translates every op in b's op range by (dx, dy).
// Deferred chrome owned by b's subtree is shifted too so finalizeChrome
// places backgrounds/borders at the post-move geometry.
func (e *engine) shiftBoxOps(boxNode *box, deltaX, deltaY float64) {
	if deltaX == 0 && deltaY == 0 || boxNode == nil {
		return
	}

	if boxNode.opEnd >= boxNode.opStart {
		for k := boxNode.opStart; k <= boxNode.opEnd && k < len(e.ops); k++ {
			e.ops[k].X += deltaX
			e.ops[k].Y += deltaY
		}
	}

	shiftDeferredChrome(e.deferredChrome, boxNode, deltaX, deltaY)
}

// shiftDeferredChrome translates the chrome ops of every deferred entry owned
// by b's subtree so finalizeChrome places backgrounds/borders at the new
// geometry.
func shiftDeferredChrome(entries []chromeEntry, boxNode *box, deltaX, deltaY float64) {
	if len(entries) == 0 {
		return
	}

	inSubtree := markBoxSubtree(boxNode)

	for idx := range entries {
		if _, ok := inSubtree[entries[idx].b]; !ok {
			continue
		}

		for j := range entries[idx].ops {
			entries[idx].ops[j].X += deltaX
			entries[idx].ops[j].Y += deltaY
		}
	}
}

// markBoxSubtree returns the set of boxes in b's subtree (b included).
func markBoxSubtree(boxNode *box) map[*box]struct{} {
	inSubtree := map[*box]struct{}{}

	var mark func(*box)
	mark = func(posX *box) {
		if posX == nil {
			return
		}

		inSubtree[posX] = struct{}{}

		for _, c := range posX.children {
			mark(c)
		}
	}
	mark(boxNode)

	return inSubtree
}

func collapseMargins(acc, boxN float64) float64 {
	if acc <= 0 && boxN <= 0 {
		return acc + boxN
	}

	if acc > boxN {
		return acc
	}

	return boxN
}
