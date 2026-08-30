package pdf

import (
	"bytes"
	"strings"
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
	shaperPool = sync.Pool{ //nolint:gochecknoglobals // pooled shapers are immutable after init
		New: func() any { return &shaping.HarfbuzzShaper{} },
	}
	segmenterPool = sync.Pool{ //nolint:gochecknoglobals // pooled segmenters are immutable after init
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
func ShapeRun(s string, fnt *Font, size float64) ShapedRun {
	text := ShapeTextFont(s, fnt)
	runes := []rune(text)
	advances := make([]float64, len(runes))

	if fnt != nil {
		for i, r := range runes {
			advances[i] = fnt.AdvanceInPoints(r, size)
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
func ShapeTextFontWithFeatures(text string, fnt *Font, features []shaping.FontFeature) string {
	if text == "" {
		return text
	}

	feats := mergeFontFeatures(text, features)

	needShape := ShapeNeeded(text) || len(feats) > 0
	if !needShape {
		return text
	}

	if run, ok := tryShapeOpenType(text, fnt, feats); ok && run.text != "" {
		return run.text
	}

	if ShapeNeeded(text) {
		return ShapeText(text)
	}

	return text
}

func tryShapeOpenType(str string, fnt *Font, features []shaping.FontFeature) (shapedRun, bool) {
	if fnt == nil {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}
	// GSUB covers Arabic ligation; halt/palt live in GPOS and are requested via
	// FontFeatures — allow the OT path when either applies.
	if !fnt.hasGSUB() && len(features) == 0 {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	face, faceOK := fnt.gotextFace()
	if !faceOK {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	rev := fnt.reverseCmap()
	text := []rune(str)

	seg, segOK := segmenterPool.Get().(*shaping.Segmenter)
	if !segOK {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	defer segmenterPool.Put(seg)

	shaper, shaperOK := shaperPool.Get().(*shaping.HarfbuzzShaper)
	if !shaperOK {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	defer shaperPool.Put(shaper)

	// Size left at zero so we do not import golang.org/x/image. Keep
	// typesetting within the allowlisted direct modules. Glyph IDs are
	// still correct; advances come from our Font hmtx below.
	inputs := seg.Split(shaping.Input{ //nolint:exhaustruct // intentional zero-value fields
		Text:         text,
		RunStart:     0,
		RunEnd:       len(text),
		Direction:    di.DirectionLTR,
		Face:         face,
		FontFeatures: features,
	}, singleFaceMap{face})

	outRunes, ok := collectShapedRunes(shaper, inputs, face, features, rev)
	if !ok {
		return shapedRun{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return shapedRun{text: string(outRunes)}, true
}

// collectShapedRunes shapes the inputs and maps the resulting glyphs back
// to Unicode via the reverse cmap; false when a glyph cannot be mapped.
func collectShapedRunes(
	shaper *shaping.HarfbuzzShaper,
	inputs []shaping.Input,
	face *gtfont.Face,
	features []shaping.FontFeature,
	rev map[uint16]rune,
) ([]rune, bool) {
	totalGlyphsHint := 0
	for _, inVal := range inputs {
		totalGlyphsHint += len(inVal.Text)
	}

	outRunes := make([]rune, 0, totalGlyphsHint)

	for _, inVal := range inputs {
		if inVal.Face == nil {
			inVal.Face = face
		}

		inVal.FontFeatures = features

		out := shaper.Shape(inVal)
		for _, g := range out.Glyphs {
			if g.GlyphID == gtfont.EmptyGlyph {
				continue
			}

			gid := uint16(g.GlyphID) //nolint:gosec // font cmap is uint16; larger ids cannot reverse-map

			r, ok := rev[gid]
			if !ok {
				return nil, false
			}

			outRunes = append(outRunes, r)
		}
	}

	return outRunes, true
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

// cjkPunctThreshold is the lowest codepoint of the CJK punctuation/script
// ranges halt/palt targets; runes below it are never CJK.
const cjkPunctThreshold = 0x3000

// cjkPunctFeatures is the shared halt/palt feature list returned by
// cjkPunctFontFeatures; consumers only read it.
var cjkPunctFeatures = []shaping.FontFeature{ //nolint:gochecknoglobals // immutable feature list, never mutated
	{Tag: ot.MustNewTag("halt"), Value: 1},
	{Tag: ot.MustNewTag("palt"), Value: 1},
}

// cjkPunctFontFeatures enables halt/palt for runs that include CJK or
// East-Asian punctuation when CSS did not request features explicitly.
func cjkPunctFontFeatures(s string) []shaping.FontFeature {
	if !textNeedsCJKFeatures(s) {
		return nil
	}

	return cjkPunctFeatures
}

func textNeedsCJKFeatures(s string) bool {
	for _, rVal := range s {
		if rVal < cjkPunctThreshold {
			continue // all CJK/EA-punct ranges start at or above U+3000
		}

		if unicode.In(rVal, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			return true
		}
		// CJK punctuation / fullwidth forms often targeted by halt/palt.
		if rVal >= 0x3000 && rVal <= 0x303F {
			return true
		}

		if rVal >= 0xFF00 && rVal <= 0xFFEF {
			return true
		}
	}

	return false
}

func splitCSSList(val string) []string {
	var parts []string

	cur := make([]byte, 0, len(val))

	inQ := byte(0)

	for i := range len(val) {
		cnt := val[i]
		if inQ != 0 {
			cur = append(cur, cnt)

			if cnt == inQ {
				inQ = 0
			}

			continue
		}

		if cnt == '"' || cnt == '\'' {
			inQ = cnt
			cur = append(cur, cnt)

			continue
		}

		if cnt == ',' {
			parts = append(parts, string(cur))
			cur = cur[:0]

			continue
		}

		cur = append(cur, cnt)
	}

	if len(cur) > 0 {
		parts = append(parts, string(cur))
	}

	return parts
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

// parseFeatureTag splits an optional quoted tag from the trailing value
// part, mirroring CSS font-feature-settings syntax.
func parseFeatureTag(part string) (string, string, bool) {
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
		if end != featureEndTagIdx {
			return "", "", false
		}

		return part[1:end], trimSpace(part[end+1:]), true
	}

	if len(part) < featureTagLen {
		return "", "", false
	}

	return part[:4], trimSpace(part[4:]), true
}

// parseFeatureCount parses a decimal feature value, bounded to uint16 per
// the OpenType feature-value width.
func parseFeatureCount(rest string) (uint32, bool) {
	count := 0

	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0, false
		}

		count = count*decimalBase + int(c-'0')
		if count > maxUint16Val {
			return 0, false
		}
	}

	return uint32(count), true //nolint:gosec // count bounded by maxUint16Val above
}

func parseOneFontFeature(part string) (ot.Tag, uint32, bool) {
	part = trimSpace(part)
	if part == "" {
		return 0, 0, false
	}

	tagStr, rest, ok := parseFeatureTag(part)
	if !ok {
		return 0, 0, false
	}

	val := uint32(1)

	if rest != "" {
		switch rest {
		case "on", "On", "ON":
			val = 1
		case "off", "Off", "OFF":
			val = 0
		default:
			count, ok := parseFeatureCount(rest)
			if !ok {
				return 0, 0, false
			}

			val = count
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

			rVal := rune(cp)
			if prev, ok := out[gid]; ok {
				if cmapRuneScore(rVal) <= cmapRuneScore(prev) {
					continue
				}
			}

			out[gid] = rVal
		}

		f.rev = out
	})

	return f.rev
}

func cmapRuneScore(r rune) int {
	switch {
	case r >= 0xFE70 && r <= 0xFEFF:
		return arabicFormsB // Arabic presentation forms-B
	case r >= 0xFB50 && r <= 0xFDFF:
		return arabicFormsA // Arabic presentation forms-A
	default:
		return 1
	}
}
