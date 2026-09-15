package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

// manyLinkLines builds a document that mimics a wrapped link index: each link
// occupies three vertically adjacent line rectangles, and the line widths
// differ the way wrapped text does.
func manyLinkLines(t *testing.T, merge bool) []byte {
	t.Helper()

	const (
		linksPerPage = 20
		linesPerLink = 3
		linkPitchPt  = 36.0
		linePitchPt  = 10.0
	)

	doc := NewDocument()
	page := doc.AddPage(595, 842)

	for link := range linksPerPage {
		uri := fmt.Sprintf("https://example.com/tutorial/%04d/introduction-to-a-long-tutorial", link)
		top := 800.0 - float64(link)*linkPitchPt

		for line := range linesPerLink {
			lineTop := top - float64(line)*linePitchPt
			rect := [4]float64{60, lineTop - 9, 320 - float64(line*30), lineTop}

			if merge {
				page.AddLinkURI(rect, uri)

				continue
			}

			// Pre-merge writer behavior: one annotation per line, appended raw.
			page.annots = append(page.annots, annotation{
				rect:     rect,
				uri:      uri,
				annotRef: doc.newObject(),
			})
		}
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return buf.Bytes()
}

func TestAddLinkURIMergesWrappedLineRects(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	page := doc.AddPage(595, 842)

	const uri = "https://example.com/cpp-tutorial/introduction"
	first := page.AddLinkURI([4]float64{60, 700, 300, 712}, uri)
	second := page.AddLinkURI([4]float64{60, 688, 260, 700}, uri)
	third := page.AddLinkURI([4]float64{60, 676, 300, 688}, uri)

	if len(page.annots) != 1 {
		t.Fatalf("annots = %d, want 1 merged annotation", len(page.annots))
	}

	if first == 0 || first != second || second != third {
		t.Fatalf("merged calls returned refs %d, %d, %d; want one non-zero ref", first, second, third)
	}

	want := [4]float64{60, 676, 300, 712}
	if page.annots[0].rect != want {
		t.Fatalf("merged rect = %v, want %v", page.annots[0].rect, want)
	}
}

func TestAddLinkURIDoesNotMergeDistinctTargets(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	page := doc.AddPage(595, 842)

	// Same rectangle, different URI: overlapping rectangles with different
	// targets stay separate annotations.
	page.AddLinkURI([4]float64{60, 700, 300, 712}, "https://example.com/a")
	page.AddLinkURI([4]float64{60, 700, 300, 712}, "https://example.com/b")

	// Same URI but a vertical gap larger than the merge tolerance.
	page.AddLinkURI([4]float64{60, 650, 300, 662}, "https://example.com/c")
	page.AddLinkURI([4]float64{60, 620, 300, 632}, "https://example.com/c")

	// Same URI, vertically adjacent, but no horizontal overlap.
	page.AddLinkURI([4]float64{60, 580, 120, 592}, "https://example.com/d")
	page.AddLinkURI([4]float64{200, 568, 300, 580}, "https://example.com/d")

	if len(page.annots) != 6 {
		t.Fatalf("annots = %d, want 6 (no unsafe merges)", len(page.annots))
	}
}

func TestAddLinkDestMergesSameTargetOnly(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	src := doc.AddPage(595, 842)
	dest := doc.AddPage(595, 842)

	first := src.AddLinkDest([4]float64{60, 700, 300, 712}, dest, 40, 700)
	second := src.AddLinkDest([4]float64{60, 688, 300, 700}, dest, 40, 700)

	if len(src.annots) != 1 || first != second {
		t.Fatalf("same-target dest links: annots = %d, refs = %d/%d, want one annotation",
			len(src.annots), first, second)
	}

	// Same page, different position: distinct destinations never merge.
	src.AddLinkDest([4]float64{60, 676, 300, 688}, dest, 40, 800)
	src.AddLinkDest([4]float64{60, 664, 300, 676}, dest, 41, 700)

	if len(src.annots) != 3 {
		t.Fatalf("annots = %d, want 3", len(src.annots))
	}

	// A nil destination adds nothing.
	if ref := src.AddLinkDest([4]float64{60, 600, 300, 612}, nil, 0, 0); ref != 0 {
		t.Fatalf("nil dest ref = %d, want 0", ref)
	}

	if len(src.annots) != 3 {
		t.Fatalf("nil dest changed annots to %d, want 3", len(src.annots))
	}
}

func TestTaggedDocumentDoesNotMergeLinks(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetInfo("Title", "Tagged links")
	page := doc.AddPage(595, 842)

	const uri = "https://example.com/tagged"

	page.AddLinkURI([4]float64{60, 700, 300, 712}, uri)
	page.AddLinkURI([4]float64{60, 688, 300, 700}, uri)

	if len(page.annots) != 2 {
		t.Fatalf("tagged annots = %d, want 2 (one structure element each)", len(page.annots))
	}
}

func TestLinkMergeReducesAnnotCountAndFileSize(t *testing.T) {
	t.Parallel()

	before := manyLinkLines(t, false)
	after := manyLinkLines(t, true)

	beforeAnnots := bytes.Count(before, []byte("/Subtype /Link"))
	afterAnnots := bytes.Count(after, []byte("/Subtype /Link"))

	t.Logf("before: %d annots, %d bytes", beforeAnnots, len(before))
	t.Logf("after: %d annots, %d bytes", afterAnnots, len(after))

	if beforeAnnots != 60 {
		t.Fatalf("before annots = %d, want 60 (one per wrapped line)", beforeAnnots)
	}

	if afterAnnots != 20 {
		t.Fatalf("after annots = %d, want 20 (one per link)", afterAnnots)
	}

	if len(after) >= len(before) {
		t.Fatalf("merged file = %d bytes, want smaller than %d bytes", len(after), len(before))
	}
}
