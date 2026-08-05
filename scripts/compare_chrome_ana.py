#!/usr/bin/env python3
"""Phase 11 compare harness: structure metrics vs output/chrome_ana.pdf."""
from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import fitz  # PyMuPDF
except ImportError:
    print("need pymupdf: pip install pymupdf", file=sys.stderr)
    sys.exit(2)


def metrics(path: Path) -> dict:
    doc = fitz.open(path)
    page = doc[0]
    under_flag = 0
    spans = 0
    for b in page.get_text("dict")["blocks"]:
        for line in b.get("lines", []):
            for s in line.get("spans", []):
                spans += 1
                if s.get("flags", 0) & 4:
                    under_flag += 1
    horiz = 0
    for d in page.get_drawings():
        for item in d.get("items", []):
            if item[0] == "l" and abs(item[1].y - item[2].y) < 0.5 and abs(item[1].x - item[2].x) > 3:
                horiz += 1
    return {
        "pages": doc.page_count,
        "p1_links": len(page.get_links()),
        "p1_spans": spans,
        "p1_underline_flags": under_flag,
        "p1_horiz_lines": horiz,
        "bytes": path.stat().st_size,
    }


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    chrome = root / "output" / "chrome_ana.pdf"
    ours = root / "output" / "wiki-ana-de-armas.pdf"
    if not chrome.exists() or not ours.exists():
        print("missing chrome_ana.pdf or wiki-ana-de-armas.pdf", file=sys.stderr)
        return 1
    c, o = metrics(chrome), metrics(ours)
    print("chrome", c)
    print("ours  ", o)
    print(f"page_ratio ours/chrome = {o['pages']/max(c['pages'],1):.2f}")
    # Soft gates (tighten as fidelity improves)
    ok = True
    if o["p1_horiz_lines"] < 10 and o["p1_underline_flags"] < 5:
        print("FAIL: links not visibly underlined on page 1")
        ok = False
    if o["pages"] > c["pages"] * 5:
        print(f"WARN: page count {o['pages']} >> chrome {c['pages']}")
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
