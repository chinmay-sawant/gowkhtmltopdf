package convert

import (
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/settings"
)

// SimplifyChromeCSS is the default opt-in chrome-strip stylesheet for
// --simplify-dom. It hides common landmark chrome only — no site-specific
// IDs/classes. MediaWiki extras live in SimplifyMediaWikiCSS (profile).
//
// Approach: append display:none rules so chrome nodes stay in the DOM but
// are excluded from layout. No extra origins are fetched.
const SimplifyChromeCSS = `
nav, footer, aside,
[role="navigation"], [role="contentinfo"], [role="complementary"],
nav.site-nav {
  display: none !important;
}
`

// SimplifyMediaWikiCSS adds MediaWiki Vector/Minerva chrome selectors on top
// of the landmark sheet when --simplify-dom-profile=mediawiki.
const SimplifyMediaWikiCSS = `
#mw-navigation,
.mw-jump-link {
  display: none !important;
}
`

// SimplifyDOMEnabled reports whether chrome-strip heuristics apply for this
// page (global or object web.simplifydom). Default is off.
func SimplifyDOMEnabled(global, obj settings.Web) bool {
	return global.SimplifyDOM || obj.SimplifyDOM
}

// SimplifyDOMProfile returns the effective simplify profile ("" or "mediawiki").
// Object setting wins when non-empty; otherwise global.
func SimplifyDOMProfile(global, obj settings.Web) string {
	p := strings.ToLower(strings.TrimSpace(obj.SimplifyDOMProfile))
	if p == "" {
		p = strings.ToLower(strings.TrimSpace(global.SimplifyDOMProfile))
	}

	switch p {
	case "mediawiki", "wiki", "mw":
		return "mediawiki"
	default:
		return ""
	}
}

// AppendSimplifySheet appends chrome-strip stylesheet(s) when enabled.
// profile "mediawiki" also appends MediaWiki-specific hide rules.
func AppendSimplifySheet(sheets []*css.Stylesheet, enabled bool, profile string) []*css.Stylesheet {
	if !enabled {
		return sheets
	}

	sheet, err := css.Parse(SimplifyChromeCSS)
	if err != nil || sheet == nil {
		return sheets
	}

	sheets = append(sheets, sheet)

	if strings.EqualFold(strings.TrimSpace(profile), "mediawiki") {
		mw, err := css.Parse(SimplifyMediaWikiCSS)
		if err == nil && mw != nil {
			sheets = append(sheets, mw)
		}
	}

	return sheets
}
