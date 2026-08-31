package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	transformFuncTranslatex = "translatex"
	transformFuncScalex     = "scalex"
	transformFuncSkewx      = "skewx"
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
	rad := deg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)

	return Matrix2D{A: c, B: s, C: -s, D: c} //nolint:exhaustruct // intentional zero fields
}

// SkewXDeg returns a skewX matrix (CSS degrees).
func SkewXDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / 180)

	return Matrix2D{A: 1, D: 1, C: t} //nolint:exhaustruct // intentional zero fields
}

// SkewYDeg returns a skewY matrix (CSS degrees).
func SkewYDeg(deg float64) Matrix2D {
	t := math.Tan(deg * math.Pi / 180)

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
	return transformOriginSpec{X: 50, Y: 50, XPercent: true, YPercent: true}
}

func resolveTransformOrigin(spec transformOriginSpec, boxNode *box) (float64, float64) {
	if boxNode == nil {
		return 0, 0
	}

	var originX, originY float64

	if spec.XPercent {
		originX = boxNode.x + boxNode.w*spec.X/100
	} else {
		originX = boxNode.x + spec.X
	}

	if spec.YPercent {
		originY = boxNode.y + boxNode.height*spec.Y/100
	} else {
		originY = boxNode.y + spec.Y
	}

	return originX, originY
}

// parseTransformList parses a CSS transform function list into a single matrix.
// Returns ok=false for unrecognized input (caller keeps prior/initial).
// "none" yields identity with ok=true and has=false.
func parseTransformList(value string, fontSize float64) (Matrix2D, bool, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return IdentityMatrix(), false, false
	}

	if strings.EqualFold(value, cssDisplayNone) {
		return IdentityMatrix(), false, true
	}

	matrix := IdentityMatrix()
	has := false
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

		fm, fok := parseOneTransformFunc(name, args, fontSize)
		if !fok {
			return IdentityMatrix(), false, false
		}
		// Left-to-right post-multiply: M = M * Fi
		matrix = matrix.Mul(fm)
		has = true
		rest = next
	}

	return matrix, has, true
}

func splitTransformFunc(s string) (string, string, string, bool) {
	text := strings.TrimSpace(s)
	idx := scanTransformFuncName(text)

	if idx == 0 {
		return "", "", "", false
	}

	name := strings.ToLower(text[:idx])

	body := strings.TrimSpace(text[idx:])
	if len(body) == 0 || body[0] != '(' {
		return "", "", "", false
	}

	args, rest, ok := scanTransformParens(body)

	return name, args, rest, ok
}

// scanTransformFuncName returns the length of the leading identifier of a
// transform function name (letters and hyphens).
func scanTransformFuncName(text string) int {
	idx := 0

	for idx < len(text) && ((text[idx] >= 'a' && text[idx] <= 'z') ||
		(text[idx] >= 'A' && text[idx] <= 'Z') || text[idx] == '-') {
		idx++
	}

	return idx
}

// scanTransformParens extracts the content of the top-level (...) group from
// text (which must start with '(') and returns it plus the trimmed remainder.
func scanTransformParens(text string) (string, string, bool) {
	depth := 0

	for jdx := range len(text) {
		switch text[jdx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(text[1:jdx]), strings.TrimSpace(text[jdx+1:]), true
			}
		}
	}

	return "", "", false
}

func parseOneTransformFunc(name, args string, fontSize float64) (Matrix2D, bool) {
	parts := splitTransformArgs(args)

	switch name {
	case "matrix":
		return parseMatrixFunc(parts)
	case "translate", transformFuncTranslatex, "translatey":
		return parseTranslateFunc(name, parts, fontSize)
	case "scale", transformFuncScalex, "scaley":
		return parseScaleFunc(name, parts)
	case "rotate":
		return parseRotateFunc(parts)
	case "skew", transformFuncSkewx, "skewy":
		return parseSkewFunc(name, parts)
	default:
		// 3D / perspective / matrix3d: reject (non-goal)
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}
}

// parseMatrixFunc parses matrix(a,b,c,d,e,f) coefficients.
func parseMatrixFunc(parts []string) (Matrix2D, bool) {
	if len(parts) != 6 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	vals := make([]float64, 6)

	for idx, p := range parts {
		// CSS matrix() takes six <number>s (user units ≈ px at 96dpi).
		val, ok := parseUnitless(p)
		if !ok {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}

		if idx >= 4 {
			// e,f: translate in px → pt for our canvas.
			vals[idx] = pxToPt(val)
		} else {
			vals[idx] = val
		}
	}
	// CSS matrix(a,b,c,d,e,f): x'=ax+cy+e, y'=bx+dy+f
	return Matrix2D{A: vals[0], B: vals[1], C: vals[2], D: vals[3], E: vals[4], F: vals[5]}, true
}

// parseTranslateFunc parses translate(tx[, ty]) and its translatex/translatey
// one-axis variants.
func parseTranslateFunc(name string, parts []string, fontSize float64) (Matrix2D, bool) {
	if name == transformFuncTranslatex || name == "translatey" {
		return parseSingleAxisTranslate(name, parts, fontSize)
	}

	return parseTwoArgTranslate(parts, fontSize)
}

// parseSingleAxisTranslate parses translatex(tx) / translatey(ty).
func parseSingleAxisTranslate(name string, parts []string, fontSize float64) (Matrix2D, bool) {
	if len(parts) != 1 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	axisLen, _, isPct, isOK := parseTransformLength(parts[0], fontSize)
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}
	if isPct {
		axisLen = 0
	}

	if name == transformFuncTranslatex {
		return Translate(axisLen, 0), true
	}

	return Translate(0, axisLen), true
}

// parseTwoArgTranslate parses translate(tx[, ty]).
func parseTwoArgTranslate(parts []string, fontSize float64) (Matrix2D, bool) {
	if len(parts) < 1 || len(parts) > 2 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	xLen, _, xPct, isOK := parseTransformLength(parts[0], fontSize)
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}
	if xPct {
		xLen = 0
	}

	yLen := 0.0
	if len(parts) == 2 {
		yLenTmp, _, yPct, ok2 := parseTransformLength(parts[1], fontSize)
		if !ok2 {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}
		if yPct {
			yLenTmp = 0
		}
		yLen = yLenTmp
	}

	return Translate(xLen, yLen), true
}

// parseScaleFunc parses scale(sx[, sy]) and its scalex/scaley one-axis
// variants.
func parseScaleFunc(name string, parts []string) (Matrix2D, bool) {
	if name == transformFuncScalex || name == "scaley" {
		return parseSingleAxisScale(name, parts)
	}

	return parseTwoArgScale(parts)
}

// parseSingleAxisScale parses scalex(sx) / scaley(sy).
func parseSingleAxisScale(name string, parts []string) (Matrix2D, bool) {
	if len(parts) != 1 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	axisScale, isOK := parseUnitless(parts[0])
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	if name == transformFuncScalex {
		return Scale(axisScale, 1), true
	}

	return Scale(1, axisScale), true
}

// parseTwoArgScale parses scale(sx[, sy]).
func parseTwoArgScale(parts []string) (Matrix2D, bool) {
	if len(parts) < 1 || len(parts) > 2 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	xScale, isOK := parseUnitless(parts[0])
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	yScale := xScale
	if len(parts) == 2 {
		yScale, isOK = parseUnitless(parts[1])
		if !isOK {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}
	}

	return Scale(xScale, yScale), true
}

// parseRotateFunc parses rotate(deg).
func parseRotateFunc(parts []string) (Matrix2D, bool) {
	if len(parts) != 1 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	deg, ok := parseAngleDeg(parts[0])
	if !ok {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	return RotateDeg(deg), true
}

// parseSkewFunc parses skew(ax[, ay]) and its skewx/skewy one-axis variants.
func parseSkewFunc(name string, parts []string) (Matrix2D, bool) {
	if name == transformFuncSkewx || name == "skewy" {
		return parseSingleAxisSkew(name, parts)
	}

	return parseTwoArgSkew(parts)
}

// parseSingleAxisSkew parses skewx(deg) / skewy(deg).
func parseSingleAxisSkew(name string, parts []string) (Matrix2D, bool) {
	if len(parts) != 1 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	deg, isOK := parseAngleDeg(parts[0])
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	if name == transformFuncSkewx {
		return SkewXDeg(deg), true
	}

	return SkewYDeg(deg), true
}

// parseTwoArgSkew parses skew(ax[, ay]).
func parseTwoArgSkew(parts []string) (Matrix2D, bool) {
	if len(parts) < 1 || len(parts) > 2 {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	xDeg, isOK := parseAngleDeg(parts[0])
	if !isOK {
		return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
	}

	yDeg := 0.0
	if len(parts) == 2 {
		yDeg, isOK = parseAngleDeg(parts[1])
		if !isOK {
			return Matrix2D{}, false //nolint:exhaustruct // intentional zero fields
		}
	}

	return SkewXDeg(xDeg).Mul(SkewYDeg(yDeg)), true
}

func splitTransformArgs(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}

	// Most transform functions take at most 4 args (translate/scale/rotate/
	// skew/matrix6). The stack array avoids a heap slice per function; a 5th
	// append grows onto the heap exactly like the old nil-slice append did.
	var buf [4]string
	parts := buf[:0]

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

		return v * 0.9, true
	case strings.HasSuffix(cssSheet, "turn"):
		v, err := strconv.ParseFloat(strings.TrimSpace(cssSheet[:len(cssSheet)-4]), 64)
		if err != nil {
			return 0, false
		}

		return v * 360, true
	default:
		// Unitless angles are invalid in modern CSS; accept bare numbers as deg
		// for authoring convenience in fixtures.
		v, err := strconv.ParseFloat(cssSheet, 64)

		return v, err == nil
	}
}

// parseTransformLength parses a translate length (px/pt/em/%).
// For percentages it returns pct=value and isPct=true so callers can defer
// resolution against the border box at layout time (spec: % of border-box).
// Absolute lengths return pt and isPct=false. em uses font-size.
func parseTransformLength(cssS string, fontSize float64) (float64, float64, bool, bool) {
	cssS = strings.TrimSpace(cssS)
	if cssS == "0" {
		return 0, 0, false, true
	}

	val, unit, ok := css.ParseLength(cssS)
	if !ok {
		// bare number = px per CSS Transforms for matrix e/f; for translate too historically
		if f, err := strconv.ParseFloat(cssS, 64); err == nil {
			return pxToPt(f), 0, false, true
		}

		return 0, 0, false, false
	}

	if unit == "%" {
		return 0, val, true, true
	}

	if unit == "rem" {
		return pxToPt(16) * val, 0, false, true //nolint:mnd // cssPxRoot 16 inlined
	}

	if pt, ok := lengthToPt(val, unit, fontSize); ok {
		return pt, 0, false, true
	}

	return 0, 0, false, false
}

// parseTransformOrigin parses CSS transform-origin (1–3 values; z ignored).
func parseTransformOrigin(value string, fontSize float64) (transformOriginSpec, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
	}

	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 3 {
		return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
	}

	spec := defaultTransformOrigin()

	switch len(parts) {
	case 1:
		val, pct, ok := parseTransformOriginToken(parts[0], fontSize)
		if !ok {
			return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
		}

		tok := strings.ToLower(parts[0])
		if tok == cssVerticalAlignTop || tok == cssVerticalAlignBottom {
			spec.Y, spec.YPercent = val, pct
			spec.X, spec.XPercent = 50, true
		} else {
			spec.X, spec.XPercent = val, pct
			spec.Y, spec.YPercent = 50, true
		}
	case 2, 3:
		// Optional third value (z) ignored for 2D print.
		if !applyTransformOriginPair(&spec, parts[0], parts[1], fontSize) {
			return transformOriginSpec{}, false //nolint:exhaustruct // intentional zero fields
		}
	}

	return spec, true
}

// parseTransformOriginToken parses one transform-origin value into its value
// and whether it is a percentage.
func parseTransformOriginToken(tok string, fontSize float64) (float64, bool, bool) {
	tok = strings.ToLower(strings.TrimSpace(tok))
	switch tok {
	case floatLeft, cssVerticalAlignTop:
		return 0, true, true
	case fxCenter:
		return 50, true, true
	case floatRight, cssVerticalAlignBottom:
		return 100, true, true
	}

	if lv, unit, lok := css.ParseLength(tok); lok && unit == "%" {
		return lv, true, true
	}

	if pt, _, isPct, lok := parseTransformLength(tok, fontSize); lok && !isPct {
		return pt, false, true
	}

	return 0, false, false
}

// applyTransformOriginPair resolves a two-value horizontal/vertical origin
// pair, swapping axes when the values come in vertical-first order.
func applyTransformOriginPair(spec *transformOriginSpec, first, second string, fontSize float64) bool {
	firstTok, secondTok := strings.ToLower(first), strings.ToLower(second)
	firstIsY := firstTok == cssVerticalAlignTop || firstTok == cssVerticalAlignBottom
	secondIsY := secondTok == cssVerticalAlignTop || secondTok == cssVerticalAlignBottom
	secondIsX := secondTok == floatLeft || secondTok == floatRight

	valA, pageA, oka := parseTransformOriginToken(first, fontSize)
	valB, pbox, okb := parseTransformOriginToken(second, fontSize)

	if !oka || !okb {
		return false
	}

	if firstIsY && (secondIsX || !secondIsY) {
		// vertical horizontal → swap
		spec.Y, spec.YPercent = valA, pageA
		spec.X, spec.XPercent = valB, pbox
	} else {
		spec.X, spec.XPercent = valA, pageA
		spec.Y, spec.YPercent = valB, pbox
	}

	return true
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
	if value == "" || value == cssDisplayNone {
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

// ApplyRotate implements the CSS `rotate` longhand (CSS Transforms Level 2).
// It parses a single angle (deg/rad/grad/turn or bare number as deg) and
// composes it into style.Transform / style.HasTransform. On invalid input the
// style is unchanged and false is returned. The composition uses post-multiply
// (style.Transform = style.Transform * RotateDeg) to match parseTransformList
// ordering and to allow successive individual properties to accumulate.
func ApplyRotate(style *ResolvedStyle, value string) bool {
	if style == nil {
		return false
	}
	v := strings.TrimSpace(value)
	if v == "" || strings.EqualFold(v, cssDisplayNone) || strings.EqualFold(v, "initial") {
		return false
	}
	// CSS rotate may be "<angle>" or "z <angle>" or "<axis> <angle>" (3d is ignored).
	// For 2D print we take the last angle token.
	fields := strings.Fields(v)
	if len(fields) == 0 {
		return false
	}
	angleTok := fields[len(fields)-1]
	// Axis tokens (x/y/z) are ignored for 2D; 3d rotate is out of scope.
	if len(fields) > 1 {
		last := strings.ToLower(angleTok)
		if last == "x" || last == "y" || last == "z" {
			// e.g. "rotate: z 12deg" weird order – fall back to first angle
			angleTok = fields[0]
		}
	}
	deg, ok := parseAngleDeg(angleTok)
	if !ok {
		// Try legacy bare numbers that parseAngleDeg already handles; second fallback
		// via parseUnitless (treat as deg) for author convenience.
		if f, err := strconv.ParseFloat(strings.TrimSpace(angleTok), 64); err == nil {
			deg = f
			ok = true
		}
	}
	if !ok {
		return false
	}
	rot := RotateDeg(deg)
	style.Transform = style.Transform.Mul(rot)
	style.HasTransform = true
	return true
}

// ApplyScale implements the CSS `scale` longhand. It parses one or two
// unitless numbers (sx [sy]); a single number yields uniform scale.
// Invalid input leaves style unchanged. Composition is post-multiply like
// parseTransformList.
func ApplyScale(style *ResolvedStyle, value string) bool {
	if style == nil {
		return false
	}
	v := strings.TrimSpace(value)
	if v == "" || strings.EqualFold(v, cssDisplayNone) || strings.EqualFold(v, "initial") {
		return false
	}
	// scale may be comma-separated in some authoring; normalize to spaces.
	v = strings.ReplaceAll(v, ",", " ")
	parts := strings.Fields(v)
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}
	sx, ok := parseUnitless(parts[0])
	if !ok {
		return false
	}
	sy := sx
	if len(parts) >= 2 {
		if s2, ok2 := parseUnitless(parts[1]); ok2 {
			sy = s2
		} else {
			return false
		}
	}
	// Third value (z) ignored for 2D print.
	sc := Scale(sx, sy)
	style.Transform = style.Transform.Mul(sc)
	style.HasTransform = true
	return true
}

// ApplyTranslate implements the CSS `translate` longhand (CSS Transforms Level 2).
// It parses one or two lengths (tx [ty]); percentages are deferred and resolved
// against the border box at layout time, bare numbers are treated as px. The
// absolute part is post-multiplied into style.Transform and the percent part
// is stored in TranslateX/YPercent for resolution in stampBoxTransforms.
func ApplyTranslate(style *ResolvedStyle, value string, fsize float64) bool {
	if style == nil {
		return false
	}
	v := strings.TrimSpace(value)
	if v == "" || strings.EqualFold(v, cssDisplayNone) || strings.EqualFold(v, "initial") {
		return false
	}
	v = strings.ReplaceAll(v, ",", " ")
	parts := strings.Fields(v)
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}
	tx, txPct, txIsPct, ok := parseTransformLength(parts[0], fsize)
	if !ok {
		return false
	}
	ty, tyPct, tyIsPct, tyOk := 0.0, 0.0, false, true
	if len(parts) >= 2 {
		var ok2 bool
		ty, tyPct, tyIsPct, ok2 = parseTransformLength(parts[1], fsize)
		if !ok2 {
			return false
		}
		tyOk = ok2
		_ = tyOk
	}
	// Third value (z) ignored for 2D print.
	tr := Translate(tx, ty)
	style.Transform = style.Transform.Mul(tr)
	if txIsPct {
		style.TranslateXPercent = txPct
	}
	if tyIsPct {
		style.TranslateYPercent = tyPct
	}
	style.HasTransform = true
	return true
}

// stampBoxTransforms walks the laid-out tree and stamps composed 2D
// transform matrices (origin baked) onto display-list ops. Parent transforms
// compose around children; sibling flow geometry is unchanged.
//
// A single []bool covered bitmap replaces per-node map[int]struct{} sets —
// multi-page tables allocate tens of thousands of boxes and the map path was
// a top alloc_space hotspot.
func stampBoxTransforms(boxNode *box, parentAccum Matrix2D, ops []Op) {
	if boxNode == nil || len(ops) == 0 {
		return
	}

	covered := make([]bool, len(ops))
	stampBoxTransformsRec(boxNode, parentAccum, ops, covered)
}

// restampBoxTransforms rebases already-stamped transforms after pagination or
// another flow pass has moved the owning boxes. The transform origin is stored
// in document coordinates, so retaining the old matrix would move transformed
// inline chrome away from its box when that box crosses a page boundary.
func restampBoxTransforms(boxNode *box, ops []Op) {
	needsStamp := false

	for idx := range ops {
		if !ops[idx].XformSet {
			continue
		}

		needsStamp = true
		ops[idx].Xform = IdentityMatrix()
		ops[idx].XformSet = false
	}

	if needsStamp {
		stampBoxTransforms(boxNode, IdentityMatrix(), ops)
	}
}

func stampBoxTransformsRec(boxNode *box, parentAccum Matrix2D, ops []Op, covered []bool) {
	if boxNode == nil {
		return
	}

	accum := parentAccum
	sty := boxNode.style

	if sty != nil && sty.HasTransform {
		tform := sty.Transform
		if sty.TranslateXPercent >= 0 || sty.TranslateYPercent >= 0 {
			tx, ty := 0.0, 0.0
			if sty.TranslateXPercent >= 0 {
				tx = sty.TranslateXPercent / 100 * boxNode.w
			}
			if sty.TranslateYPercent >= 0 {
				ty = sty.TranslateYPercent / 100 * boxNode.height
			}
			tform = tform.Mul(Translate(tx, ty))
		}
		ox, oy := resolveTransformOrigin(sty.TransformOrigin, boxNode)
		baked := BakeOrigin(tform, ox, oy)
		accum = parentAccum.Mul(baked)
	}

	for _, c := range boxNode.children {
		stampBoxTransformsRec(c, accum, ops, covered)
	}

	// Mark child-owned ranges, stamp exclusive ops, then clear for siblings.
	for _, c := range boxNode.children {
		markBoxOpsCovered(c, ops, covered, true)
	}

	stampExclusiveTransformOps(boxNode, accum, ops, covered)
	stampExclusiveOpacityOps(boxNode, ops, covered)
	stampCoveredOpacityOps(boxNode, ops, covered)

	for _, c := range boxNode.children {
		markBoxOpsCovered(c, ops, covered, false)
	}
}

// markBoxOpsCovered records or clears the display-list ops owned by child boxNode.
func markBoxOpsCovered(boxNode *box, ops []Op, covered []bool, record bool) {
	if !boxOwnsOps(boxNode) {
		return
	}

	end := boxNode.opEnd
	if end >= len(ops) {
		end = len(ops) - 1
	}

	for idx := boxNode.opStart; idx <= end; idx++ {
		covered[idx] = record
	}
}

// boxOwnsOps reports whether boxNode has a non-empty range of exclusive ops.
func boxOwnsOps(boxNode *box) bool {
	return boxNode.opEnd >= boxNode.opStart && boxNode.opStart >= 0
}

// stampExclusiveTransformOps applies the accumulated transform to ops owned
// exclusively by boxNode; child-owned ops keep the child's own transform.
func stampExclusiveTransformOps(boxNode *box, accum Matrix2D, ops []Op, covered []bool) {
	if accum.IsIdentity() || !boxOwnsOps(boxNode) {
		return
	}

	end := boxNode.opEnd
	if end >= len(ops) {
		end = len(ops) - 1
	}

	for idx := boxNode.opStart; idx <= end; idx++ {
		if covered[idx] {
			continue
		}

		ops[idx].Xform = accum
		ops[idx].XformSet = true
	}
}

// stampExclusiveOpacityOps multiplies boxNode's opacity onto its exclusive ops.
func stampExclusiveOpacityOps(boxNode *box, ops []Op, covered []bool) {
	if boxNode.style == nil || boxNode.style.Opacity >= 1 || !boxOwnsOps(boxNode) {
		return
	}

	end := boxNode.opEnd
	if end >= len(ops) {
		end = len(ops) - 1
	}

	opacityBase := boxNode.style.Opacity

	for idx := boxNode.opStart; idx <= end; idx++ {
		if covered[idx] {
			continue
		}

		opacity := opacityBase
		if ops[idx].PaintOpacity > 0 && ops[idx].PaintOpacity < 1 {
			opacity *= ops[idx].PaintOpacity
		}

		ops[idx].PaintOpacity = opacity
	}
}

// stampCoveredOpacityOps composes boxNode's ancestor opacity through
// child-owned ops (opacity composites through descendants).
func stampCoveredOpacityOps(boxNode *box, ops []Op, covered []bool) {
	if boxNode.style == nil || boxNode.style.Opacity >= 1 || !boxOwnsOps(boxNode) {
		return
	}

	end := boxNode.opEnd
	if end >= len(ops) {
		end = len(ops) - 1
	}

	opacityBase := boxNode.style.Opacity

	for idx := boxNode.opStart; idx <= end; idx++ {
		if !covered[idx] {
			continue
		}

		if ops[idx].PaintOpacity > 0 && ops[idx].PaintOpacity < 1 {
			ops[idx].PaintOpacity = opacityBase * ops[idx].PaintOpacity
		} else {
			ops[idx].PaintOpacity = opacityBase
		}
	}
}

// pdfCTMFromCSS returns the PDF cm matrix equivalent to applying cssXform in
// canvas space (y-down) given canvas→PDF map C(x,y)=(ml+x, K-y).
//
//	CTM = C ∘ cssXform ∘ C⁻¹
func pdfCTMFromCSS(
	matrix Matrix2D, pageIdx int, contentH float64, opts PaintOptions, pageH float64,
) (float64, float64, float64, float64, float64, float64) {
	margL := opts.MarginLeft
	keyK := pageH - opts.MarginTop + float64(pageIdx)*contentH
	coeffA := matrix.A
	coeffB := -matrix.B
	coeffC := -matrix.C
	coeffD := matrix.D
	offsetE := (1-matrix.A)*margL + matrix.C*keyK + matrix.E
	offsetF := keyK*(1-matrix.D) + matrix.B*margL - matrix.F

	return coeffA, coeffB, coeffC, coeffD, offsetE, offsetF
}
