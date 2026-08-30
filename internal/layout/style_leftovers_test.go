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
		"stroke-dasharray":  "5pt 10pt 15pt",
		"stroke-dashoffset": "2pt",
		"stroke-linecap":    "round",
		"stroke-linejoin":   "bevel",
		"stroke-miterlimit": "4",
	}
	applyRestProps(&s, raw, ctx, nil)

	if len(s.StrokeDashArray) != 3 || s.StrokeDashArray[0] != 5 || s.StrokeDashArray[1] != 10 || s.StrokeDashArray[2] != 15 {
		t.Errorf("mismatch on stroke-dasharray: %v", s.StrokeDashArray)
	}
	if s.StrokeDashOffset != 2 || s.StrokeLineCap != "round" || s.StrokeLineJoin != "bevel" || s.StrokeMiterLimit != 4 {
		t.Errorf("mismatch on stroke props: off=%v, cap=%q, join=%q, miter=%v",
			s.StrokeDashOffset, s.StrokeLineCap, s.StrokeLineJoin, s.StrokeMiterLimit)
	}
}
