package imageout

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestResolveImageViewport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		width, height int
		want          imageViewport
	}{
		{
			name: "defaults",
			want: imageViewport{WidthPx: 1024, HeightPx: 0, WidthPt: 768, HeightPt: 768},
		},
		{
			name:  "width only",
			width: 1400,
			want:  imageViewport{WidthPx: 1400, HeightPx: 0, WidthPt: 1050, HeightPt: 1050},
		},
		{
			name:   "width and height",
			width:  1024,
			height: 900,
			want:   imageViewport{WidthPx: 1024, HeightPx: 900, WidthPt: 768, HeightPt: 675},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveImageViewport(testCase.width, testCase.height); got != testCase.want {
				t.Fatalf("resolveImageViewport(%d, %d) = %+v, want %+v",
					testCase.width, testCase.height, got, testCase.want)
			}
		})
	}
}

// TestPrepareImageDocumentLinkMediaUsesLayoutViewport checks that linked
// stylesheet media queries evaluate against the layout viewport in points:
// Width 1024 is 768pt, so (max-width: 1100px) = 825pt matches and
// (min-width: 1100px) does not. Before the resolved viewport helper, 1024 was
// passed as points and the two answers were swapped.
func TestPrepareImageDocumentLinkMediaUsesLayoutViewport(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/index.html":
			writer.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(writer, `<html><head>
<link rel="stylesheet" media="(max-width: 1100px)" href="/max-1100.css">
<link rel="stylesheet" media="(min-width: 1100px)" href="/min-1100.css">
</head><body>media</body></html>`)
		case "/max-1100.css":
			writer.Header().Set("Content-Type", "text/css")
			_, _ = io.WriteString(writer, ".from-max-1100 { color: red }")
		case "/min-1100.css":
			writer.Header().Set("Content-Type", "text/css")
			_, _ = io.WriteString(writer, ".from-min-1100 { color: blue }")
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatal(err)
	}

	obj := &settings.PdfObject{
		Page: server.URL + "/index.html",
		Load: settings.DefaultLoadPage(),
	}
	imgSet := &settings.ImageGlobal{
		Width: 1024,
	}

	prep, _, err := prepareImageDocument(t.Context(), loader, obj, settings.PdfGlobal{}, imgSet, nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	classes := sheetClasses(prep.Sheets)
	if !classes["from-max-1100"] {
		t.Errorf("(max-width: 1100px) link was skipped; classes = %v", classes)
	}

	if classes["from-min-1100"] {
		t.Errorf("(min-width: 1100px) link was collected; classes = %v", classes)
	}
}

// sheetClasses flattens the class names in every selector of sheets.
func sheetClasses(sheets []*css.Stylesheet) map[string]bool {
	out := map[string]bool{}

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, selector := range rule.Selectors {
				for _, part := range selector.Parts {
					for _, class := range part.Classes {
						out[class] = true
					}
				}
			}
		}
	}

	return out
}
