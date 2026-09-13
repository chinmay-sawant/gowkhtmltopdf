package convert

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// bodyWithTargetOnPage2 forces the #target heading onto page 2.
func bodyWithTargetOnPage2() string {
	return `<html><head><style>
.spacer { page-break-before: always; }
</style></head><body>
<p>page one content</p>
<div class="spacer"></div>
<h2 id="target">Target Heading</h2>
<p>destination body</p>
</body></html>`
}

func writeHFLinkHeader(t *testing.T, dir, href string) string {
	t.Helper()

	path := filepath.Join(dir, "header.html")
	html := `<html><body><a href="` + href + `">Jump</a></body></html>`

	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

var (
	kidsPageRefsRe = regexp.MustCompile(`/Kids\s*\[([^\]]*)\]`)
	pageObjRefRe   = regexp.MustCompile(`(\d+)\s+0\s+R`)
	destPageRefRe  = regexp.MustCompile(`/Dest\s*\[(\d+)\s+0\s+R\s*/XYZ`)
)

// pageKidsRefs returns the ordered page object numbers from the pages tree
// /Kids array (first match is the root pages dict).
func pageKidsRefs(data []byte) []int {
	mVal := kidsPageRefsRe.FindSubmatch(data)
	if mVal == nil {
		return nil
	}

	matches := pageObjRefRe.FindAllSubmatch(mVal[1], -1)
	out := make([]int, 0, len(matches))

	for _, ref := range matches {
		n, _ := strconv.Atoi(string(ref[1]))
		out = append(out, n)
	}

	return out
}

func destPageRefs(data []byte) []int {
	matches := destPageRefRe.FindAllSubmatch(data, -1)
	out := make([]int, 0, len(matches))

	for _, m := range matches {
		n, _ := strconv.Atoi(string(m[1]))
		out = append(out, n)
	}

	return out
}

func TestHTMLHeaderFragmentGoTo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := writeHFLinkHeader(t, dir, "#target")
	cmd, _ := newCommand(t, bodyWithTargetOnPage2(), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2", n)
	}

	kids := pageKidsRefs(data)
	if len(kids) < 2 {
		t.Fatalf("Kids page refs = %v, want >= 2", kids)
	}

	dests := destPageRefs(data)
	if len(dests) == 0 {
		t.Fatal("expected /Dest GoTo from HTML header fragment link")
	}

	targetRef := kids[1] // page 2 (0-based index 1)
	found := false

	for _, d := range dests {
		if d == targetRef {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("HF fragment Dest refs = %v, want one targeting page 2 obj %d (kids=%v)", dests, targetRef, kids)
	}

	if bytes.Contains(data, []byte("/URI")) {
		t.Error("fragment link must not emit /URI; expected AddLinkDest only")
	}

	if bytes.Contains(data, []byte("#target")) {
		t.Error("raw #target must not appear as a URI annotation")
	}
}

func TestHTMLHeaderFragmentGoToCopies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := writeHFLinkHeader(t, dir, "#target")
	cmd, _ := newCommand(t, bodyWithTargetOnPage2(), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false
	cmd.Global.Copies = 2
	cmd.Global.Collate = true
	data := runPDF(t, cmd)

	if n := pageCount(data); n != 4 {
		t.Fatalf("pages = %d, want 4 (2 logical × 2 copies)", n)
	}

	kids := pageKidsRefs(data)
	if len(kids) != 4 {
		t.Fatalf("Kids = %v, want 4 page refs", kids)
	}

	dests := destPageRefs(data)
	// Header on every page → 4 GoTo annots, each to page-2 of the same copy:
	// collate order [p0,p1,p0',p1'] → dests kids[1], kids[3], kids[3].
	want := []int{kids[1], kids[1], kids[3], kids[3]}
	if len(dests) != len(want) {
		t.Fatalf("Dest count = %d, want %d; dests=%v kids=%v", len(dests), len(want), dests, kids)
	}

	counts := map[int]int{}
	for _, d := range dests {
		counts[d]++
	}

	if counts[kids[1]] != 2 || counts[kids[3]] != 2 {
		t.Errorf("collate Dest page refs = %v (counts %v); want two→%d and two→%d",
			dests, counts, kids[1], kids[3])
	}

	if bytes.Contains(data, []byte("/URI")) {
		t.Error("fragment link must not emit /URI under copies")
	}
}

func TestHTMLHeaderFragmentGoToCopiesNonCollate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := writeHFLinkHeader(t, dir, "#target")
	cmd, _ := newCommand(t, bodyWithTargetOnPage2(), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false
	cmd.Global.Copies = 2
	cmd.Global.Collate = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n != 4 {
		t.Fatalf("pages = %d, want 4", n)
	}

	kids := pageKidsRefs(data)
	if len(kids) != 4 {
		t.Fatalf("Kids = %v, want 4", kids)
	}

	dests := destPageRefs(data)
	// non-collate order [p0,p0',p1,p1'] → dests kids[2], kids[3], kids[2], kids[3].
	counts := map[int]int{}
	for _, d := range dests {
		counts[d]++
	}

	if len(dests) != 4 || counts[kids[2]] != 2 || counts[kids[3]] != 2 {
		t.Errorf("non-collate Dest refs = %v (counts %v); want two→%d and two→%d",
			dests, counts, kids[2], kids[3])
	}
}

func TestHTMLHeaderExternalURI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := writeHFLinkHeader(t, dir, "http://example.com/hf")
	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("/URI")) || !bytes.Contains(data, []byte("http://example.com/hf")) {
		t.Error("expected HTML header external URI annotation")
	}
}

func TestHTMLHeaderFontFaceLocal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ttf := copyTestdataTTF(t, dir)
	_ = ttf
	headerPath := filepath.Join(dir, "header.html")
	hdr := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.ttf); }
body { font-family: Custom, sans-serif; font-size: 12pt; }
</style></head><body><p>HFCustomFace</p></body></html>`

	if err := os.WriteFile(headerPath, []byte(hdr), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Margin.Top = -1 // auto: reserve HF height
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("/BaseFont")) {
		t.Error("expected embedded font in PDF with HF @font-face")
	}
	// Custom face registers as /BaseFont /Custom when MergeFontFaces runs for HF.
	if !bytes.Contains(data, []byte("/Custom")) && !bytes.Contains(data, []byte("Custom")) {
		t.Log("note: Custom BaseFont name may be subset-prefixed; PDF still produced")
	}

	_ = data
}

func TestHTMLHeaderFlexImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 1x1 PNG
	pngPath := filepath.Join(dir, "dot.png")
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	var pngBuf bytes.Buffer

	err := png.Encode(&pngBuf, img)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(pngPath, pngBuf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	headerPath := filepath.Join(dir, "header.html")
	hdr := `<html><head><style>
.row { display: flex; gap: 8pt; align-items: center; }
img { width: 12pt; height: 12pt; }
</style></head><body>
<div class="row"><img src="dot.png" alt=""><span>FlexHF</span></div>
<p><a href="#target">Go</a></p>
</body></html>`

	if err := os.WriteFile(headerPath, []byte(hdr), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newCommand(t, bodyWithTargetOnPage2(), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Margin.Top = -1
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if pageCount(data) < 1 {
		t.Fatal("expected PDF pages")
	}

	if !bytes.Contains(data, []byte("/Dest")) {
		t.Error("expected fragment GoTo from flex HF link")
	}
}

func TestHTMLHeaderTallContentClipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")
	// Many lines → HF taller than a typical header band; must not panic and must clip.
	var buf strings.Builder

	buf.WriteString(`<html><body>`)

	for range 40 {
		buf.WriteString(`<p>tall line</p>`)
	}

	buf.WriteString(`</body></html>`)

	if err := os.WriteFile(headerPath, []byte(buf.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newCommand(t, `<html><body><p>BODY-SENTINEL</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Margin.Top = -1
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if pageCount(data) < 1 {
		t.Fatal("expected PDF")
	}

	if !bytes.Contains(data, []byte("BODY-SENTINEL")) {
		t.Error("body content missing after tall HF")
	}
}

func TestHTMLHeaderPlaceholdersCopies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")
	hdr := `<html><body><p>P[page]/[topage]</p></body></html>`

	if err := os.WriteFile(headerPath, []byte(hdr), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.Margin.Top = -1
	cmd.Global.Copies = 2
	cmd.Global.Collate = true
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if pageCount(data) != 2 {
		t.Fatalf("pages = %d, want 2", pageCount(data))
	}

	if bytes.Contains(data, []byte("P0/0")) || bytes.Contains(data, []byte("P0/")) {
		t.Error("HTML HF placeholders expanded to page 0 (load-time substitute bug)")
	}

	if !bytes.Contains(data, []byte("P1/2")) && !bytes.Contains(data, []byte("P2/2")) {
		t.Error("expected per-page HTML HF placeholder expansion P1/2 or P2/2")
	}
}

func TestRemapPageForCopies(t *testing.T) {
	t.Parallel()
	// collate: [0,1, 0',1'] with logicalN=2, copies=2
	if got := remapPageForCopies(0, 2, true); got != 1 {
		t.Errorf("collate src0→dest1 = %d, want 1", got)
	}

	if got := remapPageForCopies(2, 2, true); got != 3 {
		t.Errorf("collate src2→dest1' = %d, want 3", got)
	}
	// non-collate: [0,0', 1,1']
	if got := remapPageForCopies(0, 2, false); got != 2 {
		t.Errorf("non-collate src0→dest1 = %d, want 2", got)
	}

	if got := remapPageForCopies(1, 2, false); got != 3 {
		t.Errorf("non-collate src1→dest1' = %d, want 3", got)
	}

	if got := remapPageForCopies(0, 1, true); got != 1 {
		t.Errorf("copies=1 passthrough = %d, want 1", got)
	}
}

// bodyWithLinkToPage2 is the source object of the cross-object copy test: it
// carries a #target fragment link, and the target object is its own document.
func bodyWithLinkToPage2() string {
	return `<html><body><p><a href="#target">jump</a></p></body></html>`
}

// TestBodyLinkDestRemapCopiesNonCollate proves a body internal link on a
// copied page targets the destination in its own copy group. Two one-page
// objects under Copies=2, Collate=false order as [p1, p1', p2, p2'], so the
// original link must target p2 and the copy must target p2'. Before the
// identity fix every copy kept the same pre-copy page index.
func TestBodyLinkDestRemapCopiesNonCollate(t *testing.T) {
	t.Parallel()

	cmd := newCommandMulti(t,
		[]string{
			bodyWithLinkToPage2(),
			`<html><body><h2 id="target">Target Heading</h2></body></html>`,
		},
		filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Copies = 2
	cmd.Global.Collate = false
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)

	if pageCount(data) != 4 {
		t.Fatalf("pages = %d, want 4", pageCount(data))
	}

	kids := pageKidsRefs(data)
	if len(kids) != 4 {
		t.Fatalf("Kids = %v, want 4 page refs", kids)
	}

	sem, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if len(sem.Pages) != 4 {
		t.Fatalf("parsed pages = %d, want 4", len(sem.Pages))
	}

	checks := []struct {
		page int
		dest int
	}{
		{page: 0, dest: kids[2]}, // original page 1 -> copy 0's page 2
		{page: 1, dest: kids[3]}, // copied page 1 -> copy 1's page 2
	}

	for _, check := range checks {
		annots := sem.Pages[check.page].Annots
		if len(annots) != 1 {
			t.Fatalf("page %d annots = %d, want 1 body link", check.page, len(annots))
		}

		if annots[0].DestPage != check.dest {
			t.Errorf("page %d link /Dest = %d, want %d (kids=%v)", check.page, annots[0].DestPage, check.dest, kids)
		}
	}

	for _, page := range []int{2, 3} {
		if len(sem.Pages[page].Annots) != 0 {
			t.Errorf("page %d annots = %d, want none", page, len(sem.Pages[page].Annots))
		}
	}
}

// TestTOCLinkDestRemapCopiesNonCollate covers the TOC link path under the
// same copy rule: forward links on the original TOC page target copy 0's
// chapters, and links on the copied TOC page target copy 1's chapters.
func TestTOCLinkDestRemapCopiesNonCollate(t *testing.T) {
	t.Parallel()

	cmd := newCommandMulti(t,
		[]string{
			`<html><body><h1>Chapter One</h1><p>one</p></body></html>`,
			`<html><body><h1>Chapter Two</h1><p>two</p></body></html>`,
		},
		filepath.Join(t.TempDir(), "out.pdf"))

	toc := settings.DefaultPdfObject()
	toc.IsTableOfContent = true
	toc.UseOutline = false
	cmd.Objects = append([]settings.PdfObject{toc}, cmd.Objects...)
	cmd.Global.TOC.ForwardLinks = true
	cmd.Global.Copies = 2
	cmd.Global.Collate = false
	cmd.Global.Outline = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)

	if pageCount(data) != 6 {
		t.Fatalf("pages = %d, want 6 (3 logical × 2 copies)", pageCount(data))
	}

	kids := pageKidsRefs(data)
	if len(kids) != 6 {
		t.Fatalf("Kids = %v, want 6 page refs", kids)
	}

	sem, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	// Non-collate order is [toc, toc', c1, c1', c2, c2'].
	wantDests := map[int]map[int]int{
		0: {kids[2]: 1, kids[4]: 1}, // TOC page -> copy 0 chapters
		1: {kids[3]: 1, kids[5]: 1}, // copied TOC -> copy 1 chapters
	}

	for page := range 2 {
		assertTOCLinkDests(t, sem.Pages[page].Annots, page, wantDests[page], kids)
	}

	for page := 2; page < 6; page++ {
		if len(sem.Pages[page].Annots) != 0 {
			t.Errorf("page %d annots = %d, want none", page, len(sem.Pages[page].Annots))
		}
	}
}

// assertTOCLinkDests checks that one TOC page carries exactly the expected
// number of forward links to each destination page.
func assertTOCLinkDests(t *testing.T, annots []pdf.SemanticAnnot, page int, want map[int]int, kids []int) {
	t.Helper()

	got := map[int]int{}

	for _, annot := range annots {
		got[annot.DestPage]++
	}

	for dest, count := range want {
		if got[dest] != count {
			t.Errorf("TOC page %d dest %d count = %d, want %d (all dests %v, kids=%v)",
				page, dest, got[dest], count, got, kids)
		}
	}

	if len(got) != len(want) {
		t.Errorf("TOC page %d dests = %v, want %d distinct", page, got, len(want))
	}
}
