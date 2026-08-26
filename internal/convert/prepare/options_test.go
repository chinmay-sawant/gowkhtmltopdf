package prepare_test

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
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
