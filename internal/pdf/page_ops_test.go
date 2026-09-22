package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"math"
	"testing"
)

// opsTestObject is one synthetic PDF object. Numbering is implicit: the
// slice index plus one. A non-nil stream makes it a stream object.
type opsTestObject struct {
	body     string
	stream   []byte
	compress bool
}

// buildOpsTestPDF serializes a single-page PDF with an exact xref table so
// ParsePageOps is exercised on raw operator bytes.
func buildOpsTestPDF(t *testing.T, objects []opsTestObject) []byte {
	t.Helper()

	var out bytes.Buffer

	out.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects)+1)

	for index, object := range objects {
		number := index + 1
		offsets[number] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", number)

		if object.stream == nil {
			out.WriteString(object.body)
			out.WriteString("\nendobj\n")

			continue
		}

		payload := object.stream
		dict := object.body

		if object.compress {
			var compressed bytes.Buffer

			writer := zlib.NewWriter(&compressed)
			if _, err := writer.Write(payload); err != nil {
				t.Fatalf("compress stream: %v", err)
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("close compressor: %v", err)
			}

			payload = compressed.Bytes()
			dict += " /Filter /FlateDecode"
		}

		fmt.Fprintf(&out, "%s /Length %d\nstream\n", dict, len(payload))
		out.Write(payload)
		out.WriteString("\nendstream\nendobj\n")
	}

	xrefPos := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")

	for number := 1; number <= len(objects); number++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[number])
	}

	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefPos)

	return out.Bytes()
}

// opsTestBase builds the four boilerplate objects (catalog, pages, page,
// contents) for a single-page 400x400 test document. pageResources is the
// full /Resources dictionary body.
func opsTestBase(content string, compress bool, pageResources string) []opsTestObject {
	if pageResources == "" {
		pageResources = "<< >>"
	}

	return []opsTestObject{
		{body: "<< /Type /Catalog /Pages 2 0 R >>"},
		{body: "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"},
		{
			body: "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 400] " +
				"/Contents 4 0 R /Resources " + pageResources + " >>",
		},
		{body: "<< >>", stream: []byte(content), compress: compress},
	}
}

func parseOpsTestPDF(t *testing.T, data []byte) *PageOps {
	t.Helper()

	ops, err := ParsePageOps(data)
	if err != nil {
		t.Fatalf("ParsePageOps: %v", err)
	}

	return ops
}

const opsTestTolerance = 1e-6

func opsTestClose(got, want float64) bool {
	return math.Abs(got-want) <= opsTestTolerance
}

func opsTestColorClose(got, want [3]float64) bool {
	return opsTestClose(got[0], want[0]) &&
		opsTestClose(got[1], want[1]) &&
		opsTestClose(got[2], want[2])
}

func assertOpsTestText(
	t *testing.T, ops *PageOps, text string, xPos, yPos, size float64, font string, color [3]float64,
) {
	t.Helper()

	for _, run := range ops.Texts {
		if run.Text != text {
			continue
		}

		if !opsTestClose(run.X, xPos) || !opsTestClose(run.Y, yPos) {
			t.Errorf("text %q at (%.3f, %.3f), want (%.3f, %.3f)", text, run.X, run.Y, xPos, yPos)
		}

		if !opsTestClose(run.Size, size) {
			t.Errorf("text %q size %.3f, want %.3f", text, run.Size, size)
		}

		if run.Font != font {
			t.Errorf("text %q font %q, want %q", text, run.Font, font)
		}

		if !opsTestColorClose(run.Color, color) {
			t.Errorf("text %q color %v, want %v", text, run.Color, color)
		}

		return
	}

	t.Fatalf("no %q text run found (%d runs)", text, len(ops.Texts))
}

// Plain Td plus Tj carries the fill color set before BT.
func TestPageOpsTextColor(t *testing.T) {
	t.Parallel()

	content := "0.102 0.239 0.427 rg\nBT\n50 60 Td\n(Hello) Tj\nET\n"
	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, opsTestBase(content, false, "")))

	if ops.Pages != 1 {
		t.Fatalf("Pages = %d, want 1", ops.Pages)
	}

	assertOpsTestText(t, ops, "Hello", 50, 60, 0, "", [3]float64{0.102, 0.239, 0.427})
}

// The same stream behind /Filter /FlateDecode reads identically. This is the
// shape every committed sample uses.
func TestPageOpsFlate(t *testing.T) {
	t.Parallel()

	content := "0.102 0.239 0.427 rg\nBT\n50 60 Td\n(Hello) Tj\nET\n"
	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, opsTestBase(content, true, "")))

	assertOpsTestText(t, ops, "Hello", 50, 60, 0, "", [3]float64{0.102, 0.239, 0.427})
}

// Tf size and the resolved /BaseFont land on the text run.
func TestPageOpsFontAndSize(t *testing.T) {
	t.Parallel()

	objects := opsTestBase("/F0 18 Tf\nBT\n50 60 Td\n(Hello) Tj\nET\n", false, "<< /Font << /F0 5 0 R >> >>")
	objects = append(objects, opsTestObject{
		body: "<< /Type /Font /Subtype /Type1 /BaseFont /LiberationSans-Bold >>",
	})

	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, objects))

	assertOpsTestText(t, ops, "Hello", 50, 60, 18, "LiberationSans-Bold", [3]float64{0, 0, 0})
}

// A stroked path records one segment per edge with stroke color and width.
func TestPageOpsStroke(t *testing.T) {
	t.Parallel()

	content := "0.8 0.8 0.8 RG\n0.75 w\n10 20 m\n100 20 l\nS\n"
	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, opsTestBase(content, false, "")))

	if len(ops.Strokes) != 1 {
		t.Fatalf("Strokes = %d segments, want 1", len(ops.Strokes))
	}

	segment := ops.Strokes[0]
	if !opsTestClose(segment.X1, 10) || !opsTestClose(segment.Y1, 20) ||
		!opsTestClose(segment.X2, 100) || !opsTestClose(segment.Y2, 20) {
		t.Errorf("segment = (%.3f, %.3f) -> (%.3f, %.3f), want (10, 20) -> (100, 20)",
			segment.X1, segment.Y1, segment.X2, segment.Y2)
	}

	if !opsTestColorClose(segment.Color, [3]float64{0.8, 0.8, 0.8}) {
		t.Errorf("segment color %v, want (0.8, 0.8, 0.8)", segment.Color)
	}

	if !opsTestClose(segment.Width, 0.75) {
		t.Errorf("segment width %.3f, want 0.75", segment.Width)
	}
}

// A filled rectangle records its page box and fill color.
func TestPageOpsFillRect(t *testing.T) {
	t.Parallel()

	content := "1 0 0 rg\n10 20 50 30 re\nf\n"
	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, opsTestBase(content, false, "")))

	if len(ops.Fills) != 1 {
		t.Fatalf("Fills = %d boxes, want 1", len(ops.Fills))
	}

	fill := ops.Fills[0]
	if !opsTestClose(fill.X, 10) || !opsTestClose(fill.Y, 20) ||
		!opsTestClose(fill.W, 50) || !opsTestClose(fill.H, 30) {
		t.Errorf("fill = (%.3f, %.3f) %.3fx%.3f, want (10, 20) 50x30", fill.X, fill.Y, fill.W, fill.H)
	}

	if !opsTestColorClose(fill.Color, [3]float64{1, 0, 0}) {
		t.Errorf("fill color %v, want (1, 0, 0)", fill.Color)
	}
}

// An image Do with "w 0 0 h x y cm" records that box.
func TestPageOpsImageBox(t *testing.T) {
	t.Parallel()

	objects := opsTestBase("q\n40 0 0 30 120 240 cm\n/Im0 Do\nQ\n", false, "<< /XObject << /Im0 5 0 R >> >>")
	objects = append(objects, opsTestObject{
		body:   "<< /Type /XObject /Subtype /Image /Width 4 /Height 4 /ColorSpace /DeviceRGB /BitsPerComponent 8 >>",
		stream: make([]byte, 48),
	})

	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, objects))

	if len(ops.Images) != 1 {
		t.Fatalf("Images = %d boxes, want 1", len(ops.Images))
	}

	box := ops.Images[0]
	if !opsTestClose(box.X, 120) || !opsTestClose(box.Y, 240) ||
		!opsTestClose(box.W, 40) || !opsTestClose(box.H, 30) {
		t.Errorf("image box = (%.3f, %.3f) %.3fx%.3f, want (120, 240) 40x30", box.X, box.Y, box.W, box.H)
	}
}

// Text inside a Form XObject placed with a non-zero cm reports the page
// point, form placement included, and keeps its own fill color.
func TestPageOpsFormPlacement(t *testing.T) {
	t.Parallel()

	objects := opsTestBase("q\n1 0 0 1 100 200 cm\n/Fm0 Do\nQ\n", false, "<< /XObject << /Fm0 5 0 R >> >>")
	objects = append(objects, opsTestObject{
		body:   "<< /Type /XObject /Subtype /Form /BBox [0 0 400 400] /Resources << >> >>",
		stream: []byte("0 0 1 rg\nBT\n10 20 Td\n(Inside) Tj\nET\n"),
	})

	ops := parseOpsTestPDF(t, buildOpsTestPDF(t, objects))

	assertOpsTestText(t, ops, "Inside", 110, 220, 0, "", [3]float64{0, 0, 1})
}
