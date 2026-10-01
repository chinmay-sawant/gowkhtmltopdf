package css

import "errors"

var (
	// ErrNilContext means Apply was called without a context.
	ErrNilContext = errors.New("css: nil context")

	// ErrNilDocument means Apply was given no HTML document.
	ErrNilDocument = errors.New("css: nil document")

	// ErrBadSize means the viewport width or height is not positive.
	ErrBadSize = errors.New("css: width and height must be positive")

	// ErrBadMedia means the media type is neither screen nor print.
	ErrBadMedia = errors.New("css: media")

	errEmptySheet = errors.New("css: empty stylesheet")
	errNilSheet   = errors.New("css: nil stylesheet")
	errUnreadable = errors.New("css: html document has no tree")
)
