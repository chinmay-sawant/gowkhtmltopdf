//nolint:testpackage // tests reach into unexported state
package pdf

import (
	"bytes"
	"testing"
)

func TestLoadDefaultFaces(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if faces.Regular == nil || faces.Bold == nil || faces.Italic == nil || faces.BoldItalic == nil {
		t.Fatal("missing face")
	}

	if !faces.Bold.Bold() {
		t.Error("Bold face should report Bold()")
	}

	if !faces.Italic.Italic() {
		t.Error("Italic face should report Italic()")
	}

	if faces.Regular.PostScriptName != "LiberationSans" {
		t.Errorf("Regular name = %q", faces.Regular.PostScriptName)
	}
}

func TestFaceResolve(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if faces.Resolve(400, false) != faces.Regular {
		t.Error("400 regular")
	}

	if faces.Resolve(700, false) != faces.Bold {
		t.Error("700 bold")
	}

	if faces.Resolve(400, true) != faces.Italic {
		t.Error("italic")
	}

	if faces.Resolve(700, true) != faces.BoldItalic {
		t.Error("bold italic")
	}
}

func TestMultiFacePDFEmbed(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	p := doc.AddPage(200, 100)
	cur := p.Content()
	cur.UseEmbeddedFont("F0", faces.Regular)
	cur.UseEmbeddedFont("F1", faces.Bold)
	cur.SetFont("F0", 12)
	cur.BeginText()
	cur.TextAt(10, 50)
	cur.TextShow("Regular")
	cur.EndText()
	cur.SetFont("F1", 12)
	cur.BeginText()
	cur.TextAt(10, 30)
	cur.TextShow("Bold")
	cur.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.Bytes()
	if !bytes.Contains(out, []byte("/BaseFont /LiberationSans")) {
		t.Error("missing regular BaseFont")
	}

	if !bytes.Contains(out, []byte("/BaseFont /LiberationSans-Bold")) {
		t.Error("missing bold BaseFont")
	}
	// two FontFile2 streams
	if n := bytes.Count(out, []byte("/FontFile2")); n < 2 {
		t.Errorf("FontFile2 count = %d, want >= 2", n)
	}
}

func TestBoldWiderThanRegular(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	s := "Heading"

	var wrVal, wbVal float64

	for _, r := range s {
		wrVal += faces.Regular.AdvanceInPoints(r, 12)
		wbVal += faces.Bold.AdvanceInPoints(r, 12)
	}

	if wbVal <= wrVal {
		t.Errorf("bold width %v should be > regular %v", wbVal, wrVal)
	}
}
