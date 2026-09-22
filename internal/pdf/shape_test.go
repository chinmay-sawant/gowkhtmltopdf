package pdf

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"unicode"
)

func TestShapeTextRTLReverse(t *testing.T) {
	t.Parallel()

	in := "ab\u05d0\u05d1\u05d2cd"
	got := ShapeText(in)
	want := "ab\u05d2\u05d1\u05d0cd"

	if got != want {
		t.Fatalf("ShapeText = %q, want %q", got, want)
	}

	if ShapeText("hello") != "hello" {
		t.Fatal("LTR unchanged")
	}
}

func TestShapeRunKeepsTextAndAdvancesAligned(t *testing.T) {
	t.Parallel()
	f := testFont(t)

	run := ShapeRun("A\u0301B", f, 12)
	if len(run.Runes) != len(run.Advances) {
		t.Fatalf("runes=%d advances=%d", len(run.Runes), len(run.Advances))
	}

	if run.Text == "" || len(run.Runes) == 0 {
		t.Fatal("empty shaped run")
	}

	for i, advance := range run.Advances {
		if advance <= 0 {
			t.Errorf("advance[%d] = %v, want positive", i, advance)
		}
	}
}

func TestArabicJoiningBehProducesConnectedForms(t *testing.T) {
	t.Parallel()

	got := shapeArabicJoining("ب")
	if got == "ب" {
		t.Fatalf("expected presentation form, got %q U+%04X", got, []rune(got)[0])
	}

	got = shapeArabicJoining("بب")
	runes := []rune(got)

	if len(runes) != 2 {
		t.Fatalf("len=%d %q", len(runes), got)
	}

	if runes[0] != 0xFE91 {
		t.Errorf("first form U+%04X want FE91 (initial)", runes[0])
	}

	if runes[1] != 0xFE90 {
		t.Errorf("second form U+%04X want FE90 (final)", runes[1])
	}
}

func TestArabicLamAlefLigature(t *testing.T) {
	t.Parallel()

	got := shapeArabicJoining("لا")
	runes := []rune(got)

	if len(runes) != 1 {
		t.Fatalf("lam-alef should ligate to 1 rune, got %d %q", len(runes), got)
	}

	if runes[0] < 0xFEF5 || runes[0] > 0xFEFC {
		t.Fatalf("got U+%04X, want Lam-Alef presentation", runes[0])
	}
}

func TestShapeTextArabicPipeline(t *testing.T) {
	t.Parallel()

	str := ShapeText("اب")
	if str == "" {
		t.Fatal("empty")
	}

	for _, r := range str {
		if !(unicode.Is(unicode.Arabic, r) || (r >= 0xFB50 && r <= 0xFEFF)) {
			t.Fatalf("unexpected rune U+%04X in %q", r, str)
		}
	}
}

func TestIndicCombiningNotDroppedMidWord(t *testing.T) {
	t.Parallel()

	s := ShapeText("क्")
	if !strings.Contains(s, "क") {
		t.Fatalf("lost base ka: %q", s)
	}
}

func loadDejaVu(t *testing.T) *Font {
	t.Helper()

	const path = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("DejaVuSans not installed:", err)
	}

	fnt, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	if !fnt.hasGSUB() {
		t.Fatal("DejaVuSans expected to have GSUB")
	}

	return fnt
}

func TestShapeTextFontArabicOTJoining(t *testing.T) {
	t.Parallel()
	f := loadDejaVu(t)
	got := ShapeTextFont("بب", f)
	runes := []rune(got)

	if len(runes) != 2 {
		t.Fatalf("OT بب → %d runes %q (want 2 presentation forms)", len(runes), got)
	}
	// Visual order: final then initial (RTL draw order from typesetting).
	if runes[0] != 0xFE90 {
		t.Errorf("first U+%04X want FE90 (final)", runes[0])
	}

	if runes[1] != 0xFE91 {
		t.Errorf("second U+%04X want FE91 (initial)", runes[1])
	}
}

func TestShapeTextFontArabicOTLamAlef(t *testing.T) {
	t.Parallel()
	f := loadDejaVu(t)
	got := ShapeTextFont("لا", f)
	runes := []rune(got)

	if len(runes) != 1 {
		t.Fatalf("lam-alef OT should be 1 glyph/rune, got %d %q", len(runes), got)
	}

	if runes[0] < 0xFEF5 || runes[0] > 0xFEFC {
		t.Fatalf("got U+%04X, want Lam-Alef presentation", runes[0])
	}
}

func TestShapeTextFontFallsBackWithoutFace(t *testing.T) {
	t.Parallel()

	got := ShapeTextFont("بب", nil)
	want := ShapeText("بب")

	if got != want {
		t.Fatalf("nil face: got %q want %q", got, want)
	}
}

func TestReverseCmapUsesDeterministicCanonicalTie(t *testing.T) {
	t.Parallel()

	fnt := &Font{cmap: map[uint32]uint16{
		uint32('\u00AD'): 1,
		uint32('-'):      1,
		uint32('\u00A0'): 2,
		uint32(' '):      2,
		0xFE91:           3,
		uint32('\u0628'): 3,
	}}

	got := fnt.reverseCmap()

	if got[1] != '-' {
		t.Errorf("glyph 1 = U+%04X, want ASCII hyphen", got[1])
	}

	if got[2] != ' ' {
		t.Errorf("glyph 2 = U+%04X, want ASCII space", got[2])
	}

	if got[3] != 0xFE91 {
		t.Errorf("glyph 3 = U+%04X, want Arabic presentation form", got[3])
	}
}

func TestShapeTextFontLatinUnchanged(t *testing.T) {
	t.Parallel()

	f := loadDejaVu(t)
	if ShapeTextFont("hello", f) != "hello" {
		t.Fatal("Latin must skip OT path")
	}
}

func TestDirectModuleAllowlist(t *testing.T) {
	t.Parallel()
	// Product constraint: only the listed modules may appear as direct
	// third-party requires (transitive graph is allowed).
	//   - go-text/typesetting: OpenType shaping
	//   - tdewolff/canvas: SVG-as-image rasterization (wiki logos, etc.)
	// Also documents CGO HarfBuzz rejection: no harfbuzz CGO module allowed.
	cmd := exec.Command("go", "list", "-m", "-f", "{{if and (not .Main) (not .Indirect)}}{{.Path}}{{end}}", "all")
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}

	allowed := map[string]bool{
		"github.com/go-text/typesetting": true,
		"github.com/tdewolff/canvas":     true,
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !allowed[line] {
			t.Errorf("unexpected direct module %q (allowlist: typesetting + tdewolff/canvas; CGO HarfBuzz rejected)", line)
		}

		if strings.Contains(strings.ToLower(line), "harfbuzz") {
			t.Errorf("CGO HarfBuzz module forbidden: %q", line)
		}
	}
}

func TestCJKPunctFontFeatures(t *testing.T) {
	t.Parallel()

	feats := cjkPunctFontFeatures("。")
	if len(feats) != 2 {
		t.Fatalf("CJK punct features = %d, want halt+palt", len(feats))
	}

	if got := feats[0].Tag.String(); got != "halt" {
		t.Errorf("feat0 = %q, want halt", got)
	}

	if got := feats[1].Tag.String(); got != "palt" {
		t.Errorf("feat1 = %q, want palt", got)
	}

	if cjkPunctFontFeatures("hello") != nil {
		t.Error("Latin must not auto-enable halt/palt")
	}
}

func TestParseFontFeatureSettings(t *testing.T) {
	t.Parallel()

	feats := ParseFontFeatureSettings(`"halt" 1, "palt" on`)
	if len(feats) != 2 {
		t.Fatalf("got %d features, want 2", len(feats))
	}

	if feats[0].Tag.String() != "halt" || feats[0].Value != 1 {
		t.Errorf("halt = %+v", feats[0])
	}

	if feats[1].Tag.String() != "palt" || feats[1].Value != 1 {
		t.Errorf("palt = %+v", feats[1])
	}

	off := ParseFontFeatureSettings(`"halt" off`)
	if len(off) != 1 || off[0].Value != 0 {
		t.Errorf("halt off = %+v", off)
	}
}

func TestParseFontFeatureSettingsHist(t *testing.T) {
	t.Parallel()

	feats := ParseFontFeatureSettings(`"hist" 1`)
	if len(feats) != 1 || feats[0].Tag.String() != "hist" || feats[0].Value != 1 {
		t.Fatalf("hist = %+v, want hist=1", feats)
	}

	swsh := ParseFontFeatureSettings(`"swsh" 1, "salt" 1`)
	if len(swsh) != 2 {
		t.Fatalf("swsh/salt count = %d", len(swsh))
	}

	if swsh[0].Tag.String() != "swsh" || swsh[0].Value != 1 {
		t.Errorf("swsh = %+v", swsh[0])
	}

	if swsh[1].Tag.String() != "salt" || swsh[1].Value != 1 {
		t.Errorf("salt = %+v", swsh[1])
	}
}

func TestParseFontFeatureSettingsSaltChangesDejaVuGlyph(t *testing.T) {
	t.Parallel()

	fnt := bundledDejaVu(t)
	plain := ShapeTextFontWithFeatures("a", fnt, nil)
	salt := ShapeTextFontWithFeatures("a", fnt, ParseFontFeatureSettings(`"salt" 1`))

	if salt == plain {
		t.Fatalf("DejaVu salt should substitute glyph/text, plain=%q salt=%q", plain, salt)
	}

	if salt != "ɑ" {
		t.Fatalf("DejaVu salt a = %q, want ɑ", salt)
	}

	hist := ShapeTextFontWithFeatures("a", fnt, ParseFontFeatureSettings(`"hist" 1`))
	if hist != plain {
		t.Fatalf("DejaVu has no hist; hist a = %q, want %q", hist, plain)
	}
}

func TestShapeTextFontWithFeaturesCJKStillSafe(t *testing.T) {
	t.Parallel()
	// Face may lack halt/palt tables; requesting features must not panic
	// or break the ShapeTextFont path for CJK punctuation.
	font := loadDejaVu(t)

	got := ShapeTextFontWithFeatures("你好。", font, ParseFontFeatureSettings(`"halt" 1, "palt" 1`))
	if got == "" {
		t.Fatal("empty shaped text")
	}
}

func TestShapeTextFontWithFeaturesKernOff(t *testing.T) {
	t.Parallel()

	font := loadDejaVu(t)

	feats := ParseFontFeatureSettings(`"kern" 0`)
	if len(feats) != 1 || feats[0].Value != 0 {
		t.Fatalf("kern off parse = %+v", feats)
	}

	got := ShapeTextFontWithFeatures("AV", font, feats)
	if got == "" {
		t.Fatal("empty shaped text with kern off")
	}

	run := ShapeRunWithFeaturesLanguage("AV", font, 12, feats, "")
	if run.Text == "" || len(run.Runes) == 0 {
		t.Fatalf("ShapeRunWithFeaturesLanguage = %+v", run)
	}
}

func TestContentTextShowLanguageFeaturesParsesSettings(t *testing.T) {
	t.Parallel()

	fnt := loadDejaVu(t)
	content := NewContent()
	content.UseEmbeddedFont("F0", fnt)
	content.BeginText()
	content.SetFont("F0", 12)
	content.TextShowLanguageFeatures("AV", "", `"kern" 0`)
	content.EndText()

	if len(content.Bytes()) == 0 {
		t.Fatal("expected content operators for featured text")
	}
}
