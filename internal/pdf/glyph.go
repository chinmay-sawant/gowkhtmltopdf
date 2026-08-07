package pdf

import (
	"encoding/binary"
)

// GlyphPoint is one TrueType contour point in font units.
type GlyphPoint struct {
	X, Y float64
	On   bool // on-curve
}

// GlyphContours returns the outline contours for rune r in font units
// (y-up). Missing glyphs return nil.
func (f *Font) GlyphContours(r rune) [][]GlyphPoint {
	g := f.GlyphID(r)
	if g == 0 && r != 0 {
		return nil
	}

	return f.glyphContoursID(g, 0, 0, 1, 1)
}

func (f *Font) glyphContoursID(glyphID uint16, deltaX, deltaY, scaleX, scaleY float64) [][]GlyphPoint {
	raw := f.glyphOutline(glyphID)
	if len(raw) < glyfHeaderSize {
		return nil
	}

	numContours := int16(binary.BigEndian.Uint16(raw[0:2])) //nolint:gosec // numContours is int16 per glyf spec
	if numContours < 0 {
		return f.compositeContours(raw, deltaX, deltaY, scaleX, scaleY)
	}

	return f.simpleContours(raw, int(numContours), deltaX, deltaY, scaleX, scaleY)
}

// decodeAxis reads the delta-encoded coordinates for one axis of a simple
// glyph. shortFlag/signFlag select the X or Y encoding bits; the shared byte
// position advances as the format requires (X deltas precede Y deltas).
func decodeAxis(raw, flags []byte, pos, nPts int, shortFlag, signFlag byte) ([]float64, int, bool) {
	vals := make([]float64, nPts)

	var posVal int16

	for idx := range nPts {
		flag := flags[idx]
		if flag&shortFlag != 0 { // *_SHORT_VECTOR
			if pos >= len(raw) {
				return nil, pos, false
			}

			val := int16(raw[pos])
			pos++

			if flag&signFlag == 0 {
				val = -val
			}

			posVal += val
		} else if flag&signFlag == 0 { // signed 16-bit delta
			if pos+2 > len(raw) {
				return nil, pos, false
			}

			posVal += int16(binary.BigEndian.Uint16(raw[pos:])) //nolint:gosec // delta is int16 per glyf spec
			pos += 2
		}

		vals[idx] = float64(posVal)
	}

	return vals, pos, true
}

// readEndPts reads the end-point indices of a simple glyph's contours.
func readEndPts(raw []byte, numContours int) ([]int, bool) {
	if numContours == 0 || len(raw) < glyfHeaderSize+uint16Bytes*numContours {
		return nil, false
	}

	endPts := make([]int, numContours)
	for i := range numContours {
		endPts[i] = int(binary.BigEndian.Uint16(raw[glyfHeaderSize+uint16Bytes*i:]))
	}

	return endPts, true
}

// decodeGlyphFlags expands the (possibly repeated) point-flag bytes,
// skipping the instruction block that follows them.
func decodeGlyphFlags(raw []byte, pos, nPts int) ([]byte, int, bool) {
	if pos+2 > len(raw) {
		return nil, pos, false
	}

	insLen := int(binary.BigEndian.Uint16(raw[pos:]))

	pos += uint16Bytes + insLen
	if pos > len(raw) {
		return nil, pos, false
	}

	flags := make([]byte, nPts)

	for idx := 0; idx < nPts; {
		if pos >= len(raw) {
			return nil, pos, false
		}

		flag := raw[pos]
		pos++
		flags[idx] = flag
		idx++

		if flag&glyfRepeatFlag != 0 { // REPEAT
			if pos >= len(raw) {
				return nil, pos, false
			}

			rep := int(raw[pos])
			pos++

			for r := 0; r < rep && idx < nPts; r++ {
				flags[idx] = flag
				idx++
			}
		}
	}

	return flags, pos, true
}

// assembleContours groups the decoded point coordinates into contours.
func assembleContours(
	endPts []int,
	xsVal, ysVal []float64,
	flags []byte,
	scaleX, scaleY, deltaX, deltaY float64,
) [][]GlyphPoint {
	out := make([][]GlyphPoint, 0, len(endPts))

	start := 0
	for _, end := range endPts {
		if end < start || end >= len(xsVal) {
			break
		}

		c := make([]GlyphPoint, 0, end-start+1)
		for i := start; i <= end; i++ {
			c = append(c, GlyphPoint{X: xsVal[i]*scaleX + deltaX, Y: ysVal[i]*scaleY + deltaY, On: flags[i]&glyfOnCurve != 0})
		}

		out = append(out, c)
		start = end + 1
	}

	return out
}

func (f *Font) simpleContours(raw []byte, numContours int, deltaX, deltaY, scaleX, scaleY float64) [][]GlyphPoint {
	endPts, valid := readEndPts(raw, numContours)
	if !valid {
		return nil
	}

	nPts := endPts[numContours-1] + 1

	flags, pos, valid := decodeGlyphFlags(raw, glyfHeaderSize+uint16Bytes*numContours, nPts)
	if !valid {
		return nil
	}

	xsVal, pos, valid := decodeAxis(raw, flags, pos, nPts, glyfXShortVector, glyfXSameOrPos)
	if !valid {
		return nil
	}

	ysVal, _, valid := decodeAxis(raw, flags, pos, nPts, glyfYShortVector, glyfYSameOrPos)
	if !valid {
		return nil
	}

	return assembleContours(endPts, xsVal, ysVal, flags, scaleX, scaleY, deltaX, deltaY)
}

// readComponentArgs decodes a composite component's argument words.
func readComponentArgs(raw []byte, pos int, flags uint16) (float64, float64, int, bool) {
	if flags&glyfArgWords != 0 { // ARG_1_AND_2_ARE_WORDS
		if pos+uint32Bytes > len(raw) {
			return 0, 0, pos, false
		}

		axVal, ayVal := 0.0, 0.0
		if flags&glyfArgsXYValues != 0 { // ARGS_ARE_XY_VALUES
			axVal = float64(int16(binary.BigEndian.Uint16(raw[pos:])))   //nolint:gosec // args are int16 per glyf spec
			ayVal = float64(int16(binary.BigEndian.Uint16(raw[pos+2:]))) //nolint:gosec // args are int16 per glyf spec
		}

		return axVal, ayVal, pos + uint32Bytes, true
	}

	if pos+int16Bytes > len(raw) {
		return 0, 0, pos, false
	}

	axVal, ayVal := 0.0, 0.0
	if flags&glyfArgsXYValues != 0 {
		axVal = float64(int8(raw[pos]))
		ayVal = float64(int8(raw[pos+1]))
	}

	return axVal, ayVal, pos + int16Bytes, true
}

// decodeCompositeComponent reads a composite glyph component's offset and
// scale values from raw at pos, returning them plus the next position.
func decodeCompositeComponent(raw []byte, pos int, flags uint16) (float64, float64, float64, float64, int, bool) {
	axVal, ayVal, next, ok := readComponentArgs(raw, pos, flags)
	if !ok {
		return 0, 0, 1, 1, next, false
	}

	csxVal, csyVal := 1.0, 1.0
	pos = next

	switch {
	case flags&glyfHaveScale != 0: // WE_HAVE_A_SCALE
		if pos+scaleBytes > len(raw) {
			return 0, 0, 1, 1, pos, false
		}

		s := f2dot14(raw, pos)
		csxVal, csyVal = s, s
		pos += scaleBytes
	case flags&glyfHaveXYScale != 0: // WE_HAVE_AN_X_AND_Y_SCALE
		if pos+xyScaleBytes > len(raw) {
			return 0, 0, 1, 1, pos, false
		}

		csxVal = f2dot14(raw, pos)
		csyVal = f2dot14(raw, pos+scaleBytes)
		pos += xyScaleBytes
	case flags&glyfHaveTwoByTwo != 0: // WE_HAVE_A_TWO_BY_TWO
		if pos+twoByTwoBytes > len(raw) {
			return 0, 0, 1, 1, pos, false
		}
		// approximate with x/y scales only
		csxVal = f2dot14(raw, pos)
		csyVal = f2dot14(raw, pos+twoByTwoSecondOff)
		pos += twoByTwoBytes
	}

	return axVal, ayVal, csxVal, csyVal, pos, true
}

// f2dot14 decodes a TrueType F2DOT14 scale value.
func f2dot14(raw []byte, pos int) float64 {
	//nolint:gosec // scale is F2DOT14 per glyf spec
	return float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / f2dot14Scale
}

func (f *Font) compositeContours(raw []byte, deltaX, deltaY, scaleX, scaleY float64) [][]GlyphPoint {
	var out [][]GlyphPoint

	pos := glyfHeaderSize
	for pos+uint32Bytes <= len(raw) {
		flags := binary.BigEndian.Uint16(raw[pos:])
		child := binary.BigEndian.Uint16(raw[pos+2:])

		axVal, ayVal, csxVal, csyVal, next, ok := decodeCompositeComponent(raw, pos+uint32Bytes, flags)
		if !ok {
			break
		}

		pos = next

		sub := f.glyphContoursID(child, deltaX+axVal*scaleX, deltaY+ayVal*scaleY, scaleX*csxVal, scaleY*csyVal)
		out = append(out, sub...)

		if flags&glyfMoreComponents == 0 { // MORE_COMPONENTS
			break
		}
	}

	return out
}

// FlattenContour expands TrueType quadratic on/off-curve points into a
// polyline of on-curve points (font units).
func FlattenContour(contour []GlyphPoint, steps int) []GlyphPoint {
	if len(contour) == 0 {
		return nil
	}

	if steps < minCurveSteps {
		steps = 4
	}
	// ensure contour is closed logically
	pts := make([]GlyphPoint, len(contour))
	copy(pts, contour)

	var out []GlyphPoint

	count := len(pts)
	idx := 0

	for idx < count {
		cur := pts[idx]
		if cur.On {
			out = append(out, cur)

			next := pts[(idx+1)%count]
			if next.On {
				idx++

				continue
			}
			// cur on, next off: quadratic to following on (or midpoint)
			ctrl := next
			end := pts[(idx+midpointDiv)%count]

			if !end.On {
				end = midpointOf(ctrl, end)
			}

			out = appendFlattened(out, cur, ctrl, end, steps)
			idx += 2

			continue
		}
		// off-curve start: treat previous implied mid as start
		prev := pts[(idx-1+count)%count]
		start := prev

		if !prev.On {
			start = midpointOf(prev, cur)
			out = append(out, start)
		}

		ctrl := cur
		end := pts[(idx+1)%count]

		if !end.On {
			end = midpointOf(ctrl, end)
		}

		out = appendFlattened(out, start, ctrl, end, steps)
		idx++
	}

	return out
}

// midpointOf returns the on-curve point midway between two points.
func midpointOf(a, b GlyphPoint) GlyphPoint {
	return GlyphPoint{X: (a.X + b.X) / midpointDiv, Y: (a.Y + b.Y) / midpointDiv, On: true}
}

// appendFlattened appends the flattened quadratic samples between start and
// end with ctrl as the off-curve control point.
func appendFlattened(out []GlyphPoint, start, ctrl, end GlyphPoint, steps int) []GlyphPoint {
	for s := 1; s <= steps; s++ {
		t := float64(s) / float64(steps)
		out = append(out, GlyphPoint{
			X:  quad(start.X, ctrl.X, end.X, t),
			Y:  quad(start.Y, ctrl.Y, end.Y, t),
			On: true,
		})
	}

	return out
}

func quad(p0, p1, p2, t float64) float64 {
	u := 1 - t

	return u*u*p0 + 2*u*t*p1 + t*t*p2
}

// ContourBounds returns min/max of flattened points.
func ContourBounds(pts []GlyphPoint) (float64, float64, float64, float64) {
	if len(pts) == 0 {
		return 0, 0, 0, 0
	}

	minX, minY := pts[0].X, pts[0].Y
	maxX, maxY := minX, minY

	for _, page := range pts[1:] {
		if page.X < minX {
			minX = page.X
		}

		if page.Y < minY {
			minY = page.Y
		}

		if page.X > maxX {
			maxX = page.X
		}

		if page.Y > maxY {
			maxY = page.Y
		}
	}

	return minX, minY, maxX, maxY
}
