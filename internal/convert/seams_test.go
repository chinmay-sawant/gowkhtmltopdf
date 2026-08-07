package convert

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/settings"
)

func TestRunRequiresExplicitOutputSink(t *testing.T) {
	req := &Request{Global: settings.DefaultPdfGlobal()}

	err := Run(t.Context(), req, io.Discard, nil)
	if !errors.Is(err, ErrMissingOutput) {
		t.Fatalf("Run error = %v, want %v", err, ErrMissingOutput)
	}
}

func TestRunRequiresDedicatedOutlineSink(t *testing.T) {
	global := settings.DefaultPdfGlobal()
	global.DumpOutline = true
	req := &Request{Global: global, Output: &bytes.Buffer{}}

	err := Run(t.Context(), req, io.Discard, nil)
	if !errors.Is(err, ErrMissingOutlineOutput) {
		t.Fatalf("Run error = %v, want %v", err, ErrMissingOutlineOutput)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("sink failed")
}

func TestRunPropagatesDocumentWriterError(t *testing.T) {
	req := NewPDFRequest(settings.DefaultPdfGlobal(), []settings.PdfObject{{
		Page: "inline:<html><body><p>writer failure</p></body></html>",
	}}, failingWriter{}, &bytes.Buffer{})

	err := Run(t.Context(), req, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "write output") {
		t.Fatalf("Run error = %v, want wrapped output-sink error", err)
	}
}

func TestModeSpecificRequestConstructors(t *testing.T) {
	global := settings.DefaultPdfGlobal()
	objects := []settings.PdfObject{{Page: "inline:<html></html>"}}

	pdfReq := NewPDFRequest(global, objects, &bytes.Buffer{}, &bytes.Buffer{})
	if err := pdfReq.ValidatePDF(); err != nil {
		t.Fatalf("PDF request validation: %v", err)
	}

	if err := pdfReq.ValidateImage(); !errors.Is(err, ErrMissingImageSettings) {
		t.Fatalf("PDF request as image = %v, want %v", err, ErrMissingImageSettings)
	}

	imageReq := NewImageRequest(global, settings.DefaultImageGlobal(), objects, &bytes.Buffer{})
	if err := imageReq.ValidateImage(); err != nil {
		t.Fatalf("image request validation: %v", err)
	}

	if err := imageReq.ValidatePDF(); !errors.Is(err, ErrUnexpectedImageSettings) {
		t.Fatalf("image request as PDF = %v, want %v", err, ErrUnexpectedImageSettings)
	}
}

func TestPrepareDocumentBindsSharedResourceContext(t *testing.T) {
	lp := settings.DefaultLoadPage()
	lp.InlineHTML = []byte(`<html><head><style>body { color: #123456 }</style></head><body>hello</body></html>`)
	lp.InlineBase = "https://example.test/reports/"
	loader := load.NewLoader(settings.LoadGlobal{})

	prep, err := PrepareDocument(t.Context(), loader, "ignored", lp, nil, PrepareOptions{
		ViewportW:   500,
		ViewportH:   700,
		MediaType:   "print",
		ObjectIndex: 1,
	}, io.Discard)
	if err != nil {
		t.Fatalf("PrepareDocument: %v", err)
	}

	if prep.Root == nil || prep.Resource == nil {
		t.Fatal("preparation did not return the document")
	}

	if prep.Resources.Loader != loader || prep.Resources.Base != lp.InlineBase || !reflect.DeepEqual(prep.Resources.Load, lp) {
		t.Fatalf("resource context = %+v, want loader/base/load binding", prep.Resources)
	}

	if len(prep.Sheets) != 1 || len(prep.Sheets[0].Rules) == 0 {
		t.Fatalf("sheets = %+v, want one parsed stylesheet", prep.Sheets)
	}

	res, err := prep.Resources.Fetch(t.Context(), "data:text/plain,hello")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if string(res.Body) != "hello" || !strings.HasPrefix(res.URL, "data:") {
		t.Fatalf("fetched resource = %+v", res)
	}
}

func TestPrepareDocumentPreservesSkipForCallerPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	global := settings.LoadGlobal{}
	loader := load.NewLoader(global)
	lp := settings.DefaultLoadPage()
	lp.LoadErrorHandling = settings.LoadErrorSkip

	prep, err := PrepareDocument(t.Context(), loader, srv.URL, lp, nil, PrepareOptions{}, io.Discard)
	if err != nil {
		t.Fatalf("PrepareDocument: %v", err)
	}

	if prep == nil || prep.Resource == nil || !prep.Resource.Skip || prep.Root != nil {
		t.Fatalf("skipped preparation = %+v, want skipped resource and nil root", prep)
	}
}
