#!/usr/bin/env python3
"""Version / compliance smokes for the Python architecture diagram.

Mirrors the four Go sample folders under ``output/pdf-{1.7,2.0}{,-compliance}/``
but nests them under ``output/python/`` so Python API samples stay together:

  output/python/pdf-1.7/architecture-diagram.pdf              (--pdf-version 1.7)
  output/python/pdf-1.7-compliance/architecture-diagram.pdf   (--pdf-profile a3a-ua1)
  output/python/pdf-2.0/architecture-diagram.pdf              (--pdf-version 2.0)
  output/python/pdf-2.0-compliance/architecture-diagram.pdf   (--pdf-profile a4-ua2)

Same source template as ``generate.py``
(``testdata/golden/python_api/architecture-diagram.html``). A bare version
flag is not a PDF/A or PDF/UA claim; only the ``*-compliance/`` dirs set a
profile.

Run from the repository root (also invoked by ``make python-api``):

    python3 testdata/golden/python_api/generate_compliance.py
"""

import argparse
import os
import sys
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

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

import generate  # noqa: E402

OUTPUT_NAME = "architecture-diagram.pdf"
PDF_FILE_MODE = 0o600
WANT_PAGES = generate.WANT_PAGES

# (subdir under output/python, Document kwargs)
COMPLIANCE_TARGETS = (
    ("pdf-1.7", {"pdf_version": "1.7"}),
    ("pdf-1.7-compliance", {"pdf_profile": "a3a-ua1"}),
    ("pdf-2.0", {"pdf_version": "2.0"}),
    ("pdf-2.0-compliance", {"pdf_profile": "a4-ua2"}),
)


def render_document(input_path, **version_kwargs):
    # type: (Path, object) -> bytes
    html = input_path.read_bytes()
    base_url = input_path.resolve().parent.as_uri() + "/"
    document = Document(
        pages=[Page(source=Content.from_html(html, base=base_url))],
        page_size="A4",
        margin=Margin(top=0.0, right=0.0, bottom=0.0, left=0.0),
        background=True,
        smart_shrinking=False,
        allow_local_files=True,
        **version_kwargs
    )
    return document.pdf()


def write_file(path, data, mode=PDF_FILE_MODE):
    # type: (Path, bytes, int) -> None
    parent = os.path.dirname(os.fspath(path))
    if parent:
        os.makedirs(parent, exist_ok=True)
    with open(path, "wb") as sink:
        sink.write(data)
    os.chmod(path, mode)


def run(argv=None):
    # type: (list | None) -> int
    parser = argparse.ArgumentParser(
        prog="generate_compliance",
        description=(
            "Render architecture-diagram.html into "
            "output/python/pdf-{1.7,2.0}{,-compliance}/"
        ),
    )
    parser.add_argument(
        "--output-root",
        default="",
        help="Parent of the pdf-* dirs (default: output/python)",
    )
    args = parser.parse_args(argv)

    input_path, _default_output, repo_root = generate.resolve_template_paths()
    if args.output_root:
        output_root = Path(args.output_root)
        if not output_root.is_absolute():
            output_root = (Path.cwd() / output_root).resolve()
    else:
        output_root = repo_root / generate.SAMPLE_DIRECTORY

    for subdir, kwargs in COMPLIANCE_TARGETS:
        pdf = render_document(input_path, **kwargs)
        got = generate.page_count(pdf)
        if got != WANT_PAGES:
            raise generate.UnexpectedPageCountError(
                "unexpected page count: {0}/{1} pages = {2}, want {3}".format(
                    subdir, OUTPUT_NAME, got, WANT_PAGES
                )
            )
        if not pdf.startswith(b"%PDF-"):
            raise RuntimeError("{0}: missing %PDF- header".format(subdir))

        out_path = output_root / subdir / OUTPUT_NAME
        write_file(out_path, pdf)
        print(
            "generated {0} ({1} pages, {2} bytes, {3})".format(
                out_path, WANT_PAGES, len(pdf), kwargs
            )
        )
    return 0


def main():
    # type: () -> None
    try:
        raise SystemExit(run())
    except Exception as exc:  # noqa: BLE001 - CLI boundary
        print("generate_compliance:", exc, file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
