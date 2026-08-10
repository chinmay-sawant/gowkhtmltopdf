// Liberation Sans is (c) Red Hat / Ascender Corp., SIL Open Font License 1.1.
// These copies ship a dependable Latin family (regular/bold/italic/bold-italic)
// without requiring system fonts at runtime.
package assets

import (
	"bytes"
	_ "embed"
)

//go:embed LiberationSans-Regular.ttf
var liberationSansRegularTTF []byte

//go:embed LiberationSans-Bold.ttf
var liberationSansBoldTTF []byte

//go:embed LiberationSans-Italic.ttf
var liberationSansItalicTTF []byte

//go:embed LiberationSans-BoldItalic.ttf
var liberationSansBoldItalicTTF []byte

//go:embed DejaVuSans-Regular.ttf
var unicodeFallbackRegularTTF []byte

//go:embed DejaVuSans-Bold.ttf
var unicodeFallbackBoldTTF []byte

// LiberationSansRegular returns an isolated copy of the embedded regular
// face, so callers cannot mutate the package-owned asset.
func LiberationSansRegular() []byte {
	return bytes.Clone(liberationSansRegularTTF)
}

//go:embed LiberationSerif-Regular.ttf
var liberationSerifRegularTTF []byte

//go:embed LiberationSerif-Bold.ttf
var liberationSerifBoldTTF []byte

//go:embed LiberationSerif-Italic.ttf
var liberationSerifItalicTTF []byte

//go:embed LiberationSerif-BoldItalic.ttf
var liberationSerifBoldItalicTTF []byte

//go:embed LiberationMono-Regular.ttf
var liberationMonoRegularTTF []byte

//go:embed LiberationMono-Bold.ttf
var liberationMonoBoldTTF []byte

//go:embed LiberationMono-Italic.ttf
var liberationMonoItalicTTF []byte

//go:embed LiberationMono-BoldItalic.ttf
var liberationMonoBoldItalicTTF []byte

func LiberationSerifRegular() []byte { return bytes.Clone(liberationSerifRegularTTF) }

func LiberationSerifBold() []byte { return bytes.Clone(liberationSerifBoldTTF) }

func LiberationSerifItalic() []byte { return bytes.Clone(liberationSerifItalicTTF) }

func LiberationSerifBoldItalic() []byte { return bytes.Clone(liberationSerifBoldItalicTTF) }

func LiberationMonoRegular() []byte { return bytes.Clone(liberationMonoRegularTTF) }

func LiberationMonoBold() []byte { return bytes.Clone(liberationMonoBoldTTF) }

func LiberationMonoItalic() []byte { return bytes.Clone(liberationMonoItalicTTF) }

func LiberationMonoBoldItalic() []byte { return bytes.Clone(liberationMonoBoldItalicTTF) }

// LiberationSansBold returns an immutable copy of the embedded bold face.
func LiberationSansBold() []byte {
	return bytes.Clone(liberationSansBoldTTF)
}

// LiberationSansItalic returns an immutable copy of the embedded italic face.
func LiberationSansItalic() []byte {
	return bytes.Clone(liberationSansItalicTTF)
}

// LiberationSansBoldItalic returns an immutable copy of the embedded bold
// italic face.
func LiberationSansBoldItalic() []byte {
	return bytes.Clone(liberationSansBoldItalicTTF)
}

// UnicodeFallbackRegular returns the embedded Unicode fallback face.
func UnicodeFallbackRegular() []byte { return bytes.Clone(unicodeFallbackRegularTTF) }

// UnicodeFallbackBold returns the embedded bold Unicode fallback face.
func UnicodeFallbackBold() []byte { return bytes.Clone(unicodeFallbackBoldTTF) }
