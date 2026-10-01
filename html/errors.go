package html

import "errors"

// errEmpty means the parser returned no tree.
var errEmpty = errors.New("html: empty document")
