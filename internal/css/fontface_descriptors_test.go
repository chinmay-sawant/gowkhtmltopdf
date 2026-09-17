package css

import "testing"

// TestParseFontFaceDescriptors pins the three @font-face descriptors the
// registry needs for face selection: font-weight, font-style, and
// unicode-range. Without them all faces of a family look identical to
// selection, which is the learncpp-1 failure.
func TestParseFontFaceDescriptors(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `@font-face {
		font-family: 'Open Sans';
		font-style: normal;
		font-weight: 700;
		src: url(open-sans-normal-latin-ext-700.woff2);
		unicode-range: U+0100-02AF, U+0304, U+4??, U+0000-00FF;
	}`)

	if len(str.FontFaces) != 1 {
		t.Fatalf("font-faces = %d, want 1", len(str.FontFaces))
	}

	face := str.FontFaces[0]
	if face.Weight != 700 {
		t.Errorf("Weight = %d, want 700", face.Weight)
	}

	if !face.StyleSet || face.Italic {
		t.Errorf("StyleSet = %v, Italic = %v, want explicit normal", face.StyleSet, face.Italic)
	}

	want := []UnicodeRange{
		{Lo: 0x0100, Hi: 0x02AF},
		{Lo: 0x0304, Hi: 0x0304},
		{Lo: 0x0400, Hi: 0x04FF},
		{Lo: 0x0000, Hi: 0x00FF},
	}
	if len(face.UnicodeRanges) != len(want) {
		t.Fatalf("UnicodeRanges = %+v, want %+v", face.UnicodeRanges, want)
	}

	for i, span := range want {
		if face.UnicodeRanges[i] != span {
			t.Errorf("UnicodeRanges[%d] = %+v, want %+v", i, face.UnicodeRanges[i], span)
		}
	}
}

// TestParseFontFaceDescriptorsAbsent keeps the single-face family contract:
// with no descriptors the face carries zero values and no range restriction.
func TestParseFontFaceDescriptorsAbsent(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `@font-face { font-family: X; src: url(x.woff) }`)

	if len(str.FontFaces) != 1 {
		t.Fatalf("font-faces = %d, want 1", len(str.FontFaces))
	}

	face := str.FontFaces[0]
	if face.Weight != 0 || face.Italic || face.StyleSet || len(face.UnicodeRanges) != 0 {
		t.Errorf("descriptors = %+v, want zero values", face)
	}
}

func TestParseUnicodeRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want []UnicodeRange
	}{
		{"single", "U+26", []UnicodeRange{{Lo: 0x26, Hi: 0x26}}},
		{"span", "U+0100-02AF", []UnicodeRange{{Lo: 0x0100, Hi: 0x02AF}}},
		{"wildcard", "u+4??", []UnicodeRange{{Lo: 0x0400, Hi: 0x04FF}}},
		{"max", "U+0-10FFFF", []UnicodeRange{{Lo: 0, Hi: 0x10FFFF}}},
		{"mixed garbage", "bogus, U+0041, U+zz, U+0042", []UnicodeRange{{Lo: 0x41, Hi: 0x41}, {Lo: 0x42, Hi: 0x42}}},
		{"reversed", "U+0300-0100", nil},
		{"too high", "U+110000", nil},
		{"too many digits", "U+1234567", nil},
		{"wildcard then digit", "U+4?1", nil},
		{"empty", "", nil},
		{"missing prefix", "0100-02AF", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ParseUnicodeRanges(test.spec)
			if len(got) != len(test.want) {
				t.Fatalf("ParseUnicodeRanges(%q) = %+v, want %+v", test.spec, got, test.want)
			}

			for i, span := range test.want {
				if got[i] != span {
					t.Errorf("ParseUnicodeRanges(%q)[%d] = %+v, want %+v", test.spec, i, got[i], span)
				}
			}
		})
	}
}

func TestUnicodeRangeCovers(t *testing.T) {
	t.Parallel()

	span := UnicodeRange{Lo: 0x0100, Hi: 0x02AF}
	if !span.Covers(0x0100) || !span.Covers(0x02AF) || !span.Covers(0x01A2) {
		t.Error("Covers must include both endpoints and the interior")
	}

	if span.Covers(0x00FF) || span.Covers(0x02B0) {
		t.Error("Covers must exclude codepoints outside the span")
	}
}

func TestFontFaceWeightDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val    string
		want   int
		wantOK bool
	}{
		{"normal", 400, true},
		{"bold", 700, true},
		{"500", 500, true},
		{"100", 100, true},
		{"1000", 1000, true},
		{"bolder", 0, false},
		{"lighter", 0, false},
		{"0", 0, false},
		{"1001", 0, false},
		{"", 0, false},
	}

	for _, test := range tests {
		got, ok := fontWeightDescriptor(test.val)
		if got != test.want || ok != test.wantOK {
			t.Errorf("fontWeightDescriptor(%q) = (%d, %v), want (%d, %v)", test.val, got, ok, test.want, test.wantOK)
		}
	}
}

func TestFontFaceStyleDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val        string
		wantItalic bool
		wantOK     bool
	}{
		{"normal", false, true},
		{"italic", true, true},
		{"oblique", true, true},
		{"oblique 10deg", true, true},
		{"BOLD", false, false},
		{"", false, false},
	}

	for _, test := range tests {
		gotItalic, ok := italicDescriptor(test.val)
		if gotItalic != test.wantItalic || ok != test.wantOK {
			t.Errorf("italicDescriptor(%q) = (%v, %v), want (%v, %v)", test.val, gotItalic, ok, test.wantItalic, test.wantOK)
		}
	}
}
