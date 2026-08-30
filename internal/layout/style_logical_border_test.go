//nolint:cyclop,funlen,lll,wsl,varnamelen,exhaustruct,usetesting,testpackage // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestLogicalBorderBlockAndInline(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	// Box 1: horizontal-tb default
	s1 := initialStyle()
	raw1 := map[string]string{
		"border-block":  "2pt solid red",
		"border-inline": "4pt dashed blue",
	}
	applyRestProps(&s1, raw1, ctx, nil)

	if s1.BorderTop.Width != 2 || s1.BorderBottom.Width != 2 {
		t.Errorf("expected BorderTop/Bottom Width 2, got top=%v, bot=%v", s1.BorderTop.Width, s1.BorderBottom.Width)
	}
	if s1.BorderLeft.Width != 4 || s1.BorderRight.Width != 4 {
		t.Errorf("expected BorderLeft/Right Width 4, got left=%v, right=%v", s1.BorderLeft.Width, s1.BorderRight.Width)
	}
	if s1.BorderTop.Color != [3]float64{1, 0, 0} || s1.BorderLeft.Color != [3]float64{0, 0, 1} {
		t.Errorf("color mismatch on box1: top=%v, left=%v", s1.BorderTop.Color, s1.BorderLeft.Color)
	}

	// Box 2: vertical-rl
	s2 := initialStyle()
	s2.WritingMode = writingModeVerticalRL
	raw2 := map[string]string{
		"border-block-start":  "3pt solid #00ff00",
		"border-block-end":    "5pt dotted #646464",
		"border-inline-start": "6pt solid #323232",
		"border-inline-end":   "7pt solid #505050",
	}
	applyRestProps(&s2, raw2, ctx, nil)

	// In vertical-rl: block-start is right, block-end is left, inline-start is top, inline-end is bottom
	if s2.BorderRight.Width != 3 || s2.BorderLeft.Width != 5 {
		t.Errorf("expected BorderRight 3, BorderLeft 5, got right=%v, left=%v", s2.BorderRight.Width, s2.BorderLeft.Width)
	}
	if s2.BorderTop.Width != 6 || s2.BorderBottom.Width != 7 {
		t.Errorf("expected BorderTop 6, BorderBottom 7, got top=%v, bot=%v", s2.BorderTop.Width, s2.BorderBottom.Width)
	}

	// Box 3: direction: rtl in horizontal-tb
	s3 := initialStyle()
	s3.Direction = "rtl"
	raw3 := map[string]string{
		"border-inline-start-width": "10pt",
		"border-inline-start-style": "solid",
		"border-inline-start-color": "red",
		"border-inline-end-width":   "20pt",
		"border-inline-end-style":   "dashed",
		"border-inline-end-color":   "blue",
	}
	applyRestProps(&s3, raw3, ctx, nil)

	// In horizontal-tb RTL: inline-start is right, inline-end is left
	if s3.BorderRight.Width != 10 || s3.BorderLeft.Width != 20 {
		t.Errorf("expected BorderRight 10, BorderLeft 20 for RTL, got right=%v, left=%v", s3.BorderRight.Width, s3.BorderLeft.Width)
	}
	if s3.BorderRight.Style != "solid" || s3.BorderLeft.Style != "dashed" {
		t.Errorf("style mismatch on RTL box3: right=%v, left=%v", s3.BorderRight.Style, s3.BorderLeft.Style)
	}
}

func TestLogicalCornerRadii(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	// Horizontal LTR: start-start is TopLeft, start-end is TopRight, end-start is BottomLeft, end-end is BottomRight
	s1 := initialStyle()
	raw1 := map[string]string{
		"border-start-start-radius": "10pt",
		"border-start-end-radius":   "20pt",
		"border-end-start-radius":   "30pt",
		"border-end-end-radius":     "40pt",
	}
	applyRestProps(&s1, raw1, ctx, nil)

	if s1.BorderRadiusTopLeft != 10 || s1.BorderRadiusTopRight != 20 ||
		s1.BorderRadiusBottomLeft != 30 || s1.BorderRadiusBottomRight != 40 {
		t.Errorf("mismatch on logical corner radii: TL=%v, TR=%v, BL=%v, BR=%v",
			s1.BorderRadiusTopLeft, s1.BorderRadiusTopRight, s1.BorderRadiusBottomLeft, s1.BorderRadiusBottomRight)
	}

	// Side radii: block-start is top (TL, TR = 15), inline-end is right (TR, BR = 25 -> TR overwritten with 25)
	s2 := initialStyle()
	raw2a := map[string]string{
		"border-block-start-radius": "15pt",
	}
	applyRestProps(&s2, raw2a, ctx, nil)
	raw2b := map[string]string{
		"border-inline-end-radius": "25pt",
	}
	applyRestProps(&s2, raw2b, ctx, nil)

	// block-start is top (TL, TR = 15), inline-end is right (TR, BR = 25 -> TR overwritten with 25)
	if s2.BorderRadiusTopLeft != 15 || s2.BorderRadiusTopRight != 25 || s2.BorderRadiusBottomRight != 25 {
		t.Errorf("mismatch on side radii: TL=%v, TR=%v, BR=%v",
			s2.BorderRadiusTopLeft, s2.BorderRadiusTopRight, s2.BorderRadiusBottomRight)
	}
}

func TestMarginBreakProperty(t *testing.T) {
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
		"margin-break": "keep",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.MarginBreak != "keep" {
		t.Errorf("expected MarginBreak 'keep', got %q", s.MarginBreak)
	}
}
