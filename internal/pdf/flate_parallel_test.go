package pdf

import (
	"bytes"
	"compress/zlib"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestFlateStateResetMatchesFreshWriter pins the byte-identity assumption
// behind the parallel page-stream path: a reused flateState (Reset then write)
// emits exactly the bytes a fresh zlib writer at the same level emits. If this
// ever fails, parallel compression has changed output bytes.
func TestFlateStateResetMatchesFreshWriter(t *testing.T) {
	t.Parallel()

	inputs := [][]byte{
		{},
		[]byte("q 1 0 0 1 10 20 cm 0 0 m 100 100 l S Q\n"),
		bytes.Repeat([]byte("representative invoice row 42 amount 1099.00\n"), 300),
	}

	for index, raw := range inputs {
		fresh := &flateState{}
		fresh.zw, _ = zlib.NewWriterLevel(&fresh.buf, zlib.DefaultCompression)
		_, _ = fresh.zw.Write(raw)
		_ = fresh.zw.Close()

		want := append([]byte(nil), fresh.buf.Bytes()...)

		reused := &flateState{}
		reused.zw, _ = zlib.NewWriterLevel(&reused.buf, zlib.DefaultCompression)

		if got := reused.compress(raw); !bytes.Equal(got, want) {
			t.Errorf("input %d: reused state emitted %d bytes, fresh writer emitted %d",
				index, len(got), len(want))
		}
	}
}

// TestRetainedPoolMatchesSerialFlate pins that the worker set emits exactly
// the serial flateBytes bytes for every stream, in input order.
func TestRetainedPoolMatchesSerialFlate(t *testing.T) {
	t.Parallel()

	raws := make([][]byte, 0, 24)

	for i := range 24 {
		stream := []byte("q 1 0 0 1 " + strconv.Itoa(i) + " 0 cm\n")
		stream = append(stream, bytes.Repeat([]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. "), 40)...)
		stream = append(stream, []byte("Q\n")...)
		raws = append(raws, stream)
	}

	want := make([][]byte, len(raws))

	for index := range raws {
		want[index] = flateBytes(raws[index])
	}

	got := retainedPageFlate.compress(raws)

	if len(got) != len(want) {
		t.Fatalf("pool returned %d streams, want %d", len(got), len(want))
	}

	for index := range want {
		if !bytes.Equal(got[index], want[index]) {
			t.Fatalf("stream %d: pool output (%d bytes) differs from serial flateBytes (%d bytes)",
				index, len(got[index]), len(want[index]))
		}
	}
}

// TestSinglePageStaysOnSerialPath pins the single-page rule: a one-page
// document never selects the worker path, so writing it cannot start the
// retained pool.
func TestSinglePageStaysOnSerialPath(t *testing.T) {
	t.Parallel()

	if flatePagesInParallel(1) {
		t.Error("single-page document selected the parallel flate path")
	}

	if runtime.GOMAXPROCS(0) > 1 && !flatePagesInParallel(2) {
		t.Error("two-page document did not select the parallel flate path")
	}
}

// buildParallelDoc builds a multi-page document whose write goes through the
// retained pool (more than one page, compression on).
func buildParallelDoc(t *testing.T, pages int) *Document {
	t.Helper()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc := fixedDoc(t)

	for index := range pages {
		page := doc.AddPage(595.276, 841.89)
		content := page.Content()
		content.UseEmbeddedFont("F1", fnt)
		content.BeginText()
		content.SetFont("F1", 10)
		content.TextAt(20, 800)
		content.TextShow("parallel page " + strconv.Itoa(index))
		content.EndText()
		page.AddLinkURI([4]float64{20, 780, 120, 800}, "https://example.test/"+strconv.Itoa(index))
	}

	return doc
}

// TestSerialAndParallelPageStreamsByteIdentical builds the same 24-page
// document twice: once with GOMAXPROCS forced to 1 so the writer takes the
// serial flateBytes path, once with the process default so the retained pool
// compresses the streams. The two files must be byte-identical. This is the
// end-to-end form of the PERFT-20 byte-identity proof.
//
//nolint:paralleltest // GOMAXPROCS is process-global; this test must not share the process with parallel writers.
func TestSerialAndParallelPageStreamsByteIdentical(t *testing.T) {
	serial := func() []byte {
		previous := runtime.GOMAXPROCS(1)
		defer runtime.GOMAXPROCS(previous)

		return writePDF(t, buildParallelDoc(t, 24))
	}()

	parallelOut := writePDF(t, buildParallelDoc(t, 24))

	if !bytes.Equal(serial, parallelOut) {
		t.Fatalf("serial and parallel page compression produced different files (%d vs %d bytes)",
			len(serial), len(parallelOut))
	}
}

// TestParallelCompressionDeterministic pins the PERFT-21 contract: two builds
// of the same multi-page document serialize to identical bytes when the
// retained pool compresses the page streams, the pages all survive, and a
// second WriteTo of one of the documents stays identical.
func TestParallelCompressionDeterministic(t *testing.T) {
	t.Parallel()

	first := writePDF(t, buildParallelDoc(t, 40))
	second := writePDF(t, buildParallelDoc(t, 40))

	if !bytes.Equal(first, second) {
		t.Fatalf("two builds of the same 40-page document are not byte-identical (%d vs %d bytes)",
			len(first), len(second))
	}

	semantic, err := ParseSemantic(first)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if got := semantic.PageCount(); got != 40 {
		t.Fatalf("page count = %d, want 40", got)
	}

	doc := buildParallelDoc(t, 40)

	var out, again bytes.Buffer

	if _, err := doc.WriteTo(&out); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if _, err := doc.WriteTo(&again); err != nil {
		t.Fatalf("second WriteTo: %v", err)
	}

	if !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Error("repeated WriteTo of a parallel-compressed document is not deterministic")
	}
}

// TestParallelCompressionXrefOffsetsExact verifies every xref entry of a
// parallel-compressed document points at its own "N 0 obj" so the worker
// ordering cannot skew byte counts.
func TestParallelCompressionXrefOffsetsExact(t *testing.T) {
	t.Parallel()

	out := writePDF(t, buildParallelDoc(t, 24))
	lines := strings.Split(string(out), "\n")
	xrefIdx := findLine(lines)

	if xrefIdx < 0 {
		t.Fatal("no xref")
	}

	startxref := -1

	for i := xrefIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "startxref") {
			startxref = i

			break
		}
	}

	if startxref < 0 {
		t.Fatal("no startxref")
	}

	offsets := parseXrefEntries(t, lines, xrefIdx, startxref)
	if len(offsets) == 0 {
		t.Fatal("parsed zero xref entries")
	}

	for obj, off := range offsets {
		want := strconv.Itoa(obj) + " 0 obj"

		if !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Errorf("object %d offset %d does not start with %q", obj, off, want)
		}
	}
}
