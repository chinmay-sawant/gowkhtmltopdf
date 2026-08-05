package pdf

import (
	"bytes"
	"sync"

	"github.com/go-text/typesetting/di"
	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
)

// gotextFaceCache maps *Font → *gtfont.Face (or false-sentinel on failure).
var gotextFaceCache sync.Map

type gotextFaceEntry struct {
	face *gtfont.Face
	ok   bool
}

var (
	shaperPool = sync.Pool{
		New: func() any { return &shaping.HarfbuzzShaper{} },
	}
	segmenterPool = sync.Pool{
		New: func() any { return &shaping.Segmenter{} },
	}
)

// ShapedGlyph is one output glyph from ShapeRun (OpenType or fallback).
type ShapedGlyph struct {
	GID     uint16
	Rune    rune    // Unicode CID for PDF Identity-H when reverse-cmap succeeds
	Advance float64 // font units
	Cluster int
}

// ShapedRun is the result of ShapeRun.
type ShapedRun struct {
	Glyphs []ShapedGlyph
	Text   string // reverse-cmap Unicode string when fully mapped; else ""
	OT     bool   // true when OpenType shaping produced Glyphs
}

// ShapeTextFont shapes s for PDF/image emission using OpenType when f has a
// GSUB table and reverse-cmap can map every shaped glyph to Unicode. Otherwise
// it falls back to presentation-form ShapeText.
func ShapeTextFont(s string, f *Font) string {
	if s == "" {
		return s
	}
	if !ShapeNeeded(s) {
		return s
	}
	if run, ok := tryShapeOpenType(s, f); ok && run.Text != "" {
		return run.Text
	}
	return ShapeText(s)
}

// ShapeRun returns glyph-level shaping. When OpenType succeeds, OT is true and
// Text holds reverse-cmap Unicode (visual order) suitable for Type0 Identity-H.
// On failure Glyphs is empty and callers should use ShapeText.
func ShapeRun(s string, f *Font) ShapedRun {
	if s == "" || !ShapeNeeded(s) {
		return ShapedRun{}
	}
	if run, ok := tryShapeOpenType(s, f); ok {
		return run
	}
	return ShapedRun{}
}

func tryShapeOpenType(s string, f *Font) (ShapedRun, bool) {
	if f == nil || !f.hasGSUB() {
		return ShapedRun{}, false
	}
	face, ok := gotextFace(f)
	if !ok {
		return ShapedRun{}, false
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
		Text:      text,
		RunStart:  0,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      face,
	}, singleFaceMap{face})

	outGlyphs := make([]ShapedGlyph, 0, len(text))
	outRunes := make([]rune, 0, len(text))
	for _, in := range inputs {
		if in.Face == nil {
			in.Face = face
		}
		out := shaper.Shape(in)
		for _, g := range out.Glyphs {
			if g.GlyphID == gtfont.EmptyGlyph {
				continue
			}
			gid := uint16(g.GlyphID)
			r, ok := rev[gid]
			if !ok {
				return ShapedRun{}, false
			}
			adv := 0.0
			if int(gid) < len(f.advance) {
				adv = float64(f.advance[gid])
			}
			outGlyphs = append(outGlyphs, ShapedGlyph{
				GID:     gid,
				Rune:    r,
				Advance: adv,
				Cluster: g.ClusterIndex,
			})
			outRunes = append(outRunes, r)
		}
	}
	return ShapedRun{
		Glyphs: outGlyphs,
		Text:   string(outRunes),
		OT:     true,
	}, true
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

func gotextFace(f *Font) (*gtfont.Face, bool) {
	if f == nil || len(f.data) == 0 {
		return nil, false
	}
	if v, ok := gotextFaceCache.Load(f); ok {
		e := v.(gotextFaceEntry)
		return e.face, e.ok
	}
	face, err := gtfont.ParseTTF(bytes.NewReader(f.data))
	e := gotextFaceEntry{face: face, ok: err == nil && face != nil}
	if actual, loaded := gotextFaceCache.LoadOrStore(f, e); loaded {
		e = actual.(gotextFaceEntry)
	}
	return e.face, e.ok
}

// reverseCmap maps glyph id → preferred Unicode (presentation forms win).
func (f *Font) reverseCmap() map[uint16]rune {
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
	return out
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
