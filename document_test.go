//nolint:exhaustruct,funlen,wsl,testpackage,copyloopvar
package gowkhtmltopdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestContentValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content Content
		want    []error
	}{
		{name: "none", content: Content{}, want: []error{ErrInvalidContent}},
		{name: "empty html", content: Content{HTML: []byte{}}, want: []error{ErrInvalidContent, ErrEmptyHTML}},
		{name: "html", content: HTML([]byte("<p>hello</p>"), ""), want: nil},
		{name: "html with base", content: HTML([]byte("<img src=\"asset.png\">"), "https://example.test/report/"), want: nil},
		{name: "file", content: File("/tmp/report.html"), want: nil},
		{name: "url", content: URL("https://example.test/report"), want: nil},
		{
			name:    "html and file",
			content: Content{HTML: []byte("<p>hello</p>"), File: "/tmp/report.html"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "html and url",
			content: Content{HTML: []byte("<p>hello</p>"), URL: "https://example.test/report"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "file and url",
			content: Content{File: "/tmp/report.html", URL: "https://example.test/report"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "base without source",
			content: Content{Base: "https://example.test/"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "base with file",
			content: Content{Base: "https://example.test/", File: "/tmp/report.html"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "base with url",
			content: Content{Base: "https://example.test/", URL: "https://example.test/report"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "blank file is not a source",
			content: Content{File: " \t"},
			want:    []error{ErrInvalidContent},
		},
		{
			name:    "blank url is not a source",
			content: Content{URL: " \t"},
			want:    []error{ErrInvalidContent},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.content.Validate()
			if len(testCase.want) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Validate() error = nil, want %v", testCase.want)
			}

			for _, want := range testCase.want {
				if !errors.Is(err, want) {
					t.Errorf("Validate() error = %v, want errors.Is(..., %v)", err, want)
				}
			}
		})
	}
}

func TestHTMLCopiesBytes(t *testing.T) {
	t.Parallel()

	source := []byte("<p>before</p>")
	content := HTML(source, "https://example.test/")
	source[0] = 'X'

	if got, want := string(content.HTML), "<p>before</p>"; got != want {
		t.Fatalf("HTML() stored %q, want %q", got, want)
	}

	content.HTML[3] = 'Y'
	if got, want := string(source), "Xp>before</p>"; got != want {
		t.Fatalf("mutating Content.HTML changed source: got %q, want %q", got, want)
	}
}

func TestNewDocumentCopiesPages(t *testing.T) {
	t.Parallel()

	pages := []Page{{Source: File("before.html")}}
	document := NewDocument(pages...)
	pages[0].Source = URL("https://example.test/after")

	if got, want := document.Pages[0].Source.File, "before.html"; got != want {
		t.Fatalf("NewDocument() page source = %q, want %q", got, want)
	}
	if document.Pages == nil {
		t.Fatal("NewDocument() returned nil Pages")
	}
}

func TestDocumentValidate(t *testing.T) {
	t.Parallel()

	validPage := Page{Source: HTML([]byte("<p>page</p>"), "")}
	validCover := Page{Source: File("cover.html")}

	tests := []struct {
		name       string
		document   *Document
		want       []error
		wantInText string
	}{
		{
			name:     "nil",
			document: nil,
			want:     []error{ErrNilDocument},
		},
		{
			name:     "empty",
			document: &Document{},
			want:     []error{ErrNoRenderablePDFObjects},
		},
		{
			name:     "toc only",
			document: &Document{TOC: &TOC{Caption: "Contents"}},
			want:     []error{ErrNoRenderablePDFObjects},
		},
		{
			name:     "cover only",
			document: &Document{Cover: &validCover},
		},
		{
			name:     "pages only",
			document: &Document{Pages: []Page{validPage}},
		},
		{
			name:     "cover toc pages",
			document: &Document{Cover: &validCover, TOC: &TOC{}, Pages: []Page{validPage}},
		},
		{
			name:       "invalid cover",
			document:   &Document{Cover: &Page{}},
			want:       []error{ErrInvalidContent},
			wantInText: "cover:",
		},
		{
			name:       "invalid page",
			document:   &Document{Pages: []Page{validPage, {}}},
			want:       []error{ErrInvalidContent},
			wantInText: "pages[1]:",
		},
		{
			name:     "negative copies",
			document: &Document{Pages: []Page{validPage}, Copies: -1},
			want:     []error{ErrInvalidPDFCopies},
		},
		{
			name:     "zero copies uses default",
			document: &Document{Pages: []Page{validPage}, Copies: 0},
		},
		{
			name:     "invalid page size",
			document: &Document{Pages: []Page{validPage}, PageSize: "A4X"},
			want:     []error{ErrInvalidPageSize},
		},
		{
			name:     "invalid orientation",
			document: &Document{Pages: []Page{validPage}, Orientation: "diagonal"},
			want:     []error{ErrInvalidOrientation},
		},
		{
			name:     "invalid pdf version",
			document: &Document{Pages: []Page{validPage}, PDFVersion: "9.9"},
			want:     []error{ErrInvalidPDFVersion},
		},
		{
			name:     "invalid pdf profile",
			document: &Document{Pages: []Page{validPage}, PDFProfile: "not-a-profile"},
			want:     []error{ErrInvalidPDFProfile},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.document.Validate()
			if len(testCase.want) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Validate() error = nil, want %v", testCase.want)
			}

			for _, want := range testCase.want {
				if !errors.Is(err, want) {
					t.Errorf("Validate() error = %v, want errors.Is(..., %v)", err, want)
				}
			}
			if testCase.wantInText != "" && !strings.Contains(err.Error(), testCase.wantInText) {
				t.Errorf("Validate() error = %q, want path %q", err, testCase.wantInText)
			}
		})
	}
}

func TestImageDocumentValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		document *ImageDocument
		want     []error
	}{
		{name: "nil", document: nil, want: []error{ErrNilImageDocument}},
		{name: "empty", document: &ImageDocument{}, want: []error{ErrInvalidContent}},
		{name: "html", document: &ImageDocument{Source: HTML([]byte("<p>image</p>"), "")}},
		{name: "file", document: &ImageDocument{Source: File("image.html")}},
		{name: "url", document: &ImageDocument{Source: URL("https://example.test/image")}},
		{
			name: "multiple sources",
			document: &ImageDocument{
				Source: Content{HTML: []byte("<p>image</p>"), URL: "https://example.test/image"},
			},
			want: []error{ErrInvalidContent},
		},
		{
			name:     "invalid format",
			document: &ImageDocument{Source: File("image.html"), Format: "webp"},
			want:     []error{ErrInvalidImageFormat},
		},
		{name: "jpeg format", document: &ImageDocument{Source: File("image.html"), Format: "JPEG"}},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.document.Validate()
			if len(testCase.want) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Validate() error = nil, want %v", testCase.want)
			}

			for _, want := range testCase.want {
				if !errors.Is(err, want) {
					t.Errorf("Validate() error = %v, want errors.Is(..., %v)", err, want)
				}
			}
		})
	}
}

func TestPublicDocumentOptionsAreRepresentable(t *testing.T) {
	t.Parallel()

	trueValue := true
	document := Document{
		PageSize:        "Letter",
		WidthMM:         210,
		HeightMM:        297,
		Orientation:     "landscape",
		Margin:          Margin{Top: 1, Right: 2, Bottom: 3, Left: 4},
		Title:           "Report",
		PDFVersion:      "1.7",
		PDFProfile:      "a3a",
		Copies:          2,
		Collate:         true,
		Outline:         &trueValue,
		OutlineDepth:    4,
		Background:      &trueValue,
		SmartShrinking:  &trueValue,
		Compression:     &trueValue,
		ResolveRelLinks: &trueValue,
		Header:          &HeaderFooter{Left: "[page]", FontName: "Arial"},
		Footer:          &HeaderFooter{Right: "[topage]"},
		AllowLocalFiles: true,
		FontPaths:       []string{"fonts"},
		UseSystemFonts:  true,
		Network:         &NetworkPolicy{AllowedSchemes: []string{"https"}},
		Pages: []Page{{
			Source:           HTML([]byte("<p>report</p>"), ""),
			IncludeInOutline: &trueValue,
			ExternalLinks:    &trueValue,
			LocalLinks:       &trueValue,
		}},
	}

	if err := document.Validate(); err != nil {
		t.Fatalf("Document.Validate() error = %v, want nil", err)
	}

	imageDocument := ImageDocument{
		Source:          File("image.html"),
		Width:           800,
		Height:          600,
		Format:          "png",
		Quality:         90,
		SmartWidth:      &trueValue,
		Transparent:     true,
		Crop:            &Crop{Left: 1, Top: 2, Width: 300, Height: 200},
		AllowLocalFiles: true,
		Network:         &NetworkPolicy{AllowedSchemes: []string{"https"}},
		Background:      &trueValue,
	}

	if err := imageDocument.Validate(); err != nil {
		t.Fatalf("ImageDocument.Validate() error = %v, want nil", err)
	}

	if !bytes.Equal(document.Pages[0].Source.HTML, []byte("<p>report</p>")) {
		t.Fatal("document page HTML was not retained")
	}
}
