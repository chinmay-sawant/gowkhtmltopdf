#!/usr/bin/env python3
"""Render the Python API architecture template to PDF.

This is the file-on-disk twin of ``testdata/golden/api/generate.go``: resolve
``architecture-diagram.html``, build a ``Document`` with the same option bag
(page size, zero margins, background, smart_shrinking, allow_local_files),
convert, assert page count, write bytes.

For the typical Python utility shape (HTML as in-memory bytes, no template
file), see ``generate_inline.py``.

Run from the repository root (also invoked by ``make python-api``):

    python3 testdata/golden/python_api/generate.py

Writes (overwriting if present):

  1. output/python/architecture-diagram.pdf

The v1 one-shot ABI accepts inline HTML only, so this reads the template
file and passes ``Content.from_html`` with a ``file://`` base URL (same
pattern as ``convert_file_to_pdf``). Go's generator can use ``File(path)``
directly; Python cannot until the handle-based ABI lands.
"""
import argparse
import os
import sys
from pathlib import Path

# Allow in-tree runs before pip install -e.
_REPO_CANDIDATES = [
    Path(__file__).resolve().parents[3],
    Path.cwd(),
]
for _root in _REPO_CANDIDATES:
    _src = _root / "bindings" / "python" / "src"
    if _src.is_dir():
        sys.path.insert(0, str(_src))
        break

from gowkhtmltopdf import Content, Document, Margin, Page  # noqa: E402

API_DIRECTORY = "testdata/golden/python_api"
SAMPLE_DIRECTORY = "output/python"
INPUT_NAME = "architecture-diagram.html"
OUTPUT_NAME = "architecture-diagram.pdf"
PDF_FILE_MODE = 0o600
WANT_PAGES = 5


class TemplateNotFoundError(FileNotFoundError):
    """Raised when architecture-diagram.html cannot be resolved."""


class UnexpectedPageCountError(RuntimeError):
    """Raised when the rendered PDF page count is not WANT_PAGES."""


def page_count(pdf):
    # type: (bytes) -> int
    return pdf.count(b"/Type /Page\n")


def is_python_api_template(path):
    # type: (Path) -> bool
    """True when path is testdata/golden/python_api/architecture-diagram.html."""
    return path.name == INPUT_NAME and path.parent.name == "python_api"


def unique_paths(*paths):
    # type: (str) -> list
    out = []
    seen = set()
    for path in paths:
        if not path:
            continue
        absolute = str(Path(path).resolve())
        if absolute in seen:
            continue
        seen.add(absolute)
        out.append(absolute)
    return out


def same_path(a, b):
    # type: (str, str) -> bool
    try:
        return Path(a).resolve() == Path(b).resolve()
    except OSError:
        return a == b


def resolve_template_paths(working_dir=None, source_file=None):
    # type: (...) -> tuple
    """Return (input, default_output, repo_root).

    Accepts invocation from the repository root, the python_api directory
    itself, or a compiled copy whose source directory is available.
    """
    cwd = working_dir if working_dir is not None else Path.cwd()
    source_dir = (
        source_file.parent
        if source_file is not None
        else Path(__file__).resolve().parent
    )

    candidates = [
        cwd / INPUT_NAME,
        cwd / API_DIRECTORY / INPUT_NAME,
        source_dir / INPUT_NAME,
    ]
    checked = []
    seen = set()

    for candidate in candidates:
        absolute = candidate.resolve()
        key = str(absolute)
        if key in seen:
            continue
        seen.add(key)
        checked.append(key)

        if not absolute.exists():
            continue
        if absolute.is_dir():
            raise IsADirectoryError(
                "template path is a directory: {0}".format(absolute)
            )
        if not is_python_api_template(absolute):
            # testdata/golden/architecture-diagram.html and api/ share the
            # basename but are not this generator's source.
            continue

        api_dir = absolute.parent
        # template lives at <repo>/testdata/golden/python_api/<file>
        root = api_dir.parent.parent.parent
        default_output = root / SAMPLE_DIRECTORY / OUTPUT_NAME
        return absolute, default_output, root

    raise TemplateNotFoundError(
        "template not found {0!r}; checked {1}".format(
            INPUT_NAME, ", ".join(checked)
        )
    )


def write_file(path, data, mode=PDF_FILE_MODE):
    # type: (Path, bytes, int) -> None
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(data)
    os.chmod(path, mode)


def render_document(input_path):
    # type: (Path) -> bytes
    html = input_path.read_bytes()
    base_url = input_path.resolve().parent.as_uri() + "/"
    document = Document(
        pages=[Page(source=Content.from_html(html, base=base_url))],
        page_size="A4",
        margin=Margin(top=0.0, right=0.0, bottom=0.0, left=0.0),
        background=True,
        smart_shrinking=False,
        allow_local_files=True,
    )
    return document.pdf()


def run(argv=None):
    # type: (list | None) -> int
    parser = argparse.ArgumentParser(
        prog="generate",
        description="Render testdata/golden/python_api/architecture-diagram.html",
    )
    parser.add_argument(
        "--output",
        default="",
        help="PDF output path (default: output/python/architecture-diagram.pdf)",
    )
    args = parser.parse_args(argv)

    input_path, default_output, _repo_root = resolve_template_paths()
    output_path = Path(args.output) if args.output else default_output
    if not output_path.is_absolute():
        output_path = (Path.cwd() / output_path).resolve()

    pdf = render_document(input_path)
    got = page_count(pdf)
    if got != WANT_PAGES:
        raise UnexpectedPageCountError(
            "unexpected page count: {0} pages = {1}, want {2}".format(
                input_path, got, WANT_PAGES
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
        print("generate:", exc, file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
