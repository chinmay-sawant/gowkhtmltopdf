package layout

import "errors"

var (
	// ErrNilContext means Lay was called without a context.
	ErrNilContext = errors.New("layout: nil context")

	// ErrNilDocument means Lay was given no styled document.
	ErrNilDocument = errors.New("layout: nil document")

	errNoImage = errors.New("layout: no image")
)
