//nolint:testpackage,cyclop,wsl // discovery fixtures patch SFNT bytes
package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFontsLoadsDirAndFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join("assets", "LiberationSans-Regular.ttf")

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read asset: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "Face.ttf"), data, 0o600); err != nil {
		t.Fatalf("write Face.ttf: %v", err)
	}

	filePath := filepath.Join(dir, "Direct.ttf")
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatalf("write Direct.ttf: %v", err)
	}

	junk := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(junk, []byte("not a font"), 0o600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	report := DiscoverFonts([]string{dir, filePath, junk, filepath.Join(dir, "missing-dir")})
	if report.Loaded < 2 {
		t.Fatalf("loaded=%d skips=%v", report.Loaded, report.Skips)
	}

	if report.Skipped < 2 {
		t.Fatalf("expected skips for junk file + missing path; got %+v", report.Skips)
	}

	var log bytes.Buffer

	report.Log(&log)
	got := log.String()
	if !strings.Contains(got, "font discovery:") || !strings.Contains(got, "loaded") {
		t.Fatalf("discovery log = %q", got)
	}

	reg := report.Registry
	if reg == nil || reg.Lookup([]string{"Liberation Sans"}, 400, false) == nil {
		t.Fatal("expected Liberation Sans in registry")
	}
}

func TestDiscoverFontsSkipsOTTOAndVariable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	otto := make([]byte, 12)
	copy(otto[0:4], []byte("OTTO"))
	if err := os.WriteFile(filepath.Join(dir, "cff.otf"), otto, 0o600); err != nil {
		t.Fatal(err)
	}

	base, err := os.ReadFile(filepath.Join("assets", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	variable := injectTable(t, base, "fvar", make([]byte, 16))
	if err := os.WriteFile(filepath.Join(dir, "var.ttf"), variable, 0o600); err != nil {
		t.Fatal(err)
	}

	report := DiscoverFonts([]string{dir})
	if report.Loaded != 0 {
		t.Fatalf("loaded=%d, want 0", report.Loaded)
	}

	reasons := strings.Join(func() []string {
		out := make([]string, len(report.Skips))
		for i, s := range report.Skips {
			out[i] = s.Reason
		}

		return out
	}(), "; ")
	if !strings.Contains(reasons, "CFF/OTTO") && !strings.Contains(reasons, "OTTO") {
		t.Fatalf("expected OTTO skip, got %q", reasons)
	}

	if !strings.Contains(reasons, "variable") && !strings.Contains(reasons, "fvar") {
		t.Fatalf("expected variable skip, got %q", reasons)
	}
}

func TestParseTTFRejectsVariableFvar(t *testing.T) {
	t.Parallel()

	base, err := os.ReadFile(filepath.Join("assets", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseTTF(injectTable(t, base, "fvar", make([]byte, 8)))
	if !errors.Is(err, errFontVariableRejected) {
		t.Fatalf("ParseTTF variable: %v", err)
	}
}

func TestLookupStyleOverrideAndNearest(t *testing.T) {
	t.Parallel()

	regular := testFontWithStyle(t, 0)
	bold := testFontWithStyle(t, 1)
	italic := testFontWithStyle(t, 2)
	boldItalic := testFontWithStyle(t, 3)

	reg := NewRegistry()
	for _, fnt := range []*Font{regular, bold, italic, boldItalic} {
		reg.AddFamilyAlias("Styled", fnt)
	}

	cases := []struct {
		weight int
		italic bool
		want   *Font
	}{
		{400, false, regular},
		{700, false, bold},
		{400, true, italic},
		{700, true, boldItalic},
	}
	for _, tc := range cases {
		got := reg.Lookup([]string{"Styled"}, tc.weight, tc.italic)
		if got != tc.want {
			t.Fatalf("Lookup(%d,%v)=%p want %p", tc.weight, tc.italic, got, tc.want)
		}
	}

	// Missing bold-italic falls back to nearest score (bold or italic).
	thin := NewRegistry()
	thin.AddFamilyAlias("Thin", regular)
	thin.AddFamilyAlias("Thin", bold)
	got := thin.Lookup([]string{"Thin"}, 700, true)
	if got != bold {
		t.Fatal("expected nearest bold when bold-italic missing")
	}
}

func TestGeorgiaDirectorySelectsGeorgia(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writePatchedLiberationSerif(t, filepath.Join(dir, "Georgia.ttf"), "Georgia")

	reg := ScanFontDirs([]string{dir})
	fnt := reg.Lookup([]string{"Georgia"}, 400, false)
	if fnt == nil {
		t.Fatal("expected Georgia from font-path directory")
	}

	found := false
	for _, n := range fnt.FamilyNames() {
		if strings.EqualFold(strings.TrimSpace(n), "Georgia") {
			found = true
		}
	}
	if !found {
		t.Fatalf("family names=%v", fnt.FamilyNames())
	}
}

func TestGelasioDoesNotRenameToGeorgia(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writePatchedLiberationSerif(t, filepath.Join(dir, "Gelasio.ttf"), "Gelasio")

	reg := ScanFontDirs([]string{dir})
	if fnt := reg.Lookup([]string{"Georgia"}, 400, false); fnt != nil {
		t.Fatalf("Gelasio must not satisfy Georgia; got %v", fnt.FamilyNames())
	}

	if fnt := reg.Lookup([]string{"Gelasio"}, 400, false); fnt == nil {
		t.Fatal("expected Gelasio exact match")
	}
}

func TestFamilyNameDiffersFromFileName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// File name is unrelated; family comes from the name table.
	writePatchedLiberationSerif(t, filepath.Join(dir, "random-file.ttf"), "Georgia")

	reg := ScanFontDirs([]string{dir})
	if reg.Lookup([]string{"random-file"}, 400, false) != nil {
		t.Fatal("file stem must not become the CSS family")
	}

	if reg.Lookup([]string{"Georgia"}, 400, false) == nil {
		t.Fatal("expected name-table family Georgia")
	}
}

func TestNotoKRSubsetReparseAfterSubset(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "fonts", "NotoSansKR-HangulSubset.ttf")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	fnt, err := ParseTTF(data)
	if err != nil {
		t.Fatalf("ParseTTF Noto KR: %v", err)
	}

	sub, err := subsetFont(fnt, []rune("안녕 Hangul"), subsetUnicode)
	if err != nil {
		t.Fatalf("subset: %v", err)
	}

	again, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("reparse subset: %v", err)
	}

	if again.GlyphID('안') == 0 {
		t.Fatal("expected Hangul glyph after subset reparse")
	}
}

func TestSameDisplayNameDifferentBytesIsolated(t *testing.T) {
	t.Parallel()

	leftData, err := os.ReadFile(filepath.Join("assets", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	rightData, err := os.ReadFile(filepath.Join("assets", "LiberationSerif-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	left, err := ParseTTF(leftData)
	if err != nil {
		t.Fatal(err)
	}

	right, err := ParseTTF(rightData)
	if err != nil {
		t.Fatal(err)
	}

	left.PostScriptName = "SharedName"
	right.PostScriptName = "SharedName"

	if bytes.Equal(left.fingerprint[:], right.fingerprint[:]) {
		t.Fatal("expected distinct fingerprints")
	}

	reg := NewRegistry()
	reg.AddFamilyAlias("Shared", left)
	reg.AddFamilyAlias("Shared", right)

	got := reg.Lookup([]string{"Shared"}, 400, false)
	if got == nil {
		t.Fatal("lookup miss")
	}
	// First registered wins on equal style score.
	if got != left {
		t.Fatal("expected first-registered face on style tie")
	}
}

func writePatchedLiberationSerif(t *testing.T, dst, family string) {
	t.Helper()

	src, err := os.ReadFile(filepath.Join("assets", "LiberationSerif-Regular.ttf"))
	if err != nil {
		t.Fatalf("read Liberation Serif: %v", err)
	}

	patched := patchFamilyInPlace(t, src, "Liberation Serif", family)
	if err := os.WriteFile(dst, patched, 0o600); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// patchFamilyInPlace replaces oldFamily with newFamily space-padded to the
// same length in ASCII and UTF-16BE name records.
func patchFamilyInPlace(t *testing.T, data []byte, oldFamily, newFamily string) []byte {
	t.Helper()

	if len(newFamily) > len(oldFamily) {
		t.Fatalf("new family %q longer than %q", newFamily, oldFamily)
	}

	out := append([]byte(nil), data...)
	padded := newFamily + strings.Repeat(" ", len(oldFamily)-len(newFamily))

	oldASCII := []byte(oldFamily)
	newASCII := []byte(padded)
	offset := 0
	for {
		idx := bytes.Index(out[offset:], oldASCII)
		if idx < 0 {
			break
		}

		idx += offset
		copy(out[idx:idx+len(oldASCII)], newASCII)
		offset = idx + len(oldASCII)
	}

	oldUTF := utf16BE(oldFamily)
	newUTF := utf16BE(padded)
	offset = 0
	for {
		idx := bytes.Index(out[offset:], oldUTF)
		if idx < 0 {
			break
		}

		idx += offset
		copy(out[idx:idx+len(oldUTF)], newUTF)
		offset = idx + len(oldUTF)
	}

	return out
}

func utf16BE(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r>>8), byte(r))
	}

	return out
}

// injectTable appends a synthetic table so ParseTTF can see an extra tag
// (used to inject fvar for variable-font rejection tests).
func injectTable(t *testing.T, sfnt []byte, tag string, payload []byte) []byte {
	t.Helper()

	if len(tag) != 4 {
		t.Fatalf("tag %q", tag)
	}

	if len(sfnt) < 12 {
		t.Fatal("sfnt too short")
	}

	numTables := int(binary.BigEndian.Uint16(sfnt[4:6]))
	dirEnd := 12 + 16*numTables

	if dirEnd > len(sfnt) {
		t.Fatal("truncated directory")
	}

	const inserted = 16

	out := make([]byte, 0, len(sfnt)+inserted+len(payload))
	out = append(out, sfnt[:12]...)
	binary.BigEndian.PutUint16(out[4:6], uint16(numTables+1)) //nolint:gosec // test table count

	for i := range numTables {
		rec := append([]byte(nil), sfnt[12+16*i:12+16*(i+1)]...)
		off := binary.BigEndian.Uint32(rec[8:12])
		binary.BigEndian.PutUint32(rec[8:12], off+inserted)
		out = append(out, rec...)
	}

	rec := make([]byte, 16)
	copy(rec[0:4], tag)
	binary.BigEndian.PutUint32(rec[8:12], uint32(len(sfnt)+inserted)) //nolint:gosec // test fixture
	binary.BigEndian.PutUint32(rec[12:16], uint32(len(payload)))      //nolint:gosec // test fixture length
	out = append(out, rec...)
	out = append(out, sfnt[dirEnd:]...)
	out = append(out, payload...)

	return out
}
