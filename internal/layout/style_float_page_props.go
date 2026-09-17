//nolint:cyclop,exhaustruct,mnd,varnamelen,wsl // page-float property parsers + place nudge
package layout

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// CSS Page Floats apply group (css-page-floats-3 lite):
//   - float-offset: <length-percentage> — length nudge on placeFloat
//   - float-reference: inline | column | region | page — inline documents the
//     current BFC; page/column/region store as Partial (no pagination change)
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
