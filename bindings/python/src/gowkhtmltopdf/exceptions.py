"""Exception hierarchy and status-code mapping for gowkhtmltopdf.

The shared library reports failures as an integer status code plus a
message string. ``error_from_status`` turns that pair into a typed Python
exception. Status 1 (invalid argument) additionally sniffs the message for
known engine sentinel phrases and records the matching sentinel class on
the exception's ``sentinel`` attribute, so callers can compare identity
(``exc.sentinel is ErrInvalidPageSize``) instead of parsing message text.
"""

from typing import Optional


class GowkhtmltopdfError(Exception):
    """Base class for every error this package raises."""


class ConversionError(GowkhtmltopdfError):
    """A conversion call failed inside the shared library.

    Attributes:
        code: Integer status code from the C ABI (see the committed header
            ``bindings/c/include/gowkhtmltopdf.h`` for the table).
        message: Human readable diagnostic reported by the engine.
    """

    def __init__(self, code, message):
        # type: (int, str) -> None
        super(GowkhtmltopdfError, self).__init__(
            "status {0}: {1}".format(code, message)
        )
        self.code = code
        self.message = message


class InvalidArgumentError(ConversionError):
    """The engine rejected an argument value (status 1).

    Attributes:
        sentinel: One of the Err* sentinel classes below when the message
            matched a known phrase, else ``None``.
    """

    def __init__(self, code, message, sentinel=None):
        # type: (int, str, Optional[type]) -> None
        super(InvalidArgumentError, self).__init__(code, message)
        self.sentinel = sentinel


class LoadDeniedError(ConversionError):
    """A local-file ACL or network-policy rule denied a resource (status 2)."""


class RenderError(ConversionError):
    """Layout, paint, pagination, or PDF encoding failed (status 3)."""


class ConversionTimeoutError(ConversionError):
    """The caller deadline elapsed before conversion finished (status 4)."""


class ResourceLimitError(ConversionError):
    """An engine ceiling was exceeded, such as the copies limit (status 5)."""


class InternalEngineError(ConversionError):
    """Unexpected internal failure inside the engine (status 6)."""


# Sentinel classes. They mirror the stable errors.Is targets of the Go
# Document API (api.go, document_validate.go). They are real exception
# classes so model-level validation can raise them directly, and the
# status-code mapper attaches the class itself to ``exc.sentinel``.


class ErrEmptyContent(GowkhtmltopdfError):
    """The document supplied empty HTML content."""


class ErrInvalidContent(GowkhtmltopdfError):
    """The content source shape is invalid.

    Zero or multiple sources were supplied, base accompanied a non-HTML
    source, or the URL was not absolute HTTP(S).
    """


class ErrNoPageObjects(GowkhtmltopdfError):
    """The document defines neither pages nor a cover to render."""


class ErrInvalidPageSize(GowkhtmltopdfError):
    """The page size name is not one the engine knows."""


class ErrInvalidOrientation(GowkhtmltopdfError):
    """The orientation is neither portrait nor landscape."""


class ErrInvalidPDFVersion(GowkhtmltopdfError):
    """The requested PDF version string is unsupported."""


class ErrInvalidPDFProfile(GowkhtmltopdfError):
    """The requested PDF conformance profile is unknown."""


class ErrMissingOutput(GowkhtmltopdfError):
    """The conversion finished without producing output bytes."""


# Substrings searched case-insensitively inside status-1 messages. Order
# matters: specific phrases must be checked before generic ones ("output").
_SENTINEL_SUBSTRINGS = (
    ("empty html", ErrEmptyContent),
    ("exactly one", ErrInvalidContent),
    ("page objects", ErrNoPageObjects),
    ("renderable", ErrNoPageObjects),
    ("page size", ErrInvalidPageSize),
    ("orientation", ErrInvalidOrientation),
    ("pdf version", ErrInvalidPDFVersion),
    ("profile", ErrInvalidPDFProfile),
    ("output", ErrMissingOutput),
)

# Status code -> exception class, mirroring the table documented in
# bindings/c/include/gowkhtmltopdf.h.
_STATUS_CLASSES = {
    1: InvalidArgumentError,
    2: LoadDeniedError,
    3: RenderError,
    4: ConversionTimeoutError,
    5: ResourceLimitError,
    6: InternalEngineError,
}


def sniff_sentinel(message):
    # type: (str) -> Optional[type]
    """Return the sentinel class whose phrase appears in message, else None."""
    lowered = (message or "").lower()
    for needle, sentinel in _SENTINEL_SUBSTRINGS:
        if needle in lowered:
            return sentinel
    return None


def error_from_status(code, message):
    # type: (int, str) -> ConversionError
    """Build the typed exception for an ABI status code and message."""
    cls = _STATUS_CLASSES.get(int(code))
    if cls is None:
        return ConversionError(int(code), message)
    if cls is InvalidArgumentError:
        return InvalidArgumentError(int(code), message, sniff_sentinel(message))
    return cls(int(code), message)
