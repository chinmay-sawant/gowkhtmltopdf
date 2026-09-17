#!/usr/bin/env python3
"""Per-page forensics for a PDF artifact.

Built for the v0.2.7 LearnCpp drill (plans/0.2.7/learncpp) to compare a PDF
before and after engine fixes with the same numbers, and reusable for future
real-site drills.

For every page it reports:
  - word count and text bounding box
  - non-white pixel coverage at a fixed DPI (visible ink)
  - drawing count, link annotation count
  - embedded font base names used by the page

It flags pages that contain text spans with (near) zero rendered ink while
the text layer still reports their words: the signature of text painted under
an opaque fill (the LearnCpp 34/37 blank pages defect). The check is per span
on purpose. Page-level coverage alone false-positived on genuinely sparse
pages (a TOC tail or a footer-only page carries little ink overall but every
span is rendered), so coverage is reported for context and does not decide
the flag.

Usage:
  python3 scripts/pdf_page_forensics.py file.pdf [--json out.json] [--dpi 80]
  python3 scripts/pdf_page_forensics.py file.pdf --compare baseline.json
  python3 scripts/pdf_page_forensics.py file.pdf --thumb-dir /tmp/shots --contact sheet.png

--compare prints a delta summary against an earlier --json report: page count,
total words, total link annots, font list diff, suspicious-page sets, and
per-page coverage min/median/max. Deltas are reported, never asserted; a full
pipeline run is expected to differ from the --no-images baseline.
"""
from __future__ import annotations

import argparse
import json
import statistics
import sys
from pathlib import Path

try:
    import fitz  # PyMuPDF
except ImportError:
    print("need pymupdf: pip install pymupdf", file=sys.stderr)
    raise SystemExit(2)

# A pixel counts as ink when any RGB channel is below this, for both the page
# coverage raster and the per-span check.
INK_CHANNEL = 250

# A page is flagged only when a text span's own region shows (near) zero
# rendered ink while the text layer still reports its characters: the real
# covered-text signal. The old page-level test (>= 20 words and coverage <
# 0.02) false-positived on genuinely sparse pages, where every span is
# rendered but the page as a whole carries little ink. Coverage stays in the
# report for context; it no longer decides the flag.
SPAN_INK_FLOOR = 0.01  # non-white fraction below this counts as no ink
SPAN_MIN_PIXELS = 4  # skip spans that rasterize to a sub-glyph sliver

# Keys a comparable report must carry. Both this script's --json output and
# the committed LearnCpp evidence files use them.
COMPARE_KEYS = (
    "pages",
    "total_words",
    "total_link_annots",
    "fonts",
    "suspicious_pages",
    "per_page",
)


def coverage_stats(report: dict) -> tuple[float, float, float] | None:
    """min/median/max visible coverage across a report's pages."""
    values = [row["coverage"] for row in report["per_page"]]
    if not values:
        return None
    return min(values), statistics.median(values), max(values)


def load_baseline(path: Path) -> dict | None:
    """Read and validate a previous report; None with a message on failure."""
    try:
        report = json.loads(path.read_text())
    except (OSError, json.JSONDecodeError) as exc:
        print(f"compare: baseline unreadable: {path}: {exc}", file=sys.stderr)
        return None
    missing = [key for key in COMPARE_KEYS if key not in report]
    if missing:
        print(
            f"compare: baseline missing keys ({', '.join(missing)}): {path}",
            file=sys.stderr,
        )
        return None
    return report


def _delta(current: int, baseline: int) -> str:
    diff = current - baseline
    return "no change" if diff == 0 else f"{diff:+d}"


def _page_list(pages: list[int]) -> str:
    if not pages:
        return "(none)"
    shown = ", ".join(str(p) for p in pages[:20])
    if len(pages) > 20:
        shown += f" (+{len(pages) - 20} more)"
    return shown


def print_comparison(current: dict, baseline: dict, baseline_path: Path) -> None:
    """Print current-vs-baseline deltas; never asserts equality."""
    current_suspicious = set(current["suspicious_pages"])
    baseline_suspicious = set(baseline["suspicious_pages"])
    entered = sorted(current_suspicious - baseline_suspicious)
    left = sorted(baseline_suspicious - current_suspicious)

    fonts_added = sorted(set(current["fonts"]) - set(baseline["fonts"]))
    fonts_removed = sorted(set(baseline["fonts"]) - set(current["fonts"]))

    print(f"## comparison vs baseline: {baseline_path}")
    print(
        f"{'pages:':<18}{baseline['pages']} -> {current['pages']} "
        f"({_delta(current['pages'], baseline['pages'])})"
    )
    print(
        f"{'total words:':<18}{baseline['total_words']} -> {current['total_words']} "
        f"({_delta(current['total_words'], baseline['total_words'])})"
    )
    print(
        f"{'link annots:':<18}{baseline['total_link_annots']} -> "
        f"{current['total_link_annots']} "
        f"({_delta(current['total_link_annots'], baseline['total_link_annots'])})"
    )
    if not fonts_added and not fonts_removed:
        print(f"{'fonts:':<18}identical: {', '.join(current['fonts']) or '(none)'}")
    else:
        print(f"{'fonts added:':<18}{', '.join(fonts_added) or '(none)'}")
        print(f"{'fonts removed:':<18}{', '.join(fonts_removed) or '(none)'}")
    print(
        f"{'suspicious pages:':<18}{len(baseline_suspicious)} -> "
        f"{len(current_suspicious)} "
        f"({_delta(len(current_suspicious), len(baseline_suspicious))})"
    )
    print(f"  entered suspicious set: {_page_list(entered)}")
    print(f"  left suspicious set: {_page_list(left)}")
    print("coverage min/median/max:")
    for label, report in (("baseline", baseline), ("current", current)):
        stats = coverage_stats(report)
        if stats is None:
            print(f"  {label}: (no pages)")
        else:
            print(f"  {label}: {stats[0]:.5f} / {stats[1]:.5f} / {stats[2]:.5f}")


def spans_missing_ink(page: fitz.Page, pix: fitz.Pixmap) -> int:
    """Count text spans whose rendered bbox shows (near) zero non-white ink.

    `pix` is a raster of `page`. Whitespace-only spans are skipped: they sit
    on the page background and a blank region around a space is not evidence
    of covered text. The bbox is inset by one pixel so anti-aliased edges of a
    neighbouring glyph cannot lend ink to a covered span.
    """
    scale = pix.width / page.rect.width if page.rect.width else 0.0
    if scale <= 0:
        return 0
    samples = memoryview(pix.samples)
    step = pix.n
    missing = 0
    for block in page.get_text("dict")["blocks"]:
        if block.get("type") != 0:
            continue
        for line in block.get("lines", []):
            for span in line.get("spans", []):
                if not any(ch.isalnum() for ch in span.get("text", "")):
                    continue
                x0, y0, x1, y1 = span["bbox"]
                ix0 = max(int(x0 * scale) + 1, 0)
                iy0 = max(int(y0 * scale) + 1, 0)
                ix1 = min(int(x1 * scale), pix.width)
                iy1 = min(int(y1 * scale), pix.height)
                area = (ix1 - ix0) * (iy1 - iy0)
                if area < SPAN_MIN_PIXELS:
                    continue
                lit = 0
                for y in range(iy0, iy1):
                    base = y * pix.width * step
                    for x in range(ix0, ix1):
                        i = base + x * step
                        if (
                            samples[i] < INK_CHANNEL
                            or samples[i + 1] < INK_CHANNEL
                            or samples[i + 2] < INK_CHANNEL
                        ):
                            lit += 1
                if lit / area < SPAN_INK_FLOOR:
                    missing += 1
    return missing


def page_metrics(page: fitz.Page, dpi: int) -> tuple[dict, int]:
    """Per-page metrics and the count of text spans with no visible ink."""
    words = page.get_text("words")
    drawings = page.get_drawings()
    # page.annots() misses the writer's link objects in this PDF; get_links()
    # parses them reliably, so links come from there and annots stays as the
    # raw supported-annotation count.
    annots = len(list(page.annots())) if page.first_annot else 0
    links = len(page.get_links())
    fonts = sorted({f[3] for f in page.get_fonts(full=True)})

    if words:
        xs0 = min(w[0] for w in words)
        ys0 = min(w[1] for w in words)
        xs1 = max(w[2] for w in words)
        ys1 = max(w[3] for w in words)
        text_box = [round(xs0, 1), round(ys0, 1), round(xs1, 1), round(ys1, 1)]
    else:
        text_box = None

    pix = page.get_pixmap(dpi=dpi)
    total = pix.width * pix.height
    lit = 0
    samples = memoryview(pix.samples)
    step = pix.n
    for i in range(0, total * step, step):
        if (
            samples[i] < INK_CHANNEL
            or samples[i + 1] < INK_CHANNEL
            or samples[i + 2] < INK_CHANNEL
        ):
            lit += 1
    coverage = round(lit / total, 5) if total else 0.0

    missing_ink = spans_missing_ink(page, pix)

    row = {
        "page": page.number + 1,
        "words": len(words),
        "text_box": text_box,
        "coverage": coverage,
        "drawings": len(drawings),
        "annots": annots,
        "link_annots": links,
        "fonts": fonts,
        "page_size": [round(page.rect.width, 1), round(page.rect.height, 1)],
    }
    return row, missing_ink


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("pdf", type=Path)
    ap.add_argument("--json", type=Path, help="write the full report as JSON")
    ap.add_argument(
        "--compare",
        type=Path,
        metavar="BASELINE.json",
        help="previous report JSON; print a delta summary against it",
    )
    ap.add_argument(
        "--dpi",
        type=int,
        default=80,
        help="raster DPI for coverage and the per-span ink check (default 80)",
    )
    ap.add_argument("--thumb-dir", type=Path, help="write page PNGs here")
    ap.add_argument("--contact", type=Path, help="write a contact sheet PNG here")
    args = ap.parse_args()

    if not args.pdf.exists():
        print(f"missing PDF: {args.pdf}", file=sys.stderr)
        return 2

    if args.thumb_dir:
        args.thumb_dir.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(args.pdf)
    metrics = [page_metrics(page, args.dpi) for page in doc]
    rows = [row for row, _ in metrics]

    if args.thumb_dir:
        for row in rows:
            pix = doc[row["page"] - 1].get_pixmap(dpi=args.dpi)
            pix.save(args.thumb_dir / f"page{row['page']:03d}.png")

    if args.contact:
        try:
            from PIL import Image
        except ImportError:
            print("contact sheet needs pillow: pip install pillow", file=sys.stderr)
            return 2
        thumbs = []
        for row in rows:
            pix = doc[row["page"] - 1].get_pixmap(dpi=40)
            thumbs.append(Image.frombytes("RGB", (pix.width, pix.height), pix.samples))
        if thumbs:
            cols = 8
            w, h = thumbs[0].size
            rows_n = (len(thumbs) + cols - 1) // cols
            sheet = Image.new("RGB", (cols * w, rows_n * h), "white")
            for i, thumb in enumerate(thumbs):
                sheet.paste(thumb, ((i % cols) * w, (i // cols) * h))
            args.contact.parent.mkdir(parents=True, exist_ok=True)
            sheet.save(args.contact)

    total_words = sum(r["words"] for r in rows)
    total_links = sum(r["link_annots"] for r in rows)
    suspicious = [row["page"] for row, missing_ink in metrics if missing_ink]
    all_fonts = sorted({f for r in rows for f in r["fonts"]})
    current = {
        "pdf": str(args.pdf),
        "pages": len(rows),
        "total_words": total_words,
        "total_link_annots": total_links,
        "fonts": all_fonts,
        "suspicious_pages": suspicious,
        "per_page": rows,
    }

    print(f"# PDF forensics: {args.pdf}")
    print()
    print(f"pages: {len(rows)} | words: {total_words} | link annots: {total_links}")
    print(f"fonts: {', '.join(all_fonts) if all_fonts else '(none)'}")
    print(f"low-ink pages with a text layer (possible covered text): {len(suspicious)}")
    if suspicious:
        shown = ", ".join(str(p) for p in suspicious[:20])
        more = "" if len(suspicious) <= 20 else f" (+{len(suspicious) - 20} more)"
        print(f"  pages: {shown}{more}")
    print()
    print("| page | words | coverage | drawings | links | main font |")
    print("|------|-------|----------|----------|-------|-----------|")
    for r in rows:
        main_font = r["fonts"][0] if r["fonts"] else "-"
        print(
            f"| {r['page']} | {r['words']} | {r['coverage']:.3f} | "
            f"{r['drawings']} | {r['link_annots']} | {main_font} |"
        )

    compare_failed = False
    if args.compare:
        baseline = load_baseline(args.compare)
        if baseline is None:
            compare_failed = True
        else:
            print()
            print_comparison(current, baseline, args.compare)

    if args.json:
        args.json.parent.mkdir(parents=True, exist_ok=True)
        args.json.write_text(json.dumps(current, indent=2))

    return 2 if compare_failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
