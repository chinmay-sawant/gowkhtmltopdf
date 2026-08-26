//nolint:exhaustruct,wsl,testpackage,usetesting,lll // same-package tests exercise the native adapter boundary.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDocumentPDFAndImageProduceMagic(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte("<html><body><h1>hello</h1></body></html>"), "")})
	pdf, err := document.PDF(context.Background())
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", pdf[:min(len(pdf), 8)])
	}

	image, err := (&ImageDocument{Source: HTML([]byte("<html><body><h1>hello</h1></body></html>"), ""), Format: "png"}).Image(context.Background())
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	if !bytes.HasPrefix(image, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("image is not PNG")
	}
}

func TestDocumentOutputAndOutlinePreflight(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: File("report.html")})
	if err := document.WritePDF(context.Background(), nil); !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("nil output error = %v", err)
	}
	if err := document.WritePDFOutline(context.Background(), &bytes.Buffer{}, nil); !errors.Is(err, ErrMissingPDFOutlineOutput) {
		t.Fatalf("nil outline error = %v", err)
	}
	if err := document.WritePDF(context.Background(), &bytes.Buffer{}); err == nil {
		t.Fatal("expected file load error for nonexistent report")
	}
}

func TestDocumentOutlineAndContextCancellation(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte("<html><body><h1>Chapter</h1></body></html>"), "")})
	var pdfOutput bytes.Buffer
	var outlineOutput bytes.Buffer
	if err := document.WritePDFOutline(t.Context(), &pdfOutput, &outlineOutput); err != nil {
		t.Fatalf("WritePDFOutline: %v", err)
	}
	if !bytes.Contains(outlineOutput.Bytes(), []byte("<outline")) {
		t.Fatalf("outline output = %q, want outline XML", outlineOutput.Bytes())
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := document.WritePDF(ctx, &bytes.Buffer{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled WritePDF error = %v, want context.Canceled", err)
	}
}

func TestDocumentAdapterCopiesHTMLAtExecutionBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("<html><body>before</body></html>")
	document := NewDocument(Page{Source: HTML(source, "")})
	document.FontPaths = []string{"fonts-a"}
	source[15] = 'X'

	req := document.toPDFRequest(&bytes.Buffer{}, nil, false)
	document.Pages[0].Source.HTML[0] = 'Z'
	document.FontPaths[0] = "fonts-b"

	if got := string(req.Objects[0].Load.InlineHTML); got != "<html><body>before</body></html>" {
		t.Fatalf("request HTML = %q after document mutation", got)
	}
	if got := req.Global.FontPaths; len(got) != 1 || got[0] != "fonts-a" {
		t.Fatalf("request FontPaths = %v after document mutation", got)
	}

	var output bytes.Buffer
	if err := document.WritePDF(t.Context(), &output); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatalf("WritePDF prefix = %q", output.Bytes()[:min(len(output.Bytes()), 8)])
	}
}

func TestDocumentValidationErrorsReachOnError(t *testing.T) {
	t.Parallel()

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := NewDocument(Page{Source: HTML([]byte("<p>ok</p>"), "")})
		document.OnError = func(message string) { messages = append(messages, message) }

		err := document.WritePDF(nil, &bytes.Buffer{}) //nolint:staticcheck // intentional nil ctx
		if !errors.Is(err, ErrNilContext) {
			t.Fatalf("WritePDF(nil ctx) = %v, want ErrNilContext", err)
		}
		if len(messages) != 1 || !strings.Contains(messages[0], ErrNilContext.Error()) {
			t.Fatalf("OnError messages = %v", messages)
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := NewDocument(Page{Source: HTML([]byte("<p>ok</p>"), "")})
		document.OnError = func(message string) { messages = append(messages, message) }

		err := document.WritePDF(t.Context(), nil)
		if !errors.Is(err, ErrMissingPDFOutput) {
			t.Fatalf("WritePDF(nil writer) = %v, want ErrMissingPDFOutput", err)
		}
		if len(messages) != 1 {
			t.Fatalf("OnError messages = %v", messages)
		}
	})

	t.Run("validate failure", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := &Document{
			Pages:   []Page{{Source: HTML([]byte("<p>ok</p>"), "")}},
			Copies:  -1,
			OnError: func(message string) { messages = append(messages, message) },
		}

		err := document.WritePDF(t.Context(), &bytes.Buffer{})
		if !errors.Is(err, ErrInvalidPDFCopies) {
			t.Fatalf("WritePDF(invalid) = %v, want ErrInvalidPDFCopies", err)
		}
		if len(messages) != 1 {
			t.Fatalf("OnError messages = %v", messages)
		}
	})
}

func TestImageDocumentValidationErrorsReachOnError(t *testing.T) {
	t.Parallel()

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := &ImageDocument{
			Source:  HTML([]byte("<p>ok</p>"), ""),
			OnError: func(message string) { messages = append(messages, message) },
		}

		err := document.WriteImage(nil, &bytes.Buffer{}) //nolint:staticcheck // intentional nil ctx
		if !errors.Is(err, ErrNilContext) {
			t.Fatalf("WriteImage(nil ctx) = %v, want ErrNilContext", err)
		}
		if len(messages) != 1 || !strings.Contains(messages[0], ErrNilContext.Error()) {
			t.Fatalf("OnError messages = %v", messages)
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := &ImageDocument{
			Source:  HTML([]byte("<p>ok</p>"), ""),
			OnError: func(message string) { messages = append(messages, message) },
		}

		err := document.WriteImage(t.Context(), nil)
		if !errors.Is(err, ErrMissingImageOutput) {
			t.Fatalf("WriteImage(nil writer) = %v, want ErrMissingImageOutput", err)
		}
		if len(messages) != 1 {
			t.Fatalf("OnError messages = %v", messages)
		}
	})

	t.Run("validate failure", func(t *testing.T) {
		t.Parallel()

		var messages []string
		document := &ImageDocument{
			Source:  Content{},
			OnError: func(message string) { messages = append(messages, message) },
		}

		err := document.WriteImage(t.Context(), &bytes.Buffer{})
		if !errors.Is(err, ErrInvalidContent) {
			t.Fatalf("WriteImage(invalid) = %v, want ErrInvalidContent", err)
		}
		if len(messages) != 1 {
			t.Fatalf("OnError messages = %v", messages)
		}
	})
}
