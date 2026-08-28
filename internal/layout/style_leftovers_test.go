//nolint:cyclop,funlen,lll,wsl,varnamelen,exhaustruct,usetesting,testpackage // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestLeftoversPropsWave5(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"clip-path":            "inset(10px)",
		"overflow-clip-margin": "5pt",
		"scroll-margin":        "10pt 20pt",
		"stroke-dasharray":     "5pt 10pt 15pt",
		"stroke-dashoffset":    "2pt",
		"stroke-linecap":       "round",
		"stroke-linejoin":      "bevel",
		"stroke-miterlimit":    "4",
		"fill-rule":            "evenodd",
		"clip-rule":            "evenodd",
		"shape-rendering":      "crispEdges",
		"text-anchor":          "middle",
		"dominant-baseline":    "central",
		"alignment-baseline":   "middle",
		"transform-box":        "fill-box",
		"transform-style":      "preserve-3d",
		"perspective":          "500pt",
		"perspective-origin":   "50pt 100pt",
		"backface-visibility":  "hidden",
		"ruby-align":           "space-around",
		"ruby-position":        "over",
		"ruby-merge":           "collapse",
		"ruby-overhang":        "start",
		"page":                 "cover",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.ClipPath != "inset(10px)" || s.OverflowClipMargin != 5 {
		t.Errorf("mismatch on clip-path/overflow-clip-margin: path=%q, margin=%v", s.ClipPath, s.OverflowClipMargin)
	}
	if s.ScrollMarginTop != 10 || s.ScrollMarginRight != 20 || s.ScrollMarginBottom != 10 || s.ScrollMarginLeft != 20 {
		t.Errorf("mismatch on scroll-margin: top=%v, right=%v, bot=%v, left=%v",
			s.ScrollMarginTop, s.ScrollMarginRight, s.ScrollMarginBottom, s.ScrollMarginLeft)
	}
	if len(s.StrokeDashArray) != 3 || s.StrokeDashArray[0] != 5 || s.StrokeDashArray[1] != 10 || s.StrokeDashArray[2] != 15 {
		t.Errorf("mismatch on stroke-dasharray: %v", s.StrokeDashArray)
	}
	if s.StrokeDashOffset != 2 || s.StrokeLineCap != "round" || s.StrokeLineJoin != "bevel" || s.StrokeMiterLimit != 4 {
		t.Errorf("mismatch on stroke props: off=%v, cap=%q, join=%q, miter=%v",
			s.StrokeDashOffset, s.StrokeLineCap, s.StrokeLineJoin, s.StrokeMiterLimit)
	}
	if s.FillRule != "evenodd" || s.ClipRule != "evenodd" || s.ShapeRendering != "crispEdges" || s.TextAnchor != "middle" {
		t.Errorf("mismatch on SVG rules/text: fill=%q, clip=%q, shape=%q, anchor=%q",
			s.FillRule, s.ClipRule, s.ShapeRendering, s.TextAnchor)
	}
	if s.TransformBox != "fill-box" || s.TransformStyle != "preserve-3d" || s.Perspective != 500 || s.BackfaceVisibility != "hidden" {
		t.Errorf("mismatch on transform box/style/persp: box=%q, style=%q, persp=%v, backface=%q",
			s.TransformBox, s.TransformStyle, s.Perspective, s.BackfaceVisibility)
	}
	if s.RubyAlign != "space-around" || s.RubyPosition != "over" || s.RubyMerge != "collapse" || s.RubyOverhang != "start" {
		t.Errorf("mismatch on ruby props: align=%q, pos=%q, merge=%q, overhang=%q",
			s.RubyAlign, s.RubyPosition, s.RubyMerge, s.RubyOverhang)
	}
	if s.Page != "cover" {
		t.Errorf("mismatch on page: %q", s.Page)
	}
}
