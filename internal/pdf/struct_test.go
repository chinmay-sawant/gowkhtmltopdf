package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var refRe = regexp.MustCompile(`(\d+) 0 R`)

// buildRichDoc builds a 2-page document exercising every feature.
func buildRichDoc(t *testing.T) []byte {
	t.Helper()

	doc := NewDocument()
	doc.SetCreationTime(time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	doc.SetInfo("Title", "Rich")
	doc.SetInfo("Author", "tester")

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	for idx := range 2 {
		page := doc.AddPage(300, 300)
		cur := page.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.Save()
		cur.SetFillColor(0.2, 0.4, 0.6)
		cur.Rect(10, 10, 100, 100)
		cur.Fill()
		cur.SetStrokeColor(1, 0, 0)
		cur.SetLineWidth(2)
		cur.MoveTo(0, 0)
		cur.LineTo(300, 300)
		cur.Stroke()
		cur.BeginText()
		cur.SetFont("F1", 12)
		cur.TextAt(20, 280)
		cur.TextShow(fmt.Sprintf("page %d", idx+1))
		cur.EndText()

		if err := cur.AddPNGImage("Im1", 200, 200, 80, 80, makePNG(t, false)); err != nil {
			t.Fatalf("AddPNGImage: %v", err)
		}

		cur.Restore()

		if idx == 0 {
			page.AddLinkURI([4]float64{10, 10, 110, 30}, "https://example.com")
			page.AddLinkDest([4]float64{10, 40, 110, 60}, 1, 50, 150)
		}
	}

	doc.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title: "root",
		Children: []*Outline{
			{Title: "one", PageRef: "1 0 R", X: 10, Y: 20},
			{Title: "two", Children: []*Outline{{Title: "two-a", PageRef: "2 0 R"}}},
		},
	})

	return writePDF(t, doc)
}

// parseObjects extracts "N 0 obj" … "endobj" spans and checks xref offsets.
func parseObjects(t *testing.T, out []byte) (map[int][]byte, map[int]int) {
	t.Helper()

	str := string(out)

	// trailer info
	trailerIdx := strings.Index(str, "trailer")
	if trailerIdx < 0 {
		t.Fatal("no trailer")
	}

	tr := str[trailerIdx:]
	if !strings.Contains(tr, "/Root ") || !strings.Contains(tr, "/Info ") {
		t.Error("trailer missing /Root or /Info")
	}

	// every "N 0 obj" must be matched by "endobj"
	re := regexp.MustCompile(`(\d+) 0 obj`)
	objs := map[int][]byte{}
	offsets := map[int]int{}

	for _, m := range re.FindAllStringSubmatchIndex(str, -1) {
		idVal, _ := strconv.Atoi(str[m[2]:m[3]])
		start := m[0]

		endMark := strings.Index(str[start:], "endobj")
		if endMark < 0 {
			t.Fatalf("object %d lacks endobj", idVal)
		}

		offsets[idVal] = start
		objs[idVal] = []byte(str[start : start+endMark+len("endobj")])
	}

	// xref offsets must agree with "N 0 obj" positions
	xrefIdx := strings.Index(str, "xref")
	lines := strings.Split(str[xrefIdx:], "\n")
	count := 0

	for i, l := range lines {
		fields := strings.Fields(l)
		if i >= 2 && len(fields) == 3 {
			obj := count
			count++

			off, err := strconv.Atoi(fields[0])
			if err != nil {
				t.Fatalf("bad xref offset %q", fields[0])
			}

			if obj == 0 {
				continue
			}

			want, ok := offsets[obj]
			if !ok {
				t.Errorf("xref points at object %d that is not written", obj)

				continue
			}

			if off != want {
				t.Errorf("xref offset %d for object %d, want %d", off, obj, want)
			}
		}
	}

	// every referenced object must exist
	for idVal := range objs {
		for _, rm := range refRe.FindAllStringSubmatch(string(objs[idVal]), -1) {
			ref, _ := strconv.Atoi(rm[1])
			if ref <= 0 {
				continue
			}

			if _, ok := objs[ref]; !ok {
				t.Errorf("object %d references missing object %d", idVal, ref)
			}
		}
	}

	return objs, offsets
}

// decompress streams and sanity-check dicts.
func checkStreams(t *testing.T, objs map[int][]byte) {
	t.Helper()

	streams := 0

	for id, body := range objs {
		_ = id

		if !bytes.Contains(body, []byte("stream")) {
			continue
		}

		streams++

		idx := bytes.Index(body, []byte("stream\n"))
		if idx < 0 {
			t.Errorf("object has stream marker without data")

			continue
		}

		dict := string(body[:idx])
		data := bytes.TrimSuffix(body[idx+len("stream\n"):], []byte("\nendstream"))

		if strings.Contains(dict, "/FlateDecode") {
			zreader, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				t.Errorf("zlib stream header failed: %v", err)

				continue
			}

			dec, err := io.ReadAll(zreader)
			zreader.Close()

			if err != nil {
				t.Errorf("zlib stream failed: %v", err)

				continue
			}

			if len(dec) == 0 {
				t.Errorf("empty decompressed stream")
			}
		}
	}

	if streams < 2 {
		t.Errorf("expected several streams, got %d", streams)
	}
}

func TestRichDocStructure(t *testing.T) {
	t.Parallel()
	out := buildRichDoc(t)
	objs, _ := parseObjects(t, out)
	checkStreams(t, objs)
}

func TestWriteToContract(t *testing.T) {
	t.Parallel()
	d := fixedDoc(t)
	d.AddPage(100, 100)

	var buf bytes.Buffer

	count, err := d.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if count != int64(buf.Len()) {
		t.Errorf("WriteTo returned %d, buffer has %d", count, buf.Len())
	}

	if buf.Len() == 0 {
		t.Error("empty output")
	}
}
