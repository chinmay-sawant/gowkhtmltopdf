#!/usr/bin/env python3
"""Runnable example: both snippet styles produce an invoice PDF.

Run from anywhere with a built library on the search path:

    cd bindings/python && python examples/invoice.py

Writes invoice.pdf (Document style) and invoice_high_level.pdf
(convert_html_to_pdf style) into the current directory.
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from gowkhtmltopdf import (  # noqa: E402
    Content,
    Document,
    PDFOptions,
    Page,
    convert_html_to_pdf,
)

HTML = (
    b"<html><body>"
    b"<h1>Invoice</h1>"
    b"<p>Invoice #42</p>"
    b"<p>Total: $19.00</p>"
    b"</body></html>"
)


def main():
    doc = Document(
        pages=[Page(source=Content(html=HTML))],
        page_size="A4",
    )
    document_bytes = doc.pdf()
    assert document_bytes.startswith(b"%PDF-"), "missing %PDF- header"

    high_level_bytes = convert_html_to_pdf(
        html=HTML,
        options=PDFOptions(page_size="A4", orientation="portrait"),
    )
    assert high_level_bytes.startswith(b"%PDF-"), "missing %PDF- header"

    with open("invoice.pdf", "wb") as sink:
        sink.write(document_bytes)
    with open("invoice_high_level.pdf", "wb") as sink:
        sink.write(high_level_bytes)

    print("invoice.pdf: {0} bytes".format(len(document_bytes)))
    print("invoice_high_level.pdf: {0} bytes".format(len(high_level_bytes)))


if __name__ == "__main__":
    main()
