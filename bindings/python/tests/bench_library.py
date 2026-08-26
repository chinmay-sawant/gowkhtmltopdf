#!/usr/bin/env python3
"""Public Python API library benchmarks.

Mirrors ``BenchmarkLibraryPDF`` / ``BenchmarkLibraryImage`` in
``document_bench_test.go``. Uses the same dirty report template
(``testdata/golden/benchmarks/templates/report.html.tmpl``) with 20 invoice
rows per requested page. Template expansion happens before the timer;
``Document.pdf()`` / ``ImageDocument.image()`` remain inside the timed calls.

Run via Makefile (rebuilds the c-shared library first):

    make python-benchmarks

Or directly after ``CGO_ENABLED=1 make c-shared``:

    PYTHONPATH=bindings/python/src python3 bindings/python/tests/bench_library.py

Optional env:

- ``GOWKHTMLTOPDF_BENCH_SIZES`` comma list of page/tile counts
  (default: 2,5,10,20,50,100,200,250,500)
- ``GOWKHTMLTOPDF_BENCH_RUNS`` timed iterations per size (default: 10)
- ``GOWKHTMLTOPDF_BENCH_WARMUP`` warmup iterations per size (default: 1)
"""

from __future__ import annotations

import os
import statistics
import sys
import time
from pathlib import Path

_REPO_ROOT = Path(__file__).resolve().parents[3]
_SRC = _REPO_ROOT / "bindings" / "python" / "src"
if str(_SRC) not in sys.path:
    sys.path.insert(0, str(_SRC))

from gowkhtmltopdf import (  # noqa: E402
    Content,
    Document,
    ImageDocument,
    Page,
)
from gowkhtmltopdf import _lib  # noqa: E402

_TEMPLATE = (
    _REPO_ROOT
    / "testdata"
    / "golden"
    / "benchmarks"
    / "templates"
    / "report.html.tmpl"
)

_DEFAULT_SIZES = (2, 5, 10, 20, 50, 100, 200, 250, 500)
_ROWS_PER_PAGE = 20


def _parse_sizes(raw: str | None) -> tuple[int, ...]:
    if not raw or not raw.strip():
        return _DEFAULT_SIZES
    values = []
    for part in raw.split(","):
        part = part.strip()
        if not part:
            continue
        values.append(int(part))
    if not values:
        raise ValueError("GOWKHTMLTOPDF_BENCH_SIZES produced no sizes")
    return tuple(values)


def _env_int(name: str, default: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    return int(raw)


def expand_report_html(page_count: int) -> bytes:
    """Expand report.html.tmpl the same way document_bench_test.go does."""
    source = _TEMPLATE.read_text(encoding="utf-8")
    if "{{range .Pages}}" not in source or "{{range .Rows}}" not in source:
        raise RuntimeError("benchmark template missing Pages/Rows ranges: {0}".format(_TEMPLATE))

    head, rest = source.split("{{range .Pages}}", 1)
    page_block, tail = rest.rsplit("{{end}}", 1)
    page_prefix, rows_rest = page_block.split("{{range .Rows}}", 1)
    row_line, page_suffix = rows_rest.split("{{end}}", 1)

    pages_html = []
    for page_index in range(page_count):
        page_number = page_index + 1
        first_class = " first" if page_index == 0 else ""
        section = page_prefix.replace("{{if .First}} first{{end}}", first_class)
        section = section.replace("{{.Number}}", str(page_number))

        rows_html = []
        for row_index in range(_ROWS_PER_PAGE):
            line = row_index + 1
            row = row_line
            row = row.replace("{{.Number}}", str(line))
            row = row.replace(
                "{{.SKU}}",
                "SKU-{0:03d}-{1:03d}".format(page_number, line),
            )
            row = row.replace(
                "{{.Description}}",
                "Platform operations and support service {0}".format(line),
            )
            row = row.replace(
                "{{.Quantity}}",
                str((line + page_index) % 7 + 1),
            )
            row = row.replace(
                "{{.Amount}}",
                "{0}.{1:02d}".format(page_number * line, (page_index + line) % 100),
            )
            rows_html.append(row)

        pages_html.append(section + "".join(rows_html) + page_suffix)

    rendered = head + "".join(pages_html) + tail
    if "{{" in rendered:
        raise RuntimeError("template actions remained after rendering")
    return rendered.encode("utf-8")

def library_pdf_document(page_count: int) -> Document:
    html = expand_report_html(page_count)
    return Document(pages=[Page(source=Content.from_html(html))])


def library_image_document(tile_count: int) -> ImageDocument:
    parts = [
        "<!doctype html><html><head><meta charset=\"utf-8\"><style>",
        "body { margin: 0; font-family: sans-serif; }",
        ".grid { display: flex; flex-wrap: wrap; gap: 8px; padding: 8px; }",
        ".tile { width: 120px; height: 56px; background: #d9e2ec; color: #17324d;",
        "  border: 1px solid #52718d; padding: 8px; box-sizing: border-box; }",
        "</style></head><body><div class=\"grid\">",
    ]
    for tile in range(1, tile_count + 1):
        parts.append('<div class="tile">Tile {0}</div>'.format(tile))
    parts.append("</div></body></html>")
    return ImageDocument(
        source=Content.from_html("".join(parts).encode("utf-8")),
        width=1024,
        height=512,
        format="png",
    )


def _format_duration(seconds: float) -> str:
    if seconds >= 1.0:
        return "{0:.3f}s".format(seconds)
    return "{0:.2f}ms".format(seconds * 1000.0)


def _median(samples: list[float]) -> float:
    return statistics.median(samples)


def _run_timed(label: str, size: int, unit: str, fn, warmup: int, runs: int) -> None:
    for _ in range(warmup):
        payload = fn()
        if not payload:
            raise RuntimeError("{0}: empty output during warmup".format(label))

    samples: list[float] = []
    last = b""
    for _ in range(runs):
        started = time.perf_counter()
        last = fn()
        samples.append(time.perf_counter() - started)

    median_s = _median(samples)
    print(
        "{0}/{1}{2}: median {3}  (n={4}, size={5} bytes)".format(
            label,
            size,
            unit,
            _format_duration(median_s),
            runs,
            len(last),
        )
    )


def bench_pdf(sizes: tuple[int, ...], warmup: int, runs: int) -> None:
    print("BenchmarkPythonLibraryPDF (report.html.tmpl, 20 rows/page)")
    for page_count in sizes:
        document = library_pdf_document(page_count)

        def convert(doc=document, expected=page_count):
            pdf = doc.pdf()
            if not pdf.startswith(b"%PDF-"):
                raise RuntimeError("Document.pdf output is not a PDF")
            pages = pdf.count(b"/Type /Page\n")
            if pages != expected:
                raise RuntimeError(
                    "Document.pdf pages = {0}, want {1}".format(pages, expected)
                )
            return pdf

        _run_timed("PDF", page_count, "Pages", convert, warmup, runs)


def bench_image(sizes: tuple[int, ...], warmup: int, runs: int) -> None:
    print("BenchmarkPythonLibraryImage (inline tile grid)")
    for tile_count in sizes:
        document = library_image_document(tile_count)

        def convert(doc=document):
            png = doc.image()
            if not png.startswith(b"\x89PNG\r\n\x1a\n"):
                raise RuntimeError("ImageDocument.image output is not a PNG")
            return png

        _run_timed("Image", tile_count, "Tiles", convert, warmup, runs)


def main() -> int:
    try:
        lib_path = _lib.find_library_path()
    except FileNotFoundError as exc:
        print("bench_library: {0}".format(exc), file=sys.stderr)
        print(
            "build with: CGO_ENABLED=1 make c-shared",
            file=sys.stderr,
        )
        return 2

    if not _TEMPLATE.is_file():
        print("bench_library: missing template {0}".format(_TEMPLATE), file=sys.stderr)
        return 2

    sizes = _parse_sizes(os.environ.get("GOWKHTMLTOPDF_BENCH_SIZES"))
    warmup = _env_int("GOWKHTMLTOPDF_BENCH_WARMUP", 1)
    runs = _env_int("GOWKHTMLTOPDF_BENCH_RUNS", 10)
    if warmup < 0 or runs < 1:
        print("bench_library: warmup must be >= 0 and runs >= 1", file=sys.stderr)
        return 2

    print("library: {0}".format(lib_path))
    print("template: {0}".format(_TEMPLATE))
    print("sizes: {0}".format(",".join(str(s) for s in sizes)))
    print("warmup={0} runs={1}".format(warmup, runs))
    print()

    bench_pdf(sizes, warmup, runs)
    print()
    bench_image(sizes, warmup, runs)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
