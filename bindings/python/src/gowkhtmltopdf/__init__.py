"""gowkhtmltopdf: in-process Python bindings for the gowkhtmltopdf engine.

Two usage styles, both backed by a ctypes-loaded c-shared library:

    from gowkhtmltopdf import Document, Page, Content

    doc = Document(
        pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
        page_size="A4",
    )
    pdf_bytes = doc.pdf()

Or the flat helper:

    from gowkhtmltopdf import convert_html_to_pdf, PDFOptions

    pdf_bytes = convert_html_to_pdf(
        b"<html><body><h1>Invoice #42</h1></body></html>",
        options=PDFOptions(page_size="A4"),
    )

The shared library is located and loaded only when a conversion runs;
building model objects never touches it.
"""

from .exceptions import (
    ConversionError,
    ConversionTimeoutError,
    ErrEmptyContent,
    ErrInvalidContent,
    ErrInvalidOrientation,
    ErrInvalidPDFProfile,
    ErrInvalidPageSize,
    ErrInvalidPDFVersion,
    ErrMissingOutput,
    ErrNoPageObjects,
    GowkhtmltopdfError,
    InternalEngineError,
    InvalidArgumentError,
    LoadDeniedError,
    RenderError,
    ResourceLimitError,
    error_from_status,
    sniff_sentinel,
)
from .document import (
    Content,
    Crop,
    Document,
    HeaderFooter,
    ImageDocument,
    ImageOptions,
    Margin,
    NetworkPolicy,
    Page,
    PDFOptions,
    TOC,
    compatible_network_policy,
    restricted_network_policy,
)
from .api import (
    convert_file_to_pdf,
    convert_html_to_image,
    convert_html_to_pdf,
    convert_url_to_pdf,
)

__version__ = "0.2.4"

#: Upstream settings-surface identifier (api.go LibraryVersion), distinct
#: from the project release in __version__.
library_version = "0.12.7-dev"


def abi_version():
    # type: () -> int
    """Return the ABI revision of the loaded shared library (always 1 today).

    Raises ImportError when the library is missing or built for another ABI.
    """
    from ._lib import abi_version as _abi_version

    return _abi_version()


def library_version_string():
    # type: () -> str
    """Return the runtime version string reported by the shared library."""
    from ._lib import library_version_string as _lvs

    return _lvs()


__all__ = [
    "GowkhtmltopdfError",
    "ConversionError",
    "InvalidArgumentError",
    "LoadDeniedError",
    "RenderError",
    "ConversionTimeoutError",
    "ResourceLimitError",
    "InternalEngineError",
    "ErrEmptyContent",
    "ErrInvalidContent",
    "ErrNoPageObjects",
    "ErrInvalidPageSize",
    "ErrInvalidOrientation",
    "ErrInvalidPDFVersion",
    "ErrInvalidPDFProfile",
    "ErrMissingOutput",
    "error_from_status",
    "sniff_sentinel",
    "Content",
    "Page",
    "Margin",
    "HeaderFooter",
    "TOC",
    "Crop",
    "NetworkPolicy",
    "compatible_network_policy",
    "restricted_network_policy",
    "PDFOptions",
    "ImageOptions",
    "Document",
    "ImageDocument",
    "convert_html_to_pdf",
    "convert_file_to_pdf",
    "convert_url_to_pdf",
    "convert_html_to_image",
    "__version__",
    "library_version",
    "abi_version",
    "library_version_string",
]
