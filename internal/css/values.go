package css

import (
	"math"
	"strconv"
	"strings"
)

// ParseInline parses a style="" attribute value into declarations.
func ParseInline(style string) []Declaration {
	return parseDeclarations(style)
}

// parseDeclarations splits a declaration block on top-level ';' and parses
// each "prop: value" pair. Garbage pairs are skipped.
func parseDeclarations(block string) []Declaration {
	if strings.TrimSpace(block) == "" {
		return nil
	}

	parts := splitTopLevel(block, ';')
	decls := make([]Declaration, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		colon := strings.IndexByte(part, ':')
		if colon < 0 {
			continue
		}

		prop := strings.ToLower(strings.TrimSpace(part[:colon]))
		value := strings.TrimSpace(part[colon+1:])

		if prop == "" || value == "" {
			continue
		}

		important := isImportant(value)
		if important {
			value = stripImportant(value)
		}

		if !validPropName(prop) {
			continue
		}

		decls = append(decls, Declaration{Prop: prop, Value: value, Important: important})
	}

	return decls
}

func validPropName(page string) bool {
	if page == "" {
		return false
	}

	for i := range len(page) {
		c := page[i]
		if !(c == '-' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}

	return true
}

// isImportant reports whether a declaration value carries !important
// (whitespace between ! and important is allowed).
func isImportant(val string) bool {
	val = strings.TrimSpace(val)

	const word = "important"

	if len(val) < len(word)+1 {
		return false
	}

	if !strings.EqualFold(val[len(val)-len(word):], word) {
		return false
	}

	rest := strings.TrimRight(val[:len(val)-len(word)], " \t")

	return strings.HasSuffix(rest, "!")
}

// stripImportant removes a trailing !important (any case, optional space)
// from a declaration value.
func stripImportant(val string) string {
	if !isImportant(val) {
		return val
	}

	val = strings.TrimRight(val, " \t")
	val = val[:len(val)-len("important")]
	val = strings.TrimRight(val, " \t")
	val = strings.TrimSuffix(val, "!")

	return strings.TrimSpace(val)
}

// ParseLength parses a CSS length: number + unit, where bare numbers are
// pixels. Units: px, pt, pc, in, cm, mm, em, rem, ex, ch, %, vw, vh.
func ParseLength(val string) (float64, string, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, "", false
	}

	idx := scanLengthNumber(val)
	if idx == 0 {
		return 0, "", false
	}

	num, err := strconv.ParseFloat(val[:idx], 64)
	if err != nil {
		return 0, "", false
	}

	unit := strings.ToLower(strings.TrimSpace(val[idx:]))
	if unit == "" {
		unit = "px"
	}

	if !isLengthUnit(unit) {
		return 0, "", false
	}

	return num, unit, true
}

// scanLengthNumber returns the index of the end of the numeric prefix of val
// (optional sign, digits and one '.', per CSS lengths).
func scanLengthNumber(val string) int {
	idx := 0
	if val[0] == '+' || val[0] == '-' {
		idx++
	}

	for idx < len(val) && (val[idx] >= '0' && val[idx] <= '9' || val[idx] == '.') {
		idx++
	}

	return idx
}

// isLengthUnit reports whether unit is one of the CSS units ParseLength
// accepts.
func isLengthUnit(unit string) bool {
	switch unit {
	case "px", "pt", "pc", "in", "cm", "mm", "em", "rem", "ex", "ch", "%", "vw", "vh":
		return true
	}

	return false
}

// ParseNumber parses a bare number, e.g. line-height or font-weight.
func ParseNumber(val string) (float64, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, false
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

// ParseColor parses #rgb, #rrggbb, #rrggbbaa, rgb()/rgba() with integer,
// float or percentage channels, hsl()/hsla(), and a named-color subset. It
// returns RGB in 0..255 and alpha in 0..1; ok=false for anything unrecognized.
//
//nolint:cyclop // linear dispatch across color forms; extraction would obscure it
func ParseColor(val string) (int, int, int, float64, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, 0, 0, 0, false
	}
	// CSS variables: var(--name, fallback) — resolve fallback only (no custom props).
	// ponytail: ParseColor accepts bare var() without a prop map (API is color-
	// string only). Layout resolves custom props via ResolveCustomProps before
	// color parse; upgrade if ParseColor gains a props argument.
	if strings.HasPrefix(strings.ToLower(val), "var(") {
		if fb, okFB := cssVarFallback(val); okFB {
			return ParseColor(fb)
		}

		return 0, 0, 0, 0, false
	}

	if val[0] == '#' {
		return parseHexColor(val[1:])
	}

	low := val

	for i := range len(val) {
		if val[i] >= 'A' && val[i] <= 'Z' {
			low = strings.ToLower(val)

			break
		}
	}

	if low == "transparent" {
		return 0, 0, 0, 0, true
	}

	if name, found := namedColors()[low]; found {
		return name[0], name[1], name[2], 1, true
	}

	if strings.HasPrefix(low, "rgb") {
		return parseRGBColor(val, low)
	}

	if strings.HasPrefix(low, "hsl") {
		return parseHSLColor(val, low)
	}

	return 0, 0, 0, 0, false
}

// parseHexColor parses #rgb, #rgba, #rrggbb and #rrggbbaa forms (hex is the
// content after '#').
func parseHexColor(hex string) (int, int, int, float64, bool) {
	switch len(hex) {
	case hexRGBLen:
		if !isHex(hex) {
			return 0, 0, 0, 0, false
		}

		return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), 1, true
	case hexRGBALen:
		if !isHex(hex) {
			return 0, 0, 0, 0, false
		}

		return hexNibble(hex[0]), hexNibble(hex[1]), hexNibble(hex[2]), float64(hexNibble(hex[3])) / maxRGBChannel, true
	case hexRRGGBBLen:
		if !isHex(hex) {
			return 0, 0, 0, 0, false
		}

		return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), 1, true
	case hexRRGGBBAALen:
		if !isHex(hex) {
			return 0, 0, 0, 0, false
		}

		return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), float64(hexByte(hex[6:8])) / maxRGBChannel, true
	}

	return 0, 0, 0, 0, false
}

// parseRGBColor parses rgb()/rgba() with integer, float or percentage
// channels (low is the lower-cased original). The channel list is scanned in
// place between '(' and ')': no split slice, no per-channel allocation except
// the trimmed channel strings themselves.
//
//nolint:cyclop // single-pass in-place channel scan; splitting loses the no-alloc property
func parseRGBColor(val, low string) (int, int, int, float64, bool) {
	open := strings.IndexByte(val, '(')
	closeIdx := strings.LastIndexByte(val, ')')

	if open < 0 || closeIdx < open {
		return 0, 0, 0, 0, false
	}

	channels := hexRGBLen
	if strings.HasPrefix(low, "rgba") {
		channels = rgbaChannelCount
	}

	var vals [rgbaChannelCount]float64

	alphaRaw := ""

	idx := open + 1
	for channel := range channels {
		start := idx

		for idx < closeIdx && val[idx] != ',' {
			idx++
		}

		if channel == rgbaChannelCount-1 {
			alphaRaw = val[start:idx]
		}

		chVal, ok := parseColorChannel(val[start:idx])
		if !ok {
			return 0, 0, 0, 0, false
		}

		vals[channel] = chVal
		idx++
	}

	// the scan must have consumed exactly the channel list: anything between
	// the last comma and ')' (including a trailing comma) is a mismatch
	if idx != closeIdx+1 {
		return 0, 0, 0, 0, false
	}

	red := clampByte(vals[0])
	green := clampByte(vals[1])
	blue := clampByte(vals[2])

	if channels != rgbaChannelCount {
		return red, green, blue, 1, true
	}

	alpha := vals[3]
	if strings.HasSuffix(strings.TrimSpace(alphaRaw), "%") {
		alpha /= maxRGBChannel
	}

	return red, green, blue, clampAlpha(alpha), true
}

// parseHSLColor parses hsl()/hsla() with hue in degrees (optional deg suffix),
// saturation and lightness as percentages, and optional alpha like rgba().
func parseHSLColor(val, low string) (int, int, int, float64, bool) {
	open := strings.IndexByte(val, '(')
	closeIdx := strings.LastIndexByte(val, ')')

	if open < 0 || closeIdx < open {
		return 0, 0, 0, 0, false
	}

	channels := hslChannelCount
	if strings.HasPrefix(low, "hsla") {
		channels = rgbaChannelCount
	}

	hue, sat, light, alpha, alphaRaw, parsed := parseHSLChannels(val, open+1, closeIdx, channels)
	if !parsed {
		return 0, 0, 0, 0, false
	}

	red, green, blue := hslToRGB(hue, sat, light)

	if channels != rgbaChannelCount {
		return red, green, blue, 1, true
	}

	if strings.HasSuffix(strings.TrimSpace(alphaRaw), "%") {
		alpha /= maxRGBChannel
	}

	return red, green, blue, clampAlpha(alpha), true
}

func parseHSLChannels(val string, idx, closeIdx, channels int) (float64, float64, float64, float64, string, bool) {
	var chans [rgbaChannelCount]float64

	alphaRaw := ""

	for channel := range channels {
		start := idx

		for idx < closeIdx && val[idx] != ',' {
			idx++
		}

		raw := val[start:idx]
		if channel == rgbaChannelCount-1 {
			alphaRaw = raw
		}

		chVal, parsed := parseOneHSLChannel(channel, raw)
		if !parsed {
			return 0, 0, 0, 0, "", false
		}

		chans[channel] = chVal
		idx++
	}

	if idx != closeIdx+1 {
		return 0, 0, 0, 0, "", false
	}

	return chans[0], chans[1], chans[hslChannelCount-1], chans[rgbaChannelCount-1], alphaRaw, true
}

func parseOneHSLChannel(channel int, raw string) (float64, bool) {
	switch channel {
	case 0:
		return parseHueChannel(raw)
	case 1, hslChannelCount - 1:
		return parsePercentUnit(raw)
	default:
		return parseColorChannel(raw)
	}
}

func parseHueChannel(chVal string) (float64, bool) {
	chVal = strings.TrimSpace(chVal)
	if strings.HasSuffix(strings.ToLower(chVal), "deg") {
		chVal = strings.TrimSpace(chVal[:len(chVal)-3])
	}

	if chVal == "" || strings.Contains(chVal, "%") {
		return 0, false
	}

	f, err := strconv.ParseFloat(chVal, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

func parsePercentUnit(chVal string) (float64, bool) {
	chVal = strings.TrimSpace(chVal)
	if !strings.HasSuffix(chVal, "%") {
		return 0, false
	}

	f, err := strconv.ParseFloat(strings.TrimSuffix(chVal, "%"), 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

func hslToRGB(hue, sat, light float64) (int, int, int) {
	hue = math.Mod(hue, hslCircleDeg)
	if hue < 0 {
		hue += hslCircleDeg
	}

	sat = clampUnit(sat / percentScale)
	light = clampUnit(light / percentScale)

	chroma := (1 - math.Abs(2*light-1)) * sat
	hSector := hue / hslSectorDeg
	xVal := chroma * (1 - math.Abs(math.Mod(hSector, hslEvenPeriod)-1))
	mVal := light - chroma/hslChromaHalf

	red, green, blue := hslSectorRGB(hSector, chroma, xVal)

	return clampByte((red + mVal) * maxRGBChannel),
		clampByte((green + mVal) * maxRGBChannel),
		clampByte((blue + mVal) * maxRGBChannel)
}

func hslSectorRGB(hSector, chroma, xVal float64) (float64, float64, float64) {
	switch {
	case hSector < 1:
		return chroma, xVal, 0
	case hSector < hslSectorYG:
		return xVal, chroma, 0
	case hSector < hslSectorGC:
		return 0, chroma, xVal
	case hSector < hslSectorCB:
		return 0, xVal, chroma
	case hSector < hslSectorBM:
		return xVal, 0, chroma
	default:
		return chroma, 0, xVal
	}
}

func clampUnit(fVal float64) float64 {
	if fVal < 0 {
		return 0
	}

	if fVal > 1 {
		return 1
	}

	return fVal
}

// parseColorChannel parses one rgb()/rgba() channel: a number with an
// optional trailing '%' scaling 0..100 onto 0..255.
func parseColorChannel(chVal string) (float64, bool) {
	chVal = strings.TrimSpace(chVal)
	if strings.HasSuffix(chVal, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(chVal, "%"), 64)
		if err != nil {
			return 0, false
		}

		return f * maxRGBChannel / percentScale, true
	}

	f, err := strconv.ParseFloat(chVal, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

// clampAlpha clamps an alpha value to 0..1.
func clampAlpha(alpha float64) float64 {
	if alpha > 1 {
		return 1
	}

	if alpha < 0 {
		return 0
	}

	return alpha
}

// cssVarFallback extracts the fallback from var(--name, fallback). Nested
// var() in the fallback is not expanded further here.
func cssVarFallback(v string) (string, bool) {
	_, fb, ok := parseVarFn(v)

	return fb, ok && fb != ""
}

// parseVarFn parses a top-level var(--name) or var(--name, fallback).
// ok is false when v is not a var() function.
func parseVarFn(val string) (string, string, bool) {
	val = strings.TrimSpace(val)
	if len(val) < 6 || !strings.EqualFold(val[:4], "var(") {
		return "", "", false
	}

	inner := val[4:]
	if !strings.HasSuffix(inner, ")") {
		return "", "", false
	}

	inner = strings.TrimSpace(inner[:len(inner)-1])
	depth := 0

	for idx := range len(inner) {
		switch inner[idx] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				name := strings.ToLower(strings.TrimSpace(inner[:idx]))
				fallback := strings.TrimSpace(inner[idx+1:])

				return name, fallback, name != ""
			}
		}
	}

	name := strings.ToLower(strings.TrimSpace(inner))

	return name, "", name != ""
}

// ResolveVar expands CSS var() references in v using lookup(--name).
// Unresolved var() uses the CSS fallback when present; otherwise the empty
// string (caller treats as invalid / keeps the prior cascaded value).
// Nested var() expands up to a small depth.
func ResolveVar(val2 string, lookup func(name string) (string, bool)) string {
	val2 = strings.TrimSpace(val2)
	for range 16 {
		if !strings.HasPrefix(strings.ToLower(val2), "var(") {
			return val2
		}

		name, fallback, ok := parseVarFn(val2)
		if !ok {
			return val2
		}

		if lookup != nil {
			if val, found := lookup(name); found && strings.TrimSpace(val) != "" {
				val2 = strings.TrimSpace(val)

				continue
			}
		}

		if fallback != "" {
			val2 = fallback

			continue
		}

		return ""
	}

	return val2
}

// ResolveVars expands every var() function embedded in a CSS value. ResolveVar
// handles a value whose complete input is one var() function; declarations such
// as "1px solid var(--line)" need the same substitution while preserving their
// surrounding tokens.
func ResolveVars(value string, lookup func(name string) (string, bool)) string {
	if !strings.Contains(strings.ToLower(value), "var(") {
		return value
	}

	var out strings.Builder

	start := 0
	for start < len(value) {
		idx := indexVarFunction(value, start)
		if idx < 0 {
			out.WriteString(value[start:])

			break
		}

		out.WriteString(value[start:idx])

		end := matchingVarParen(value, idx+varFunctionPrefixLen)
		if end < 0 {
			out.WriteString(value[idx:])

			break
		}

		out.WriteString(ResolveVar(value[idx:end+1], lookup))
		start = end + 1
	}

	return out.String()
}

const varFunctionPrefixLen = len("var")

func indexVarFunction(value string, start int) int {
	for idx := start; idx+4 <= len(value); idx++ {
		if strings.EqualFold(value[idx:idx+4], "var(") {
			return idx
		}
	}

	return -1
}

func matchingVarParen(value string, open int) int {
	depth := 0

	for idx := open; idx < len(value); idx++ {
		switch value[idx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx
			}
		}
	}

	return -1
}

// ResolveCustomProps walks a custom-property graph: the inherited overlay
// plus declared --* values, with var() chains expanded once using a memo and
// a cycle stack (cycles resolve to invalid → empty). The single place
// custom-property policy lives.
func ResolveCustomProps(declared, inherited map[string]string) map[string]string {
	work := make(map[string]string, len(inherited)+len(declared))
	for k, v := range inherited {
		work[k] = v
	}

	for k, v := range declared {
		work[k] = v
	}

	memo := map[string]string{}

	var eval func(name string, stack map[string]bool) string

	eval = func(name string, stack map[string]bool) string {
		if v, ok := memo[name]; ok {
			return v
		}

		raw, ok := work[name]
		if !ok || !strings.Contains(strings.ToLower(raw), "var(") {
			memo[name] = raw

			return raw
		}

		if stack[name] {
			return ""
		}

		stack[name] = true
		val := ResolveVar(raw, func(n string) (string, bool) {
			s := eval(n, stack)
			_, exists := work[n]

			return s, exists && strings.TrimSpace(s) != ""
		})

		delete(stack, name)

		memo[name] = val

		return val
	}
	for name := range work {
		eval(name, map[string]bool{})
	}

	return memo
}

func isHex(s string) bool {
	for i := range len(s) {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}

	return true
}

func hexNibble(buf byte) int {
	switch {
	case buf >= '0' && buf <= '9':
		n := int(buf - '0')

		return n*16 + n
	case buf >= 'a' && buf <= 'f':
		n := int(buf-'a') + hexLetterBase

		return n*16 + n
	case buf >= 'A' && buf <= 'F':
		n := int(buf-'A') + hexLetterBase

		return n*16 + n
	}

	return 0
}

func hexByte(s string) int {
	hi := hexVal(s[0])
	lo := hexVal(s[1])

	return hi*16 + lo
}

func hexVal(buf byte) int {
	switch {
	case buf >= '0' && buf <= '9':
		return int(buf - '0')
	case buf >= 'a' && buf <= 'f':
		return int(buf-'a') + hexLetterBase
	case buf >= 'A' && buf <= 'F':
		return int(buf-'A') + hexLetterBase
	}

	return 0
}

func clampByte(fVal float64) int {
	if fVal < 0 {
		return 0
	}

	if fVal > maxRGBChannel {
		return maxRGBChannel
	}

	return int(fVal + roundHalfUp)
}

// ParseFontFamily splits a font-family value on commas and trims quotes.
func ParseFontFamily(value string) []string {
	var out []string

	for {
		comma := strings.IndexByte(value, ',')

		part := value
		if comma >= 0 {
			part = value[:comma]
		}

		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"'")

		if part != "" {
			out = append(out, part)
		}

		if comma < 0 {
			break
		}

		value = value[comma+1:]
	}

	return out
}

// namedColorTable is the CSS2 system colors plus greys/orange and common web
// names used by fixtures and layout tests (ponytail: not the full CSS Color 4
// list). Cached at package level so color parsing does not allocate a new map
// on every call; callers must treat the map as read-only.
var namedColorTable = map[string][3]int{ //nolint:gochecknoglobals // read-only cache; avoids per-call allocation
	// CSS2 core
	"black": {0, 0, 0}, "silver": {192, 192, 192}, "gray": {128, 128, 128},
	"grey": {128, 128, 128}, "white": {255, 255, 255}, "maroon": {128, 0, 0},
	"red": {255, 0, 0}, "purple": {128, 0, 128}, "fuchsia": {255, 0, 255},
	"green": {0, 128, 0}, "lime": {0, 255, 0}, "olive": {128, 128, 0},
	"yellow": {255, 255, 0}, "navy": {0, 0, 128}, "blue": {0, 0, 255},
	"teal": {0, 128, 128}, "aqua": {0, 255, 255},
	// Common aliases / CSS3 extras used in sheets
	"cyan": {0, 255, 255}, "magenta": {255, 0, 255}, "orange": {255, 165, 0},
	"brown": {165, 42, 42}, "pink": {255, 192, 203}, "gold": {255, 215, 0},
	"darkgray": {169, 169, 169}, "darkgrey": {169, 169, 169},
	"lightgray": {211, 211, 211}, "lightgrey": {211, 211, 211},
	"darkgreen": {0, 100, 0}, "darkblue": {0, 0, 139}, "darkred": {139, 0, 0},
	"darkorange": {255, 140, 0}, "lightblue": {173, 216, 230},
	"lightgreen": {144, 238, 144}, "lightyellow": {255, 255, 224},
	"coral": {255, 127, 80}, "crimson": {220, 20, 60}, "indigo": {75, 0, 130},
	"khaki": {240, 230, 140}, "lavender": {230, 230, 250}, "violet": {238, 130, 238},
	"tan": {210, 180, 140}, "salmon": {250, 128, 114}, "seagreen": {46, 139, 87},
	"steelblue": {70, 130, 180}, "turquoise": {64, 224, 208}, "wheat": {245, 222, 179},
	"orangered": {255, 69, 0}, "tomato": {255, 99, 71}, "whitesmoke": {245, 245, 245},
	"gainsboro": {220, 220, 220}, "rebeccapurple": {102, 51, 153},
}

// namedColors returns the shared named-color table (read-only).
func namedColors() map[string][3]int {
	return namedColorTable
}
