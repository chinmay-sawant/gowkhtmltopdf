package pdf

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// reBaseFontRef matches a /BaseFont name in a PDF object body.
var reBaseFontRef = regexp.MustCompile(`/BaseFont\s*/([^\s/\[\]<>()]+)`)

// reFontDescriptorRef matches a /FontDescriptor indirect reference.
var reFontDescriptorRef = regexp.MustCompile(`/FontDescriptor\s+(\d+)\s+0\s+R`)

// reFontNameRef matches a FontDescriptor /FontName.
var reFontNameRef = regexp.MustCompile(`/FontName\s*/([^\s/\[\]<>()]+)`)

// reDescendantFontsRef matches a Type0 /DescendantFonts array reference.
var reDescendantFontsRef = regexp.MustCompile(`/DescendantFonts\s*\[\s*(\d+)\s+0\s+R\s*\]`)

// rePDFObject splits raw PDF bytes into "N 0 obj ... endobj" bodies.
var rePDFObject = regexp.MustCompile(`(?s)(\d+) 0 obj\s*(.*?)\s*endobj`)

// subsetNameRE is the name shape validators expect for an embedded subset:
// six uppercase letters, a plus sign, then the untagged base name.
var subsetNameRE = regexp.MustCompile(`^[A-Z]{6}\+[^+\s]+$`)

// parsePDFObjects maps object number -> raw body for direct byte-level checks.
func parsePDFObjects(pdf []byte) map[int]string {
	objs := map[int]string{}

	for _, match := range rePDFObject.FindAllSubmatch(pdf, -1) {
		n, err := strconv.Atoi(string(match[1]))
		if err != nil {
			continue
		}

		objs[n] = string(match[2])
	}

	return objs
}

// baseFontNames returns every /BaseFont name in document order.
func baseFontNames(pdf []byte) []string {
	matches := reBaseFontRef.FindAllSubmatch(pdf, -1)
	names := make([]string, 0, len(matches))

	for _, m := range matches {
		names = append(names, string(m[1]))
	}

	return names
}

// assertTaggedBaseFontName fails the test unless the PDF names the face whose
// untagged name is plain with a six-letter subset tag. The trailing boundary
// keeps "LiberationSans" from matching its Bold sibling.
func assertTaggedBaseFontName(t *testing.T, pdf []byte, plain string) {
	t.Helper()

	re := regexp.MustCompile(`/BaseFont /([A-Z]{6}\+` + regexp.QuoteMeta(plain) + `)(?:\s|/|>>)`)

	if re.FindSubmatch(pdf) == nil {
		t.Fatalf("no /BaseFont name with a six-letter subset tag for %q", plain)
	}
}

// TestSubsetTagShapeDeterministic checks the tag itself: six uppercase
// letters and the same tag whenever the same subset program is rebuilt.
func TestSubsetTagShapeDeterministic(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	first, err := subsetFont(fnt, []rune("Hello"), subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	second, err := subsetFont(fnt, []rune("Hello"), subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	tag := subsetTag(first)
	if len(tag) != subsetTagLen {
		t.Fatalf("tag %q has length %d, want %d", tag, len(tag), subsetTagLen)
	}

	for _, r := range tag {
		if r < 'A' || r > 'Z' {
			t.Fatalf("tag %q contains non-uppercase-letter rune %q", tag, r)
		}
	}

	if got := subsetTag(second); got != tag {
		t.Fatalf("tags unstable for identical subsets: %q != %q", tag, got)
	}
}

// TestSubsetTagStableAcrossRuns converts the same document twice and asserts
// every BaseFont name is byte-identical, well formed, and tagged.
func TestSubsetTagStableAcrossRuns(t *testing.T) {
	t.Parallel()

	build := func() []byte {
		fnt, err := DefaultFont()
		if err != nil {
			t.Fatal(err)
		}

		doc := fixedDoc(t)
		doc.SetCompression(false)

		p := doc.AddPage(100, 100)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginText()
		cur.SetFont("F1", 12)
		cur.TextAt(5, 5)
		cur.TextShow("Hello")
		cur.EndText()

		return writePDF(t, doc)
	}

	first := baseFontNames(build())
	second := baseFontNames(build())

	if len(first) == 0 {
		t.Fatal("no BaseFont names in output")
	}

	if !slices.Equal(first, second) {
		t.Fatalf("BaseFont names not stable across runs: %v != %v", first, second)
	}

	for _, name := range first {
		if !subsetNameRE.MatchString(name) {
			t.Errorf("BaseFont %q is not a tagged subset name", name)
		}
	}
}

// TestSimpleFontDescriptorNameMatchesBaseFont proves the simple-font
// cross-references agree: FontDescriptor /FontName equals /BaseFont, and both
// carry the subset tag.
func TestSimpleFontDescriptorNameMatchesBaseFont(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc := fixedDoc(t)
	doc.SetCompression(false)

	p := doc.AddPage(200, 100)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fnt)
	cur.BeginText()
	cur.SetFont("F1", 12)
	cur.TextAt(10, 50)
	cur.TextShow("Regular")
	cur.EndText()

	objs := parsePDFObjects(writePDF(t, doc))
	checked := 0

	for _, body := range objs {
		if !strings.Contains(body, "/Subtype /TrueType") {
			continue
		}

		base := reBaseFontRef.FindStringSubmatch(body)

		desc := reFontDescriptorRef.FindStringSubmatch(body)
		if base == nil || desc == nil {
			t.Fatalf("TrueType font missing BaseFont or FontDescriptor: %s", body[:min(120, len(body))])
		}

		if !subsetNameRE.MatchString(base[1]) {
			t.Errorf("BaseFont %q is not a tagged subset name", base[1])
		}

		fname := reFontNameRef.FindStringSubmatch(objs[mustAtoi(desc[1])])
		if fname == nil {
			t.Fatalf("FontDescriptor %s missing FontName", desc[1])
		}

		if fname[1] != base[1] {
			t.Errorf("FontDescriptor /FontName %q != /BaseFont %q", fname[1], base[1])
		}

		checked++
	}

	if checked == 0 {
		t.Fatal("no TrueType font objects found")
	}
}
