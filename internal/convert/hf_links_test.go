package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
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
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
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
	m := kidsPageRefsRe.FindSubmatch(data)
	if m == nil {
		return nil
	}
	var out []int
	for _, ref := range pageObjRefRe.FindAllSubmatch(m[1], -1) {
		n, _ := strconv.Atoi(string(ref[1]))
		out = append(out, n)
	}
	return out
}

func destPageRefs(data []byte) []int {
	var out []int
	for _, m := range destPageRefRe.FindAllSubmatch(data, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		out = append(out, n)
	}
	return out
}

func TestHTMLHeaderFragmentGoTo(t *testing.T) {
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
	// collate order [p0,p1,p0',p1'] → dests kids[1], kids[1], kids[3], kids[3].
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

func TestRemapPageForCopies(t *testing.T) {
	// collate: [0,1, 0',1'] with logicalN=2, copies=2
	if got := remapPageForCopies(1, 0, 2, 2, true); got != 1 {
		t.Errorf("collate src0→dest1 = %d, want 1", got)
	}
	if got := remapPageForCopies(1, 2, 2, 2, true); got != 3 {
		t.Errorf("collate src2→dest1' = %d, want 3", got)
	}
	// non-collate: [0,0', 1,1']
	if got := remapPageForCopies(1, 0, 2, 2, false); got != 2 {
		t.Errorf("non-collate src0→dest1 = %d, want 2", got)
	}
	if got := remapPageForCopies(1, 1, 2, 2, false); got != 3 {
		t.Errorf("non-collate src1→dest1' = %d, want 3", got)
	}
	if got := remapPageForCopies(1, 0, 2, 1, true); got != 1 {
		t.Errorf("copies=1 passthrough = %d, want 1", got)
	}
}
