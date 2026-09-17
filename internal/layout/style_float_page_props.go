//nolint:cyclop,exhaustruct,mnd,varnamelen,wsl // page-float property parsers + place nudge
package layout

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// CSS Page Floats apply group (css-page-floats-3 lite):
//   - float-offset: <length-percentage> — length nudge on placeFloat
//   - float-reference: inline | column | region | page
//       inline  = current BFC (CSS2 floats)
//       page    = page content box (x=0, width=viewport)
//       column  = parent BFC when nested inside a multicol ancestor
//       region  = stored; treated as inline (no CSS Regions)
//
// float-defer stays Unsupported (no apply arm, no defer model).

const (
	floatRefInline = "inline"
	floatRefColumn = "column"
	floatRefRegion = "region"
	floatRefPage   = "page"
)

// applyFloatPageProps owns float-offset and float-reference.
func applyFloatPageProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "float-offset":
		if pt, pct, ok := parseFloatOffset(value, fsize, ctx); ok {
			style.FloatOffset = pt
			style.FloatOffsetPercent = pct
		}
	case "float-reference":
		if ref, ok := parseFloatReference(value); ok {
			style.FloatReference = ref
		}
	default:
		return false
	}

	return true
}

func parseFloatOffset(raw string, fsize float64, ctx *styleContext) (float64, float64, bool) {
	value := normalizeCSSValue(raw)
	if value == "" {
		return 0, -1, false
	}

	val, unit, ok := css.ParseLength(value)
	if !ok {
		return 0, -1, false
	}

	if unit == "%" {
		_ = ctx

		return 0, val, true
	}

	pt, converted := lengthToPt(val, unit, fsize)
	if !converted {
		return 0, -1, false
	}

	return pt, -1, true
}

func parseFloatReference(raw string) (string, bool) {
	switch normalizeCSSValue(raw) {
	case floatRefInline, floatRefColumn, floatRefRegion, floatRefPage:
		return normalizeCSSValue(raw), true
	default:
		return "", false
	}
}

// nudgeFloatOffset shifts a just-built float box by float-offset along the
// block axis (positive moves down). Percent resolves against the float's own
// border-box height (lite). Extracted so placeFloat stays a thin caller.
func nudgeFloatOffset(e *engine, fbox *box, sty ResolvedStyle) {
	if fbox == nil {
		return
	}

	var dy float64

	switch {
	case sty.FloatOffsetPercent >= 0:
		dy = fbox.height * sty.FloatOffsetPercent / 100
	case sty.FloatOffset != 0:
		dy = e.scalePt(sty.FloatOffset)
	default:
		return
	}

	if dy == 0 {
		return
	}

	fbox.y += dy
	e.shiftBoxOps(fbox, 0, dy)
}

// floatReferenceBox returns the containing block used to place a float.
// inline / region: current BFC. page: page content box. column: parent BFC
// when the float sits in a nested BFC inside a multicol ancestor.
func (e *engine) floatReferenceBox(
	node *html.Node, sty ResolvedStyle, contentX, contentW, flowY float64,
) (x, w, y float64) {
	switch sty.FloatReference {
	case floatRefPage:
		w = 0
		if e != nil {
			w = e.opts.Width
		}

		if w <= 0 {
			w = contentW
		}

		return 0, w, flowY
	case floatRefColumn:
		if x, w, ok := e.columnReferenceBox(node, contentX, contentW); ok {
			return x, w, flowY
		}
	}

	return contentX, contentW, flowY
}

// columnReferenceBox reports the parent BFC (the column box) when the float
// is nested inside a multicol ancestor. Direct-in-column floats already use
// the column BFC, so this is a no-op unless a nested formatting context
// pushed a tighter box.
func (e *engine) columnReferenceBox(
	node *html.Node, contentX, contentW float64,
) (x, w float64, ok bool) {
	if e == nil || !hasMulticolAncestor(e, node) {
		return 0, 0, false
	}

	if n := len(e.bfcStack); n > 0 {
		parent := e.bfcStack[n-1]
		if parent != nil && parent.contentW > 0 &&
			(parent.contentX != contentX || parent.contentW != contentW) {
			return parent.contentX, parent.contentW, true
		}
	}

	return 0, 0, false
}

func hasMulticolAncestor(e *engine, n *html.Node) bool {
	if e == nil || n == nil {
		return false
	}

	for p := n.Parent; p != nil; p = p.Parent {
		st := e.stylePtr(p)
		if st == nil {
			continue
		}

		if st.ColumnCount > 1 || st.ColumnWidth >= 0 || st.ColumnHeight >= 0 {
			return true
		}
	}

	return false
}

// pinFloatToReference reports whether placeFloat should pack against the
// resolved reference box instead of the current BFC edges.
func pinFloatToReference(sty ResolvedStyle, refX, refW, contentX, contentW float64) bool {
	switch sty.FloatReference {
	case floatRefPage:
		return true
	case floatRefColumn:
		return refX != contentX || refW != contentW
	default:
		return false
	}
}
