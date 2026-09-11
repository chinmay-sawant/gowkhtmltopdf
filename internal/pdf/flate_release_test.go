package pdf

import (
	"bytes"
	"testing"
)

// flateReleasePageCount is larger than maxPageFlateWorkers so Write walks
// more than one compression window, including a short last window.
const flateReleasePageCount = 20

// TestFlateReleaseTwoWritesByteIdentical pins PERF3-30: windowed
// flate-then-finalize emits the same bytes on two independent builds of the
// same multi-page document, a second Write of a finalized document stays
// identical and does not panic, and each page's Content.Bytes is empty after
// Write (PDF-06).
func TestFlateReleaseTwoWritesByteIdentical(t *testing.T) {
	t.Parallel()

	first := writePDF(t, buildParallelDoc(t, flateReleasePageCount))
	second := writePDF(t, buildParallelDoc(t, flateReleasePageCount))

	if !bytes.Equal(first, second) {
		t.Fatalf("two builds of the same %d-page document are not byte-identical (%d vs %d bytes)",
			flateReleasePageCount, len(first), len(second))
	}

	doc := buildParallelDoc(t, flateReleasePageCount)

	var out, again bytes.Buffer

	if _, err := doc.WriteTo(&out); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if !bytes.Equal(first, out.Bytes()) {
		t.Fatalf("third build differs from the first (%d vs %d bytes)", len(out.Bytes()), len(first))
	}

	requireFlateReleaseBuffersEmpty(t, doc)

	if _, err := doc.WriteTo(&again); err != nil {
		t.Fatalf("second WriteTo: %v", err)
	}

	if !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Fatal("repeated WriteTo after windowed flate-and-release is not deterministic")
	}

	semantic, err := ParseSemantic(out.Bytes())
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if got := semantic.PageCount(); got != flateReleasePageCount {
		t.Fatalf("page count = %d, want %d", got, flateReleasePageCount)
	}
}

// requireFlateReleaseBuffersEmpty requires every page builder to have dropped
// its raw buffer after Write, matching the PDF-06 contract in
// content_lifetime_test.go without editing that file.
func requireFlateReleaseBuffersEmpty(t *testing.T, doc *Document) {
	t.Helper()

	for _, page := range doc.pages {
		if page.content == nil {
			t.Fatalf("page %d content builder was dropped; Page.Content must stay callable", page.index)
		}

		if got := len(page.content.Bytes()); got != 0 {
			t.Errorf("page %d Content.Bytes len = %d after write, want 0 (PDF-06)",
				page.index, got)
		}

		if got := page.content.buf.Cap(); got != 0 {
			t.Errorf("page %d raw content cap = %d after write, want 0 (backing array not released)",
				page.index, got)
		}

		// Page.Content stays callable after release; this must not panic.
		_ = page.Content().Bytes()
	}
}
