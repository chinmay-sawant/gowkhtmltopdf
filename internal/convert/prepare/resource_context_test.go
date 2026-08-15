package prepare_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
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

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // the test only needs the default HTTP loader
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
