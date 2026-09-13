package pdf

import (
	"bytes"
	"errors"
	"testing"
)

// TestContentLifetimeRawBuffersReleasedAfterWrite pins the PDF-06 contract:
// once every page's content stream has been safely materialized into its
// indirect object, the raw uncompressed bytes.Buffer that built the page must
// not stay reachable through Page.Content(). The page stream lives on in
// Document.objects (pdf.go:35-40); the builder buffer is dead weight at that
// point.
//
// PDF-06 releases the buffer inside finalizePage (pdf.go) right after the
// page resources are built, so the bytes are gone during serialization rather
// than after the whole document is dropped.
func TestContentLifetimeRawBuffersReleasedAfterWrite(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)

	first := doc.AddPage(200, 200)
	first.Content().Rect(10, 20, 30, 40)
	first.Content().Fill()

	second := doc.AddPage(200, 200)
	second.Content().Rect(50, 60, 70, 80)
	second.Content().Fill()

	requireRawBuffersPresentBeforeWrite(t, doc)

	var out bytes.Buffer
	if _, err := doc.WriteTo(&out); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	requireRawBuffersReleasedAfterWrite(t, doc)

	semDoc, err := ParseSemantic(out.Bytes())
	if err != nil {
		t.Fatalf("ParseSemantic after release: %v", err)
	}

	if got := semDoc.PageCount(); got != 2 {
		t.Fatalf("page count after release = %d, want 2", got)
	}

	// Releasing the builder buffer must not change serialization: a second
	// write of the same finalized document stays byte-identical.
	var again bytes.Buffer
	if _, err := doc.WriteTo(&again); err != nil {
		t.Fatalf("second WriteTo: %v", err)
	}

	if !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Fatal("repeat WriteTo after release is not deterministic")
	}
}

// requireRawBuffersPresentBeforeWrite requires every page builder to hold raw
// content before the document is written.
func requireRawBuffersPresentBeforeWrite(t *testing.T, doc *Document) {
	t.Helper()

	for _, page := range doc.pages {
		if page.content == nil {
			t.Fatalf("page %d has no content builder before write", page.index)
		}

		if page.content.buf.Len() == 0 {
			t.Fatalf("precondition: page %d raw content is empty before write", page.index)
		}
	}
}

// requireRawBuffersReleasedAfterWrite requires every page builder to have
// dropped its raw buffer after Write. Capacity is the part that actually pins
// memory; Len alone would also go to zero under a plain Reset that keeps the
// backing array.
func requireRawBuffersReleasedAfterWrite(t *testing.T, doc *Document) {
	t.Helper()

	for _, page := range doc.pages {
		if page.content == nil {
			t.Fatalf("page %d content builder was dropped; Page.Content must stay callable", page.index)
		}

		if got := page.content.buf.Len(); got != 0 {
			t.Errorf("page %d raw content len = %d after write, want 0 (PDF-06 release contract)",
				page.index, got)
		}

		if got := page.content.buf.Cap(); got != 0 {
			t.Errorf("page %d raw content cap = %d after write, want 0 (backing array not released)",
				page.index, got)
		}
	}
}

// TestContentLifetimeCompressionOffStreamStaysIntact guards the uncompressed
// path. With compression off, setStream (pdf.go:289-297) stores the exact
// slice returned by Content.Bytes(), so the page object aliases the builder
// array. Whatever PDF-06 does to the builder must not blank, shift, or reuse
// those bytes before the object is serialized, and repeat writes must stay
// byte-identical.
func TestContentLifetimeCompressionOffStreamStaysIntact(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)
	doc.SetCompression(false)

	first := doc.AddPage(200, 200)
	first.Content().Rect(11, 22, 33, 44)
	first.Content().Fill()

	second := doc.AddPage(200, 200)
	second.Content().Rect(55, 66, 77, 88)
	second.Content().Fill()

	var out bytes.Buffer
	if _, err := doc.WriteTo(&out); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	for _, want := range [][]byte{[]byte("11 22 33 44 re"), []byte("55 66 77 88 re")} {
		if !bytes.Contains(out.Bytes(), want) {
			t.Errorf("uncompressed output lost %q", want)
		}
	}

	if _, err := ParseSemantic(out.Bytes()); err != nil {
		t.Fatalf("ParseSemantic(compression off): %v", err)
	}

	var again bytes.Buffer
	if _, err := doc.WriteTo(&again); err != nil {
		t.Fatalf("second WriteTo: %v", err)
	}

	if !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Fatal("repeat WriteTo with compression off is not deterministic")
	}
}

// TestContentLifetimeFailedFinalizeIsTerminal guards the failure policy that
// makes the per-page release safe. finalizePage releases page 1's raw buffer
// before a later page can fail, for example when a page marks a font that the
// compliant profile cannot embed. A retry could then serialize an empty page
// 1, so a mutating finalize failure becomes sticky (Document.failFinalize):
// every later Write returns the same error and writes no bytes. The raw
// stream cannot be rebuilt once released, so no silent content loss is
// possible.
func TestContentLifetimeFailedFinalizeIsTerminal(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3a,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "lifetime terminal")

	first := doc.AddPage(200, 200)
	first.Content().Rect(10, 20, 30, 40)
	first.Content().Fill()

	second := doc.AddPage(200, 200)
	// A used font with no registered face fails closed inside finalizePage
	// (content.go fonts) after page 1 was already finalized and released.
	second.content.fontUses["FMissing"] = ""

	var failed bytes.Buffer

	firstErr := doc.Write(&failed)
	if firstErr == nil {
		t.Fatal("first Write succeeded, want missing-font error")
	}

	if failed.Len() != 0 {
		t.Fatalf("failed Write wrote %d bytes, want 0", failed.Len())
	}

	if len(first.content.Bytes()) != 0 {
		t.Fatal("finalizePage did not release the first page raw content")
	}

	// Fixing the document must not turn the retry into a PDF whose page 1
	// stream is empty.
	second.Content().UseEmbeddedFont("FMissing", fnt)

	var retry bytes.Buffer

	retryErr := doc.Write(&retry)
	if !errors.Is(retryErr, firstErr) {
		t.Fatalf("retry Write error = %v, want sticky %v", retryErr, firstErr)
	}

	if retry.Len() != 0 {
		t.Fatalf("terminal finalize wrote %d bytes, want 0", retry.Len())
	}
}
