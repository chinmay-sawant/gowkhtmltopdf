package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

var errReviewFlush = errors.New("review sink: flush failed")

type reviewFailOnceWriter struct {
	bytes.Buffer
	failed bool
}

func (w *reviewFailOnceWriter) Write(data []byte) (int, error) {
	if !w.failed {
		w.failed = true

		return 0, errReviewFlush
	}

	count, err := w.Buffer.Write(data)
	if err != nil {
		return count, fmt.Errorf("write retry sink: %w", err)
	}

	return count, nil
}

type reviewPrefixErrorWriter struct {
	limit   int
	written int
}

func (w *reviewPrefixErrorWriter) Write(data []byte) (int, error) {
	if w.written >= w.limit {
		return 0, errReviewFlush
	}

	count := len(data)
	if remaining := w.limit - w.written; count > remaining {
		count = remaining
	}

	w.written += count

	return count, errReviewFlush
}

func TestEmptyPageContentIsSerializedAsAnEmptyStream(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)
	doc.SetCompression(false)
	doc.AddPage(200, 200)

	out := writePDF(t, doc)
	if _, err := ParseSemantic(out); err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if !strings.Contains(string(out), "/Length 0 >>\nstream\n\nendstream\nendobj") {
		t.Fatalf("empty page content is not serialized as an empty stream:\n%s", out)
	}
}

func TestWriteToCountExcludesBufferedBytesAfterFlushError(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)
	doc.AddPage(200, 200)

	sink := &reviewPrefixErrorWriter{limit: 7, written: 0}

	count, err := doc.WriteTo(sink)
	if !errors.Is(err, errReviewFlush) {
		t.Fatalf("WriteTo error = %v, want %v", err, errReviewFlush)
	}

	if count != int64(sink.written) {
		t.Fatalf("WriteTo count = %d, sink accepted %d bytes", count, sink.written)
	}
}

func TestDocumentCanRetryAfterLateSinkError(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)
	doc.AddPage(200, 200)

	failedSink := &reviewFailOnceWriter{Buffer: bytes.Buffer{}, failed: false}

	if _, err := doc.WriteTo(failedSink); !errors.Is(err, errReviewFlush) {
		t.Fatalf("first WriteTo error = %v, want %v", err, errReviewFlush)
	}

	var retry bytes.Buffer
	if err := doc.Write(&retry); err != nil {
		t.Fatalf("retry Write: %v", err)
	}

	fresh := fixedDoc(t)
	fresh.AddPage(200, 200)

	if want := writePDF(t, fresh); !bytes.Equal(retry.Bytes(), want) {
		t.Fatal("retry output differs from a fresh document")
	}
}

func TestRepeatedJPEGImageReusesXObject(t *testing.T) {
	t.Parallel()

	doc := fixedDoc(t)
	doc.SetCompression(false)
	page := doc.AddPage(100, 100)
	content := page.Content()
	imageData := makeJPEG(t)

	if err := content.AddJPEGImage("J0", 0, 0, 10, 10, imageData); err != nil {
		t.Fatalf("first image: %v", err)
	}

	if err := content.AddJPEGImage("J1", 20, 20, 10, 10, imageData); err != nil {
		t.Fatalf("repeated image: %v", err)
	}

	if content.imageRefs["J0"].ref != content.imageRefs["J1"].ref {
		t.Fatalf("repeated JPEG refs = %v/%v, want one XObject", content.imageRefs["J0"].ref, content.imageRefs["J1"].ref)
	}

	if len(content.imageDedup) != 1 {
		t.Fatalf("dedup entries = %d, want 1", len(content.imageDedup))
	}
}
