package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/html"

// applyAutoContentBottom pins the content-flow bottom edge for auto-sized
// native value controls (meter, progress) and textareas, whose height comes
// from their content rather than a length declaration. The caller keeps
// applying padding and border after this endpoint.
func (e *engine) applyAutoContentBottom(
	node *html.Node, style ResolvedStyle, boxStyle *ResolvedStyle, widget bool, curY float64,
) float64 {
	if widget && style.Height < 0 {
		// Native value controls use their intrinsic font-sized control height
		// when auto-sized. Treating them as ordinary text blocks adds the
		// line-height and authored padding a second time, producing the
		// oversized meter/progress tracks in fixture-56.
		curY = e.nativeWidgetAutoContentBottom(style)
	}

	if node.Name == "textarea" && style.Height < 0 && style.HeightPercent < 0 {
		curY = e.textareaAutoContentBottom(style, node, boxStyle, curY)
	}

	return curY
}
