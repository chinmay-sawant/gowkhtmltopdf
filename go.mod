module github.com/chinmay-sawant/gowkhtmltopdf

go 1.26

toolchain go1.26.4

require (
	github.com/chinmay-sawant/blinkless v0.0.0-00010101000000-000000000000
	github.com/go-text/typesetting v0.3.4
	github.com/tdewolff/canvas v0.0.0-20260803134256-8e86b9abb917
)

// Local sibling checkout: the blinkless layout engine is developed next to
// this repository and is not required from the module proxy. CI images must
// check out blinkless at this relative path, or drop this replace and pin a
// published version.
replace github.com/chinmay-sawant/blinkless => ../blinkless

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	codeberg.org/go-pdf/fpdf v0.11.1 // indirect
	github.com/BurntSushi/freetype-go v0.0.0-20160129220410-b763ddbfe298 // indirect
	github.com/BurntSushi/graphics-go v0.0.0-20160129215708-b43f31a4a966 // indirect
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/BurntSushi/xgbutil v0.0.0-20190907113008-ad855c713046 // indirect
	github.com/ByteArena/poly2tri-go v0.0.0-20170716161910-d102ad91854f // indirect
	github.com/benoitkugler/textlayout v0.3.2 // indirect
	github.com/benoitkugler/textprocessing v0.0.6 // indirect
	github.com/go-fonts/latin-modern v0.3.3 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/srwiley/scanx v0.0.0-20190309010443-e94503791388 // indirect
	github.com/tdewolff/font v0.0.0-20260424075104-b5eeb1e23189 // indirect
	github.com/tdewolff/minify/v2 v2.24.13 // indirect
	github.com/tdewolff/parse/v2 v2.8.12 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/image v0.41.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	modernc.org/knuth v0.5.5 // indirect
	modernc.org/token v1.1.0 // indirect
	star-tex.org/x/tex v0.7.1 // indirect
)
