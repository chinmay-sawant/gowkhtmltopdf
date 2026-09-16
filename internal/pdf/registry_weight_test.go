package pdf

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

// weightedFont parses a real Liberation Sans face with its OS/2 usWeightClass
// rewritten. It stands in for the webfont case where a family ships one file
// per weight and the macStyle bold bit is not set.
func weightedFont(t *testing.T, weight int) *Font {
	t.Helper()

	data := assets.LiberationSansRegular()

	tables, err := parseTableDirectory(data)
	if err != nil {
		t.Fatalf("parseTableDirectory: %v", err)
	}

	os2, ok := tables["OS/2"]
	if !ok || len(os2) < os2WeightClassEnd {
		t.Fatal("Liberation Sans Regular has no OS/2 table")
	}

	//nolint:gosec // weight is 100..1000
	binary.BigEndian.PutUint16(os2[4:os2WeightClassEnd], uint16(weight))

	fnt, err := ParseTTF(data)
	if err != nil {
		t.Fatalf("ParseTTF: %v", err)
	}

	fnt.PostScriptName = fmt.Sprintf("Weighted-%d", weight)

	if got := fnt.WeightClass(); got != weight {
		t.Fatalf("patched face WeightClass = %d, want %d", got, weight)
	}

	return fnt
}

// TestRegistryLookupSelectsFaceByWeight pins CSS weight matching inside one
// family: a 400/500/700 trio must resolve to three distinct faces even though
// the heaviest file is registered first. The old boolean macStyle match kept
// the first face because no file set the bold bit.
func TestRegistryLookupSelectsFaceByWeight(t *testing.T) {
	t.Parallel()

	bold := weightedFont(t, 700)
	medium := weightedFont(t, 500)
	regular := weightedFont(t, 400)

	if regular.macStyle&1 != 0 {
		t.Fatal("test face unexpectedly sets the macStyle bold bit")
	}

	reg := NewRegistry()
	// Registration order mirrors the real page: the 700 file is first.
	for _, fnt := range []*Font{bold, medium, regular} {
		reg.AddFamilyAlias("Weighted", fnt)
	}

	for _, testCase := range []struct {
		want   *Font
		weight int
	}{
		{regular, 400},
		{medium, 500},
		{bold, 700},
	} {
		got := reg.Lookup([]string{"Weighted"}, testCase.weight, false)
		if got == nil {
			t.Fatalf("Lookup(weight %d) = nil, want %q", testCase.weight, testCase.want.PostScriptName)
		}

		if got != testCase.want {
			t.Errorf("Lookup(weight %d) picked %q, want %q", testCase.weight, got.PostScriptName, testCase.want.PostScriptName)
		}
	}
}

// TestWeightClassParsesShortOS2Table covers the bundled DejaVu faces, whose
// OS/2 table is version 1 and only 86 bytes long: usWeightClass (offset 4) is
// present even though the version-2 capHeight field (offset 88) is not. The
// old 90-byte gate read the bold face as 400.
func TestWeightClassParsesShortOS2Table(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if got := faces.UnicodeFallbackBold.WeightClass(); got != fontWeightBoldMin {
		t.Errorf("DejaVu Sans Bold WeightClass = %d, want %d", got, fontWeightBoldMin)
	}

	if !faces.UnicodeFallbackBold.Bold() {
		t.Error("DejaVu Sans Bold must report bold")
	}

	if got := faces.UnicodeFallback.WeightClass(); got != fontWeightDefault {
		t.Errorf("DejaVu Sans WeightClass = %d, want %d", got, fontWeightDefault)
	}
}
