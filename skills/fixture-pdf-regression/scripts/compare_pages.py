#!/usr/bin/env python3
"""Compare one page region between two PDFs (good vs current).

Usage:
  python3 compare_pages.py GOOD.pdf CUR.pdf --page 3 --out /tmp/cmp \\
    --words Surface,ImageConverter,Contract --y-min 640
"""

from __future__ import annotations

import argparse
from collections import Counter
from pathlib import Path

import fitz


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("good")
    ap.add_argument("current")
    ap.add_argument("--page", type=int, required=True, help="1-based page number")
    ap.add_argument("--out", type=Path, required=True)
    ap.add_argument("--words", default="", help="comma-separated words to locate")
    ap.add_argument("--y-min", type=float, default=0.0)
    ap.add_argument("--scale", type=float, default=2.0)
    args = ap.parse_args()

    args.out.mkdir(parents=True, exist_ok=True)
    needles = [w for w in args.words.split(",") if w.strip()]

    for label, path in (("good", args.good), ("cur", args.current)):
        doc = fitz.open(path)
        page = doc[args.page - 1]
        pix = page.get_pixmap(matrix=fitz.Matrix(args.scale, args.scale), alpha=False)
        png = args.out / f"{label}-p{args.page:02d}.png"
        pix.save(str(png))
        print(f"=== {label} pages={doc.page_count} wrote {png} ===")

        if needles:
            print("--- word hits ---")
            for w in page.get_text("words"):
                if w[1] < args.y_min:
                    continue
                if w[4] in needles:
                    print(f"  ({w[0]:.1f},{w[1]:.1f}) {w[4]}")

        print("--- fills (y>=y-min, w>40) ---")
        for d in page.get_drawings():
            fill = d.get("fill")
            rect = d.get("rect")
            if not fill or not rect or rect.y0 < args.y_min or rect.width < 40:
                continue
            rgb = tuple(round(c, 3) for c in fill[:3])
            print(
                f"  fill={rgb} ({rect.x0:.1f},{rect.y0:.1f}) "
                f"{rect.width:.1f}x{rect.height:.1f}"
            )

        kinds = Counter()
        for d in page.get_drawings():
            for it in d.get("items") or []:
                kinds[it[0]] += 1
        print(f"--- drawing kinds {dict(kinds)} ---")
        doc.close()


if __name__ == "__main__":
    main()
