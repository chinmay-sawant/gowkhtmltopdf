package prepare_test

import (
	"io"
	"math"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

type buildOptionsCase struct {
	name   string
	global settings.Web
	object settings.Web
	image  settings.Web
	want   prepare.Options
}

func TestBuildOptionsSharedAcrossPDFAndImageLayers(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct // table rows intentionally omit unused Web layers
	cases := []buildOptionsCase{
		{
			name: "defaults",
			want: baseWant(),
		},
		{
			name:   "global simplify",
			global: settings.Web{SimplifyDOM: true},
			want:   withSimplify(baseWant(), true, ""),
		},
		{
			name:   "object profile wins over global",
			global: settings.Web{SimplifyDOMProfile: "mediawiki"},
			object: settings.Web{
				SimplifyDOM: true, SimplifyDOMProfile: "wiki",
			},
			want: withSimplify(baseWant(), true, "mediawiki"),
		},
		{
			name: "image web enables simplify with empty object",
			image: settings.Web{
				SimplifyDOM: true, SimplifyDOMProfile: "mw",
			},
			want: withSimplify(baseWant(), true, "mediawiki"),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertBuildOptionsCase(t, testCase)
		})
	}
}

func baseWant() prepare.Options {
	return prepare.Options{ //nolint:exhaustruct // Simplify filled by withSimplify
		ViewportW: 100, ViewportH: 200, MediaType: "print", ObjectIndex: 1,
	}
}

func withSimplify(opts prepare.Options, enabled bool, profile string) prepare.Options {
	opts.SimplifyDOM = enabled
	opts.SimplifyProfile = profile

	return opts
}

func assertBuildOptionsCase(t *testing.T, testCase buildOptionsCase) {
	t.Helper()

	pdfOpts := prepare.BuildOptions(100, 200, "print", 1, testCase.global, testCase.object)
	imageOpts := prepare.BuildOptions(
		100, 200, "print", 1, testCase.global, testCase.image, testCase.object,
	)

	if !testCase.image.SimplifyDOM && testCase.image.SimplifyDOMProfile == "" {
		if !reflect.DeepEqual(pdfOpts, imageOpts) {
			t.Fatalf("empty image web: pdf=%+v image=%+v", pdfOpts, imageOpts)
		}

		if !reflect.DeepEqual(pdfOpts, testCase.want) {
			t.Fatalf("pdf BuildOptions = %+v, want %+v", pdfOpts, testCase.want)
		}

		return
	}

	if !reflect.DeepEqual(imageOpts, testCase.want) {
		t.Fatalf("image BuildOptions = %+v, want %+v", imageOpts, testCase.want)
	}
}

func TestBuildOptionsNormalizesMedia(t *testing.T) {
	t.Parallel()

	opts := prepare.BuildOptions(100, 200, " PRINT ", 1)
	if opts.MediaType != "print" {
		t.Fatalf("MediaType = %q, want print", opts.MediaType)
	}
}

func TestPrepareDocumentRejectsBadOptions(t *testing.T) {
	t.Parallel()

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{}) //nolint:exhaustruct // default HTTP loader
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	lineP := settings.DefaultLoadPage()
	lineP.InlineHTML = []byte(`<html><body>x</body></html>`)

	//nolint:exhaustruct // table rows exercise one bad field at a time
	cases := []struct {
		name string
		opts prepare.Options
	}{
		{name: "negative viewport", opts: prepare.Options{ViewportW: -1, ViewportH: 100, MediaType: "print"}},
		{name: "nan viewport", opts: prepare.Options{ViewportW: math.NaN(), ViewportH: 100, MediaType: "print"}},
		{name: "unknown media", opts: prepare.Options{ViewportW: 100, ViewportH: 100, MediaType: "tv"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := prepare.Document(t.Context(), loader, "ignored", lineP, nil, testCase.opts, io.Discard); err == nil {
				t.Fatal("Document accepted invalid options")
			}
		})
	}
}
