# gowkhtmltopdf (Python)

In-process Python bindings for the gowkhtmltopdf HTML-to-PDF engine. The
package loads `libgowkhtmltopdf` (a Go `-buildmode=c-shared` library) with
stdlib `ctypes`; there is no subprocess and no compiled Python extension.

Requires Python 3.8+. Linux is the first-supported platform; macOS and
Windows builds follow the wheel matrix.

## Document style

Mirrors the Go `Document` API:

```python
from gowkhtmltopdf import Document, Page, Content

doc = Document(
    pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
    page_size="A4",
)
pdf_bytes: bytes = doc.pdf()  # or doc.pdf(timeout=30)
```

## Helper style

```python
from gowkhtmltopdf import convert_html_to_pdf, PDFOptions

pdf_bytes = convert_html_to_pdf(
    html=b"<html><body><h1>Invoice #42</h1><p>Total: $19.00</p></body></html>",
    options=PDFOptions(page_size="A4", orientation="portrait"),
)

with open("invoice.pdf", "wb") as f:
    f.write(pdf_bytes)
```

Images work the same way via `ImageDocument` or
`convert_html_to_image(html, options=ImageOptions(width=1024))`.

The full build, install, security (ACL / NetworkPolicy), and ABI
stability guide lives in [documentation/python.md](../../documentation/python.md).
