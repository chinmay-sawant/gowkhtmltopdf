#!/usr/bin/env python3
"""Regenerate output/python/ fixture PDFs through the Python Document API.

Mirrors the body-fixture loop of ``make samples``, but writes under
``output/python/`` via ``convert_file_to_pdf`` (inline HTML + file:// base).

Also writes version / compliance smokes for fixture-21 and fixture-56 into
``output/python/pdf-{1.7,2.0}{,-compliance}/``, matching the Go sample
four-way split (nested under python/).

Run from the repository root (also invoked by ``make samples-python``):

    python3 testdata/golden/python_api/generate_samples.py

Notes on v1 ABI limits vs ``make samples``:

- Header/footer HTML companions (fixture-36-*-html) are not transmitted by
  the one-shot C ABI, so fixture-36 is body-only.
- ``--font-path`` / system font flags are not on ``GwkPdfOptions``, so
  fixture-27 and font-family stacks use engine defaults.
- Showcase TOC/HF/outline and live Wikipedia smokes stay Go/CLI-only.
"""

from __future__ import annotations

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

from gowkhtmltopdf import convert_file_to_pdf  # noqa: E402

import generate  # noqa: E402

PDF_FILE_MODE = 0o600

COMPLIANCE_FIXTURES = (
    "fixture-21-detailed-report.html",
    "fixture-56-architecture-diagram.html",
)

COMPLIANCE_TARGETS = (
    ("pdf-1.7", {"pdf_version": "1.7"}),
    ("pdf-1.7-compliance", {"pdf_profile": "a3a-ua1"}),
    ("pdf-2.0", {"pdf_version": "2.0"}),
    ("pdf-2.0-compliance", {"pdf_profile": "a4-ua2"}),
)


def resolve_repo_root(working_dir=None, source_file=None):
    # type: (...) -> Path
    cwd = working_dir if working_dir is not None else Path.cwd()
    source_dir = (
        source_file.parent
        if source_file is not None
        else Path(__file__).resolve().parent
    )
    for candidate in (cwd, source_dir.parent.parent.parent):
        root = candidate.resolve()
        if (root / "testdata" / "golden").is_dir() and (
            root / "bindings" / "python" / "src"
        ).is_dir():
            return root
    raise FileNotFoundError("could not resolve repository root from {0}".format(cwd))


def list_body_fixtures(golden_dir):
    # type: (Path) -> list
    fixtures = []
    for path in sorted(golden_dir.glob("fixture-*.html")):
        name = path.name
        if name.endswith("-header.html") or name.endswith("-footer.html"):
            continue
        fixtures.append(path)
    return fixtures


def write_pdf(path, data):
    # type: (Path, bytes) -> None
    parent = os.path.dirname(os.fspath(path))
    if parent:
        os.makedirs(parent, exist_ok=True)
    with open(path, "wb") as sink:
        sink.write(data)
    os.chmod(path, PDF_FILE_MODE)


def convert_one(source, out_path, **options):
    # type: (Path, Path, object) -> bytes
    pdf = convert_file_to_pdf(str(source), out_path=None, **options)
    if not pdf.startswith(b"%PDF-"):
        raise RuntimeError("{0}: missing %PDF- header".format(source.name))
    write_pdf(out_path, pdf)
    return pdf


def run(argv=None):
    # type: (list | None) -> int
    parser = argparse.ArgumentParser(
        prog="generate_samples",
        description="Convert golden fixtures into output/python/ via the Python API",
    )
    parser.add_argument(
        "--output-root",
        default="",
        help="Sample root (default: output/python)",
    )
    parser.add_argument(
        "--skip-compliance",
        action="store_true",
        help="Skip fixture-21/56 version and profile smokes",
    )
    args = parser.parse_args(argv)

    repo_root = resolve_repo_root()
    golden_dir = repo_root / "testdata" / "golden"
    if args.output_root:
        output_root = Path(args.output_root)
        if not output_root.is_absolute():
            output_root = (Path.cwd() / output_root).resolve()
    else:
        output_root = repo_root / generate.SAMPLE_DIRECTORY

    os.makedirs(output_root, exist_ok=True)

    # Wipe regenerable fixture samples only (keep manual leftovers out of the wipe
    # by matching the fixture-*.pdf pattern make samples uses).
    for stale in output_root.glob("fixture-*.pdf"):
        stale.unlink()

    fixtures = list_body_fixtures(golden_dir)
    if not fixtures:
        raise FileNotFoundError("no fixture-*.html bodies under {0}".format(golden_dir))

    failures = []  # type: list
    for source in fixtures:
        out_path = output_root / (source.stem + ".pdf")
        try:
            pdf = convert_one(source, out_path, allow_local_files=True)
        except Exception as exc:  # noqa: BLE001 - per-fixture soft fail
            failures.append((source.name, str(exc)))
            print(
                "warning: skipped {0}: {1}".format(source.name, exc),
                file=sys.stderr,
            )
            continue
        pages = pdf.count(b"/Type /Page\n")
        print(
            "generated {0} ({1} pages, {2} bytes)".format(
                out_path, pages, len(pdf)
            )
        )

    if not args.skip_compliance:
        for subdir, _kwargs in COMPLIANCE_TARGETS:
            target_dir = output_root / subdir
            os.makedirs(target_dir, exist_ok=True)
            for stale in target_dir.glob("fixture-*.pdf"):
                stale.unlink()

        for fixture_name in COMPLIANCE_FIXTURES:
            source = golden_dir / fixture_name
            if not source.is_file():
                raise FileNotFoundError("missing compliance fixture {0}".format(source))
            stem = source.stem
            for subdir, kwargs in COMPLIANCE_TARGETS:
                out_path = output_root / subdir / (stem + ".pdf")
                try:
                    pdf = convert_one(
                        source,
                        out_path,
                        allow_local_files=True,
                        **kwargs
                    )
                except Exception as exc:  # noqa: BLE001 - per-fixture soft fail
                    failures.append(
                        ("{0}/{1}".format(subdir, fixture_name), str(exc))
                    )
                    print(
                        "warning: skipped {0}/{1}: {2}".format(
                            subdir, fixture_name, exc
                        ),
                        file=sys.stderr,
                    )
                    continue
                pages = pdf.count(b"/Type /Page\n")
                print(
                    "generated {0} ({1} pages, {2} bytes, {3})".format(
                        out_path, pages, len(pdf), kwargs
                    )
                )

    print(
        "samples-python fixtures: {0} bodies under {1}".format(
            len(fixtures), output_root
        )
    )
    if failures:
        print(
            "warning: {0} conversion(s) skipped (v1 ABI limits such as "
            "missing font-path / header-footer):".format(len(failures)),
            file=sys.stderr,
        )
        for name, err in failures:
            print("  - {0}: {1}".format(name, err), file=sys.stderr)
        # Soft-fail like make samples wiki smoke: remaining artifacts still land.
    return 0


def main():
    # type: () -> None
    try:
        raise SystemExit(run())
    except Exception as exc:  # noqa: BLE001 - CLI boundary
        print("generate_samples:", exc, file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
