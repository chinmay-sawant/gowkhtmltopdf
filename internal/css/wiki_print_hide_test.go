package css

import (
	"strings"
	"testing"
)

func TestWikiPrintHideSelectorsParse(t *testing.T) {
	t.Parallel()

	rule := `#mw-navigation,.noprint,.mw-jump-link,.mw-portlet-lang,.toc .tocnumber{display:none}`
	rule2 := `.vector-page-toolbar,.vector-header-start > *:not(.mw-logo),.vector-header-end,#mw-panel-toc,#vector-sticky-header,#p-lang-btn,.vector-menu-checkbox,nav,#vector-page-titlebar-toc,#footer{display:none}`

	for _, src := range []string{rule, rule2} {
		sheet, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%s): %v", src[:40], err)
		}

		t.Logf("src rules=%d selectors=%d", len(sheet.Rules), len(sheet.Rules[0].Selectors))

		for idx, s := range sheet.Rules[0].Selectors {
			var buf strings.Builder
			for _, page := range s.Parts {
				buf.WriteString(page.Tag)

				for _, c := range page.Classes {
					buf.WriteByte('.')
					buf.WriteString(c)
				}

				if page.ID != "" {
					buf.WriteByte('#')
					buf.WriteString(page.ID)
				}

				for _, ps := range page.Pseudos {
					buf.WriteByte(':')
					buf.WriteString(ps.Name)
				}

				buf.WriteByte(' ')
			}

			t.Logf("  %d: %s", idx, buf.String())
		}
	}
}
