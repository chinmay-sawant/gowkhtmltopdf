package pdf

import (
	"bytes"
	"testing"
)

func TestLoadDefaultFaces(t *testing.T) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if fs.Regular == nil || fs.Bold == nil || fs.Italic == nil || fs.BoldItalic == nil {
		t.Fatal("missing face")
	}

	if !fs.Bold.Bold() {
		t.Error("Bold face should report Bold()")
	}

	if !fs.Italic.Italic() {
		t.Error("Italic face should report Italic()")
	}

	if fs.Regular.PostScriptName != "LiberationSans" {
		t.Errorf("Regular name = %q", fs.Regular.PostScriptName)
	}
}

func TestFaceResolve(t *testing.T) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if fs.Resolve(400, false) != fs.Regular {
		t.Error("400 regular")
	}

	if fs.Resolve(700, false) != fs.Bold {
		t.Error("700 bold")
	}

	if fs.Resolve(400, true) != fs.Italic {
		t.Error("italic")
	}

	if fs.Resolve(700, true) != fs.BoldItalic {
		t.Error("bold italic")
	}
}

func TestMultiFacePDFEmbed(t *testing.T) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	d := NewDocument()
	p := d.AddPage(200, 100)
	c := p.Content()
	c.UseEmbeddedFont("F0", fs.Regular)
	c.UseEmbeddedFont("F1", fs.Bold)
	c.SetFont("F0", 12)
	c.BeginText()
	c.TextAt(10, 50)
	c.TextShow("Regular")
	c.EndText()
	c.SetFont("F1", 12)
	c.BeginText()
	c.TextAt(10, 30)
	c.TextShow("Bold")
	c.EndText()

	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
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
	fs, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	s := "Heading"

	var wr, wb float64

	for _, r := range s {
		wr += fs.Regular.AdvanceInPoints(r, 12)
		wb += fs.Bold.AdvanceInPoints(r, 12)
	}

	if wb <= wr {
		t.Errorf("bold width %v should be > regular %v", wb, wr)
	}
}
