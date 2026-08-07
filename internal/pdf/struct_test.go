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
	d := NewDocument()
	d.SetCreationTime(time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	d.SetInfo("Title", "Rich")
	d.SetInfo("Author", "tester")
	f, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}
	for i := 0; i < 2; i++ {
		p := d.AddPage(300, 300)
		c := p.Content()
		c.UseEmbeddedFont("F1", f)
		c.Save()
		c.SetFillColor(0.2, 0.4, 0.6)
		c.Rect(10, 10, 100, 100)
		c.Fill()
		c.SetStrokeColor(1, 0, 0)
		c.SetLineWidth(2)
		c.MoveTo(0, 0)
		c.LineTo(300, 300)
		c.Stroke()
		c.BeginText()
		c.SetFont("F1", 12)
		c.TextAt(20, 280)
		c.TextShow(fmt.Sprintf("page %d", i+1))
		c.EndText()
		if err := c.AddPNGImage("Im1", 200, 200, 80, 80, makePNG(t, false)); err != nil {
			t.Fatalf("AddPNGImage: %v", err)
		}
		c.Restore()
		if i == 0 {
			p.AddLinkURI([4]float64{10, 10, 110, 30}, "https://example.com")
			p.AddLinkDest([4]float64{10, 40, 110, 60}, 1, 50, 150)
		}
	}
	d.SetOutline(&Outline{
		Title: "root",
		Children: []*Outline{
			{Title: "one", PageRef: "1 0 R", X: 10, Y: 20},
			{Title: "two", Children: []*Outline{{Title: "two-a", PageRef: "2 0 R"}}},
		},
	})
	return writePDF(t, d)
}

// parseObjects extracts "N 0 obj" … "endobj" spans and checks xref offsets.
func parseObjects(t *testing.T, out []byte) (map[int][]byte, map[int]int) {
	t.Helper()
	s := string(out)

	// trailer info
	trailerIdx := strings.Index(s, "trailer")
	if trailerIdx < 0 {
		t.Fatal("no trailer")
	}
	tr := s[trailerIdx:]
	if !strings.Contains(tr, "/Root ") || !strings.Contains(tr, "/Info ") {
		t.Error("trailer missing /Root or /Info")
	}

	// every "N 0 obj" must be matched by "endobj"
	re := regexp.MustCompile(`(\d+) 0 obj`)
	objs := map[int][]byte{}
	offsets := map[int]int{}
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		id, _ := strconv.Atoi(s[m[2]:m[3]])
		start := m[0]
		endMark := strings.Index(s[start:], "endobj")
		if endMark < 0 {
			t.Fatalf("object %d lacks endobj", id)
		}
		offsets[id] = start
		objs[id] = []byte(s[start : start+endMark+len("endobj")])
	}

	// xref offsets must agree with "N 0 obj" positions
	xrefIdx := strings.Index(s, "xref")
	lines := strings.Split(s[xrefIdx:], "\n")
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
	for id := range objs {
		for _, rm := range refRe.FindAllStringSubmatch(string(objs[id]), -1) {
			ref, _ := strconv.Atoi(rm[1])
			if ref <= 0 {
				continue
			}
			if _, ok := objs[ref]; !ok {
				t.Errorf("object %d references missing object %d", id, ref)
			}
		}
	}
	return objs, offsets
}

// decompress streams and sanity-check dicts
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
			zr, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				t.Errorf("zlib stream header failed: %v", err)
				continue
			}
			dec, err := io.ReadAll(zr)
			zr.Close()
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
	out := buildRichDoc(t)
	objs, _ := parseObjects(t, out)
	checkStreams(t, objs)
}

func TestWriteToContract(t *testing.T) {
	d := fixedDoc(t)
	d.AddPage(100, 100)
	var buf bytes.Buffer
	n, err := d.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Errorf("WriteTo returned %d, buffer has %d", n, buf.Len())
	}
	if buf.Len() == 0 {
		t.Error("empty output")
	}
}
