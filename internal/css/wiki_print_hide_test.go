package css

import (
	"strings"
	"testing"
)

func TestWikiPrintHideSelectorsParse(t *testing.T) {
	rule := `#mw-navigation,.noprint,.mw-jump-link,.mw-portlet-lang,.toc .tocnumber{display:none}`
	rule2 := `.vector-page-toolbar,.vector-header-start > *:not(.mw-logo),.vector-header-end,#mw-panel-toc,#vector-sticky-header,#p-lang-btn,.vector-menu-checkbox,nav,#vector-page-titlebar-toc,#footer{display:none}`

	for _, src := range []string{rule, rule2} {
		sheet, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%s): %v", src[:40], err)
		}

		t.Logf("src rules=%d selectors=%d", len(sheet.Rules), len(sheet.Rules[0].Selectors))

		for i, s := range sheet.Rules[0].Selectors {
			var b strings.Builder
			for _, p := range s.Parts {
				b.WriteString(p.Tag)

				for _, c := range p.Classes {
					b.WriteByte('.')
					b.WriteString(c)
				}

				if p.ID != "" {
					b.WriteByte('#')
					b.WriteString(p.ID)
				}

				for _, ps := range p.Pseudos {
					b.WriteByte(':')
					b.WriteString(ps.Name)
				}

				b.WriteByte(' ')
			}

			t.Logf("  %d: %s", i, b.String())
		}
	}
}
