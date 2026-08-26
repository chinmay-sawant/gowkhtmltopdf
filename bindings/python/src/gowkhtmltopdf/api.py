"""High-level one-call helpers mirroring the issue contract.

``convert_html_to_pdf`` and ``convert_html_to_image`` are sugar over the
Document / ImageDocument models; they accept either a prebuilt options
object or per-call keyword overrides.
"""

import dataclasses
from typing import Optional

from .document import (
    Content,
    Document,
    ImageDocument,
    ImageOptions,
    Page,
    PDFOptions,
)


def _as_html_bytes(html):
    # type: (object) -> bytes
    if isinstance(html, str):
        return html.encode("utf-8")
    if isinstance(html, (bytes, bytearray)):
        return bytes(html)
    raise TypeError("html must be str or bytes")


def convert_html_to_pdf(
    html,
    options=None,  # type: Optional[PDFOptions]
    **overrides  # type: object
):
    # type: (...) -> bytes
    """Convert inline HTML to PDF bytes in one call.

    ``options`` may be omitted for engine defaults. Keyword overrides are
    applied onto a copy of the options before serialization; ``timeout``
    (seconds) is accepted as an override too. Note that local file
    references inside ``html`` resolve relative to the current working
    directory because the content becomes an inline document.
    """
    timeout = overrides.pop("timeout", None)
    resolved = (options if options is not None else PDFOptions()).update(
        **overrides
    )
    content = Content.from_html(_as_html_bytes(html))
    kwargs = resolved.to_kwargs()
    option_base = kwargs.pop("base_url", None)
    if option_base:
        content.base = option_base
    doc = Document(pages=[Page(source=content)], **kwargs)
    return doc.pdf(timeout=timeout)


def convert_file_to_pdf(source, out_path=None, **options):
    # type: (str, Optional[str], object) -> bytes
    """Read an HTML file, convert it, and optionally write a PDF file.

    The top-level file is read by this helper; local resources it
    references (linked CSS, images) resolve relative to the source file's
    directory. ``allow_local_files`` defaults to True here unless
    overridden. Extra keyword arguments go to the Document constructor.
    """
    from pathlib import Path

    timeout = options.pop("timeout", None)
    base_url = options.pop("base_url", None)
    options.setdefault("allow_local_files", True)
    src_path = Path(source).resolve()
    if base_url is None:
        base_url = src_path.parent.as_uri() + "/"
    with open(src_path, "rb") as handle:
        data = handle.read()
    content = Content.from_html(data, base=base_url)
    doc = Document(pages=[Page(source=content)], **options)
    pdf_bytes = doc.pdf(timeout=timeout)
    if out_path is not None:
        with open(out_path, "wb") as sink:
            sink.write(pdf_bytes)
    return pdf_bytes


def convert_url_to_pdf(url, **options):
    # type: (str, object) -> bytes
    """Not implemented: URL sources need the handle-based ABI.

    Use the gowkhtmltopdf CLI for URL input today.
    """
    raise NotImplementedError(
        "URL source lands with the handle-based ABI;"
        " use the gowkhtmltopdf CLI for URL input today"
    )


def convert_html_to_image(
    html,
    options=None,  # type: Optional[ImageOptions]
    **overrides  # type: object
):
    # type: (...) -> bytes
    """Convert inline HTML to an encoded image (PNG by default)."""
    timeout = overrides.pop("timeout", None)
    resolved = (options if options is not None else ImageOptions()).update(
        **overrides
    )
    content = Content.from_html(_as_html_bytes(html))
    image_doc = ImageDocument(source=content, **{
        field.name: getattr(resolved, field.name)
        for field in dataclasses.fields(resolved)
        if field.name != "timeout_ms"
    })
    return image_doc.image(timeout=timeout)
