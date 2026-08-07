package layout

import (
	"math"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
)

// Matrix2D is a CSS/SVG-style 2D affine transform:
//
//	x' = A*x + C*y + E
//	y' = B*x + D*y + F
type Matrix2D struct {
	A, B, C, D, E, F float64
}

// IdentityMatrix returns the identity transform.
func IdentityMatrix() Matrix2D {
	return Matrix2D{A: 1, D: 1} //nolint:exhaustruct // intentional zero fields
}

// IsIdentity reports whether m is the identity (within a small epsilon).
func (m Matrix2D) IsIdentity() bool {
	const eps = 1e-9

	return math.Abs(m.A-1) < eps && math.Abs(m.D-1) < eps &&
		math.Abs(m.B) < eps && math.Abs(m.C) < eps &&
		math.Abs(m.E) < eps && math.Abs(m.F) < eps
}

// Mul returns m * n (apply n first, then m) for column vectors.
func (m Matrix2D) Mul(node Matrix2D) Matrix2D {
	return Matrix2D{
		A: m.A*node.A + m.C*node.B,
		B: m.B*node.A + m.D*node.B,
		C: m.A*node.C + m.C*node.D,
		D: m.B*node.C + m.D*node.D,
		E: m.A*node.E + m.C*node.F + m.E,
		F: m.B*node.E + m.D*node.F + m.F,
	}
}

// Translate returns a pure translation matrix.
func Translate(tx, ty float64) Matrix2D {
	return Matrix2D{A: 1, D: 1, E: tx, F: ty} //nolint:exhaustruct // intentional zero fields
}

// Scale returns a pure scale matrix about the origin.
func Scale(sx, sy float64) Matrix2D {
	return Matrix2D{A: sx, D: sy} //nolint:exhaustruct // intentional zero fields
}

// RotateDeg returns a rotation by deg degrees about the origin (CSS y-down).
func RotateDeg(deg float64) Matrix2D {
	rad := deg * math.Pi / degHalfCircle
	c, s := math.Cos(rad), math.Sin(rad)

	return Matrix2D{A: c, B: s, C: -s, D: c} //nolint:exhaustruct // intentional zero fields
}

// SkewXDeg returns a skewX matrix (CSS degrees).
func SkewXDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / degHalfCircle)

	return Matrix2D{A: 1, D: 1, C: t} //nolint:exhaustruct // intentional zero fields
}

// SkewYDeg returns a skewY matrix (CSS degrees).
func SkewYDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / degHalfCircle)

	return Matrix2D{A: 1, D: 1, B: t} //nolint:exhaustruct // intentional zero fields
}

// BakeOrigin returns T(ox,oy) * m * T(-ox,-oy).
func BakeOrigin(m Matrix2D, ox, oy float64) Matrix2D {
	return Translate(ox, oy).Mul(m).Mul(Translate(-ox, -oy))
}

// Apply maps (x,y) through m.
func (m Matrix2D) Apply(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// transformOriginSpec stores unresolved transform-origin components.
// Percentages are 0–100 against the border box; lengths are absolute pt.
type transformOriginSpec struct {
	X, Y     float64
	XPercent bool
	YPercent bool
}

func defaultTransformOrigin() transformOriginSpec {
	return transformOriginSpec{X: cssCenterPercent, Y: cssCenterPercent, XPercent: true, YPercent: true}
}

func resolveTransformOrigin(spec transformOriginSpec, b *box) (ox, oy float64) {
	if b == nil {
		return 0, 0
	}

	if spec.XPercent {
		ox = b.x + b.w*spec.X/cssPercent
	} else {
		ox = b.x + spec.X
	}

	if spec.YPercent {
		oy = b.y + b.height*spec.Y/cssPercent
	} else {
		oy = b.y + spec.Y
	}

	return ox, oy
}

// parseTransformList parses a CSS transform function list into a single matrix.
// Returns ok=false for unrecognized input (caller keeps prior/initial).
// "none" yields identity with ok=true and has=false.
func parseTransformList(value string, fs float64) (m Matrix2D, has bool, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return IdentityMatrix(), false, false
	}

	if strings.EqualFold(value, "none") {
		return IdentityMatrix(), false, true
	}

	m = IdentityMatrix()
	rest := value

	for {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}

		name, args, next, pok := splitTransformFunc(rest)
		if !pok {
			return IdentityMatrix(), false, false
		}

		fm, fok := parseOneTransformFunc(name, args, fs)
		if !fok {
			return IdentityMatrix(), false, false
		}
		// Left-to-right post-multiply: M = M * Fi
		m = m.Mul(fm)
		has = true
		rest = next
	}

	return m, has, true
}

func splitTransformFunc(s string) (name, args, rest string, ok bool) {
	s = strings.TrimSpace(s)
	idx := 0

	for idx < len(s) && ((s[idx] >= 'a' && s[idx] <= 'z') || (s[idx] >= 'A' && s[idx] <= 'Z') || s[idx] == '-') {
		idx++
	}

	if idx == 0 {
		return "", "", "", false
	}

	name = strings.ToLower(s[:idx])

	s = strings.TrimSpace(s[idx:])
	if len(s) == 0 || s[0] != '(' {
		return "", "", "", false
	}

	depth := 0

	for jdx := range len(s) {
		switch s[jdx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				args = strings.TrimSpace(s[1:jdx])
				rest = strings.TrimSpace(s[jdx+1:])

				return name, args, rest, true
			}
		}
	}

	return "", "", "", false
}

func parseOneTransformFunc(name, args string, fs float64) (Matrix2D, bool) {
	parts := splitTransformArgs(args)

	switch name {
	case "matrix":
		if len(parts) != matrixCoeffCount {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		vals := make([]float64, matrixCoeffCount)

		for idx, p := range parts {
			// CSS matrix() takes six <number>s (user units ≈ px at 96dpi).
			val, ok := parseUnitless(p)
			if !ok {
				return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
			}

			if idx >= minBoxPt {
				// e,f: translate in px → pt for our canvas.
				vals[idx] = pxToPt(val)
			} else {
				vals[idx] = val
			}
		}
		// CSS matrix(a,b,c,d,e,f): x'=ax+cy+e, y'=bx+dy+f
		return Matrix2D{A: vals[0], B: vals[1], C: vals[2], D: vals[3], E: vals[4], F: vals[5]}, true
	case "translate":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		textX, isOK := parseTransformLength(parts[0], fs)
		if !isOK {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		typeY := 0.0
		if len(parts) == two {
			typeY, isOK = parseTransformLength(parts[1], fs)
			if !isOK {
				return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
			}
		}

		return Translate(textX, typeY), true
	case "translatex":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		tx, ok := parseTransformLength(parts[0], fs)
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return Translate(tx, 0), true
	case "translatey":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		ty, ok := parseTransformLength(parts[0], fs)
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return Translate(0, ty), true
	case "scale":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		startX, isOK := parseUnitless(parts[0])
		if !isOK {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		startY := startX
		if len(parts) == two {
			startY, isOK = parseUnitless(parts[1])
			if !isOK {
				return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
			}
		}

		return Scale(startX, startY), true
	case "scalex":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		sx, ok := parseUnitless(parts[0])
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return Scale(sx, 1), true
	case "scaley":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		sy, ok := parseUnitless(parts[0])
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return Scale(1, sy), true
	case "rotate":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return RotateDeg(deg), true
	case "skew":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		axVal, isOK := parseAngleDeg(parts[0])
		if !isOK {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		absY := 0.0
		if len(parts) == two {
			absY, isOK = parseAngleDeg(parts[1])
			if !isOK {
				return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
			}
		}

		return SkewXDeg(axVal).Mul(SkewYDeg(absY)), true
	case "skewx":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return SkewXDeg(deg), true
	case "skewy":
		if len(parts) != 1 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		return SkewYDeg(deg), true
	default:
		// 3D / perspective / matrix3d: reject (non-goal)
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}
}

func splitTransformArgs(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}

	var parts []string

	start := 0
	depth := 0

	for idx := range len(args) {
		switch args[idx] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(args[start:idx]))
				start = idx + 1
			}
		}
	}

	parts = append(parts, strings.TrimSpace(args[start:]))

	return parts
}

func parseUnitless(s string) (float64, bool) {
	s = strings.TrimSpace(s)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

func parseAngleDeg(cssSheet string) (float64, bool) {
	cssSheet = strings.TrimSpace(strings.ToLower(cssSheet))
	if cssSheet == "" {
		return 0, false
	}

	switch {
	case strings.HasSuffix(cssSheet, "deg"):
		v, err := strconv.ParseFloat(strings.TrimSpace(cssSheet[:len(cssSheet)-3]), 64)

		return v, err == nil
	case strings.HasSuffix(cssSheet, "rad"):
		v, err := strconv.ParseFloat(strings.TrimSpace(cssSheet[:len(cssSheet)-3]), 64)
		if err != nil {
			return 0, false
		}

		return v * 180 / math.Pi, true
	case strings.HasSuffix(cssSheet, "grad"):
		v, err := strconv.ParseFloat(strings.TrimSpace(cssSheet[:len(cssSheet)-4]), 64)
		if err != nil {
			return 0, false
		}

		return v * turnScale, true
	case strings.HasSuffix(cssSheet, "turn"):
		v, err := strconv.ParseFloat(strings.TrimSpace(cssSheet[:len(cssSheet)-4]), 64)
		if err != nil {
			return 0, false
		}

		return v * degFullCircle, true
	default:
		// Unitless angles are invalid in modern CSS; accept bare numbers as deg
		// for authoring convenience in fixtures.
		v, err := strconv.ParseFloat(cssSheet, 64)

		return v, err == nil
	}
}

// parseTransformLength parses a translate length (px/pt/em/% of 0 → 0 for %).
// Percentages in translate resolve against the reference box at used-value time;
// at parse we store % as a fraction of a 0 base (treated as 0) unless we defer.
// For static print we resolve % translate against 0 at parse (rare) — prefer
// absolute lengths in reports. em uses font-size.
func parseTransformLength(cssS string, fsize float64) (float64, bool) {
	cssS = strings.TrimSpace(cssS)
	if cssS == "0" {
		return 0, true
	}

	val, unit, ok := css.ParseLength(cssS)
	if !ok {
		// bare number = px per CSS Transforms for matrix e/f; for translate too historically
		if f, err := strconv.ParseFloat(cssS, 64); err == nil {
			return pxToPt(f), true
		}

		return 0, false
	}

	if unit == "%" {
		// Without the border box at parse time, % translate is 0 (used value
		// would need layout). Authors should use absolute lengths for print.
		return 0, true
	}

	if unit == "rem" {
		return pxToPt(cssPxRoot) * val, true
	}

	if pt, ok := css.LengthToPt(val, unit, fsize); ok {
		return pt, true
	}

	return 0, false
}

// parseTransformOrigin parses CSS transform-origin (1–3 values; z ignored).
func parseTransformOrigin(value string, fs float64) (transformOriginSpec, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
	}

	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 3 {
		return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
	}

	spec := defaultTransformOrigin()
	parseOne := func(tok string) (v float64, pct bool, ok bool) {
		tok = strings.ToLower(strings.TrimSpace(tok))
		switch tok {
		case "left", "top":
			return 0, true, true
		case "center":
			return cssCenterPercent, true, true
		case "right", "bottom":
			return cssPercent, true, true
		}

		if lv, unit, lok := css.ParseLength(tok); lok && unit == "%" {
			return lv, true, true
		}

		if pt, lok := parseTransformLength(tok, fs); lok {
			return pt, false, true
		}

		return 0, false, false
	}

	switch len(parts) {
	case 1:
		val, pct, ok := parseOne(parts[0])
		if !ok {
			return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
		}

		tok := strings.ToLower(parts[0])
		if tok == "top" || tok == "bottom" {
			spec.Y, spec.YPercent = val, pct
			spec.X, spec.XPercent = 50, true
		} else {
			spec.X, spec.XPercent = val, pct
			spec.Y, spec.YPercent = 50, true
		}
	case two, three:
		// Two-value: horizontal then vertical (keywords may be either axis).
		a, b := strings.ToLower(parts[0]), strings.ToLower(parts[1])
		axIsY := a == "top" || a == "bottom"
		bxIsY := b == "top" || b == "bottom"
		bxIsX := b == "left" || b == "right"

		if axIsY && (bxIsX || !bxIsY) {
			// vertical horizontal → swap
			valA, pageA, oka := parseOne(parts[0])
			valB, pbox, okb := parseOne(parts[1])

			if !oka || !okb {
				return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
			}

			spec.Y, spec.YPercent = valA, pageA
			spec.X, spec.XPercent = valB, pbox
		} else {
			valA, pageA, oka := parseOne(parts[0])
			valB, pbox, okb := parseOne(parts[1])

			if !oka || !okb {
				return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
			}

			spec.X, spec.XPercent = valA, pageA
			spec.Y, spec.YPercent = valB, pbox
		}
		// Optional third value (z) ignored for 2D print.
	}

	return spec, true
}

// parseOpacityValue parses CSS opacity (number 0..1 or percentage).
func parseOpacityValue(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1, false
	}

	if strings.HasSuffix(value, "%") {
		v, err := strconv.ParseFloat(strings.TrimSpace(value[:len(value)-1]), 64)
		if err != nil {
			return 1, false
		}

		return clamp01(v / cssPercent), true
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 1, false
	}

	return clamp01(v), true
}

// parseFilterOpacity extracts opacity() from a filter list; other functions
// are ignored (blur/drop-shadow are permanent print-engine non-goals).
func parseFilterOpacity(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "none" {
		return 1, false
	}

	found := false
	acc := 1.0
	rest := value

	for {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}

		name, args, next, ok := splitTransformFunc(rest)
		if !ok {
			break
		}

		if name == "opacity" {
			if v, pok := parseOpacityValue(args); pok {
				acc *= v
				found = true
			}
		}
		// blur, brightness, etc. ignored
		rest = next
	}

	return acc, found
}

func clamp01(val float64) float64 {
	if val < 0 {
		return 0
	}

	if val > 1 {
		return 1
	}

	return val
}

// stampBoxTransforms walks the laid-out tree and stamps composed 2D
// transform matrices (origin baked) onto display-list ops. Parent transforms
// compose around children; sibling flow geometry is unchanged.
func stampBoxTransforms(b *box, parentAccum Matrix2D, ops []Op) {
	if b == nil {
		return
	}

	accum := parentAccum

	if b.style.HasTransform {
		ox, oy := resolveTransformOrigin(b.style.TransformOrigin, b)
		baked := BakeOrigin(b.style.Transform, ox, oy)
		accum = parentAccum.Mul(baked)
	}

	childCovered := map[int]struct{}{}

	for _, c := range b.children {
		stampBoxTransforms(c, accum, ops)

		if c.opEnd >= c.opStart && c.opStart >= 0 {
			for i := c.opStart; i <= c.opEnd && i < len(ops); i++ {
				childCovered[i] = struct{}{}
			}
		}
	}

	if !accum.IsIdentity() && b.opEnd >= b.opStart && b.opStart >= 0 {
		for idx := b.opStart; idx <= b.opEnd && idx < len(ops); idx++ {
			if _, covered := childCovered[idx]; covered {
				continue
			}

			ops[idx].Xform = accum
			ops[idx].XformSet = true
		}
	}
	// Opacity: multiply down the tree onto exclusive ops.
	if b.style.Opacity < 1 && b.opEnd >= b.opStart && b.opStart >= 0 {
		for idx := b.opStart; idx <= b.opEnd && idx < len(ops); idx++ {
			if _, covered := childCovered[idx]; covered {
				continue
			}

			a := b.style.Opacity
			if ops[idx].PaintOpacity > 0 && ops[idx].PaintOpacity < 1 {
				a *= ops[idx].PaintOpacity
			}

			ops[idx].PaintOpacity = a
		}
		// Children already stamped their exclusive ops; multiply ancestor opacity
		// onto covered child ops as well (opacity composites through descendants).
		for idx := b.opStart; idx <= b.opEnd && idx < len(ops); idx++ {
			if _, covered := childCovered[idx]; !covered {
				continue
			}

			a := b.style.Opacity
			if ops[idx].PaintOpacity > 0 && ops[idx].PaintOpacity < 1 {
				ops[idx].PaintOpacity = a * ops[idx].PaintOpacity
			} else {
				ops[idx].PaintOpacity = a
			}
		}
	}
}

// pdfCTMFromCSS returns the PDF cm matrix equivalent to applying cssXform in
// canvas space (y-down) given canvas→PDF map C(x,y)=(ml+x, K-y).
//
//	CTM = C ∘ cssXform ∘ C⁻¹
func pdfCTMFromCSS(m Matrix2D, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (a, b, c, d, e, f float64) {
	margL := opts.MarginLeft
	keyK := pageH - opts.MarginTop + float64(pageIdx)*contentH
	a = m.A
	b = -m.B
	c = -m.C
	d = m.D
	e = (1-m.A)*margL + m.C*keyK + m.E
	f = keyK*(1-m.D) + m.B*margL - m.F

	return a, b, c, d, e, f
}
