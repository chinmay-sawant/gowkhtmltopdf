package prepare

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const profileMediaWiki = "mediawiki"

const SimplifyChromeCSS = `
nav, footer, aside,
[role="navigation"], [role="contentinfo"], [role="complementary"],
nav.site-nav {
  display: none !important;
}
`

const SimplifyMediaWikiCSS = `
#mw-navigation,
.mw-jump-link {
  display: none !important;
}
`

func SimplifyDOMEnabled(global, object settings.Web) bool {
	return global.SimplifyDOM || object.SimplifyDOM
}

//nolint:wsl,nlreturn // profile normalization flow
func SimplifyDOMProfile(global, object settings.Web) string {
	profile := strings.ToLower(strings.TrimSpace(object.SimplifyDOMProfile))
	if profile == "" {
		profile = strings.ToLower(strings.TrimSpace(global.SimplifyDOMProfile))
	}
	if profile == profileMediaWiki || profile == "wiki" || profile == "mw" {
		return profileMediaWiki
	}
	return ""
}

//nolint:wsl,nlreturn // stylesheet append flow
func AppendSimplifySheet(sheets []*css.Stylesheet, enabled bool, profile string) []*css.Stylesheet {
	if !enabled {
		return sheets
	}

	sheet, err := css.Parse(SimplifyChromeCSS)
	if err != nil || sheet == nil {
		return sheets
	}
	sheets = append(sheets, sheet)
	if strings.EqualFold(strings.TrimSpace(profile), profileMediaWiki) {
		mw, err := css.Parse(SimplifyMediaWikiCSS)
		if err == nil && mw != nil {
			sheets = append(sheets, mw)
		}
	}
	return sheets
}
