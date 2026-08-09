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

// LiberationSansRegular returns an isolated copy of the embedded regular
// face, so callers cannot mutate the package-owned asset.
func LiberationSansRegular() []byte {
	return bytes.Clone(liberationSansRegularTTF)
}

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
