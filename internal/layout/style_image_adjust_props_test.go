//nolint:cyclop,funlen,gocognit,varnamelen // image adjustment property parsing tests
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// imageAdjustStyleFor applies one property to a fresh initial style through
// the real apply arm. The bool reports whether the group owned the property.
func imageAdjustStyleFor(t *testing.T, prop, value string) (ResolvedStyle, bool) {
	t.Helper()

	style := initialStyle()
	handled := applyImageAdjustProps(&style, prop, value, 0, nil, nil, false)

	return style, handled
}

func TestImageAdjustParsing(t *testing.T) {
	t.Parallel()

	t.Run("image-orientation", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value     string
			wantValue string
			wantAngle float64
			valid     bool
		}{
			{value: "from-image", wantValue: "from-image", valid: true},
			{value: "FROM-IMAGE", wantValue: "from-image", valid: true},
			{value: "none", wantValue: "none", valid: true},
			{value: "90deg", wantValue: "90deg", wantAngle: 90, valid: true},
			{value: "-90deg", wantValue: "270deg", wantAngle: 270, valid: true},
			{value: "100grad", wantValue: "90deg", wantAngle: 90, valid: true},
			{value: "0.5turn", wantValue: "180deg", wantAngle: 180, valid: true},
			{value: "3.141592653589793rad", wantValue: "180deg", wantAngle: 180, valid: true},
			{value: "flip", wantValue: "0deg flip", valid: true},
			{value: "90deg flip", wantValue: "90deg flip", wantAngle: 90, valid: true},
			{value: "flip 90deg", wantValue: "90deg flip", wantAngle: 90, valid: true},
			{value: "90", wantValue: "from-image"},
			{value: "from-image flip", wantValue: "from-image"},
			{value: "flip flip", wantValue: "from-image"}, //nolint:dupword // duplicate tokens must be rejected
			{value: "banana", wantValue: "from-image"},
		}

		for _, tc := range tests {
			style, handled := imageAdjustStyleFor(t, "image-orientation", tc.value)
			if !handled {
				t.Fatalf("image-orientation %q was not owned by applyImageAdjustProps", tc.value)
			}

			if style.ImageOrientation != tc.wantValue {
				t.Fatalf("image-orientation %q stored %q, want %q", tc.value, style.ImageOrientation, tc.wantValue)
			}

			if !near(style.ImageOrientationAngle, tc.wantAngle) {
				t.Fatalf("image-orientation %q angle = %v, want %v", tc.value, style.ImageOrientationAngle, tc.wantAngle)
			}
		}
	})

	t.Run("image-resolution", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value     string
			wantValue string
			wantDPI   float64
		}{
			{value: "from-image", wantValue: "from-image"},
			{value: "300dpi", wantValue: "300dpi", wantDPI: 300},
			{value: "2dppx", wantValue: "192dpi", wantDPI: 192},
			{value: "1x", wantValue: "96dpi", wantDPI: 96},
			{value: "10dpcm", wantValue: "25.4dpi", wantDPI: 25.4},
			{value: "from-image 300dpi", wantValue: "from-image 300dpi", wantDPI: 300},
			{value: "300dpi from-image", wantValue: "from-image 300dpi", wantDPI: 300},
			{value: "0dpi", wantValue: "from-image"},
			{value: "-1dpi", wantValue: "from-image"},
			{value: "1200", wantValue: "from-image"},
			{value: "banana", wantValue: "from-image"},
			{value: "from-image from-image", wantValue: "from-image"}, //nolint:dupword // duplicate tokens must be rejected
			{value: "300dpi 96dpi", wantValue: "from-image"},
		}

		for _, tc := range tests {
			style, handled := imageAdjustStyleFor(t, "image-resolution", tc.value)
			if !handled {
				t.Fatalf("image-resolution %q was not owned by applyImageAdjustProps", tc.value)
			}

			if style.ImageResolution != tc.wantValue {
				t.Fatalf("image-resolution %q stored %q, want %q", tc.value, style.ImageResolution, tc.wantValue)
			}

			if !near(style.ImageResolutionDPI, tc.wantDPI) {
				t.Fatalf("image-resolution %q dpi = %v, want %v", tc.value, style.ImageResolutionDPI, tc.wantDPI)
			}
		}
	})

	t.Run("object-view-box", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value     string
			wantValue string
		}{
			{value: "none", wantValue: "none"},
			{value: "inset(10px)", wantValue: "inset(10px)"},
			{value: "inset(1px 2px 3px 4px)", wantValue: "inset(1px 2px 3px 4px)"},
			{value: "inset( 1px  2% )", wantValue: "inset(1px 2%)"},
			{value: "INSET(10PX)", wantValue: "inset(10px)"},
			{value: "inset(10% round 5px)", wantValue: "inset(10% round 5px)"},
			{value: "inset(10% round 5px / 10px)", wantValue: "inset(10% round 5px / 10px)"},
			{value: "xywh(0 0 50% 50%)", wantValue: "xywh(0 0 50% 50%)"},
			{value: "rect(auto 10px auto 20px)", wantValue: "rect(auto 10px auto 20px)"},
			{value: "circle(50% at 50% 50%)", wantValue: "circle(50% at 50% 50%)"},
			{value: "ellipse(50% 50% at 50% 50%)", wantValue: "ellipse(50% 50% at 50% 50%)"},
			{value: "polygon(0 0, 100% 0, 50% 100%)", wantValue: "polygon(0 0, 100% 0, 50% 100%)"},
			{value: "inset()", wantValue: "none"},
			{value: "inset(10px 20px 30px 40px 50px)", wantValue: "none"},
			{value: "inset(-10px)", wantValue: "none"},
			{value: "inset(calc(10px))", wantValue: "none"},
			{value: "inset(10px round )", wantValue: "none"},
			{value: "rect(auto auto auto)", wantValue: "none"},
			{value: "xywh(0 0 5px)", wantValue: "none"},
			{value: "10px", wantValue: "none"},
			{value: "circle()", wantValue: "none"},
			{value: "frobnicate(1px)", wantValue: "none"},
		}

		for _, tc := range tests {
			style, handled := imageAdjustStyleFor(t, "object-view-box", tc.value)
			if !handled {
				t.Fatalf("object-view-box %q was not owned by applyImageAdjustProps", tc.value)
			}

			if style.ObjectViewBox != tc.wantValue {
				t.Fatalf("object-view-box %q stored %q, want %q", tc.value, style.ObjectViewBox, tc.wantValue)
			}
		}
	})

	t.Run("foreign property", func(t *testing.T) {
		t.Parallel()

		if _, handled := imageAdjustStyleFor(t, "border-left-width", "1px"); handled {
			t.Fatal("applyImageAdjustProps claimed border-left-width")
		}
	})
}

func TestImageAdjustCascadeDispatch(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="outer"><span class="probe">x</span></div></body></html>`)
	opts := Options{
		Sheets: []*css.Stylesheet{sheet(t, `
			.outer { image-orientation: 90deg flip; image-resolution: 2dppx }
			.probe { object-view-box: inset(10%) }
		`)},
		Media: "print", Width: testViewport, Height: 800,
	}
	styles := resolveStylesWith(root, opts, nil)

	outer := styleRecordsByClass(t, root, styles, "outer")[0]
	if outer.ImageOrientation != "90deg flip" || !near(outer.ImageOrientationAngle, 90) {
		t.Fatalf("outer image-orientation = %q/%v, want 90deg flip", outer.ImageOrientation, outer.ImageOrientationAngle)
	}

	if outer.ImageResolution != "192dpi" || !near(outer.ImageResolutionDPI, 192) {
		t.Fatalf("outer image-resolution = %q/%v, want 192dpi", outer.ImageResolution, outer.ImageResolutionDPI)
	}

	if outer.ObjectViewBox != "none" {
		t.Fatalf("object-view-box = %q, want none on the outer element", outer.ObjectViewBox)
	}

	probe := styleRecordsByClass(t, root, styles, "probe")[0]
	if probe.ImageOrientation != "90deg flip" || !near(probe.ImageOrientationAngle, 90) {
		t.Fatal("image-orientation did not inherit to the child")
	}

	if probe.ImageResolution != "192dpi" || !near(probe.ImageResolutionDPI, 192) {
		t.Fatal("image-resolution did not inherit to the child")
	}

	if probe.ObjectViewBox != "inset(10%)" {
		t.Fatalf("probe object-view-box = %q, want inset(10%%)", probe.ObjectViewBox)
	}
}
