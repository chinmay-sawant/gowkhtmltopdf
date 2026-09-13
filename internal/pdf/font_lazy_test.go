package pdf

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

// bundledFace identifies one embedded default face and its accessor.
type bundledFace struct {
	name string
	data func() []byte
}

// bundledFaces lists every face LoadDefaultFaces installs. Keep in sync with
// faces.go:TestLoadDefaultFaces.
func bundledFaces() []bundledFace {
	return []bundledFace{
		{"LiberationSans-Regular", assets.LiberationSansRegular},
		{"LiberationSans-Bold", assets.LiberationSansBold},
		{"LiberationSans-Italic", assets.LiberationSansItalic},
		{"LiberationSans-BoldItalic", assets.LiberationSansBoldItalic},
		{"LiberationSerif", assets.LiberationSerifRegular},
		{"LiberationSerif-Bold", assets.LiberationSerifBold},
		{"LiberationSerif-Italic", assets.LiberationSerifItalic},
		{"LiberationSerif-BoldItalic", assets.LiberationSerifBoldItalic},
		{"LiberationMono", assets.LiberationMonoRegular},
		{"LiberationMono-Bold", assets.LiberationMonoBold},
		{"LiberationMono-Italic", assets.LiberationMonoItalic},
		{"LiberationMono-BoldItalic", assets.LiberationMonoBoldItalic},
		{"DejaVuSans-UnicodeFallback", assets.UnicodeFallbackRegular},
		{"DejaVuSans-UnicodeFallback-Bold", assets.UnicodeFallbackBold},
	}
}

// fontSnapshot flattens the parsed state into a comparable value.
type fontSnapshot struct {
	unitsPerEm    int16
	indexToLocFmt int16
	numGlyphs     int
	ascender      int16
	descender     int16
	numHMetrics   int
	xMin, yMin    int16
	xMax, yMax    int16
	macStyle      uint16
	italicAngle   int16
	capHeight     int16
	advance       []int32
	lsb           []int16
	cmap          map[uint32]uint16
	tables        map[string][]byte
	data          []byte
	fingerprint   [32]byte
}

func snapshotFont(fnt *Font) fontSnapshot {
	fnt.ensureParsed()

	return fontSnapshot{
		unitsPerEm:    fnt.unitsPerEm,
		indexToLocFmt: fnt.indexToLocFmt,
		numGlyphs:     fnt.numGlyphs,
		ascender:      fnt.ascender,
		descender:     fnt.descender,
		numHMetrics:   fnt.numHMetrics,
		xMin:          fnt.xMin,
		yMin:          fnt.yMin,
		xMax:          fnt.xMax,
		yMax:          fnt.yMax,
		macStyle:      fnt.macStyle,
		italicAngle:   fnt.italicAngle,
		capHeight:     fnt.capHeight,
		advance:       fnt.advance,
		lsb:           fnt.lsb,
		cmap:          fnt.cmap,
		tables:        fnt.tables,
		data:          fnt.data,
		fingerprint:   fnt.fingerprint,
	}
}

// lazyFaceUnloaded reports whether no part of the deferred load has run.
func lazyFaceUnloaded(fnt *Font) bool {
	return fnt.data == nil && fnt.tables == nil && fnt.advance == nil &&
		fnt.cmap == nil && fnt.numGlyphs == 0
}

// TestLazyFaceDefersParse proves the load really is deferred: a freshly built
// lazy face has no bytes and no derived state until an accessor asks for it.
func TestLazyFaceDefersParse(t *testing.T) {
	t.Parallel()

	lazyFace := newLazyFont(assets.LiberationSansRegular)

	if !lazyFaceUnloaded(lazyFace) {
		t.Fatal("lazy face loaded before first use")
	}

	if g := lazyFace.GlyphID('A'); g == 0 {
		t.Fatal("no glyph for A after first accessor call")
	}

	if lazyFaceUnloaded(lazyFace) {
		t.Fatal("accessor did not trigger the deferred load")
	}
}

// assertSubsetMatchesEager runs the lazy copy through subsetting without any
// prior accessor call; the on-demand load must produce identical subset bytes.
func assertSubsetMatchesEager(t *testing.T, eager, lazyFace *Font) {
	t.Helper()

	runes := []rune("Hello ★A0")

	wantSub, err := subsetFont(eager, runes, subsetUnicode)
	if err != nil {
		t.Fatalf("eager subset: %v", err)
	}

	gotSub, err := subsetFont(lazyFace, runes, subsetUnicode)
	if err != nil {
		t.Fatalf("lazy subset: %v", err)
	}

	if !bytes.Equal(wantSub.data, gotSub.data) {
		t.Error("subset bytes differ between eager and lazy parse")
	}
}

// assertStateMatchesEager compares the full observable state of an eager and a
// lazy face after both are loaded.
func assertStateMatchesEager(t *testing.T, eager, lazyFace *Font) {
	t.Helper()

	if lazyFace.parseErr != nil {
		t.Fatalf("lazy parse error: %v", lazyFace.parseErr)
	}

	if !reflect.DeepEqual(snapshotFont(eager), snapshotFont(lazyFace)) {
		t.Error("parsed state differs between eager and lazy parse")
	}

	if !reflect.DeepEqual(eager.LoadNames(), lazyFace.LoadNames()) {
		t.Error("name table differs between eager and lazy parse")
	}

	if eager.PostScriptName != lazyFace.PostScriptName {
		t.Errorf("PostScriptName %q != %q", eager.PostScriptName, lazyFace.PostScriptName)
	}

	for _, runeValue := range "Hello ★A0" {
		if eager.GlyphID(runeValue) != lazyFace.GlyphID(runeValue) {
			t.Errorf("GlyphID(%q) %d != %d", runeValue, eager.GlyphID(runeValue), lazyFace.GlyphID(runeValue))
		}

		if eager.AdvanceInPoints(runeValue, 12) != lazyFace.AdvanceInPoints(runeValue, 12) {
			t.Errorf("AdvanceInPoints(%q) differs", runeValue)
		}
	}
}

// TestLazyFaceMatchesEager pins the PERF2-08 contract: deferring the load may
// move the work, not change it. Every bundled face is parsed once eagerly and
// once lazily, and both subset bytes and parsed state must match.
func TestLazyFaceMatchesEager(t *testing.T) {
	t.Parallel()

	for _, face := range bundledFaces() {
		t.Run(face.name, func(t *testing.T) {
			t.Parallel()

			eager, err := ParseTTF(face.data())
			if err != nil {
				t.Fatalf("eager parse: %v", err)
			}

			lazyFace := newLazyFont(face.data)

			assertSubsetMatchesEager(t, eager, lazyFace)
			assertStateMatchesEager(t, eager, lazyFace)
		})
	}
}
