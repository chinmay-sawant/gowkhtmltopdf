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

func (f *Font) glyphContoursID(g uint16, dx, dy, sx, sy float64) [][]GlyphPoint {
	raw := f.glyphOutline(g)
	if len(raw) < 10 {
		return nil
	}
	numContours := int16(binary.BigEndian.Uint16(raw[0:2]))
	if numContours < 0 {
		return f.compositeContours(raw, dx, dy, sx, sy)
	}
	return f.simpleContours(raw, int(numContours), dx, dy, sx, sy)
}

func (f *Font) simpleContours(raw []byte, numContours int, dx, dy, sx, sy float64) [][]GlyphPoint {
	if numContours == 0 || len(raw) < 10+2*numContours {
		return nil
	}
	endPts := make([]int, numContours)
	for i := 0; i < numContours; i++ {
		endPts[i] = int(binary.BigEndian.Uint16(raw[10+2*i:]))
	}
	nPts := endPts[numContours-1] + 1
	pos := 10 + 2*numContours
	if pos+2 > len(raw) {
		return nil
	}
	insLen := int(binary.BigEndian.Uint16(raw[pos:]))
	pos += 2 + insLen
	if pos > len(raw) {
		return nil
	}

	flags := make([]byte, nPts)
	for i := 0; i < nPts; {
		if pos >= len(raw) {
			return nil
		}
		fl := raw[pos]
		pos++
		flags[i] = fl
		i++
		if fl&8 != 0 { // REPEAT
			if pos >= len(raw) {
				return nil
			}
			rep := int(raw[pos])
			pos++
			for r := 0; r < rep && i < nPts; r++ {
				flags[i] = fl
				i++
			}
		}
	}

	xs := make([]float64, nPts)
	ys := make([]float64, nPts)
	var x int16
	for i := 0; i < nPts; i++ {
		fl := flags[i]
		if fl&2 != 0 { // X_SHORT_VECTOR
			if pos >= len(raw) {
				return nil
			}
			v := int16(raw[pos])
			pos++
			if fl&16 != 0 {
				x += v
			} else {
				x -= v
			}
		} else if fl&16 == 0 { // X is signed 16-bit delta
			if pos+2 > len(raw) {
				return nil
			}
			x += int16(binary.BigEndian.Uint16(raw[pos:]))
			pos += 2
		}
		xs[i] = float64(x)*sx + dx
	}
	var y int16
	for i := 0; i < nPts; i++ {
		fl := flags[i]
		if fl&4 != 0 { // Y_SHORT_VECTOR
			if pos >= len(raw) {
				return nil
			}
			v := int16(raw[pos])
			pos++
			if fl&32 != 0 {
				y += v
			} else {
				y -= v
			}
		} else if fl&32 == 0 {
			if pos+2 > len(raw) {
				return nil
			}
			y += int16(binary.BigEndian.Uint16(raw[pos:]))
			pos += 2
		}
		ys[i] = float64(y)*sy + dy
	}

	out := make([][]GlyphPoint, 0, numContours)
	start := 0
	for _, end := range endPts {
		if end < start || end >= nPts {
			break
		}
		c := make([]GlyphPoint, 0, end-start+1)
		for i := start; i <= end; i++ {
			c = append(c, GlyphPoint{X: xs[i], Y: ys[i], On: flags[i]&1 != 0})
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
		var ax, ay float64
		if flags&1 != 0 { // ARG_1_AND_2_ARE_WORDS
			if pos+4 > len(raw) {
				break
			}
			if flags&2 != 0 { // ARGS_ARE_XY_VALUES
				ax = float64(int16(binary.BigEndian.Uint16(raw[pos:])))
				ay = float64(int16(binary.BigEndian.Uint16(raw[pos+2:])))
			}
			pos += 4
		} else {
			if pos+2 > len(raw) {
				break
			}
			if flags&2 != 0 {
				ax = float64(int8(raw[pos]))
				ay = float64(int8(raw[pos+1]))
			}
			pos += 2
		}
		csx, csy := 1.0, 1.0
		if flags&8 != 0 { // WE_HAVE_A_SCALE
			if pos+2 > len(raw) {
				break
			}
			s := float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / 16384.0
			csx, csy = s, s
			pos += 2
		} else if flags&64 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
			if pos+4 > len(raw) {
				break
			}
			csx = float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / 16384.0
			csy = float64(int16(binary.BigEndian.Uint16(raw[pos+2:]))) / 16384.0
			pos += 4
		} else if flags&128 != 0 { // WE_HAVE_A_TWO_BY_TWO
			if pos+8 > len(raw) {
				break
			}
			// approximate with x/y scales only
			csx = float64(int16(binary.BigEndian.Uint16(raw[pos:]))) / 16384.0
			csy = float64(int16(binary.BigEndian.Uint16(raw[pos+6:]))) / 16384.0
			pos += 8
		}
		sub := f.glyphContoursID(child, dx+ax*sx, dy+ay*sy, sx*csx, sy*csy)
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
	if steps < 2 {
		steps = 4
	}
	// ensure contour is closed logically
	pts := make([]GlyphPoint, len(c))
	copy(pts, c)

	var out []GlyphPoint
	n := len(pts)
	i := 0
	for i < n {
		cur := pts[i]
		if cur.On {
			out = append(out, cur)
			next := pts[(i+1)%n]
			if next.On {
				i++
				continue
			}
			// cur on, next off: quadratic to following on (or midpoint)
			ctrl := next
			end := pts[(i+2)%n]
			if !end.On {
				end = GlyphPoint{
					X:  (ctrl.X + end.X) / 2,
					Y:  (ctrl.Y + end.Y) / 2,
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
			i += 2
			continue
		}
		// off-curve start: treat previous implied mid as start
		prev := pts[(i-1+n)%n]
		start := prev
		if !prev.On {
			start = GlyphPoint{X: (prev.X + cur.X) / 2, Y: (prev.Y + cur.Y) / 2, On: true}
			out = append(out, start)
		}
		ctrl := cur
		end := pts[(i+1)%n]
		if !end.On {
			end = GlyphPoint{X: (ctrl.X + end.X) / 2, Y: (ctrl.Y + end.Y) / 2, On: true}
		}
		for s := 1; s <= steps; s++ {
			t := float64(s) / float64(steps)
			out = append(out, GlyphPoint{
				X:  quad(start.X, ctrl.X, end.X, t),
				Y:  quad(start.Y, ctrl.Y, end.Y, t),
				On: true,
			})
		}
		i++
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
	for _, p := range pts[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return
}
