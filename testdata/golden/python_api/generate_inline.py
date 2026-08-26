#!/usr/bin/env python3
"""Inline-HTML Python API sample (typical Python utility style).

``generate.py`` loads ``architecture-diagram.html`` from disk. This file is
the other common caller shape: HTML lives in the program as a string/bytes,
you pass Document options, call ``.pdf()``, and write the bytes with
``os`` / ``pathlib``.

Mirrors the Go ``testdata/golden/api/generate.go`` option bag
(page size, zero margins, background, smart_shrinking, allow_local_files),
except the source is inline HTML instead of ``File(path)`` (the v1 one-shot
ABI accepts inline HTML only).

Run from the repository root (also invoked by ``make python-api``):

    python3 testdata/golden/python_api/generate_inline.py

Writes (overwriting if present):

  1. output/python/invoice-inline.pdf
"""

import argparse
import os
import sys
from pathlib import Path

_REPO_CANDIDATES = [
    Path(__file__).resolve().parents[3],
    Path.cwd(),
]
for _root in _REPO_CANDIDATES:
    _src = _root / "bindings" / "python" / "src"
    if _src.is_dir():
        sys.path.insert(0, str(_src))
        break

from gowkhtmltopdf import (  # noqa: E402
    Content,
    Document,
    Margin,
    PDFOptions,
    Page,
    convert_html_to_pdf,
)

SAMPLE_DIRECTORY = "output/python"
OUTPUT_NAME = "invoice-inline.pdf"
PDF_FILE_MODE = 0o600
WANT_PAGES = 1

# Typical Python utility pattern: HTML is data in the program, not a path.
INVOICE_HTML = """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Invoice #42</title>
  <style>
    body { color: #172033; font-family: sans-serif; font-size: 11pt; margin: 18mm; }
    h1 { color: #174a7c; font-size: 18pt; margin: 0 0 4mm; }
    .meta { margin: 0 0 8mm; color: #4a5a6a; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #a8b5c5; padding: 2mm 3mm; text-align: left; }
    th { background: #e6eef7; }
    td.amount { text-align: right; }
    .total { margin-top: 6mm; font-weight: bold; text-align: right; }
  </style>
</head>
<body>
  <h1>Invoice #42</h1>
  <p class="meta">Northline Systems - due on receipt</p>
  <table>
    <thead>
      <tr><th>SKU</th><th>Description</th><th>Qty</th><th>Amount</th></tr>
    </thead>
    <tbody>
      <tr>
        <td>SKU-001</td>
        <td>Platform operations and support service</td>
        <td>2</td>
        <td class="amount">19.00</td>
      </tr>
    </tbody>
  </table>
  <p class="total">Total: $19.00</p>
</body>
</html>
""".encode("utf-8")


class UnexpectedPageCountError(RuntimeError):
    """Raised when the rendered PDF page count is not WANT_PAGES."""


def page_count(pdf):
    # type: (bytes) -> int
    return pdf.count(b"/Type /Page\n")


def resolve_repo_root(working_dir=None, source_file=None):
    # type: (...) -> Path
    """Find the repo root that contains output/ and bindings/python/."""
    cwd = working_dir if working_dir is not None else Path.cwd()
    source_dir = (
        source_file.parent
        if source_file is not None
        else Path(__file__).resolve().parent
    )
    candidates = [
        cwd,
        source_dir.parent.parent.parent,  # testdata/golden/python_api -> repo
    ]
    seen = set()
    for candidate in candidates:
        root = candidate.resolve()
        key = str(root)
        if key in seen:
            continue
        seen.add(key)
        if (root / "bindings" / "python" / "src").is_dir() and (
            root / "testdata" / "golden"
        ).is_dir():
            return root
    raise FileNotFoundError(
        "could not resolve repository root from {0}".format(cwd)
    )

def write_file(path, data, mode=PDF_FILE_MODE):
    # type: (Path, bytes, int) -> None
    parent = os.path.dirname(os.fspath(path))
    if parent:
        os.makedirs(parent, exist_ok=True)
    with open(path, "wb") as sink:
        sink.write(data)
    os.chmod(path, mode)


def render_with_document(html):
    # type: (bytes) -> bytes
    """Document API shape, matching generate.go's option bag."""
    document = Document(
        pages=[Page(source=Content.from_html(html))],
        page_size="A4",
        orientation="portrait",
        margin=Margin(top=0.0, right=0.0, bottom=0.0, left=0.0),
        background=True,
        smart_shrinking=False,
        allow_local_files=False,
    )
    return document.pdf()


def render_with_helper(html):
    # type: (bytes) -> bytes
    """Flat helper shape used by most Python HTML-to-PDF wrappers."""
    return convert_html_to_pdf(
        html,
        options=PDFOptions(
            page_size="A4",
            orientation="portrait",
            margin=Margin(top=0.0, right=0.0, bottom=0.0, left=0.0),
            allow_local_files=False,
        ),
    )


def run(argv=None):
    # type: (list | None) -> int
    parser = argparse.ArgumentParser(
        prog="generate_inline",
        description="Render an inline HTML invoice through the Python Document API",
    )
    parser.add_argument(
        "--output",
        default="",
        help="PDF output path (default: output/python/invoice-inline.pdf)",
    )
    args = parser.parse_args(argv)

    repo_root = resolve_repo_root()
    default_output = repo_root / SAMPLE_DIRECTORY / OUTPUT_NAME
    if args.output:
        output_path = Path(args.output)
        if not output_path.is_absolute():
            output_path = (Path.cwd() / output_path).resolve()
    else:
        output_path = default_output

    # Primary sample: Document with explicit options (same idea as generate.go).
    pdf = render_with_document(INVOICE_HTML)
    got = page_count(pdf)
    if got != WANT_PAGES:
        raise UnexpectedPageCountError(
            "unexpected page count: invoice-inline pages = {0}, want {1}".format(
                got, WANT_PAGES
            )
        )

    # Sanity: helper path produces the same structure for the same HTML.
    helper_pdf = render_with_helper(INVOICE_HTML)
    if not helper_pdf.startswith(b"%PDF-"):
        raise RuntimeError("convert_html_to_pdf output is not a PDF")
    if page_count(helper_pdf) != WANT_PAGES:
        raise UnexpectedPageCountError(
            "helper page count = {0}, want {1}".format(
                page_count(helper_pdf), WANT_PAGES
            )
        )

    write_file(output_path, pdf)
    print(
        "generated {0} ({1} pages, {2} bytes)".format(
            output_path, WANT_PAGES, len(pdf)
        )
    )
    return 0


def main():
    # type: () -> None
    try:
        raise SystemExit(run())
    except Exception as exc:  # noqa: BLE001 - CLI boundary
        print("generate_inline:", exc, file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
