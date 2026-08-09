package convert

import (
	"gowkhtmltopdf/internal/convert/prepare"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/settings"
)

const profileMediaWiki = "mediawiki"

const (
	SimplifyChromeCSS    = prepare.SimplifyChromeCSS
	SimplifyMediaWikiCSS = prepare.SimplifyMediaWikiCSS
)

func SimplifyDOMEnabled(global, object settings.Web) bool {
	return prepare.SimplifyDOMEnabled(global, object)
}

func SimplifyDOMProfile(global, object settings.Web) string {
	return prepare.SimplifyDOMProfile(global, object)
}

func AppendSimplifySheet(sheets []*css.Stylesheet, enabled bool, profile string) []*css.Stylesheet {
	return prepare.AppendSimplifySheet(sheets, enabled, profile)
}
