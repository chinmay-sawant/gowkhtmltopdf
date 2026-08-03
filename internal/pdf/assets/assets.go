// Liberation Sans is (c) Red Hat / Ascender Corp., SIL Open Font License 1.1.
// These copies ship a dependable Latin family (regular/bold/italic/bold-italic)
// without requiring system fonts at runtime.
package assets

import _ "embed"

//go:embed LiberationSans-Regular.ttf
var LiberationSansRegularTTF []byte

//go:embed LiberationSans-Bold.ttf
var LiberationSansBoldTTF []byte

//go:embed LiberationSans-Italic.ttf
var LiberationSansItalicTTF []byte

//go:embed LiberationSans-BoldItalic.ttf
var LiberationSansBoldItalicTTF []byte
