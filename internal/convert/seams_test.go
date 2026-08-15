package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestRunRequiresExplicitOutputSink(t *testing.T) {
	t.Parallel()

	req := &Request{Global: settings.DefaultPdfGlobal()} //nolint:exhaustruct // intentional zero-value fields

	err := Run(t.Context(), req, io.Discard, nil)
	if !errors.Is(err, ErrMissingOutput) {
		t.Fatalf("Run error = %v, want %v", err, ErrMissingOutput)
	}
}

func TestRunValidatesRenderableObjectsBeforeContext(t *testing.T) {
	t.Parallel()

	req := &Request{ //nolint:exhaustruct // focused invalid request
		Global: settings.DefaultPdfGlobal(),
		Output: &bytes.Buffer{},
	}

	err := Run(t.Context(), req, io.Discard, nil)
	if !errors.Is(err, ErrNoRenderableObjects) {
		t.Fatalf("Run error = %v, want errors.Is(..., %v)", err, ErrNoRenderableObjects)
	}
}

func TestRunRequiresDedicatedOutlineSink(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	global.DumpOutline = true
	req := &Request{Global: global, Output: &bytes.Buffer{}} //nolint:exhaustruct // intentional zero-value fields

	err := Run(t.Context(), req, io.Discard, nil)
	if !errors.Is(err, ErrMissingOutlineOutput) {
		t.Fatalf("Run error = %v, want %v", err, ErrMissingOutlineOutput)
	}
}

type failingWriter struct{}

var errSinkFailed = errors.New("sink failed")

func (failingWriter) Write([]byte) (int, error) {
	return 0, errSinkFailed
}

func TestRunPropagatesDocumentWriterError(t *testing.T) {
	t.Parallel()

	req := NewPDFRequest(
		settings.DefaultPdfGlobal(),
		[]settings.PdfObject{{ //nolint:exhaustruct // intentional zero-value fields
			Page: "inline:<html><body><p>writer failure</p></body></html>",
		}},
		failingWriter{},
		&bytes.Buffer{},
	)

	err := Run(t.Context(), req, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "write output") {
		t.Fatalf("Run error = %v, want wrapped output-sink error", err)
	}
}

func TestModeSpecificRequestConstructors(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	//nolint:exhaustruct // intentional zero-value fields
	objects := []settings.PdfObject{
		{Page: "inline:<html><body>test</body></html>"},
	}

	pdfReq := NewPDFRequest(global, objects, &bytes.Buffer{}, &bytes.Buffer{})
	if err := pdfReq.ValidatePDF(); err != nil {
		t.Fatalf("PDF request validation: %v", err)
	}

	var out bytes.Buffer
	typedReq := &PDFRequest{
		Global:        global,
		Objects:       objects,
		Now:           nil,
		Output:        &out,
		OutlineOutput: nil,
	}

	if err := RunTypedPDF(t.Context(), typedReq, io.Discard, nil); err != nil {
		t.Fatalf("RunTypedPDF: %v", err)
	}
}

func TestPrepareDocumentBindsSharedResourceContext(t *testing.T) { //nolint:cyclop // seam binding checks many fields
	t.Parallel()

	lineP := settings.DefaultLoadPage()
	lineP.InlineHTML = []byte(`<html><head><style>body { color: #123456 }</style></head><body>hello</body></html>`)
	lineP.InlineBase = "https://example.test/reports/"
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero-value fields

	prep, err := PrepareDocument(t.Context(), loader, "ignored", lineP, nil, PrepareOptions{ //nolint:exhaustruct,lll // intentional zero-value fields
		ViewportW:   500,
		ViewportH:   700,
		MediaType:   mediaPrint,
		ObjectIndex: 1,
	}, io.Discard)
	if err != nil {
		t.Fatalf("PrepareDocument: %v", err)
	}

	if prep.Root == nil || prep.Resource == nil {
		t.Fatal("preparation did not return the document")
	}

	bound := prep.Resources.Bound()
	if bound.Loader() != loader ||
		bound.Base() != lineP.InlineBase ||
		!reflect.DeepEqual(bound.PageLoad(), lineP) {
		t.Fatalf("resource context = %+v, want loader/base/load binding", bound)
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	global := settings.LoadGlobal{} //nolint:exhaustruct // intentional zero-value fields
	loader := load.NewLoader(global)
	lp := settings.DefaultLoadPage()
	lp.LoadErrorHandling = settings.LoadErrorSkip

	prep, err := PrepareDocument(
		t.Context(), loader, srv.URL, lp, nil, PrepareOptions{}, //nolint:exhaustruct // intentional zero-value fields
		io.Discard,
	)
	if err != nil {
		t.Fatalf("PrepareDocument: %v", err)
	}

	if prep == nil || prep.Resource == nil || !prep.Resource.Skip || prep.Root != nil {
		t.Fatalf("skipped preparation = %+v, want skipped resource and nil root", prep)
	}
}
