package convert

import (
	"bytes"
	"testing"
)

// Phase 7 metadata contract: the document <title> becomes the PDF /Title when
// the caller did not override it, and a declared html lang becomes /Lang.
// /ID and XMP stay version-driven: 1.4 output is unchanged, 1.7 gains both.

func TestDocumentTitleFallsBackToHTMLTitle(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><head><title>Learn C++</title></head><body><p>body</p></body></html>`, "")
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("/Title (Learn C++)")) {
		t.Fatalf("output missing /Title from <title>\n%s", data)
	}
}

func TestExplicitTitleOverridesHTMLTitle(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><head><title>Document Title</title></head><body><p>body</p></body></html>`, "")
	cmd.Global.Title = "Invoice 42"
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("/Title (Invoice 42)")) {
		t.Fatalf("output missing explicit /Title (Invoice 42)")
	}

	if bytes.Contains(data, []byte("/Title (Document Title)")) {
		t.Fatalf("explicit title did not win over the document <title>")
	}
}

func TestDocumentLangSetsCatalogLang(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html lang="en-AU"><head><title>Lang</title></head><body><p>x</p></body></html>`, "")
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("/Lang (en-AU)")) {
		t.Fatalf("output missing /Lang (en-AU)")
	}
}

func TestUndeclaredLangLeavesCatalogLangUnset(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><head><title>No lang</title></head><body><p>x</p></body></html>`, "")
	data := runPDF(t, cmd)

	if bytes.Contains(data, []byte("/Lang")) {
		t.Fatalf("output fabricated /Lang without a declared language")
	}
}

func TestTrailerIDAndXMPFollowVersionPolicy(t *testing.T) {
	t.Parallel()

	const html = `<html><head><title>Policy</title></head><body><p>policy</p></body></html>`

	cmd14, _ := newCommand(t, html, "")
	data14 := runPDF(t, cmd14)

	if bytes.Contains(data14, []byte("/ID [")) {
		t.Fatalf("PDF 1.4 trailer grew an /ID entry")
	}

	if bytes.Contains(data14, []byte("/Type /Metadata")) {
		t.Fatalf("PDF 1.4 output grew an XMP metadata stream")
	}

	cmd17, _ := newCommand(t, html, "")
	cmd17.Global.PdfVersion = pdfVersion17
	data17 := runPDF(t, cmd17)

	if !bytes.Contains(data17, []byte("/ID [")) {
		t.Fatalf("PDF 1.7 trailer missing /ID")
	}

	if !bytes.Contains(data17, []byte("/Type /Metadata /Subtype /XML")) {
		t.Fatalf("PDF 1.7 output missing XMP metadata stream")
	}

	if !bytes.Contains(data17, []byte("/Title (Policy)")) {
		t.Fatalf("PDF 1.7 output missing the document title")
	}
}
