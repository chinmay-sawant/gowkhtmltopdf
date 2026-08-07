package pdf

import (
	"strings"
	"testing"
)

func TestFontFamilyKeysGenericsOnly(t *testing.T) {
	if got := fontFamilyKeys("Georgia"); len(got) != 1 || got[0] != "georgia" {
		t.Fatalf("Georgia keys=%v want exact [georgia]", got)
	}

	if got := fontFamilyKeys("serif"); len(got) == 0 || got[0] != "liberation serif" {
		t.Fatalf("serif keys=%v want liberation serif first", got)
	}

	if got := fontFamilyKeys("sans-serif"); len(got) == 0 || got[0] != "liberation sans" {
		t.Fatalf("sans-serif keys=%v", got)
	}
}

func TestLookupExactBeforeGeneric(t *testing.T) {
	reg := ScanFontDirs(DefaultSystemFontDirs())
	if len(reg.byFamily["liberation serif"]) == 0 {
		t.Skip("Liberation Serif not installed")
	}

	if f := reg.Lookup([]string{"Georgia"}, 400, false); f != nil {
		for _, n := range f.FamilyNames() {
			if strings.Contains(strings.ToLower(n), "georgia") {
				t.Skip("system has Georgia; cannot assert miss")
			}
		}
	}

	f := reg.Lookup([]string{"Georgia", "Liberation Serif", "serif"}, 400, false)
	if f == nil {
		t.Fatal("expected Liberation Serif from author stack")
	}

	ok := false

	for _, n := range f.FamilyNames() {
		low := strings.ToLower(n)
		if strings.Contains(low, "liberation") && strings.Contains(low, "serif") {
			ok = true
		}
	}

	if !ok {
		t.Fatalf("got face %v, want Liberation Serif", f.FamilyNames())
	}
}
