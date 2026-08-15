//nolint:testpackage,exhaustruct,gocognit,varnamelen,wsl,lll,cyclop,funlen,gocyclo,maintidx // tests verify internal tagged PDF structures and object streams
package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

//nolint:maintidx // comprehensive test for tagged PDF structure
func TestTaggedPDFUA1Structure(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	fixedTime := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Accessible Tagged Document")
	doc.SetLang("en-US")

	page := doc.AddPage(600, 800)

	// Build structure tree: Document > H1 + P + Table (TR > TH + TD) + Figure (with Alt) + Link
	root := doc.CreateStructTreeRoot()
	if root == nil {
		t.Fatal("expected non-nil StructTreeRoot for PDF/UA-1")
	}

	docElem := root.NewChild(StructTypeDocument)

	// H1
	h1 := docElem.NewChild(StructTypeH1)
	mcidH1 := page.AllocMCID(h1)

	// P
	p := docElem.NewChild(StructTypeP)
	mcidP := page.AllocMCID(p)

	// Table > TR > TH + TD
	table := docElem.NewChild(StructTypeTable)
	tr := table.NewChild(StructTypeTR)

	th := tr.NewChild(StructTypeTH)
	mcidTH := page.AllocMCID(th)

	td := tr.NewChild(StructTypeTD)
	mcidTD := page.AllocMCID(td)

	// Figure with Alt
	fig := docElem.NewChild(StructTypeFigure)
	fig.SetAlt("Company Logo Graphic")
	mcidFig := page.AllocMCID(fig)

	// Link
	linkElem := docElem.NewChild(StructTypeLink)
	linkRef := page.AddLinkURI([4]float64{50, 400, 200, 420}, "https://example.com")
	linkElem.SetObjRef(linkRef, page)
	mcidLink := page.AllocMCID(linkElem)

	// Draw content stream with marked content sequences
	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)

	// 1. Heading H1
	content.BeginMarkedContent(string(StructTypeH1), mcidH1)
	content.BeginText()
	content.SetFont("F1", 18)
	content.TextAt(50, 750)
	content.TextShow("Document Title")
	content.EndText()
	content.EndMarkedContent()

	// 2. Paragraph P
	content.BeginMarkedContent(string(StructTypeP), mcidP)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 720)
	content.TextShow("This is an accessible paragraph conforming to PDF/UA-1.")
	content.EndText()
	content.EndMarkedContent()

	// 3. Table cells (TH and TD)
	content.BeginMarkedContent(string(StructTypeTH), mcidTH)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 680)
	content.TextShow("Header Column")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeTD), mcidTD)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(150, 680)
	content.TextShow("Data Value")
	content.EndText()
	content.EndMarkedContent()

	// 4. Figure
	content.BeginMarkedContent(string(StructTypeFigure), mcidFig)
	if err := content.AddPNGImage("Im1", 50, 550, 100, 100, makePNG(t, false)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}
	content.EndMarkedContent()

	// 5. Link
	content.BeginMarkedContent(string(StructTypeLink), mcidLink)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 405)
	content.TextShow("Visit Example.com")
	content.EndText()
	content.EndMarkedContent()

	// 6. Artifact (Pagination)
	content.BeginArtifact("Pagination")
	content.BeginText()
	content.SetFont("F1", 10)
	content.TextAt(50, 30)
	content.TextShow("Page 1 of 1")
	content.EndText()
	content.EndArtifact()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write failed: %v", err)
	}

	outStr := buf.String()

	// --- 1. Header ---
	if !strings.HasPrefix(outStr, "%PDF-1.7\n") {
		t.Errorf("PDF header is not 1.7: %s", outStr[:min(len(outStr), 20)])
	}

	// --- 2. Catalog Dictionary ---
	if !strings.Contains(outStr, "/MarkInfo << /Marked true >>") {
		t.Errorf("Catalog missing /MarkInfo << /Marked true >>")
	}

	if !strings.Contains(outStr, "/Lang (en-US)") {
		t.Errorf("Catalog missing /Lang (en-US)")
	}

	if !strings.Contains(outStr, "/ViewerPreferences << /DisplayDocTitle true >>") {
		t.Errorf("Catalog missing /ViewerPreferences << /DisplayDocTitle true >>")
	}

	catRe := regexp.MustCompile(`/StructTreeRoot (\d+ 0 R)`)
	catMatches := catRe.FindStringSubmatch(outStr)

	if len(catMatches) != 2 {
		t.Fatalf("Catalog missing /StructTreeRoot reference")
	}

	structTreeRootRef := catMatches[1]
	structTreeRootObj := strings.Replace(structTreeRootRef, " 0 R", " 0 obj", 1)

	if !strings.Contains(outStr, structTreeRootObj) {
		t.Fatalf("StructTreeRoot object %s not found in output", structTreeRootObj)
	}

	// --- 3. StructTreeRoot Dictionary ---
	// Must have /Type /StructTreeRoot, /K, /ParentTree
	// Must NOT contain PDF 2.0 /Namespace
	if strings.Contains(outStr, "/Namespace") {
		t.Errorf("PDF/UA-1 output contains forbidden PDF 2.0 /Namespace")
	}

	if !strings.Contains(outStr, "<< /Type /StructTreeRoot") {
		t.Errorf("StructTreeRoot dictionary missing /Type /StructTreeRoot")
	}

	if !strings.Contains(outStr, "/ParentTree ") {
		t.Errorf("StructTreeRoot missing /ParentTree reference")
	}

	// --- 4. Page Dictionary ---
	// Must contain /StructParents 0
	if !strings.Contains(outStr, "/StructParents 0") {
		t.Errorf("Page dictionary missing /StructParents 0")
	}

	// Must contain /Tabs /S because page has annotations
	if !strings.Contains(outStr, "/Tabs /S") {
		t.Errorf("Page with annotations missing /Tabs /S")
	}

	// --- 5. StructElems ---
	// Check Document, H1, P, Table, TR, TH, TD, Figure, Link tags exist
	for _, tag := range []string{"/Document", "/H1", "/P", "/Table", "/TR", "/TH", "/TD", "/Figure", "/Link"} {
		if !strings.Contains(outStr, "/S "+tag) {
			t.Errorf("Missing StructElem with tag %s", tag)
		}
	}

	// Check Figure Alt text
	if !strings.Contains(outStr, "/Alt (Company Logo Graphic)") {
		t.Errorf("Figure StructElem missing /Alt (Company Logo Graphic)")
	}

	// Check Link OBJR reference
	annotRefStr := linkElem.AnnotRef.String()
	pageRefStr := page.ref.String()
	wantOBJR := fmt.Sprintf("<< /Type /OBJR /Obj %s /Pg %s >>", annotRefStr, pageRefStr)

	if !strings.Contains(outStr, wantOBJR) {
		t.Errorf("Link StructElem missing OBJR dict %q", wantOBJR)
	}

	// --- 6. ParentTree (Number Tree) ---
	// Check that ParentTree contains /Nums [ 0 [ ... ] ... ]
	// and that for table cells, TH and TD refs are in the array (not TR ref).
	numsRe := regexp.MustCompile(`<< /Nums \[ 0 \[ ([^\]]+) \]`)
	numsMatches := numsRe.FindStringSubmatch(outStr)

	if len(numsMatches) != 2 {
		t.Fatalf("ParentTree missing /Nums [ 0 [ ... ] ], output:\n%s", outStr)
	}

	mcidRefs := strings.Fields(numsMatches[1])
	// Should have 6 MCID refs (mcid 0..5): H1, P, TH, TD, Figure, Link
	if len(mcidRefs) != 12 { // each ref is "N 0 R" (2 tokens: N and "0 R" or "N", "0", "R" -> 6 refs * 3 fields = 18 or 6 refs)
		// Let's count "0 R" occurrences
		refCount := strings.Count(numsMatches[1], "0 R")
		if refCount != 6 {
			t.Errorf("ParentTree /Nums has %d element refs, want 6", refCount)
		}
	}

	// Verify TH and TD refs are in ParentTree, but TR ref is NOT in ParentTree
	thRefStr := th.Ref()
	tdRefStr := td.Ref()
	trRefStr := tr.Ref()

	if !strings.Contains(numsMatches[1], thRefStr) {
		t.Errorf("ParentTree missing TH ref %s", thRefStr)
	}

	if !strings.Contains(numsMatches[1], tdRefStr) {
		t.Errorf("ParentTree missing TD ref %s", tdRefStr)
	}

	if strings.Contains(numsMatches[1], trRefStr) {
		t.Errorf("ParentTree contains TR ref %s (MCID should belong to TH/TD, not TR)", trRefStr)
	}

	// --- 7. Marked Content in Content Stream ---
	decompressed := page.content.Bytes()
	contentStr := string(decompressed)

	if !strings.Contains(contentStr, "/H1 << /MCID 0 >> BDC") {
		t.Errorf("Content stream missing /H1 << /MCID 0 >> BDC")
	}

	if !strings.Contains(contentStr, "/P << /MCID 1 >> BDC") {
		t.Errorf("Content stream missing /P << /MCID 1 >> BDC")
	}

	if !strings.Contains(contentStr, "/TH << /MCID 2 >> BDC") {
		t.Errorf("Content stream missing /TH << /MCID 2 >> BDC")
	}

	if !strings.Contains(contentStr, "/TD << /MCID 3 >> BDC") {
		t.Errorf("Content stream missing /TD << /MCID 3 >> BDC")
	}

	if !strings.Contains(contentStr, "/Figure << /MCID 4 >> BDC") {
		t.Errorf("Content stream missing /Figure << /MCID 4 >> BDC")
	}

	if !strings.Contains(contentStr, "/Link << /MCID 5 >> BDC") {
		t.Errorf("Content stream missing /Link << /MCID 5 >> BDC")
	}

	if !strings.Contains(contentStr, "/Artifact << /Type /Pagination >> BDC") {
		t.Errorf("Content stream missing /Artifact << /Type /Pagination >> BDC")
	}

	emcCount := strings.Count(contentStr, "EMC\n")
	if emcCount != 7 {
		t.Errorf("Content stream has %d EMC operators, want 7", emcCount)
	}

	// --- 8. XMP Metadata ---
	if !strings.Contains(outStr, "<pdfuaid:part>1</pdfuaid:part>") {
		t.Errorf("XMP Metadata missing <pdfuaid:part>1</pdfuaid:part>")
	}

	if strings.Contains(outStr, "<pdfuaid:part>2</pdfuaid:part>") {
		t.Errorf("XMP Metadata contains forbidden <pdfuaid:part>2</pdfuaid:part>")
	}
}

//nolint:cyclop // tests dual profile XMP schemas and extensions
func TestTaggedPDFUA1DualProfile(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfileDualA3aUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Dual Profile Document")
	doc.AddPage(400, 400)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// 1. PDF/A-3 claim
	if !strings.Contains(outStr, "<pdfaid:part>3</pdfaid:part>") {
		t.Error("Dual profile XMP missing pdfaid:part 3")
	}

	if !strings.Contains(outStr, "<pdfaid:conformance>A</pdfaid:conformance>") {
		t.Error("Dual profile XMP missing pdfaid:conformance A")
	}

	// 2. PDF/UA-1 claim
	if !strings.Contains(outStr, "<pdfuaid:part>1</pdfuaid:part>") {
		t.Error("Dual profile XMP missing pdfuaid:part 1")
	}

	// 3. PDF/A Extension Schema for pdfuaid
	if !strings.Contains(outStr, "<pdfaExtension:schemas>") {
		t.Error("Dual profile XMP missing pdfaExtension:schemas")
	}

	if !strings.Contains(outStr, "<pdfaSchema:prefix>pdfuaid</pdfaSchema:prefix>") {
		t.Error("Dual profile XMP missing pdfuaid prefix in extension schema")
	}

	if !strings.Contains(outStr, "http://www.aiim.org/pdfua/ns/id/") {
		t.Error("Dual profile XMP missing PDF/UA namespace URI in extension schema")
	}
}

func TestPDFUA1EmptyTitleFailsClosed(t *testing.T) {
	t.Parallel()

	// 1. Completely missing title
	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.AddPage(200, 200)

	var buf bytes.Buffer
	writeErr := doc.Write(&buf)

	if !errors.Is(writeErr, ErrTitleRequired) {
		t.Fatalf("doc.Write with empty title err = %v, want ErrTitleRequired", writeErr)
	}

	if buf.Len() != 0 {
		t.Errorf("doc.Write produced %d bytes on empty title error, want 0", buf.Len())
	}

	// 2. Whitespace-only title
	doc2, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc2.SetInfo("Title", "   \t\n  ")
	doc2.AddPage(200, 200)

	var buf2 bytes.Buffer
	writeErr2 := doc2.Write(&buf2)

	if !errors.Is(writeErr2, ErrTitleRequired) {
		t.Fatalf("doc2.Write with whitespace title err = %v, want ErrTitleRequired", writeErr2)
	}

	if buf2.Len() != 0 {
		t.Errorf("doc2.Write produced %d bytes on whitespace title error, want 0", buf2.Len())
	}
}

func TestUnclaimed14And17EmitNoStructure(t *testing.T) {
	t.Parallel()

	// 1. PDF 1.4 unclaimed
	doc14 := NewDocument()
	doc14.SetInfo("Title", "Unclaimed 1.4")
	p14 := doc14.AddPage(200, 200)

	// Structure methods should no-op
	if root := doc14.CreateStructTreeRoot(); root != nil {
		t.Errorf("CreateStructTreeRoot on PDF 1.4 returned non-nil: %v", root)
	}

	p14.Content().BeginMarkedContent("H1", 0)
	p14.Content().EndMarkedContent()

	var buf14 bytes.Buffer
	if err := doc14.Write(&buf14); err != nil {
		t.Fatalf("doc14.Write: %v", err)
	}

	str14 := buf14.String()
	for _, forbidden := range []string{"/MarkInfo", "/StructTreeRoot", "/StructParents", "pdfuaid"} {
		if strings.Contains(str14, forbidden) {
			t.Errorf("PDF 1.4 output contains unexpected structure token %q", forbidden)
		}
	}

	// 2. PDF 1.7 unclaimed
	doc17, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc17.SetInfo("Title", "Unclaimed 1.7")
	p17 := doc17.AddPage(200, 200)

	if root := doc17.CreateStructTreeRoot(); root != nil {
		t.Errorf("CreateStructTreeRoot on unclaimed PDF 1.7 returned non-nil: %v", root)
	}

	p17.Content().BeginArtifact("Pagination")
	p17.Content().EndArtifact()

	var buf17 bytes.Buffer
	if err := doc17.Write(&buf17); err != nil {
		t.Fatalf("doc17.Write: %v", err)
	}

	str17 := buf17.String()
	for _, forbidden := range []string{"/MarkInfo", "/StructTreeRoot", "/StructParents", "pdfuaid"} {
		if strings.Contains(str17, forbidden) {
			t.Errorf("Unclaimed PDF 1.7 output contains unexpected structure token %q", forbidden)
		}
	}
}

//nolint:funlen // tests multi-page document ParentTree numbering and page mapping
func TestMultiPageParentTree(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Multi-Page ParentTree Test")

	// Page 0: has MCIDs
	page0 := doc.AddPage(300, 400)
	// Page 1: empty of MCIDs
	page1 := doc.AddPage(300, 400)
	_ = page1
	// Page 2: has MCIDs
	page2 := doc.AddPage(300, 400)

	root := doc.CreateStructTreeRoot()
	docElem := root.NewChild(StructTypeDocument)

	h1 := docElem.NewChild(StructTypeH1)
	mcid0 := page0.AllocMCID(h1)

	p := docElem.NewChild(StructTypeP)
	mcid2 := page2.AllocMCID(p)

	c0 := page0.Content()
	c0.UseEmbeddedFont("F1", fnt)
	c0.BeginMarkedContent("H1", mcid0)
	c0.BeginText()
	c0.SetFont("F1", 12)
	c0.TextAt(20, 350)
	c0.TextShow("Page 1 Heading")
	c0.EndText()
	c0.EndMarkedContent()

	c2 := page2.Content()
	c2.UseEmbeddedFont("F1", fnt)
	c2.BeginMarkedContent("P", mcid2)
	c2.BeginText()
	c2.SetFont("F1", 12)
	c2.TextAt(20, 350)
	c2.TextShow("Page 3 Paragraph")
	c2.EndText()
	c2.EndMarkedContent()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// Page 0 should have /StructParents 0
	p0Obj := strings.Replace(page0.ref.String(), " 0 R", " 0 obj", 1)
	p0Idx := strings.Index(outStr, p0Obj)
	if p0Idx < 0 || !strings.Contains(outStr[p0Idx:p0Idx+300], "/StructParents 0") {
		t.Errorf("Page 0 missing /StructParents 0")
	}

	// Page 1 should NOT have /StructParents
	p1Obj := strings.Replace(page1.ref.String(), " 0 R", " 0 obj", 1)
	p1Idx := strings.Index(outStr, p1Obj)
	if p1Idx >= 0 && strings.Contains(outStr[p1Idx:p1Idx+300], "/StructParents") {
		t.Errorf("Page 1 should not have /StructParents entry")
	}

	// Page 2 should have /StructParents 1
	p2Obj := strings.Replace(page2.ref.String(), " 0 R", " 0 obj", 1)
	p2Idx := strings.Index(outStr, p2Obj)
	if p2Idx < 0 || !strings.Contains(outStr[p2Idx:p2Idx+300], "/StructParents 1") {
		t.Errorf("Page 2 missing /StructParents 1")
	}

	// ParentTree Nums should map 0 -> [ h1Ref ] and 1 -> [ pRef ]
	numsRe := regexp.MustCompile(`<< /Nums \[ 0 \[ ([^\]]+) \] 1 \[ ([^\]]+) \] \] >>`)
	matches := numsRe.FindStringSubmatch(outStr)

	if len(matches) != 3 {
		t.Fatalf("ParentTree /Nums did not match expected pattern for 2 mapped pages, output:\n%s", outStr)
	}

	if !strings.Contains(matches[1], h1.Ref()) {
		t.Errorf("Nums key 0 does not contain h1 ref %s", h1.Ref())
	}

	if !strings.Contains(matches[2], p.Ref()) {
		t.Errorf("Nums key 1 does not contain p ref %s", p.Ref())
	}
}

// TestMultiPageStructElemUsesMCR ensures a StructElem with marked content on
// more than one page serializes /K kids as MCR dictionaries (ISO 32000-1
// §14.7.4.2). Bare MCID integers under a single /Pg are structurally invalid
// when the same element appears on multiple pages.
func TestMultiPageStructElemUsesMCR(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Multi-Page MCR Test")

	page0 := doc.AddPage(300, 400)
	page1 := doc.AddPage(300, 400)

	root := doc.CreateStructTreeRoot()
	docElem := root.NewChild(StructTypeDocument)
	pElem := docElem.NewChild(StructTypeP)

	mcid0 := page0.AllocMCID(pElem)
	mcid1 := page1.AllocMCID(pElem)

	for _, item := range []struct {
		page *Page
		mcid int
		text string
	}{
		{page0, mcid0, "First page"},
		{page1, mcid1, "Second page"},
	} {
		cur := item.page.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginMarkedContent("P", item.mcid)
		cur.BeginText()
		cur.SetFont("F1", 12)
		cur.TextAt(20, 350)
		cur.TextShow(item.text)
		cur.EndText()
		cur.EndMarkedContent()
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	out := buf.String()
	want0 := fmt.Sprintf("<< /Type /MCR /Pg %s /MCID %d >>", page0.ref.String(), mcid0)
	want1 := fmt.Sprintf("<< /Type /MCR /Pg %s /MCID %d >>", page1.ref.String(), mcid1)

	for _, want := range []string{want0, want1, "/Type /MCR"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing multi-page MCR kid %q", want)
		}
	}

	// Bare duplicate MCIDs without MCR would look like "/K [ 0 0 ]" under one /Pg.
	if strings.Contains(out, "/K [ 0 0 ]") {
		t.Errorf("StructElem /K still uses bare multi-page MCIDs without MCR")
	}
}

func TestMarkedContentHelpersAndBalancing(t *testing.T) {
	t.Parallel()

	c := NewContent()
	if c.MarkedDepth() != 0 {
		t.Errorf("initial MarkedDepth = %d, want 0", c.MarkedDepth())
	}

	c.BeginMarkedContent("H1", 0)
	if c.MarkedDepth() != 1 {
		t.Errorf("after BeginMarkedContent MarkedDepth = %d, want 1", c.MarkedDepth())
	}

	c.BeginArtifact("Header")
	if c.MarkedDepth() != 2 {
		t.Errorf("after BeginArtifact MarkedDepth = %d, want 2", c.MarkedDepth())
	}

	c.EndArtifact()
	if c.MarkedDepth() != 1 {
		t.Errorf("after EndArtifact MarkedDepth = %d, want 1", c.MarkedDepth())
	}

	c.EndMarkedContent()
	if c.MarkedDepth() != 0 {
		t.Errorf("after EndMarkedContent MarkedDepth = %d, want 0", c.MarkedDepth())
	}

	// Bare artifact without type
	c.BeginArtifact("")
	c.EndArtifact()

	raw := string(c.Bytes())
	if !strings.Contains(raw, "/H1 << /MCID 0 >> BDC\n") {
		t.Errorf("missing /H1 BDC")
	}

	if !strings.Contains(raw, "/Artifact << /Type /Header >> BDC\n") {
		t.Errorf("missing Header Artifact BDC")
	}

	if !strings.Contains(raw, "/Artifact BDC\n") {
		t.Errorf("missing bare /Artifact BDC")
	}
}

func TestTableScopeAndListStructureElements(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Table Scope and List Tagging Test")
	page := doc.AddPage(600, 800)

	root := doc.CreateStructTreeRoot()
	docElem := root.NewChild(StructTypeDocument)

	// Table with Caption, TR, TH (col, row, both scopes), TD
	tbl := docElem.NewChild(StructTypeTable)
	caption := tbl.NewChild(StructTypeCaption)
	mcidCap := page.AllocMCID(caption)

	tr := tbl.NewChild(StructTypeTR)

	thCol := tr.NewChild(StructTypeTH) // default scope: Column
	mcidTHCol := page.AllocMCID(thCol)

	thRow := tr.NewChild(StructTypeTH)
	thRow.SetTableScope("Row")
	mcidTHRow := page.AllocMCID(thRow)

	thBoth := tr.NewChild(StructTypeTH)
	thBoth.SetTableScope("Both")
	mcidTHBoth := page.AllocMCID(thBoth)

	td := tr.NewChild(StructTypeTD)
	mcidTD := page.AllocMCID(td)

	// List > LI > Lbl + LBody (with nested L)
	listElem := docElem.NewChild(StructTypeL)
	liElem := listElem.NewChild(StructTypeLI)
	lblElem := liElem.NewChild(StructTypeLbl)
	mcidLbl := page.AllocMCID(lblElem)

	lbodyElem := liElem.NewChild(StructTypeLBody)
	pElem := lbodyElem.NewChild(StructTypeP)
	mcidP := page.AllocMCID(pElem)

	nestedList := lbodyElem.NewChild(StructTypeL)
	nestedLI := nestedList.NewChild(StructTypeLI)
	nestedLbl := nestedLI.NewChild(StructTypeLbl)
	mcidNestedLbl := page.AllocMCID(nestedLbl)

	nestedLBody := nestedLI.NewChild(StructTypeLBody)
	nestedP := nestedLBody.NewChild(StructTypeP)
	mcidNestedP := page.AllocMCID(nestedP)

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)

	content.BeginMarkedContent(string(StructTypeCaption), mcidCap)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 750)
	content.TextShow("Table Caption")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeTH), mcidTHCol)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 700)
	content.TextShow("Col Header")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeTH), mcidTHRow)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(150, 700)
	content.TextShow("Row Header")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeTH), mcidTHBoth)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(250, 700)
	content.TextShow("Both Header")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeTD), mcidTD)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(350, 700)
	content.TextShow("Data")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeLbl), mcidLbl)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(50, 600)
	content.TextShow("\u2022")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeP), mcidP)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(70, 600)
	content.TextShow("List item body text")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeLbl), mcidNestedLbl)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(70, 550)
	content.TextShow("-")
	content.EndText()
	content.EndMarkedContent()

	content.BeginMarkedContent(string(StructTypeP), mcidNestedP)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(90, 550)
	content.TextShow("Nested item text")
	content.EndText()
	content.EndMarkedContent()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// Verify Structure Tags exist
	expectedTags := []string{
		"/S /Caption", "/S /Table", "/S /TR", "/S /TH", "/S /TD",
		"/S /L", "/S /LI", "/S /Lbl", "/S /LBody", "/S /P",
	}
	for _, tag := range expectedTags {
		if !strings.Contains(outStr, tag) {
			t.Errorf("expected structure tag %s in output", tag)
		}
	}

	// Verify /A Table Scope Attributes on TH StructElems
	if !strings.Contains(outStr, "<< /O /Table /Scope /Column >>") {
		t.Errorf("missing /Scope /Column attribute on TH StructElem")
	}
	if !strings.Contains(outStr, "<< /O /Table /Scope /Row >>") {
		t.Errorf("missing /Scope /Row attribute on TH StructElem")
	}
	if !strings.Contains(outStr, "<< /O /Table /Scope /Both >>") {
		t.Errorf("missing /Scope /Both attribute on TH StructElem")
	}
}

func TestAnnotationComplianceFlagsAndOBJR(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Annotation Compliance Document")
	doc.SetLang("en-US")

	page := doc.AddPage(600, 800)
	root := doc.CreateStructTreeRoot()
	docElem := root.NewChild(StructTypeDocument)

	// Add link annotation and attach via SetObjRef
	ref1 := page.AddLinkURI([4]float64{50, 400, 200, 420}, "https://example.com/objr1")
	linkElem1 := docElem.NewChild(StructTypeLink)
	linkElem1.SetObjRef(ref1, page)
	mcid1 := page.AllocMCID(linkElem1)

	// Add dest annotation and attach via AddAnnot
	ref2 := page.AddLinkDest([4]float64{50, 300, 200, 320}, 0, 50, 700)
	linkElem2 := docElem.NewChild(StructTypeLink)
	linkElem2.AddAnnot(ref2, page)
	mcid2 := page.AllocMCID(linkElem2)

	c := page.Content()
	c.UseEmbeddedFont("F1", fnt)

	c.BeginMarkedContent(string(StructTypeLink), mcid1)
	c.BeginText()
	c.SetFont("F1", 12)
	c.TextAt(50, 405)
	c.TextShow("Link 1")
	c.EndText()
	c.EndMarkedContent()

	c.BeginMarkedContent(string(StructTypeLink), mcid2)
	c.BeginText()
	c.SetFont("F1", 12)
	c.TextAt(50, 305)
	c.TextShow("Link 2")
	c.EndText()
	c.EndMarkedContent()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// 1. Check /Border [0 0 0] /F 4 and /Contents on both annotations
	if !strings.Contains(outStr, "/Type /Annot /Subtype /Link /Rect [50 400 200 420] /Border [0 0 0] /F 4") {
		t.Errorf("URI annotation missing /Border [0 0 0] /F 4")
	}

	if !strings.Contains(outStr, "/Contents (https://example.com/objr1)") {
		t.Errorf("URI annotation missing /Contents")
	}

	if !strings.Contains(outStr, "/Border [0 0 0] /F 4") || !strings.Contains(outStr, "/Dest [") {
		t.Errorf("Dest annotation missing /Border [0 0 0] /F 4")
	}

	// 2. Check /Tabs /S on page
	if !strings.Contains(outStr, "/Tabs /S") {
		t.Errorf("Page missing /Tabs /S")
	}

	// 3. Check /OBJR references in StructElems
	wantOBJR1 := fmt.Sprintf("<< /Type /OBJR /Obj %s /Pg %s >>", ref1.String(), page.ref.String())
	wantOBJR2 := fmt.Sprintf("<< /Type /OBJR /Obj %s /Pg %s >>", ref2.String(), page.ref.String())

	if !strings.Contains(outStr, wantOBJR1) {
		t.Errorf("StructElem missing OBJR 1 %q", wantOBJR1)
	}

	if !strings.Contains(outStr, wantOBJR2) {
		t.Errorf("StructElem missing OBJR 2 %q", wantOBJR2)
	}
}

func TestDuplicatePagePDFUA(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: "PDF/UA-1",
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Duplicate Page UA Test")
	page := doc.AddPage(600, 800)

	root := doc.CreateStructTreeRoot()
	docElem := root.NewChild(StructTypeDocument)
	h1 := docElem.NewChild(StructTypeH1)
	mcidH1 := page.AllocMCID(h1)

	pElem := docElem.NewChild(StructTypeP)
	mcidP := page.AllocMCID(pElem)

	c := page.Content()
	c.BeginMarkedContent("H1", mcidH1)
	c.TextAt(50, 700)
	c.TextShow("Heading 1")
	c.EndMarkedContent()

	c.BeginMarkedContent("P", mcidP)
	c.TextAt(50, 650)
	c.TextShow("Paragraph")
	c.EndMarkedContent()

	cloned, err := doc.DuplicatePage(0)
	if err != nil {
		t.Fatalf("DuplicatePage: %v", err)
	}

	if len(cloned.mcids) != len(page.mcids) {
		t.Fatalf("cloned.mcids len = %d, want %d", len(cloned.mcids), len(page.mcids))
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	if !strings.Contains(outStr, "/StructParents 0") || !strings.Contains(outStr, "/StructParents 1") {
		t.Errorf("Document missing StructParents for original and clone")
	}

	if !strings.Contains(outStr, "/Type /MCR /Pg") {
		t.Errorf("Document missing MCR dictionary for multi-page structure elements")
	}
}

func TestDuplicatePageDefault14Isolation(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetInfo("Title", "Duplicate Page 1.4 Test")
	page := doc.AddPage(600, 800)

	c := page.Content()
	c.TextAt(50, 700)
	c.TextShow("Hello 1.4")

	page.AddLinkURI([4]float64{50, 700, 200, 720}, "https://example.com")

	_, err := doc.DuplicatePage(0)
	if err != nil {
		t.Fatalf("DuplicatePage: %v", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	if strings.Contains(outStr, "/StructTreeRoot") {
		t.Errorf("Default 1.4 emitted /StructTreeRoot")
	}
	if strings.Contains(outStr, "/ParentTree") {
		t.Errorf("Default 1.4 emitted /ParentTree")
	}
	if strings.Contains(outStr, "/StructParents") {
		t.Errorf("Default 1.4 emitted /StructParents")
	}
	if strings.Contains(outStr, "/Tabs /S") {
		t.Errorf("Default 1.4 with links leaked /Tabs /S")
	}
}

func TestDefaultPathIsolation(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	if root := doc.CreateStructTreeRoot(); root != nil {
		t.Errorf("doc.CreateStructTreeRoot() on 1.4 = %v, want nil", root)
	}

	if elems := doc.HeadingStructElems(); elems != nil {
		t.Errorf("doc.HeadingStructElems() on 1.4 = %v, want nil", elems)
	}

	page := doc.AddPage(600, 800)
	dummyElem := &StructElem{Tag: StructTypeP} //nolint:exhaustruct // test dummy
	if mcid := page.AllocMCID(dummyElem); mcid != -1 {
		t.Errorf("page.AllocMCID() on 1.4 = %d, want -1", mcid)
	}

	page.AddLinkURI([4]float64{50, 500, 200, 520}, "https://example.com")

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()
	if strings.Contains(outStr, "/StructTreeRoot") {
		t.Errorf("Default 1.4 emitted /StructTreeRoot")
	}
	if strings.Contains(outStr, "/ParentTree") {
		t.Errorf("Default 1.4 emitted /ParentTree")
	}
	if strings.Contains(outStr, "/Tabs /S") {
		t.Errorf("Default 1.4 emitted /Tabs /S")
	}
}
