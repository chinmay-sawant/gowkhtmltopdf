"""Integration tests against libgowkhtmltopdf.

Every test skips when the shared library is absent so the suite stays
green on machines that have not built bindings/c yet. Build it with:

    CGO_ENABLED=1 go build -buildmode=c-shared \
        -o dist/libgowkhtmltopdf.so ./bindings/c
"""

import re
import unittest
from pathlib import Path

import gowkhtmltopdf
from gowkhtmltopdf import Content, Document, ImageOptions, PDFOptions, Page
from gowkhtmltopdf import (
    convert_file_to_pdf,
    convert_html_to_image,
    convert_html_to_pdf,
)
from gowkhtmltopdf import _lib
from gowkhtmltopdf.exceptions import ErrInvalidPageSize
from gowkhtmltopdf.exceptions import ErrNoPageObjects, InvalidArgumentError


def _find_library():
    try:
        return _lib.find_library_path()
    except Exception:
        return None


_LIB_PATH = _find_library()

_REASON = (
    "libgowkhtmltopdf not found; build with"
    " 'CGO_ENABLED=1 go build -buildmode=c-shared"
    " -o dist/libgowkhtmltopdf.so ./bindings/c' or set"
    " GOWKHTMLTOPDF_LIBRARY_PATH"
)

_REPO_ROOT = Path(__file__).resolve().parents[3]
_FIXTURE = (
    _REPO_ROOT / "testdata" / "golden" / "fixture-01-simple-invoice.html"
)

_INLINE_HTML = (
    b"<html><body><h1>Invoice #42</h1><p>Total: $19.00</p></body></html>"
)

# The engine pins dates per call (Request.now falls back to time.Now in
# internal/convert/convert.go), so two conversions of the same input can
# differ inside these fields only.
_DATE_RE = re.compile(rb"/(Creation|Mod)Date \(D:[0-9]{14}Z\)")


def _normalize_dates(data):
    return _DATE_RE.sub(b"DATE", data)


@unittest.skipUnless(_LIB_PATH is not None, _REASON)
class BindingTest(unittest.TestCase):
    def test_inline_pdf_structure(self):
        pdf_bytes = convert_html_to_pdf(_INLINE_HTML)
        self.assertGreater(len(pdf_bytes), 1024)
        self.assertTrue(pdf_bytes.startswith(b"%PDF-"))
        self.assertIn(b"%%EOF", pdf_bytes[-1024:])
        self.assertIn(b"/FontFile2", pdf_bytes)

    def test_fixture_invoice_converts(self):
        self.assertTrue(_FIXTURE.is_file(), "missing {0}".format(_FIXTURE))
        pdf_bytes = convert_file_to_pdf(str(_FIXTURE))
        self.assertGreater(len(pdf_bytes), 1024)
        self.assertTrue(pdf_bytes.startswith(b"%PDF-"))
        self.assertIn(b"/FontFile2", pdf_bytes)
        self.assertGreaterEqual(pdf_bytes.count(b"/Type /Page"), 1)

    def test_document_parity_with_helper(self):
        document_bytes = Document(
            pages=[Page(source=Content(html=_INLINE_HTML))],
            page_size="A4",
            orientation="portrait",
        ).pdf()
        helper_bytes = convert_html_to_pdf(
            _INLINE_HTML,
            options=PDFOptions(page_size="A4", orientation="portrait"),
        )
        # Byte equality holds apart from embedded creation timestamps.
        self.assertEqual(
            _normalize_dates(document_bytes),
            _normalize_dates(helper_bytes),
        )

    def test_invalid_page_size_maps_to_sentinel(self):
        opts = _lib.GwkPdfOptions.create()
        opts.page_size = b"Bogus"
        try:
            with self.assertRaises(InvalidArgumentError) as ctx:
                _lib.convert_html_to_pdf(_INLINE_HTML, opts)
            self.assertIs(ctx.exception.sentinel, ErrInvalidPageSize)
        finally:
            opts.page_size = None

    def test_empty_document_model_rejected_without_library_call(self):
        with self.assertRaises(ErrNoPageObjects):
            Document(pages=[]).validate()

    def test_html_to_image_png_magic(self):
        png_bytes = convert_html_to_image(
            b"<html><body><h1>Badge</h1></body></html>",
            options=ImageOptions(width=256, format="png"),
        )
        self.assertTrue(png_bytes.startswith(b"\x89PNG\r\n\x1a\n"))

    def test_version_and_abi_reported(self):
        self.assertEqual(gowkhtmltopdf.__version__, "0.2.5")
        self.assertEqual(gowkhtmltopdf.library_version, "0.12.7-dev")
        self.assertEqual(gowkhtmltopdf.abi_version(), 1)
        reported = gowkhtmltopdf.library_version_string()
        self.assertIsInstance(reported, str)
        self.assertGreater(len(reported), 0)

    def test_repeated_conversion_is_stable_and_frees_cleanly(self):
        first = _normalize_dates(convert_html_to_pdf(_INLINE_HTML))
        for _ in range(25):
            again = _normalize_dates(convert_html_to_pdf(_INLINE_HTML))
            self.assertEqual(first, again)


if __name__ == "__main__":
    unittest.main()
