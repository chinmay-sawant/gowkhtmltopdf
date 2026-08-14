#!/usr/bin/env python3
"""Rasterize output/*.pdf pages into frontend showcase PNGs.

Writes one PNG per PDF page with a solid white background:

  {name}.png       page 1
  {name}-2.png     page 2
  {name}-N.png     page N

Default paths (repo root relative):

  input:  output/*.pdf
  output: frontend/src/assets/showcase/

Requires pymupdf:

  pip install pymupdf

Examples:

  python3 scripts/screenshot_showcase.py
  python3 scripts/screenshot_showcase.py --dpi 96
  python3 scripts/screenshot_showcase.py --pdf output/fixture-01-simple-invoice.pdf
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import fitz  # PyMuPDF
except ImportError:
    print("need pymupdf: pip install pymupdf", file=sys.stderr)
    sys.exit(2)

# Match the committed showcase thumbs (A4 @ 96 dpi ≈ 794×1123).
DEFAULT_DPI = 96.0


def page_path(out_dir: Path, stem: str, page_num: int) -> Path:
    """Return the showcase filename for a 1-based page number."""
    if page_num == 1:
        return out_dir / f"{stem}.png"
    return out_dir / f"{stem}-{page_num}.png"


def remove_stale(out_dir: Path, stem: str, page_count: int) -> int:
    """Delete leftover PNGs for pages beyond the current PDF length."""
    removed = 0
    for path in out_dir.glob(f"{stem}*.png"):
        name = path.name
        if name == f"{stem}.png":
            page = 1
        elif name.startswith(f"{stem}-") and name.endswith(".png"):
            suffix = name[len(stem) + 1 : -4]
            if not suffix.isdigit():
                continue
            page = int(suffix)
        else:
            continue
        if page > page_count:
            path.unlink()
            removed += 1
    return removed


def render_pdf(pdf: Path, out_dir: Path, dpi: float) -> int:
    """Render every page of pdf into out_dir. Returns pages written."""
    zoom = dpi / 72.0
    matrix = fitz.Matrix(zoom, zoom)
    # alpha=False composites onto an opaque white RGB canvas.
    doc = fitz.open(pdf)
    try:
        stem = pdf.stem
        for i in range(doc.page_count):
            page = doc[i]
            pix = page.get_pixmap(
                matrix=matrix,
                colorspace=fitz.csRGB,
                alpha=False,
            )
            dest = page_path(out_dir, stem, i + 1)
            pix.save(dest.as_posix())
        removed = remove_stale(out_dir, stem, doc.page_count)
        if removed:
            print(f"  removed {removed} stale page(s) for {stem}")
        return doc.page_count
    finally:
        doc.close()


def collect_pdfs(root: Path, only: list[Path] | None) -> list[Path]:
    if only:
        pdfs = []
        for p in only:
            path = p if p.is_absolute() else root / p
            if not path.is_file():
                raise FileNotFoundError(f"PDF not found: {path}")
            pdfs.append(path.resolve())
        return pdfs
    out = root / "output"
    return sorted(p for p in out.glob("*.pdf") if p.is_file())


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(
        description="Render output/*.pdf pages to white-background showcase PNGs.",
    )
    parser.add_argument(
        "--dpi",
        type=float,
        default=DEFAULT_DPI,
        help=f"render resolution (default {DEFAULT_DPI:g})",
    )
    parser.add_argument(
        "--out",
        type=Path,
        default=None,
        help="output directory (default: frontend/src/assets/showcase)",
    )
    parser.add_argument(
        "--pdf",
        action="append",
        default=None,
        type=Path,
        help="render only this PDF (repeatable); default: all output/*.pdf",
    )
    args = parser.parse_args()

    if args.dpi <= 0:
        print("--dpi must be positive", file=sys.stderr)
        return 1

    out_dir = args.out
    if out_dir is None:
        out_dir = root / "frontend" / "src" / "assets" / "showcase"
    elif not out_dir.is_absolute():
        out_dir = root / out_dir
    out_dir.mkdir(parents=True, exist_ok=True)

    try:
        pdfs = collect_pdfs(root, args.pdf)
    except FileNotFoundError as err:
        print(err, file=sys.stderr)
        return 1

    if not pdfs:
        print("no PDFs found under output/", file=sys.stderr)
        return 1

    total_pages = 0
    for pdf in pdfs:
        pages = render_pdf(pdf, out_dir, args.dpi)
        total_pages += pages
        print(f"{pdf.name}: {pages} page(s)")

    print(f"wrote {total_pages} PNG(s) to {out_dir.relative_to(root)}")

    # Update WebP thumbnails if Pillow is installed
    try:
        from generate_showcase_thumbs import generate_all_thumbs
        print("Generating WebP thumbnails...")
        t_count, t_orig, t_thumb = generate_all_thumbs(out_dir)
        print(f"Updated {t_count} thumbnail(s) in {out_dir / 'thumbs'}")
    except ImportError:
        # If running from another working directory or Pillow not available
        try:
            sys.path.insert(0, str(Path(__file__).resolve().parent))
            from generate_showcase_thumbs import generate_all_thumbs
            print("Generating WebP thumbnails...")
            t_count, t_orig, t_thumb = generate_all_thumbs(out_dir)
            print(f"Updated {t_count} thumbnail(s) in {out_dir / 'thumbs'}")
        except Exception:
            print("Tip: Run `python3 scripts/generate_showcase_thumbs.py` to regenerate WebP thumbnails.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

