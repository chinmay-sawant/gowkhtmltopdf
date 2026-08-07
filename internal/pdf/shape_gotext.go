package pdf

import (
	"bytes"
	"sync"
	"unicode"

	"github.com/go-text/typesetting/di"
	gtfont "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
)

// gotextFace parses the go-text face once per Font (see Font.gotOnce); the
// package-level sync.Map is gone so derived data lives on and dies with the
// Font it derives from.
//
// ponytail: go-text OT shaping when GSUB available; manual Arabic/RTL fallback otherwise.
var (
	shaperPool = sync.Pool{
		New: func() any { return &shaping.HarfbuzzShaper{} },
	}
	segmenterPool = sync.Pool{
		New: func() any { return &shaping.Segmenter{} },
	}
)

// shapedRun is the internal result of OpenType shaping for reverse-cmap text.
type shapedRun struct {
	text string // reverse-cmap Unicode when fully mapped; else ""
}

// ShapedRun is the canonical text result consumed by layout-independent
// emitters. Text is shaped once, then Runes and Advances describe that exact
// shaped sequence. Keeping the advance calculation next to shaping prevents
// PDF and raster output from independently walking the pre-shaped source
// string and drifting on ligatures or presentation forms.
type ShapedRun struct {
	Text     string
	Runes    []rune
	Advances []float64 // points at the requested font size, one per Rune
}

// ShapeRun shapes s and computes advances for the resulting run. The returned
// slices are owned by the result and may be retained by the caller.
func ShapeRun(s string, f *Font, size float64) ShapedRun {
	text := ShapeTextFont(s, f)
	runes := []rune(text)
	advances := make([]float64, len(runes))

	if f != nil {
		for i, r := range runes {
			advances[i] = f.AdvanceInPoints(r, size)
		}
	}

	return ShapedRun{Text: text, Runes: runes, Advances: advances}
}

// ShapeTextFont shapes s for PDF/image emission using OpenType when f has a
// GSUB table and reverse-cmap can map every shaped glyph to Unicode. Otherwise
// it falls back to presentation-form ShapeText.
//
// ponytail: go-text OT shaping when GSUB available; manual Arabic/RTL fallback otherwise.
//
// For CJK / East-Asian punctuation runs, optional OpenType features halt and
// palt are requested via typesetting FontFeatures when the face provides them.
// CSS font-feature-settings can also be passed via ShapeTextFontWithFeatures.
func ShapeTextFont(s string, f *Font) string {
	return ShapeTextFontWithFeatures(s, f, nil)
}

// ShapeTextFontWithFeatures is ShapeTextFont with explicit OpenType feature
// tags. A nil/empty features slice still enables halt/palt for CJK text.
func ShapeTextFontWithFeatures(s string, f *Font, features []shaping.FontFeature) string {
	if s == "" {
		return s
	}

	feats := mergeFontFeatures(s, features)

	needShape := ShapeNeeded(s) || len(feats) > 0
	if !needShape {
		return s
	}

	if run, ok := tryShapeOpenType(s, f, feats); ok && run.text != "" {
		return run.text
	}

	if ShapeNeeded(s) {
		return ShapeText(s)
	}

	return s
}

func tryShapeOpenType(s string, f *Font, features []shaping.FontFeature) (shapedRun, bool) {
	if f == nil {
		return shapedRun{}, false
	}
	// GSUB covers Arabic ligation; halt/palt live in GPOS and are requested via
	// FontFeatures — allow the OT path when either applies.
	if !f.hasGSUB() && len(features) == 0 {
		return shapedRun{}, false
	}

	face, ok := f.gotextFace()
	if !ok {
		return shapedRun{}, false
	}

	rev := f.reverseCmap()
	text := []rune(s)

	seg := segmenterPool.Get().(*shaping.Segmenter)
	defer segmenterPool.Put(seg)

	shaper := shaperPool.Get().(*shaping.HarfbuzzShaper)
	defer shaperPool.Put(shaper)

	// Size left at zero so we do not import golang.org/x/image (keep
	// typesetting as the only direct third-party require). Glyph IDs are
	// still correct; advances come from our Font hmtx below.
	inputs := seg.Split(shaping.Input{
		Text:         text,
		RunStart:     0,
		RunEnd:       len(text),
		Direction:    di.DirectionLTR,
		Face:         face,
		FontFeatures: features,
	}, singleFaceMap{face})

	outRunes := make([]rune, 0, len(text))

	for _, in := range inputs {
		if in.Face == nil {
			in.Face = face
		}

		in.FontFeatures = features

		out := shaper.Shape(in)
		for _, g := range out.Glyphs {
			if g.GlyphID == gtfont.EmptyGlyph {
				continue
			}

			gid := uint16(g.GlyphID)

			r, ok := rev[gid]
			if !ok {
				return shapedRun{}, false
			}

			outRunes = append(outRunes, r)
		}
	}

	return shapedRun{text: string(outRunes)}, true
}

// ParseFontFeatureSettings parses a CSS font-feature-settings value into
// typesetting FontFeature tags. Only 4-letter OpenType tags are recognized
// (e.g. `"halt" 1, "palt" on`). Unknown junk is skipped.
func ParseFontFeatureSettings(v string) []shaping.FontFeature {
	out := []shaping.FontFeature{}

	for _, part := range splitCSSList(v) {
		part = trimSpace(part)
		if part == "" {
			continue
		}

		tag, val, ok := parseOneFontFeature(part)
		if !ok {
			continue
		}

		out = append(out, shaping.FontFeature{Tag: tag, Value: val})
	}

	return out
}

func mergeFontFeatures(s string, requested []shaping.FontFeature) []shaping.FontFeature {
	if len(requested) > 0 {
		return requested
	}

	return cjkPunctFontFeatures(s)
}

// cjkPunctFontFeatures enables halt/palt for runs that include CJK or
// East-Asian punctuation when CSS did not request features explicitly.
func cjkPunctFontFeatures(s string) []shaping.FontFeature {
	if !textNeedsCJKFeatures(s) {
		return nil
	}

	return []shaping.FontFeature{
		{Tag: ot.MustNewTag("halt"), Value: 1},
		{Tag: ot.MustNewTag("palt"), Value: 1},
	}
}

func textNeedsCJKFeatures(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			return true
		}
		// CJK punctuation / fullwidth forms often targeted by halt/palt.
		if r >= 0x3000 && r <= 0x303F {
			return true
		}

		if r >= 0xFF00 && r <= 0xFFEF {
			return true
		}
	}

	return false
}

func splitCSSList(v string) []string {
	parts := []string{}

	var cur []byte

	inQ := byte(0)

	for i := range len(v) {
		c := v[i]
		if inQ != 0 {
			cur = append(cur, c)

			if c == inQ {
				inQ = 0
			}

			continue
		}

		if c == '"' || c == '\'' {
			inQ = c
			cur = append(cur, c)

			continue
		}

		if c == ',' {
			parts = append(parts, string(cur))
			cur = cur[:0]

			continue
		}

		cur = append(cur, c)
	}

	if len(cur) > 0 {
		parts = append(parts, string(cur))
	}

	return parts
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}

	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}

	return s[i:j]
}

func parseOneFontFeature(part string) (ot.Tag, uint32, bool) {
	part = trimSpace(part)
	if part == "" {
		return 0, 0, false
	}

	var tagStr string

	rest := ""

	if part[0] == '"' || part[0] == '\'' {
		q := part[0]
		end := -1

		for i := 1; i < len(part); i++ {
			if part[i] == q {
				end = i

				break
			}
		}
		// Quoted 4-letter tag: `"halt"` → indices 0 and 5.
		if end != 5 {
			return 0, 0, false
		}

		tagStr = part[1:end]
		rest = trimSpace(part[end+1:])
	} else {
		if len(part) < 4 {
			return 0, 0, false
		}

		tagStr = part[:4]
		rest = trimSpace(part[4:])
	}

	val := uint32(1)

	if rest != "" {
		switch rest {
		case "on", "On", "ON":
			val = 1
		case "off", "Off", "OFF":
			val = 0
		default:
			n := 0

			for _, c := range rest {
				if c < '0' || c > '9' {
					return 0, 0, false
				}

				n = n*10 + int(c-'0')
			}

			val = uint32(n)
		}
	}

	return ot.MustNewTag(tagStr), val, true
}

type singleFaceMap struct{ face *gtfont.Face }

func (m singleFaceMap) ResolveFace(rune) *gtfont.Face { return m.face }

func (f *Font) hasGSUB() bool {
	if f == nil {
		return false
	}

	_, ok := f.tables["GSUB"]

	return ok
}

// gotextFace parses the go-text face once per Font (see Font.gotOnce); the
// package-level sync.Map is gone so derived data lives on and dies with the
// Font it derives from.
func (f *Font) gotextFace() (*gtfont.Face, bool) {
	if f == nil || len(f.data) == 0 {
		return nil, false
	}

	f.gotOnce.Do(func() {
		if face, err := gtfont.ParseTTF(bytes.NewReader(f.data)); err == nil && face != nil {
			f.gotFace = face
		}
	})

	return f.gotFace, f.gotFace != nil
}

// reverseCmap maps glyph id → preferred Unicode (presentation forms win).
// The map is built once from f.cmap (immutable after parse) and cached on
// the Font.
func (f *Font) reverseCmap() map[uint16]rune {
	f.revOnce.Do(func() {
		out := make(map[uint16]rune, len(f.cmap))

		for cp, gid := range f.cmap {
			if gid == 0 {
				continue
			}

			r := rune(cp)
			if prev, ok := out[gid]; ok {
				if cmapRuneScore(r) <= cmapRuneScore(prev) {
					continue
				}
			}

			out[gid] = r
		}

		f.rev = out
	})

	return f.rev
}

func cmapRuneScore(r rune) int {
	switch {
	case r >= 0xFE70 && r <= 0xFEFF:
		return 3 // Arabic presentation forms-B
	case r >= 0xFB50 && r <= 0xFDFF:
		return 2 // Arabic presentation forms-A
	default:
		return 1
	}
}
