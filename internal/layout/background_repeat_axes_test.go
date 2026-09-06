//nolint:testpackage,wsl // white-box background repeat axis coverage
package layout

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestBackgroundRepeatLonghandsKeepTheOtherAxisAtInitialRepeat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		property string
		value    string
		want     int
	}{
		{name: "block", property: "background-repeat-block", value: "repeat", want: 16},
		{name: "inline no-repeat", property: "background-repeat-inline", value: "no-repeat", want: 4},
		{name: "x", property: "background-repeat-x", value: "repeat", want: 16},
		{name: "y", property: "background-repeat-y", value: "repeat", want: 16},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sheet, err := css.Parse(fmt.Sprintf(`
.box {
  width: 40pt; height: 40pt;
  background-image: url("logo.png");
  background-size: 10pt 10pt;
  %s: %s;
}
`, testCase.property, testCase.value))
			if err != nil {
				t.Fatal(err)
			}

			root, err := html.Parse(`<html><body><div class="box"></div></body></html>`)
			if err != nil {
				t.Fatal(err)
			}

			result, err := Layout(root, Options{ //nolint:exhaustruct // image and background options under test
				Width: 200, Height: 200, Background: true,
				Sheets: []*css.Stylesheet{sheet},
				Images: func(string) ([]byte, error) {
					return tinyPNG(10, 10), nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			got := 0
			for _, op := range result.Ops {
				if op.Kind == OpImage && op.IsBackground {
					got++
				}
			}

			if got != testCase.want {
				t.Fatalf("background tile count = %d, want %d", got, testCase.want)
			}
		})
	}
}
