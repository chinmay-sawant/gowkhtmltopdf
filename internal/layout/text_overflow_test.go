package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// layoutWithWarnings runs Layout with a captured warning sink.
func layoutWithWarnings(t *testing.T, src, cssSrc string, width float64) (*Result, []string) {
	t.Helper()

	var warnings []string

	res, err := Layout(mustParse(t, src), Options{
		Width: width, Height: 200,
		Sheets:     []*css.Stylesheet{sheet(t, cssSrc)},
		Background: true,
		Warnf: func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res, warnings
}

// TestTextOverflowWarnsWithoutChangingLayout covers the measured learncpp
// defect: prose painted 7.8-14.1pt into the right margin produced no warning
// because the smart-shrink census only sees fill/stroke/image ops. The fix
// reports the overflow and leaves the layout untouched (text is excluded from
// MaxContentX, so smart shrink never rescales the page for prose).
func TestTextOverflowWarnsWithoutChangingLayout(t *testing.T) {
	t.Parallel()

	res, warnings := layoutWithWarnings(t,
		`<html><body><div class="clip"><span class="now">`+
			`https://www.learncpp.com/cpp-tutorial/a-very-long-unbreakable-token/</span></div></body></html>`,
		`.clip { width:60pt } .now { white-space:nowrap; font-size:12pt }`, 100)

	textRight := 0.0

	for _, op := range res.Ops {
		if op.Kind == OpText && op.X+op.W > textRight {
			textRight = op.X + op.W
		}
	}

	if textRight <= 100 {
		t.Fatalf("probe text right edge = %.2fpt, want an overflow past the 100pt content box", textRight)
	}

	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want exactly one text-overflow warning", warnings)
	}

	if !strings.Contains(warnings[0], "text overflows content box") {
		t.Fatalf("warning does not name the overflow: %q", warnings[0])
	}

	// Layout unchanged: text must not raise the shrink census (MaxContentX).
	if res.MaxContentX > 100+0.01 {
		t.Fatalf("MaxContentX = %.2f, want <= 100: text ops must not trigger smart shrink", res.MaxContentX)
	}
}

// TestTextFitsNoOverflowWarning is the control: no warning when every text op
// stays inside the content box.
func TestTextFitsNoOverflowWarning(t *testing.T) {
	t.Parallel()

	_, warnings := layoutWithWarnings(t,
		`<html><body><p class="fit">short prose that wraps inside the box</p></body></html>`,
		`.fit { width:80pt }`, 100)

	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none for fitting text", warnings)
	}
}
