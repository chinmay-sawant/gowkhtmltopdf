#!/usr/bin/env python3
"""Generate fixture-60/61/62 HTML audits for Implemented CSS properties.

Reads plans/0.2.6/catalog/implemented-fixture-split.json and writes
testdata/golden/fixture-6{0,1,2}-implemented-props-{a,b,c}.html

Usage:
  python3 scripts/gen-implemented-prop-fixtures.py          # all three
  python3 scripts/gen-implemented-prop-fixtures.py A        # one slice
"""

from __future__ import annotations

import html
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SPLIT = ROOT / "plans/0.2.6/catalog/implemented-fixture-split.json"
OUT_DIR = ROOT / "testdata/golden"

TINY_PNG = (
    "data:image/png;base64,"
    "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAEklEQVR42mP4z8CABzGMSmNDALfKY53W1e90AAAAAElFTkSuQmCC"
)


def effect_html(prop: str, kind: str, legacy_of: str | None) -> str:
    """Return inner HTML for the Effect cell; must apply the property somehow."""
    p = prop
    target = legacy_of or p

    # Keep demos small and print-safe.
    if p in ("display",) or target == "display":
        return (
            f'<div style="{p}:flex;gap:4px;border:1px solid #336;padding:4px;">'
            f'<span style="background:#cde;padding:2px 6px;">A</span>'
            f'<span style="background:#cde;padding:2px 6px;">B</span></div>'
        )
    if p == "visibility":
        return (
            '<span style="visibility:hidden;background:#fcc;padding:2px 6px;">hidden</span>'
            '<span> (space kept)</span>'
        )
    if p == "opacity" or target == "opacity":
        return f'<span style="{p}:0.45;background:#8af;padding:4px 8px;">opacity demo</span>'
    if p == "color" or target in ("color", "-webkit-text-fill-color"):
        return f'<span style="{p}:#0b5;font-weight:bold;">colored text</span>'
    if p.startswith("background") or target.startswith("background"):
        if p == "background":
            return (
                '<div style="background:#eef url(logo.png) no-repeat;'
                'height:36px;border:1px solid #99a;"></div>'
            )
        if p == "background-attachment" or target == "background-attachment":
            return (
                '<div style="background-image:url(logo.png);background-repeat:no-repeat;'
                f'{p}:local;background-color:#eef;height:36px;border:1px solid #99a;overflow:auto;">'
                '<div style="height:56px;font-size:8pt;">tall content scrolls over a local background</div></div>'
            )
        if "blend" in p:
            return (
                f'<div style="background-image:url(logo.png),linear-gradient(#6af,#f80);'
                f'{p}:multiply;background-size:contain,cover;height:36px;border:1px solid #99a;"></div>'
            )
        if "clip" in p:
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:content-box;'
                f'padding:10px;background-color:#eef;height:44px;border:4px dashed #246;"></div>'
            )
        if p == "background-color" or target == "background-color":
            return f'<div style="{p}:#6af;height:28px;border:1px solid #336;"></div>'
        if "image" in p or p == "background":
            return (
                '<div style="background-image:linear-gradient(#6af,#f80);'
                'height:36px;border:1px solid #99a;"></div>'
            )
        if "origin" in p:
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:content-box;'
                f'padding:10px;background-color:#eef;height:40px;border:1px solid #99a;"></div>'
            )
        if p == "background-position":
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:right bottom;'
                f'background-color:#eef;height:36px;border:1px solid #99a;"></div>'
            )
        if p == "background-position-block":
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:100%;'
                f'background-color:#eef;height:52px;border:1px solid #99a;"></div>'
            )
        if p == "background-position-inline":
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:100%;'
                f'background-color:#eef;height:36px;border:1px solid #99a;"></div>'
            )
        if p == "background-position-x":
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:center;'
                f'background-color:#eef;height:36px;border:1px solid #99a;"></div>'
            )
        if p == "background-position-y":
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:center;'
                f'background-color:#eef;height:52px;border:1px solid #99a;"></div>'
            )
        if p == "background-repeat":
            return (
                f'<div style="background-image:url(logo.png);{p}:repeat;background-size:16px 16px;'
                f'background-color:#eef;height:48px;border:1px solid #99a;"></div>'
            )
        if p in ("background-repeat-block", "background-repeat-inline"):
            height = "height:64px;" if p == "background-repeat-inline" else "height:36px;"
            return (
                f'<div style="background-image:url(logo.png);{p}:no-repeat;background-size:16px 16px;'
                f'background-color:#eef;{height}border:1px solid #99a;"></div>'
            )
        if p in ("background-repeat-x", "background-repeat-y"):
            val = "repeat-x" if p == "background-repeat-x" else "repeat-y"
            tall = "" if p == "background-repeat-x" else "height:64px;"
            base = "height:36px;" if not tall else tall
            return (
                f'<div style="background-image:url(logo.png);{p}:{val};background-size:16px 16px;'
                f'background-color:#eef;{base}border:1px solid #99a;"></div>'
            )
        if "size" in p:
            return (
                f'<div style="background-image:url(logo.png);background-repeat:no-repeat;{p}:cover;'
                f'background-color:#eef;height:40px;border:1px solid #99a;"></div>'
            )
        return f'<div style="{p}:#6af;background-color:#eef;height:28px;border:1px solid #336;"></div>'
    if "shadow" in p:
        return f'<div style="{p}:2px 2px 0 #333;background:#ffe;padding:6px 10px;display:inline-block;">shadow</div>'
    if p.startswith("font") or target.startswith("font"):
        style = f"{p}: "
        if "size" in p:
            style += "16pt"
        elif "weight" in p:
            style += "700"
        elif "style" in p:
            style += "italic"
        elif "family" in p or p == "font":
            style += '"Liberation Serif", serif'
        else:
            style += "inherit"
        return f'<span style="{html.escape(style)};">Font sample Aa</span>'
    if p in ("line-height",) or p.startswith("letter-") or p.startswith("word-spacing"):
        return f'<div style="{p}:{"1.9" if p=="line-height" else ("0.2em" if "letter" in p else "0.6em")};border:1px dashed #888;padding:4px;">line spacing sample text that wraps a little</div>'
    if p.startswith("text-align") or p == "text-align":
        return f'<div style="{p}:center;border:1px solid #888;padding:4px;">aligned</div>'
    if p.startswith("text-decoration") or p.startswith("text-underline") or p.startswith("text-emphasis"):
        return f'<span style="{p}:{"underline" if p=="text-decoration" or p.endswith("-line") else ("#c00" if "color" in p else ("2px" if "thickness" in p or "offset" in p else "wavy"))};">decorated</span>'
    if p == "text-transform":
        return f'<span style="{p}:uppercase;">Upper demo</span>'
    if p == "text-indent":
        return f'<div style="{p}:24px;border:1px solid #888;">Indented first line of a short paragraph.</div>'
    if p.startswith("white-space") or p in ("tab-size", "line-break", "overflow-wrap", "word-break", "word-wrap", "hyphens", "hyphenate-character"):
        # Latin-only sample: CJK in Liberation triggers an empty Type0 cmap on write.
        val = "pre-wrap" if "white-space" in p else (
            "4" if p == "tab-size" else (
                "auto" if "hyphen" in p else (
                    "anywhere" if p == "line-break" else "break-word"
                )
            )
        )
        return (
            f'<div style="{p}:{val};border:1px dashed #888;width:140px;font-size:9pt;">'
            f'supercalifragilistic wrapping-hyphenation sample</div>'
        )
    if p.startswith("margin"):
        return f'<div style="border:1px solid #333;display:inline-block;"><div style="{p}:10px;background:#dfd;border:1px dashed #080;">margin</div></div>'
    if p.startswith("padding"):
        return f'<div style="{p}:10px;background:#ddf;border:1px solid #228;">padding</div>'
    if "radius" in p:
        return f'<div style="{p}:12px;background:#9cf;border:2px solid #246;width:64px;height:36px;"></div>'
    if p.startswith("border") and "image" in p:
        return (
            f'<div style="border:6px solid transparent;border-image-source:url(logo.png);'
            f'border-image-slice:10;border-image-repeat:stretch;{p}:{"url(logo.png)" if p.endswith("source") else ("10" if "slice" in p or "width" in p or "outset" in p else "stretch")};'
            f'padding:8px;width:90px;">img border</div>'
        )
    if p.startswith("border") or p in ("outline",) or p.startswith("outline"):
        val = "3px solid #c40" if "style" not in p and "color" not in p and "width" not in p and "offset" not in p else (
            "3px" if "width" in p else ("#c40" if "color" in p else ("solid" if "style" in p else "4px"))
        )
        if p.startswith("outline") and "offset" in p:
            val = "4px"
        return f'<div style="{p}:{val};{"outline:2px solid #c40;" if p=="outline-offset" else ""}padding:6px;display:inline-block;">box</div>'
    if p in ("width", "height", "min-width", "max-width", "min-height", "max-height") or p.endswith("-size") or p.startswith("inline-size") or p.startswith("block-size") or "min-" in p or "max-" in p:
        return f'<div style="{p}:72px;background:#cef;border:1px solid #246;">size</div>'
    if p == "box-sizing":
        return f'<div style="{p}:border-box;width:80px;padding:10px;border:4px solid #246;background:#efc;">box</div>'
    if p.startswith("overflow"):
        return f'<div style="{p}:hidden;width:90px;height:32px;border:1px solid #333;font-size:8pt;">overflowing content that should clip here XXXXXXX</div>'
    if p in ("position",) or p.startswith("inset") or p in ("top", "right", "bottom", "left", "z-index"):
        return (
            f'<div style="position:relative;height:40px;border:1px dashed #888;">'
            f'<div style="position:absolute;{p}:{"4px" if p!="z-index" else "2"};left:4px;background:#fd8;padding:2px 6px;">pos</div></div>'
        )
    if p in ("float", "clear"):
        return (
            f'<div style="border:1px dashed #888;overflow:auto;">'
            f'<div style="{p}:left;background:#9cf;padding:4px;margin-right:6px;">F</div>'
            f'<span>following text wraps beside the float.</span></div>'
        )
    if p.startswith("-webkit-box-") and p not in ("-webkit-box-shadow", "-webkit-box-sizing"):
        if p.endswith("align"):
            return (
                f'<div style="display:-webkit-box;-webkit-box-orient:horizontal;{p}:center;height:48px;'
                f'border:1px solid #336;padding:4px;">'
                f'<span style="background:#cde;padding:2px 6px;font-size:8pt;">sm</span>'
                f'<span style="background:#9cf;padding:8px 6px;">mid</span></div>'
            )
        if p.endswith("flex"):
            return (
                f'<div style="display:-webkit-box;-webkit-box-orient:horizontal;width:140px;'
                f'border:1px solid #336;padding:4px;">'
                f'<span style="{p}:1;background:#cde;padding:4px 6px;">flex</span>'
                f'<span style="background:#fd8;padding:4px 6px;">fix</span></div>'
            )
        if "ordinal" in p:
            return (
                f'<div style="display:-webkit-box;-webkit-box-orient:horizontal;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="{p}:1;background:#fd8;padding:2px 6px;">2nd</span>'
                f'<span style="{p}:2;background:#cde;padding:2px 6px;">1st</span></div>'
            )
        if p.endswith("orient"):
            return (
                f'<div style="display:-webkit-box;{p}:vertical;-webkit-box-align:start;border:1px solid #336;padding:4px;gap:4px;width:48px;">'
                f'<span style="background:#cde;padding:2px 6px;">A</span>'
                f'<span style="background:#9cf;padding:2px 6px;">B</span></div>'
            )
        if p.endswith("pack"):
            return (
                f'<div style="display:-webkit-box;-webkit-box-orient:horizontal;{p}:center;width:140px;'
                f'border:1px solid #336;padding:4px;">'
                f'<span style="background:#cde;padding:2px 6px;">1</span>'
                f'<span style="background:#cde;padding:2px 6px;">2</span></div>'
            )
    if p.startswith("flex") or p in ("order",) or p.startswith("align-") or p.startswith("justify-") or p.startswith("place-") or p == "gap" or p.startswith("row-gap") or p.startswith("column-gap") or p.startswith("-webkit-flex") or p.startswith("-webkit-align") or p.startswith("-webkit-justify") or p == "-webkit-order":
        if p in ("flex", "-webkit-flex") or target == "flex":
            return (
                f'<div style="display:flex;width:140px;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="{p}:1 1 auto;background:#cde;padding:2px 6px;">grow</span>'
                f'<span style="background:#fd8;padding:2px 6px;">fix</span></div>'
            )
        if "basis" in p:
            return (
                f'<div style="display:flex;width:140px;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="{p}:64px;background:#cde;padding:2px 6px;">64px</span>'
                f'<span style="background:#fd8;padding:2px 6px;">rest</span></div>'
            )
        if "grow" in p:
            return (
                f'<div style="display:flex;width:140px;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="{p}:1;background:#cde;padding:2px 6px;">grow</span>'
                f'<span style="background:#fd8;padding:2px 6px;">fix</span></div>'
            )
        if "shrink" in p:
            return (
                f'<div style="display:flex;width:100px;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="{p}:1;flex-basis:80px;background:#cde;padding:2px 6px;">shrink</span>'
                f'<span style="flex-basis:80px;background:#fd8;padding:2px 6px;">keep</span></div>'
            )
        if p.endswith("self") or (legacy_of or "").endswith("self"):
            return (
                f'<div style="display:flex;align-items:stretch;height:48px;border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="background:#cde;padding:2px 6px;">A</span>'
                f'<span style="{p}:center;background:#fd8;padding:2px 6px;">self</span>'
                f'<span style="background:#cde;padding:2px 6px;">C</span></div>'
            )
        if "content" in p and p.startswith(("align", "-webkit-align")):
            return (
                f'<div style="display:flex;flex-wrap:wrap;{p}:center;height:56px;width:120px;'
                f'border:1px solid #336;padding:4px;gap:4px;">'
                f'<span style="background:#cde;padding:2px 6px;">1</span>'
                f'<span style="background:#cde;padding:2px 6px;">2</span>'
                f'<span style="background:#cde;padding:2px 6px;">3</span>'
                f'<span style="background:#cde;padding:2px 6px;">4</span></div>'
            )
        val = (
            "column" if "direction" in p else
            ("row wrap" if "flow" in p else
             ("20px" if "gap" in p else
              ("center" if p.startswith(("align", "justify", "place", "-webkit-align", "-webkit-justify")) else
               ("2" if p in ("order", "-webkit-order") else "wrap"))))
        )
        return (
            f'<div style="display:flex;{p}:{val};'
            f'border:1px solid #336;padding:4px;gap:4px;">'
            f'<span style="background:#cde;padding:2px 6px;">1</span>'
            f'<span style="background:#cde;padding:2px 6px;">2</span>'
            f'<span style="background:#cde;padding:2px 6px;">3</span></div>'
        )
    if p.startswith("grid") or p == "grid":
        return (
            f'<div style="display:grid;{p}:{"40px 40px" if "columns" in p or p.endswith("template-columns") else ("1fr 1fr" if "rows" in p else "auto")};'
            f'gap:4px;border:1px solid #336;padding:4px;">'
            f'<span style="background:#cde;padding:4px;">A</span><span style="background:#cde;padding:4px;">B</span></div>'
        )
    if "column" in p or p == "columns":
        return f'<div style="{p}:{"2" if p in ("columns","column-count") else ("12px" if "gap" in p else ("1px solid #666" if "rule" in p else "auto"))};border:1px solid #888;padding:4px;font-size:8pt;height:48px;">Multi-column sample text repeated. Multi-column sample text repeated. Multi-column sample text repeated.</div>'
    if "break" in p or p.startswith("page"):
        return f'<div style="{p}:{"avoid" if "inside" in p else "auto"};border:1px dashed #a60;padding:4px;font-size:8pt;">fragmentation: <code>{html.escape(p)}</code></div>'
    if p.startswith("list-") or p.startswith("counter") or p in ("quotes", "content"):
        return f'<div style="{p}:{"decimal" if "type" in p or p=="list-style-type" else ("inside" if "position" in p else "none")};">list/content demo</div>'
    if p in ("caption-side", "border-collapse", "border-spacing", "table-layout", "empty-cells"):
        return (
            f'<table style="{p}:{"bottom" if p=="caption-side" else ("collapse" if p=="border-collapse" else ("4px" if p=="border-spacing" else ("fixed" if p=="table-layout" else "hide")))};'
            f'width:120px;border:1px solid #333;font-size:8pt;"><caption>cap</caption><tr><td style="border:1px solid #333;">td</td><td style="border:1px solid #333;">td</td></tr></table>'
        )
    if p.startswith("transform") or p in ("translate", "rotate", "scale"):
        return f'<div style="{p}:{"rotate(8deg)" if p=="transform" else ("8deg" if p=="rotate" else ("4px" if p=="translate" else "1.1"))};background:#fd8;padding:6px;display:inline-block;">T</div>'
    if p in ("direction", "writing-mode", "unicode-bidi", "text-orientation"):
        return f'<div style="{p}:{"rtl" if p=="direction" else ("vertical-rl" if p=="writing-mode" else "normal")};border:1px solid #888;padding:4px;height:48px;">ABC 123</div>'
    if p in ("filter",) or p.startswith("mix-blend") or p == "isolation":
        return f'<div style="{p}:{"opacity(0.5)" if p=="filter" else ("multiply" if "blend" in p else "isolate")};background:#8af;padding:6px;">fx</div>'
    if p.startswith("fill") or p.startswith("stroke"):
        return (
            f'<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">'
            f'<rect x="2" y="2" width="60" height="24" style="{p}:{"#0a7" if p.startswith("fill") else ("#c40" if p=="stroke" else "2px")};fill:#cde;stroke:#333;"/></svg>'
        )
    if kind == "legacy-alias" and legacy_of:
        return effect_html(legacy_of, "longhand", None).replace(f"{legacy_of}:", f"{p}:", 1)

    # Generic fallback: apply a harmless visible declaration when possible
    return (
        f'<div style="{html.escape(p)}:inherit;border:1px solid #6a6;padding:4px;background:#f7fff7;">'
        f'<code>{html.escape(p)}</code> applied</div>'
    )


def render_fixture(key: str, meta: dict, font_family: str) -> str:
    props = meta["properties"]
    needle = {"A": "IMPLEMENTED-PROPS-A", "B": "IMPLEMENTED-PROPS-B", "C": "IMPLEMENTED-PROPS-C"}[key]
    fname = meta["file"]
    stem = fname.replace(".html", "")
    title = meta["title"]
    rows = []
    for i, row in enumerate(props, 1):
        prop = row["property"]
        desc = row["description"]
        kind = row.get("kind") or ""
        legacy = row.get("legacy_of")
        group = row.get("group") or ""
        effect = effect_html(prop, kind, legacy)
        rows.append(
            "<tr>"
            f"<td class=\"idx\">{i}</td>"
            f"<td class=\"prop\"><code>{html.escape(prop)}</code><div class=\"meta\">{html.escape(kind)} · {html.escape(group)}</div></td>"
            f"<td class=\"desc\">{html.escape(desc)}</td>"
            f"<td class=\"effect\">{effect}</td>"
            "</tr>"
        )

    body_rows = "\n".join(rows)
    return f"""<!DOCTYPE html>
<!--
  {stem}
  Proves: visual audit of Implemented CSS properties (slice {key} of 3: {len(props)} names),
  each with a plain-language description and a live Effect cell.
  Fonts: {font_family} via --font-path testdata/fonts/implemented-audit (SIL OFL 1.1).
  Images: logo.png + assets/asteria-lake.png + inline data URI.
  Expected: multi-page; envelope in fixturePageBounds.
-->
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{html.escape(title)} ({len(props)} properties)</title>
  <style>
    @page {{ size: A4; margin: 12mm; }}
    * {{ box-sizing: border-box; }}
    body {{
      font-family: "{font_family}", "Liberation Serif", "Liberation Mono", sans-serif;
      font-size: 9.5pt;
      line-height: 1.35;
      color: #1a1f24;
      margin: 0;
      padding: 0;
    }}
    header.masthead {{
      display: flex;
      align-items: center;
      gap: 12px;
      border-bottom: 3px solid #1f4b99;
      padding-bottom: 8px;
      margin-bottom: 12px;
    }}
    header.masthead img.logo {{ height: 36px; }}
    header.masthead img.hero {{ height: 48px; border-radius: 4px; }}
    header.masthead .titles h1 {{
      font-size: 16pt;
      margin: 0 0 2px 0;
      color: #1f4b99;
    }}
    header.masthead .titles p {{ margin: 0; font-size: 8.5pt; color: #445; }}
    .needle {{
      display: inline-block;
      background: #1f4b99;
      color: #fff;
      font-weight: normal;
      padding: 2px 8px;
      border-radius: 3px;
      letter-spacing: 0.04em;
      margin: 8px 0 12px;
    }}
    table.audit {{
      width: 100%;
      border-collapse: collapse;
      table-layout: fixed;
    }}
    table.audit th, table.audit td {{
      border: 1px solid #b8c0cc;
      vertical-align: top;
      padding: 6px 7px;
    }}
    table.audit th {{
      background: #e8eef8;
      font-size: 8.5pt;
      text-align: left;
    }}
    table.audit tr:nth-child(even) td {{ background: #f7f9fc; }}
    td.idx, th.idx {{ width: 20px; text-align: right; color: #667; font-size: 8pt; padding: 6px 4px; }}
    td.prop {{ width: 22%; }}
    td.prop code {{
      font-family: "Liberation Mono", monospace;
      font-size: 8.5pt;
      color: #103a7a;
      font-weight: normal;
    }}
    td.prop .meta {{ font-size: 7.5pt; color: #667; margin-top: 2px; }}
    td.desc {{ width: 38%; font-size: 8.5pt; }}
    td.effect {{ width: 34%; }}
    footer.note {{
      margin-top: 14px;
      font-size: 8pt;
      color: #556;
      border-top: 1px solid #ccd;
      padding-top: 6px;
    }}
  </style>
</head>
<body>
  <header class="masthead">
    <img class="logo" src="logo.png" alt="logo">
    <img class="hero" src="assets/asteria-lake.png" alt="sample">
    <img class="logo" src="{TINY_PNG}" alt="pixel">
    <div class="titles">
      <h1>{html.escape(title)}</h1>
      <p>Slice {key} · {len(props)} Implemented properties · font-family "{html.escape(font_family)}" · local images</p>
    </div>
  </header>

  <div class="needle">{needle}</div>
  <p style="font-size:8.5pt;margin:0 0 10px 0;">
    Each row names an Implemented property, explains the expected effect in plain words,
    and applies that property in the Effect cell so print output can be checked by eye.
    Generate with <code>--font-path testdata/fonts/implemented-audit</code>.
  </p>

  <table class="audit">
    <thead>
      <tr>
        <th class="idx">#</th>
        <th>Property</th>
        <th>What it should do</th>
        <th>Effect (live)</th>
      </tr>
    </thead>
    <tbody>
{body_rows}
    </tbody>
  </table>

  <footer class="note">
    {needle} · Liberation fonts OFL-1.1 under testdata/fonts/implemented-audit ·
    gowkhtmltopdf Implemented CSS audit · do not use network images
  </footer>
</body>
</html>
"""


def main(argv: list[str]) -> int:
    data = json.loads(SPLIT.read_text())
    font_family = data.get("font_family") or "Liberation Sans"
    wanted = {a.upper() for a in argv[1:]} if len(argv) > 1 else {"A", "B", "C"}
    for key, meta in data["fixtures"].items():
        if key not in wanted:
            continue
        out = OUT_DIR / meta["file"]
        out.write_text(render_fixture(key, meta, font_family), encoding="utf-8")
        print(f"wrote {out.relative_to(ROOT)} ({len(meta['properties'])} props)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
