#!/usr/bin/env python3
"""Generate WebP thumbnails for frontend showcase PNG images.

Scans frontend/src/assets/showcase/*.png and creates ~400px wide WebP
thumbnails in frontend/src/assets/showcase/thumbs/*.webp maintaining aspect ratio.

Requires Pillow:
  pip install Pillow

Usage:
  python3 scripts/generate_showcase_thumbs.py
  python3 scripts/generate_showcase_thumbs.py --width 400 --quality 82
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("need Pillow: pip install Pillow", file=sys.stderr)
    sys.exit(2)

DEFAULT_WIDTH = 400
DEFAULT_QUALITY = 82


def generate_thumbnail(
    src_path: Path,
    dest_path: Path,
    target_width: int = DEFAULT_WIDTH,
    quality: int = DEFAULT_QUALITY,
) -> tuple[int, int]:
    """Generate a WebP thumbnail for a single image.
    Returns (orig_size_bytes, thumb_size_bytes).
    """
    with Image.open(src_path) as img:
        orig_w, orig_h = img.size
        if orig_w > target_width:
            target_h = max(1, round(orig_h * (target_width / orig_w)))
            resample_filter = getattr(Image, 'Resampling', Image).LANCZOS
            thumb = img.resize((target_width, target_h), resample_filter)
        else:
            thumb = img.copy()

        # Handle color modes
        if thumb.mode in ('RGBA', 'LA') or (thumb.mode == 'P' and 'transparency' in thumb.info):
            bg = Image.new('RGB', thumb.size, (255, 255, 255))
            if thumb.mode != 'RGBA':
                thumb = thumb.convert('RGBA')
            bg.paste(thumb, mask=thumb.split()[3])
            thumb = bg
        elif thumb.mode != 'RGB':
            thumb = thumb.convert('RGB')

        dest_path.parent.mkdir(parents=True, exist_ok=True)
        thumb.save(dest_path, 'WEBP', quality=quality, method=6)

    orig_size = src_path.stat().st_size
    thumb_size = dest_path.stat().st_size
    return orig_size, thumb_size


def generate_all_thumbs(
    assets_dir: Path,
    thumbs_dir: Path | None = None,
    target_width: int = DEFAULT_WIDTH,
    quality: int = DEFAULT_QUALITY,
    clean_stale: bool = True,
) -> tuple[int, int, int]:
    """Generate thumbnails for all PNGs in assets_dir.
    Returns (count, total_orig_bytes, total_thumb_bytes).
    """
    if thumbs_dir is None:
        thumbs_dir = assets_dir / "thumbs"
    thumbs_dir.mkdir(parents=True, exist_ok=True)

    png_files = sorted(
        p for p in assets_dir.glob("*.png")
        if p.is_file() and p.parent == assets_dir
    )

    valid_stems = {p.stem for p in png_files}
    if clean_stale:
        for webp_file in thumbs_dir.glob("*.webp"):
            if webp_file.stem not in valid_stems:
                webp_file.unlink()
                print(f"  removed stale thumbnail: {webp_file.name}")

    total_orig = 0
    total_thumb = 0
    for png in png_files:
        dest = thumbs_dir / f"{png.stem}.webp"
        orig_b, thumb_b = generate_thumbnail(png, dest, target_width, quality)
        total_orig += orig_b
        total_thumb += thumb_b

    return len(png_files), total_orig, total_thumb


def format_bytes(b: int) -> str:
    if b >= 1_000_000:
        return f"{b / 1_000_000:.2f} MB"
    if b >= 1_000:
        return f"{b / 1_000:.1f} KB"
    return f"{b} B"


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(
        description="Generate WebP thumbnails for frontend showcase PNG images.",
    )
    parser.add_argument(
        "--width",
        type=int,
        default=DEFAULT_WIDTH,
        help=f"thumbnail width in px (default {DEFAULT_WIDTH})",
    )
    parser.add_argument(
        "--quality",
        type=int,
        default=DEFAULT_QUALITY,
        help=f"WebP quality 1-100 (default {DEFAULT_QUALITY})",
    )
    parser.add_argument(
        "--showcase-dir",
        type=Path,
        default=None,
        help="showcase assets directory (default: frontend/src/assets/showcase)",
    )
    parser.add_argument(
        "--thumbs-dir",
        type=Path,
        default=None,
        help="thumbs output directory (default: frontend/src/assets/showcase/thumbs)",
    )
    args = parser.parse_args()

    showcase_dir = args.showcase_dir
    if showcase_dir is None:
        showcase_dir = root / "frontend" / "src" / "assets" / "showcase"
    elif not showcase_dir.is_absolute():
        showcase_dir = root / showcase_dir

    thumbs_dir = args.thumbs_dir
    if thumbs_dir is not None and not thumbs_dir.is_absolute():
        thumbs_dir = root / thumbs_dir

    if not showcase_dir.is_dir():
        print(f"Showcase directory not found: {showcase_dir}", file=sys.stderr)
        return 1

    count, orig_bytes, thumb_bytes = generate_all_thumbs(
        showcase_dir,
        thumbs_dir,
        target_width=args.width,
        quality=args.quality,
    )

    ratio = (thumb_bytes / orig_bytes * 100) if orig_bytes > 0 else 0
    savings = 100 - ratio
    print(
        f"Generated {count} WebP thumbnails:\n"
        f"  Original:   {format_bytes(orig_bytes)}\n"
        f"  Thumbnails: {format_bytes(thumb_bytes)} ({ratio:.1f}% of original, {savings:.1f}% reduction)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
