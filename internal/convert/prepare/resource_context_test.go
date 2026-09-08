package prepare_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestResourceContextFetchUsesPrivateLoadSeam(t *testing.T) {
	t.Parallel()

	const canonicalHeader = "canonical"

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Resource-Policy") != canonicalHeader {
			responseWriter.WriteHeader(http.StatusForbidden)

			return
		}

		_, _ = responseWriter.Write([]byte("canonical resource"))
	}))
	defer server.Close()

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	loadPage := settings.DefaultLoadPage()
	loadPage.CustomHeaders = map[string]string{"X-Resource-Policy": canonicalHeader}
	resources := prepare.NewResourceContext(loader, server.URL+"/root.html", loadPage)

	// These fields are retained only for source compatibility while callers
	// migrate. They must not be another source of truth for Fetch.
	resources.Base = server.URL + "/attacker.html"
	resources.Loader = nil
	resources.Load.CustomHeaders["X-Resource-Policy"] = "mutated"
	resources.Load.BlockLocalFileAccess = !resources.Load.BlockLocalFileAccess

	resource, err := resources.Fetch(t.Context(), "child.txt")
	if err != nil {
		t.Fatalf("Fetch after compatibility-snapshot mutation: %v", err)
	}

	if got := string(resource.Body); got != "canonical resource" {
		t.Fatalf("resource body = %q, want canonical resource", got)
	}
}

// TestResourceContextNilLoaderDegradedPath pins the documented degraded
// behavior of NewResourceContext(nil, ...): Fetch fails, stylesheet
// collection returns nil, and font-face merging leaves the registry
// unchanged. Document rejects nil loaders up front, so this path is only
// reachable through direct construction.
func TestResourceContextNilLoaderDegradedPath(t *testing.T) {
	t.Parallel()

	resources := prepare.NewResourceContext(nil, "https://example.test/root.html", settings.DefaultLoadPage())

	if _, err := resources.Fetch(t.Context(), "child.txt"); err == nil {
		t.Fatal("Fetch on a nil-loader context must fail")
	} else if !strings.Contains(err.Error(), "no loader") {
		t.Fatalf("Fetch error = %v, want the no-loader diagnostic", err)
	}

	sheetOpts := prepare.SheetOptions{}
	if sheets := resources.CollectSheets(t.Context(), nil, sheetOpts, io.Discard); sheets != nil {
		t.Fatalf("CollectSheets = %v, want nil on a degraded context", sheets)
	}

	registry := pdf.NewRegistry()
	if got := resources.MergeFontFaces(t.Context(), registry, nil, 1, io.Discard); got != registry {
		t.Fatal("MergeFontFaces must return the input registry unchanged on a degraded context")
	}
}
