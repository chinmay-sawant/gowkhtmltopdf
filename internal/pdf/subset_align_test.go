package pdf

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestSubsetGlyfFourByteAligned(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf")
	if err != nil {
		t.Skip(err)
	}

	f, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	sub, err := subsetFont(f, []rune("汉字与假名：東京都、上海、深圳。"), subsetUnicode)
	if err != nil {
		t.Fatal(err)
	}

	sfnt, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatal(err)
	}

	loca := sfnt.tables["loca"]
	maxp := sfnt.tables["maxp"]

	n := int(binary.BigEndian.Uint16(maxp[4:6]))
	for i := 0; i <= n; i++ {
		off := binary.BigEndian.Uint32(loca[i*4:])
		if off%4 != 0 {
			t.Fatalf("loca[%d]=%d not 4-byte aligned", i, off)
		}
	}

	for _, run := range "東京都、" {
		if len(sfnt.GlyphContours(run)) == 0 {
			t.Fatalf("missing contours for %c", run)
		}
	}
	// Hint bytecode must be stripped (no fpgm/prep/cvt in subset).
	for _, run := range "東告" {
		raw := sfnt.glyphOutline(sfnt.GlyphID(run))
		if len(raw) < 12 {
			t.Fatalf("short outline for %c", run)
		}

		nc := int16(binary.BigEndian.Uint16(raw[0:2]))
		if nc < 0 {
			continue
		}

		ins := binary.BigEndian.Uint16(raw[10+2*int(nc):])
		if ins != 0 {
			t.Fatalf("%c still has %d hint bytes", run, ins)
		}
	}
}
