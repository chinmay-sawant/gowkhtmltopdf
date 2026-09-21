#!/usr/bin/env python3
"""Compare two PDFs page by page: size, ink bbox, drawings, text, images, pixels.

Usage:
    python3 compare_pdfs.py A.pdf B.pdf [--dpi 150] [--outdir DIR] [--max-rows 15]
                                        [--pixel-threshold 1.0]

Prints one block per page, then a RESULT line. Exit code 0 when page count,
page sizes (within 0.1pt), drawing, text and image signatures, and pixel
deltas all pass. Exit code 1 when anything differs. Exit code 2 on bad input.

Matching is signature-based: each drawing becomes (type, fill, stroke, rect),
each text span becomes (text, size, direction, bbox), and each image becomes
(width_px, height_px, bbox), quantized to a quarter point. Font subset ids are
ignored on purpose; a PDF writer that embeds time.Now() can still produce
matching signatures.

Pixels: each page is rasterized once in grayscale at --dpi (default 150) and
reused for the ink bbox and the pixel delta. A pixel counts as changed when
its grayscale value differs by more than 24 of 255 (PIXEL_DIFF_LIMIT). A page
fails when more than 1% of its pixels changed (--pixel-threshold,
DEFAULT_PIXEL_THRESHOLD). The pixel check catches color changes and raster
differences that get_drawings() cannot see; small embedded images that barely
move the pixel count still fail through the image rows. Without Pillow the
pixel check is skipped with a warning and RESULT cannot be MATCH.
"""

import argparse
import collections
import os
import sys

try:
    import fitz  # PyMuPDF
except ImportError:
    sys.exit("PyMuPDF (fitz) is required: pip install pymupdf")

try:
    from PIL import Image, ImageChops
except ImportError:
    Image = None
    ImageChops = None

PIXEL_DIFF_LIMIT = 24  # grayscale value delta that counts as a changed pixel
DEFAULT_PIXEL_THRESHOLD = 1.0  # percent of changed pixels that fails a page
SIZE_TOLERANCE = 0.1  # pt; a larger delta on either side is a size mismatch


def rgb(color):
    if color is None:
        return None
    return tuple(round(float(v), 3) for v in color[:3])


def q(value):
    return round(float(value) * 4) / 4


def page_size(page):
    rect = page.rect
    return (round(rect.width, 2), round(rect.height, 2))


def drawing_rows(page):
    rows = []
    for drawing in page.get_drawings():
        rect = drawing["rect"]
        rows.append(
            (
                drawing.get("type"),
                rgb(drawing.get("fill")),
                rgb(drawing.get("color")),
                (q(rect.x0), q(rect.y0), q(rect.x1), q(rect.y1)),
            )
        )
    return rows


def text_rows(page):
    rows = []
    for block in page.get_text("dict").get("blocks", []):
        for line in block.get("lines", []):
            for span in line.get("spans", []):
                text = span.get("text", "")
                if not text.strip():
                    continue
                box = span["bbox"]
                direction = span.get("dir", (1.0, 0.0))
                rows.append(
                    (
                        text,
                        round(float(span["size"]), 2),
                        tuple(round(float(v), 3) for v in direction),
                        (q(box[0]), q(box[1]), q(box[2]), q(box[3])),
                    )
                )
    return rows


def image_rows(page):
    """Image XObjects as (width_px, height_px, quantized bbox) rows.

    get_drawings() cannot see images; these rows make small images visible
    even when they barely move the pixel delta."""
    rows = []
    for info in page.get_image_info():
        box = info["bbox"]
        rows.append(
            (
                int(info.get("width", 0)),
                int(info.get("height", 0)),
                (q(box[0]), q(box[1]), q(box[2]), q(box[3])),
            )
        )
    return rows


def font_names(page):
    """Sorted base font names, informational: helps classify font substitution."""
    names = set()
    for font in page.get_fonts():
        if len(font) > 3 and font[3]:
            names.add(str(font[3]))
    return sorted(names)


def raster(page, dpi):
    """One grayscale PIL image per page, reused for ink bbox and pixel delta."""
    if Image is None:
        return None
    pixmap = page.get_pixmap(dpi=dpi, colorspace=fitz.csGRAY)
    return Image.frombytes("L", (pixmap.width, pixmap.height), pixmap.samples)


def ink_from_raster(image, dpi):
    """Ink bbox in points and ink pixel count, from an existing grayscale image."""
    mask = image.point(lambda p: 255 if p < 245 else 0)
    box = mask.getbbox()
    if box is None:
        return ((0.0, 0.0, 0.0, 0.0), 0)
    points = tuple(round(v * 72.0 / dpi, 2) for v in box)
    return (points, mask.histogram()[255])


def pixel_delta(image_a, image_b):
    """Return (percent changed, note). Mismatched rasters are compared on the overlap."""
    note = ""
    if image_a.size != image_b.size:
        width = min(image_a.width, image_b.width)
        height = min(image_a.height, image_b.height)
        note = (
            f"raster size mismatch A={image_a.size} B={image_b.size}, "
            "compared on overlap"
        )
        image_a = image_a.crop((0, 0, width, height))
        image_b = image_b.crop((0, 0, width, height))
    diff = ImageChops.difference(image_a, image_b)
    changed = diff.point(lambda p: 255 if p > PIXEL_DIFF_LIMIT else 0).histogram()[255]
    return 100.0 * changed / (image_a.width * image_a.height), note


def side_diff(rows_a, rows_b, side):
    """Return (matched_count, unmatched Counter) for the requested side."""
    counter_a = collections.Counter(rows_a)
    counter_b = collections.Counter(rows_b)
    matched = sum((counter_a & counter_b).values())
    diff = counter_a - counter_b if side == "A" else counter_b - counter_a
    return matched, diff


def print_rows(label, diff, max_rows):
    if not diff:
        return
    for row, count in list(diff.items())[:max_rows]:
        suffix = f"  x{count}" if count > 1 else ""
        print(f"    {label} {row}{suffix}")
    hidden = len(diff) - max_rows
    if hidden > 0:
        print(f"    {label} ... and {hidden} more distinct rows")


def compare_page(index, page_a, page_b, dpi, outdir, max_rows, threshold):
    """Print one page block. Return per-component mismatch counts."""
    size_a, size_b = page_size(page_a), page_size(page_b)
    print(f"page {index}: size A={size_a} B={size_b}")
    delta_x = round(size_b[0] - size_a[0], 2)
    delta_y = round(size_b[1] - size_a[1], 2)
    size_mismatch = abs(delta_x) > SIZE_TOLERANCE or abs(delta_y) > SIZE_TOLERANCE
    if size_mismatch:
        print(f"  PAGE SIZE DELTA: ({delta_x}, {delta_y}) pt (B minus A)")
    elif size_a != size_b:
        print(
            f"  size within {SIZE_TOLERANCE}pt tolerance: "
            f"delta ({delta_x}, {delta_y}) pt"
        )

    image_a, image_b = raster(page_a, dpi), raster(page_b, dpi)
    pixel_fail, pixel_skipped = 0, False
    if image_a is not None and image_b is not None:
        (box_a, px_a), (box_b, px_b) = (
            ink_from_raster(image_a, dpi),
            ink_from_raster(image_b, dpi),
        )
        print(f"  ink bbox A (pt)={box_a} ink px A={px_a}")
        print(f"  ink bbox B (pt)={box_b} ink px B={px_b}")
        delta, note = pixel_delta(image_a, image_b)
        line = f"  pixel delta: {delta:.2f}% (threshold {threshold:.2f}%)"
        if note:
            line += f" [{note}]"
        print(line)
        pixel_fail = 1 if (note or delta > threshold) else 0
    else:
        pixel_skipped = True
        print("  pixel delta: skipped (Pillow missing)")

    if outdir:
        for side, page in (("A", page_a), ("B", page_b)):
            target = f"{outdir}/{side}-p{index:02d}.png"
            page.get_pixmap(dpi=dpi).save(target)

    print(f"  fonts A={font_names(page_a)} B={font_names(page_b)}")

    draws_a, draws_b = drawing_rows(page_a), drawing_rows(page_b)
    draw_match, draw_only_a = side_diff(draws_a, draws_b, "A")
    _, draw_only_b = side_diff(draws_a, draws_b, "B")
    print(f"  draw rows A={len(draws_a)} B={len(draws_b)} matched={draw_match}")
    print_rows("-", draw_only_a, max_rows)
    print_rows("+", draw_only_b, max_rows)

    texts_a, texts_b = text_rows(page_a), text_rows(page_b)
    text_match, text_only_a = side_diff(texts_a, texts_b, "A")
    _, text_only_b = side_diff(texts_a, texts_b, "B")
    print(f"  text rows A={len(texts_a)} B={len(texts_b)} matched={text_match}")
    print_rows("-", text_only_a, max_rows)
    print_rows("+", text_only_b, max_rows)

    images_a, images_b = image_rows(page_a), image_rows(page_b)
    img_match, img_only_a = side_diff(images_a, images_b, "A")
    _, img_only_b = side_diff(images_a, images_b, "B")
    print(f"  image rows A={len(images_a)} B={len(images_b)} matched={img_match}")
    print_rows("-", img_only_a, max_rows)
    print_rows("+", img_only_b, max_rows)

    return {
        "drawings": sum(draw_only_a.values()) + sum(draw_only_b.values()),
        "text": sum(text_only_a.values()) + sum(text_only_b.values()),
        "images": sum(img_only_a.values()) + sum(img_only_b.values()),
        "size": 1 if size_mismatch else 0,
        "pixel": pixel_fail,
        "pixel_skipped": pixel_skipped,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("pdf_a", help="reference PDF (Chromium)")
    parser.add_argument("pdf_b", help="candidate PDF (gowkhtmltopdf)")
    parser.add_argument(
        "--dpi", type=int, default=150, help="DPI for ink bbox, pixels, and PNGs"
    )
    parser.add_argument("--outdir", help="write A-pNN.png and B-pNN.png here")
    parser.add_argument(
        "--max-rows", type=int, default=15, help="unmatched rows shown per side"
    )
    parser.add_argument(
        "--pixel-threshold",
        type=float,
        default=DEFAULT_PIXEL_THRESHOLD,
        help="percent of changed pixels that fails a page (default 1.0)",
    )
    args = parser.parse_args()

    docs = []
    for path in (args.pdf_a, args.pdf_b):
        try:
            docs.append(fitz.open(path))
        except Exception:
            print(f"cannot open input: {path}", file=sys.stderr)
            for doc in docs:
                doc.close()
            return 2
    doc_a, doc_b = docs

    if args.outdir:
        os.makedirs(args.outdir, exist_ok=True)

    if Image is None:
        print(
            "warning: Pillow missing, pixel check skipped (pip install pillow)",
            file=sys.stderr,
        )

    try:
        if doc_a.page_count != doc_b.page_count:
            print(f"PAGE COUNT: A={doc_a.page_count} B={doc_b.page_count}")

        unmatched = collections.Counter()
        size_mismatches = 0
        pages_over = 0
        pixel_skipped = False
        for index in range(min(doc_a.page_count, doc_b.page_count)):
            page_result = compare_page(
                index + 1,
                doc_a[index],
                doc_b[index],
                args.dpi,
                args.outdir,
                args.max_rows,
                args.pixel_threshold,
            )
            unmatched["drawings"] += page_result["drawings"]
            unmatched["text"] += page_result["text"]
            unmatched["images"] += page_result["images"]
            size_mismatches += page_result["size"]
            pages_over += page_result["pixel"]
            pixel_skipped = pixel_skipped or page_result["pixel_skipped"]

        page_count_delta = abs(doc_a.page_count - doc_b.page_count)
        unmatched_rows = sum(unmatched.values())
        print(
            f"pixel threshold: {args.pixel_threshold:.2f}% of pixels changed by "
            f">{PIXEL_DIFF_LIMIT}/255 at {args.dpi} dpi"
        )
        if page_count_delta == 0 and size_mismatches == 0 and unmatched_rows == 0:
            if pixel_skipped:
                print(
                    "RESULT: DIFFER (page count delta 0, page size mismatches 0, "
                    "0 unmatched signature rows, pixel check skipped: Pillow missing)"
                )
                return 1
            if pages_over == 0:
                print(
                    "RESULT: MATCH (page count, page sizes, drawing, text and "
                    "image signatures, and pixel deltas are equal)"
                )
                return 0
            print(
                f"RESULT: DIFFER (pixel-only: {pages_over} pages over threshold, "
                "0 unmatched rows)"
            )
            return 1
        print(
            f"RESULT: DIFFER (page count delta {page_count_delta}, "
            f"page size mismatches {size_mismatches}, "
            f"unmatched signature rows {unmatched_rows} "
            f"(drawings {unmatched['drawings']}, text {unmatched['text']}, "
            f"images {unmatched['images']}), "
            f"{pages_over} pages over pixel threshold)"
        )
        return 1
    finally:
        doc_a.close()
        doc_b.close()


if __name__ == "__main__":
    sys.exit(main())
