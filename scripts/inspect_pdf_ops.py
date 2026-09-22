#!/usr/bin/env python3
"""Dump the page operations of a PDF in PDF user space.

Prints text spans (origin, size, color, font), drawings (stroked lines and
filled rectangles with color and width), and image boxes. Coordinates use the
PDF convention: points, origin at the bottom-left of the page, y grows up.
PyMuPDF reports y downward from the top, so every y printed here is flipped.

This is the independent ruler for the canonical output checks: measure a
committed sample with it, then pin the measured values in the Go test that
compares the sample with a fresh conversion. Needs PyMuPDF:
`pip install pymupdf`.

Usage:
    python3 scripts/inspect_pdf_ops.py output/fixture-01-simple-invoice.pdf
    python3 scripts/inspect_pdf_ops.py output/fixture-01-simple-invoice.pdf --page 1
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import fitz  # PyMuPDF
except ImportError:
    print("need pymupdf: pip install pymupdf", file=sys.stderr)
    raise SystemExit(2)


def pdf_y(page: "fitz.Page", y_down: float) -> float:
    return page.rect.height - y_down


def print_text(page: "fitz.Page") -> None:
    for block in page.get_text("dict")["blocks"]:
        if block["type"] != 0:
            continue
        for line in block["lines"]:
            for span in line["spans"]:
                color = span["color"]
                red = (color >> 16) & 255
                green = (color >> 8) & 255
                blue = color & 255
                x, y = span["origin"]
                print(
                    f"text x={x:.3f} y={pdf_y(page, y):.3f} size={span['size']:.2f} "
                    f"color=#{color:06x} rgb=({red},{green},{blue}) "
                    f"font={span['font']} text={span['text']!r}"
                )


def print_drawings(page: "fitz.Page") -> None:
    for drawing in page.get_drawings():
        rect = drawing["rect"]
        print(
            f"draw type={drawing['type']} "
            f"rect=({rect.x0:.3f},{pdf_y(page, rect.y1):.3f},"
            f"{rect.x1:.3f},{pdf_y(page, rect.y0):.3f}) "
            f"width={drawing.get('width')} color={drawing.get('color')} "
            f"fill={drawing.get('fill')}"
        )
        for item in drawing["items"]:
            if item[0] == "l":
                first, second = item[1], item[2]
                print(
                    f"  line ({first.x:.3f},{pdf_y(page, first.y):.3f}) -> "
                    f"({second.x:.3f},{pdf_y(page, second.y):.3f})"
                )
            elif item[0] == "re":
                box = item[1]
                print(
                    f"  rect ({box.x0:.3f},{pdf_y(page, box.y1):.3f}) "
                    f"w={box.width:.3f} h={box.height:.3f}"
                )
            else:
                print(f"  item {item[0]}")


def print_images(page: "fitz.Page") -> None:
    for image in page.get_images(full=True):
        box = page.get_image_bbox(image)
        print(
            f"image xref={image[0]} w={image[2]} h={image[3]} "
            f"box=({box.x0:.3f},{pdf_y(page, box.y1):.3f}) "
            f"w={box.width:.3f} h={box.height:.3f}"
        )


def inspect(pdf: Path, page_number: int) -> int:
    doc = fitz.open(pdf)
    print(f"{pdf} pages={doc.page_count}")

    for index, page in enumerate(doc):
        if page_number and index + 1 != page_number:
            continue
        print(f"page={index + 1} size={page.rect.width:.3f}x{page.rect.height:.3f}")
        print_text(page)
        print_drawings(page)
        print_images(page)

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("pdf", type=Path)
    parser.add_argument(
        "--page", type=int, default=0, help="1-based page to inspect; 0 means all"
    )
    args = parser.parse_args()
    return inspect(args.pdf, args.page)


if __name__ == "__main__":
    raise SystemExit(main())
