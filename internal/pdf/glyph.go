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

	numContours := int16(binary.BigEndian.Uint16(raw[0:2]))
	if numContours < 0 {
		return f.compositeContours(raw, deltaX, deltaY, scaleX, scaleY)
	}

	return f.simpleContours(raw, int(numContours), deltaX, deltaY, scaleX, scaleY)
}

func (f *Font) simpleContours(raw []byte, numContours int, dx, dy, sx, sy float64) [][]GlyphPoint {
	if numContours == 0 || len(raw) < 10+2*numContours {
		return nil
	}

	endPts := make([]int, numContours)
	for i := range numContours {
		endPts[i] = int(binary.BigEndian.Uint16(raw[10+2*i:]))
	}

	nPts := endPts[numContours-1] + 1

	pos := glyfHeaderSize + uint16Bytes*numContours
	if pos+2 > len(raw) {
		return nil
	}

	insLen := int(binary.BigEndian.Uint16(raw[pos:]))

	pos += uint16Bytes + insLen
	if pos > len(raw) {
		return nil
	}

	flags := make([]byte, nPts)

	for idx := 0; idx < nPts; {
		if pos >= len(raw) {
			return nil
		}

		flag := raw[pos]
		pos++
		flags[idx] = flag
		idx++

		if flag&8 != 0 { // REPEAT
			if pos >= len(raw) {
				return nil
			}

			rep := int(raw[pos])
			pos++

			for r := 0; r < rep && idx < nPts; r++ {
				flags[idx] = flag
				idx++
			}
		}
	}

	xsVal := make([]float64, nPts)
	ysVal := make([]float64, nPts)

	var posX int16

	for idx := range nPts {
		flag := flags[idx]
		if flag&2 != 0 { // X_SHORT_VECTOR
			if pos >= len(raw) {
				return nil
			}

			val := int16(raw[pos])
			pos++

			if flag&16 != 0 {
				posX += val
			} else {
				posX -= val
			}
		} else if flag&16 == 0 { // X is signed 16-bit delta
			if pos+2 > len(raw) {
				return nil
			}

			posX += int16(binary.BigEndian.Uint16(raw[pos:]))
			pos += 2
		}

		xsVal[idx] = float64(posX)*sx + dx
	}

	var posY int16

	for idx := range nPts {
		flag := flags[idx]
		if flag&4 != 0 { // Y_SHORT_VECTOR
			if pos >= len(raw) {
				return nil
			}

			val := int16(raw[pos])
			pos++

			if flag&32 != 0 {
				posY += val
			} else {
				posY -= val
			}
		} else if flag&32 == 0 {
			if pos+2 > len(raw) {
				return nil
			}

			posY += int16(binary.BigEndian.Uint16(raw[pos:]))
			pos += 2
		}

		ysVal[idx] = float64(posY)*sy + dy
	}

	out := make([][]GlyphPoint, 0, numContours)

	start := 0
	for _, end := range endPts {
		if end < start || end >= nPts {
			break
		}

		c := make([]GlyphPoint, 0, end-start+1)
		for i := start; i <= end; i++ {
			c = append(c, GlyphPoint{X: xsVal[i], Y: ysVal[i], On: flags[i]&1 != 0})
		}

		out = append(out, c)
		start = end + 1
	}

	return out
}

func (f *Font) compositeContours(raw []byte, dx, dy, sx, sy float64) [][]GlyphPoint {
	var out [][]GlyphPoint

	pos := 10
	for pos+4 <= len(raw) {
		flags := binary.BigEndian.Uint16(raw[pos:])
		child := binary.BigEndian.Uint16(raw[pos+2:])
		pos += 4

		var axVal, ayVal float64

		if flags&1 != 0 { // ARG_1_AND_2_ARE_WORDS
			if pos+4 > len(raw) {
				break
			}

			if flags&2 != 0 { // ARGS_ARE_XY_VALUES
				axVal = float64(int16(binary.BigEndian.Uint16(raw[pos:])))
				ayVal = float64(int16(binary.BigEndian.Uint16(raw[pos+2:])))
			}

			pos += 4
		} else {
			if pos+2 > len(raw) {
				break
			}

			if flags&2 != 0 {
				axVal = float64(int8(raw[pos]))
				ayVal = float64(int8(raw[pos+1]))
			}

			pos += 2
		}

		csx, csy := 1.0, 1.0

		if flags&8 != 0 { // WE_HAVE_A_SCALE
			if pos+2 > len(raw) {
				break
			}

			s := float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / f2dot14Scale
			csx, csy = s, s
			pos += 2
		} else if flags&64 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
			if pos+4 > len(raw) {
				break
			}

			csx = float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / f2dot14Scale
			csy = float64(int16(binary.BigEndian.Uint16(raw[pos+2:]))) / f2dot14Scale
			pos += 4
		} else if flags&128 != 0 { // WE_HAVE_A_TWO_BY_TWO
			if pos+8 > len(raw) {
				break
			}
			// approximate with x/y scales only
			csx = float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / f2dot14Scale
			csy = float64(int16(binary.BigEndian.Uint16(raw[pos+6:]))) / f2dot14Scale
			pos += 8
		}

		sub := f.glyphContoursID(child, dx+axVal*sx, dy+ayVal*sy, sx*csx, sy*csy)
		out = append(out, sub...)

		if flags&32 == 0 { // MORE_COMPONENTS
			break
		}
	}

	return out
}

// FlattenContour expands TrueType quadratic on/off-curve points into a
// polyline of on-curve points (font units).
func FlattenContour(c []GlyphPoint, steps int) []GlyphPoint {
	if len(c) == 0 {
		return nil
	}

	if steps < minCurveSteps {
		steps = 4
	}
	// ensure contour is closed logically
	pts := make([]GlyphPoint, len(c))
	copy(pts, c)

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
				end = GlyphPoint{
					X:  (ctrl.X + end.X) / midpointDiv,
					Y:  (ctrl.Y + end.Y) / midpointDiv,
					On: true,
				}
			}

			for s := 1; s <= steps; s++ {
				t := float64(s) / float64(steps)
				out = append(out, GlyphPoint{
					X:  quad(cur.X, ctrl.X, end.X, t),
					Y:  quad(cur.Y, ctrl.Y, end.Y, t),
					On: true,
				})
			}

			idx += 2

			continue
		}
		// off-curve start: treat previous implied mid as start
		prev := pts[(idx-1+count)%count]
		start := prev

		if !prev.On {
			start = GlyphPoint{X: (prev.X + cur.X) / midpointDiv, Y: (prev.Y + cur.Y) / midpointDiv, On: true}
			out = append(out, start)
		}

		ctrl := cur
		end := pts[(idx+1)%count]

		if !end.On {
			end = GlyphPoint{X: (ctrl.X + end.X) / midpointDiv, Y: (ctrl.Y + end.Y) / midpointDiv, On: true}
		}

		for s := 1; s <= steps; s++ {
			t := float64(s) / float64(steps)
			out = append(out, GlyphPoint{
				X:  quad(start.X, ctrl.X, end.X, t),
				Y:  quad(start.Y, ctrl.Y, end.Y, t),
				On: true,
			})
		}

		idx++
	}

	return out
}

func quad(p0, p1, p2, t float64) float64 {
	u := 1 - t

	return u*u*p0 + 2*u*t*p1 + t*t*p2
}

// ContourBounds returns min/max of flattened points.
func ContourBounds(pts []GlyphPoint) (minX, minY, maxX, maxY float64) {
	if len(pts) == 0 {
		return
	}

	minX, minY = pts[0].X, pts[0].Y
	maxX, maxY = minX, minY

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

	return
}
