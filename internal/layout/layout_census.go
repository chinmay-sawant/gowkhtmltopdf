package layout

import "strings"

// textOverflowTolerancePt is the slack allowed before text painted past the
// content box is reported. Sub-point overflow comes from line-break rounding.
const textOverflowTolerancePt = 1.0

// censusOps records the right edge of fill/stroke/image ops and whether any
// link URI is a same-document fragment. opts.Width is the floor so MaxContentX
// is never zero after a real Layout (zero stays the hand-built Result signal).
// Text ops are deliberately excluded from MaxContentX: smart shrinking must
// not rescale a page because prose reaches into the margin. Text-only overflow
// is reported as a warning instead, leaving layout unchanged, because shrinking
// text to fit would change line breaking for every consumer.
func censusOps(ops []Op, opts Options) (float64, bool) {
	maxX := opts.Width
	hasFrag := false
	maxTextX := 0.0

	for idx := range ops {
		switch ops[idx].Kind {
		case OpFillRect, OpStrokeRect, OpImage:
			if ext := ops[idx].X + ops[idx].W; ext > maxX {
				maxX = ext
			}
		case OpLinkURI:
			if !hasFrag && strings.HasPrefix(ops[idx].URI, "#") {
				hasFrag = true
			}
		case OpText:
			if ext := ops[idx].X + ops[idx].W; ext > maxTextX {
				maxTextX = ext
			}
		case OpLine, OpGridRun, OpBullet, OpUnknown, opKindNoop:
			continue
		}
	}

	warnTextOverflow(opts, maxTextX)

	return maxX, hasFrag
}

// warnTextOverflow reports the widest text op when it crosses the content box.
// Text ops are invisible to the fill/stroke/image census used by smart shrink,
// so without this a page whose prose paints into the right margin converts
// silently (learncpp: 7.8-14.1pt with no warning).
func warnTextOverflow(opts Options, maxTextX float64) {
	if opts.Warnf == nil || opts.Width <= 0 {
		return
	}

	if overflow := maxTextX - opts.Width; overflow > textOverflowTolerancePt {
		opts.Warnf("layout: text overflows content box by %.1fpt (text right edge %.1f, content width %.1f)",
			overflow, maxTextX, opts.Width)
	}
}
