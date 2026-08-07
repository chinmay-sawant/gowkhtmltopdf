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
	return Matrix2D{A: 1, D: 1}
}

// IsIdentity reports whether m is the identity (within a small epsilon).
func (m Matrix2D) IsIdentity() bool {
	const eps = 1e-9
	return math.Abs(m.A-1) < eps && math.Abs(m.D-1) < eps &&
		math.Abs(m.B) < eps && math.Abs(m.C) < eps &&
		math.Abs(m.E) < eps && math.Abs(m.F) < eps
}

// Mul returns m * n (apply n first, then m) for column vectors.
func (m Matrix2D) Mul(n Matrix2D) Matrix2D {
	return Matrix2D{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

// Translate returns a pure translation matrix.
func Translate(tx, ty float64) Matrix2D {
	return Matrix2D{A: 1, D: 1, E: tx, F: ty}
}

// Scale returns a pure scale matrix about the origin.
func Scale(sx, sy float64) Matrix2D {
	return Matrix2D{A: sx, D: sy}
}

// RotateDeg returns a rotation by deg degrees about the origin (CSS y-down).
func RotateDeg(deg float64) Matrix2D {
	rad := deg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	return Matrix2D{A: c, B: s, C: -s, D: c}
}

// SkewXDeg returns a skewX matrix (CSS degrees).
func SkewXDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / 180)
	return Matrix2D{A: 1, D: 1, C: t}
}

// SkewYDeg returns a skewY matrix (CSS degrees).
func SkewYDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / 180)
	return Matrix2D{A: 1, D: 1, B: t}
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
	return transformOriginSpec{X: 50, Y: 50, XPercent: true, YPercent: true}
}

func resolveTransformOrigin(spec transformOriginSpec, b *box) (ox, oy float64) {
	if b == nil {
		return 0, 0
	}
	if spec.XPercent {
		ox = b.x + b.w*spec.X/100
	} else {
		ox = b.x + spec.X
	}
	if spec.YPercent {
		oy = b.y + b.h*spec.Y/100
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
	i := 0
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || s[i] == '-') {
		i++
	}
	if i == 0 {
		return "", "", "", false
	}
	name = strings.ToLower(s[:i])
	s = strings.TrimSpace(s[i:])
	if len(s) == 0 || s[0] != '(' {
		return "", "", "", false
	}
	depth := 0
	for j := 0; j < len(s); j++ {
		switch s[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				args = strings.TrimSpace(s[1:j])
				rest = strings.TrimSpace(s[j+1:])
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
		if len(parts) != 6 {
			return Matrix2D{}, false
		}
		vals := make([]float64, 6)
		for i, p := range parts {
			// CSS matrix() takes six <number>s (user units ≈ px at 96dpi).
			v, ok := parseUnitless(p)
			if !ok {
				return Matrix2D{}, false
			}
			if i >= 4 {
				// e,f: translate in px → pt for our canvas.
				vals[i] = pxToPt(v)
			} else {
				vals[i] = v
			}
		}
		// CSS matrix(a,b,c,d,e,f): x'=ax+cy+e, y'=bx+dy+f
		return Matrix2D{A: vals[0], B: vals[1], C: vals[2], D: vals[3], E: vals[4], F: vals[5]}, true
	case "translate":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false
		}
		tx, ok := parseTransformLength(parts[0], fs)
		if !ok {
			return Matrix2D{}, false
		}
		ty := 0.0
		if len(parts) == 2 {
			ty, ok = parseTransformLength(parts[1], fs)
			if !ok {
				return Matrix2D{}, false
			}
		}
		return Translate(tx, ty), true
	case "translatex":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		tx, ok := parseTransformLength(parts[0], fs)
		if !ok {
			return Matrix2D{}, false
		}
		return Translate(tx, 0), true
	case "translatey":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		ty, ok := parseTransformLength(parts[0], fs)
		if !ok {
			return Matrix2D{}, false
		}
		return Translate(0, ty), true
	case "scale":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false
		}
		sx, ok := parseUnitless(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		sy := sx
		if len(parts) == 2 {
			sy, ok = parseUnitless(parts[1])
			if !ok {
				return Matrix2D{}, false
			}
		}
		return Scale(sx, sy), true
	case "scalex":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		sx, ok := parseUnitless(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		return Scale(sx, 1), true
	case "scaley":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		sy, ok := parseUnitless(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		return Scale(1, sy), true
	case "rotate":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		return RotateDeg(deg), true
	case "skew":
		if len(parts) < 1 || len(parts) > 2 {
			return Matrix2D{}, false
		}
		ax, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		ay := 0.0
		if len(parts) == 2 {
			ay, ok = parseAngleDeg(parts[1])
			if !ok {
				return Matrix2D{}, false
			}
		}
		return SkewXDeg(ax).Mul(SkewYDeg(ay)), true
	case "skewx":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		return SkewXDeg(deg), true
	case "skewy":
		if len(parts) != 1 {
			return Matrix2D{}, false
		}
		deg, ok := parseAngleDeg(parts[0])
		if !ok {
			return Matrix2D{}, false
		}
		return SkewYDeg(deg), true
	default:
		// 3D / perspective / matrix3d: reject (non-goal)
		return Matrix2D{}, false
	}
}

func splitTransformArgs(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}
	var parts []string
	start := 0
	depth := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(args[start:i]))
				start = i + 1
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

func parseAngleDeg(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false
	}
	switch {
	case strings.HasSuffix(s, "deg"):
		v, err := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-3]), 64)
		return v, err == nil
	case strings.HasSuffix(s, "rad"):
		v, err := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-3]), 64)
		if err != nil {
			return 0, false
		}
		return v * 180 / math.Pi, true
	case strings.HasSuffix(s, "grad"):
		v, err := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-4]), 64)
		if err != nil {
			return 0, false
		}
		return v * 0.9, true
	case strings.HasSuffix(s, "turn"):
		v, err := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-4]), 64)
		if err != nil {
			return 0, false
		}
		return v * 360, true
	default:
		// Unitless angles are invalid in modern CSS; accept bare numbers as deg
		// for authoring convenience in fixtures.
		v, err := strconv.ParseFloat(s, 64)
		return v, err == nil
	}
}

// parseTransformLength parses a translate length (px/pt/em/% of 0 → 0 for %).
// Percentages in translate resolve against the reference box at used-value time;
// at parse we store % as a fraction of a 0 base (treated as 0) unless we defer.
// For static print we resolve % translate against 0 at parse (rare) — prefer
// absolute lengths in reports. em uses font-size.
func parseTransformLength(s string, fs float64) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "0" {
		return 0, true
	}
	v, unit, ok := css.ParseLength(s)
	if !ok {
		// bare number = px per CSS Transforms for matrix e/f; for translate too historically
		if f, err := strconv.ParseFloat(s, 64); err == nil {
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
		return pxToPt(16) * v, true
	}
	if pt, ok := css.LengthToPt(v, unit, fs); ok {
		return pt, true
	}
	return 0, false
}

// parseTransformOrigin parses CSS transform-origin (1–3 values; z ignored).
func parseTransformOrigin(value string, fs float64) (transformOriginSpec, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return transformOriginSpec{}, false
	}
	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 3 {
		return transformOriginSpec{}, false
	}
	spec := defaultTransformOrigin()
	parseOne := func(tok string) (v float64, pct bool, ok bool) {
		tok = strings.ToLower(strings.TrimSpace(tok))
		switch tok {
		case "left", "top":
			return 0, true, true
		case "center":
			return 50, true, true
		case "right", "bottom":
			return 100, true, true
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
		v, pct, ok := parseOne(parts[0])
		if !ok {
			return transformOriginSpec{}, false
		}
		tok := strings.ToLower(parts[0])
		if tok == "top" || tok == "bottom" {
			spec.Y, spec.YPercent = v, pct
			spec.X, spec.XPercent = 50, true
		} else {
			spec.X, spec.XPercent = v, pct
			spec.Y, spec.YPercent = 50, true
		}
	case 2, 3:
		// Two-value: horizontal then vertical (keywords may be either axis).
		a, b := strings.ToLower(parts[0]), strings.ToLower(parts[1])
		axIsY := a == "top" || a == "bottom"
		bxIsY := b == "top" || b == "bottom"
		bxIsX := b == "left" || b == "right"
		if axIsY && (bxIsX || !bxIsY) {
			// vertical horizontal → swap
			va, pa, oka := parseOne(parts[0])
			vb, pb, okb := parseOne(parts[1])
			if !oka || !okb {
				return transformOriginSpec{}, false
			}
			spec.Y, spec.YPercent = va, pa
			spec.X, spec.XPercent = vb, pb
		} else {
			va, pa, oka := parseOne(parts[0])
			vb, pb, okb := parseOne(parts[1])
			if !oka || !okb {
				return transformOriginSpec{}, false
			}
			spec.X, spec.XPercent = va, pa
			spec.Y, spec.YPercent = vb, pb
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
		return clamp01(v / 100), true
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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
		for i := b.opStart; i <= b.opEnd && i < len(ops); i++ {
			if _, covered := childCovered[i]; covered {
				continue
			}
			ops[i].Xform = accum
			ops[i].XformSet = true
		}
	}
	// Opacity: multiply down the tree onto exclusive ops.
	if b.style.Opacity < 1 && b.opEnd >= b.opStart && b.opStart >= 0 {
		for i := b.opStart; i <= b.opEnd && i < len(ops); i++ {
			if _, covered := childCovered[i]; covered {
				continue
			}
			a := b.style.Opacity
			if ops[i].PaintOpacity > 0 && ops[i].PaintOpacity < 1 {
				a *= ops[i].PaintOpacity
			}
			ops[i].PaintOpacity = a
		}
		// Children already stamped their exclusive ops; multiply ancestor opacity
		// onto covered child ops as well (opacity composites through descendants).
		for i := b.opStart; i <= b.opEnd && i < len(ops); i++ {
			if _, covered := childCovered[i]; !covered {
				continue
			}
			a := b.style.Opacity
			if ops[i].PaintOpacity > 0 && ops[i].PaintOpacity < 1 {
				ops[i].PaintOpacity = a * ops[i].PaintOpacity
			} else {
				ops[i].PaintOpacity = a
			}
		}
	}
}

// pdfCTMFromCSS returns the PDF cm matrix equivalent to applying cssXform in
// canvas space (y-down) given canvas→PDF map C(x,y)=(ml+x, K-y).
//
//	CTM = C ∘ cssXform ∘ C⁻¹
func pdfCTMFromCSS(m Matrix2D, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (a, b, c, d, e, f float64) {
	ml := opts.MarginLeft
	K := pageH - opts.MarginTop + float64(pageIdx)*contentH
	a = m.A
	b = -m.B
	c = -m.C
	d = m.D
	e = (1-m.A)*ml + m.C*K + m.E
	f = K*(1-m.D) + m.B*ml - m.F
	return a, b, c, d, e, f
}
