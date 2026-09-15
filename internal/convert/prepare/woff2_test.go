package prepare_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestMergeFontFacesWOFF2Registers proves the @font-face path no longer skips
// .woff2: plain, cache-busted (?v=), and data: sources decode and register the
// face. See testdata/fonts/woff2/README.md for the fixture.
func TestMergeFontFacesWOFF2Registers(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "testdata", "fonts", "woff2", "LiberationSans-Regular-latin.woff2"))
	if err != nil {
		t.Fatalf("read woff2 fixture: %v", err)
	}

	tests := []struct {
		name string
		src  string
	}{
		{"plain", `url(Custom.woff2) format("woff2")`},
		{"query", `url(Custom.woff2?v=4.7.0) format("woff2")`},
		{"data-uri", "url(data:font/woff2;base64," + base64.StdEncoding.EncodeToString(fixture) + `) format("woff2")`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/Custom.woff2" {
					http.NotFound(writer, request)

					return
				}

				_, _ = writer.Write(fixture)
			}))
			t.Cleanup(server.Close)

			loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
			if err != nil {
				t.Fatalf("new loader: %v", err)
			}

			resources := prepare.NewResourceContext(loader, server.URL+"/", settings.DefaultLoadPage())
			sheets := []*css.Stylesheet{{FontFaces: []css.FontFace{{Family: "Custom", Src: test.src}}}}

			var log bytes.Buffer

			registry := resources.MergeFontFaces(t.Context(), pdf.NewRegistry(), sheets, 1, &log)

			got := registry.Lookup([]string{"Custom"}, 400, false)
			if got == nil {
				t.Fatalf("WOFF2 @font-face did not register Custom; log=%q", log.String())
			}

			if got.PostScriptName != "Custom" {
				t.Errorf("PostScriptName = %q, want Custom", got.PostScriptName)
			}

			if strings.Contains(log.String(), "skipped") {
				t.Errorf("WOFF2 src produced a skip warning; log=%q", log.String())
			}
		})
	}
}
