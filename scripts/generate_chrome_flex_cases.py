#!/usr/bin/env python3
"""Generate the Chromium Flexbox porting inventory and HTML scaffolds."""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "test" / "Chrome"
CASES = [
    [
        "legacy-flex-algorithm",
        "Flex sizing with positive and negative free space",
        "third_party/blink/web_tests/css3/flexbox/flex-algorithm.html",
        "flex-sizing",
        "flex-sizing",
        "layout-unit",
        "A 600px row redistributes positive and negative free space using flex basis, grow, and shrink factors.",
        "row"
    ],
    [
        "legacy-flex-algorithm-minmax",
        "Flex sizing with min and max constraints",
        "third_party/blink/web_tests/css3/flexbox/flex-algorithm-min-max.html",
        "min-max-freeze",
        "flex-sizing",
        "layout-unit",
        "Clamped items freeze and the remaining free space is redistributed to the other items.",
        "minmax"
    ],
    [
        "legacy-flex-algorithm-margins",
        "Flex sizing with fixed and auto margins",
        "third_party/blink/web_tests/css3/flexbox/flex-algorithm-with-margins.html",
        "auto-margins",
        "flex-sizing",
        "layout-unit",
        "Auto margins consume positive main-axis free space without creating negative flex space.",
        "margins"
    ],
    [
        "legacy-columns-auto-size",
        "Intrinsic column sizing",
        "third_party/blink/web_tests/css3/flexbox/columns-auto-size.html",
        "intrinsic-sizing",
        "column-sizing",
        "layout-unit",
        "Automatic column sizes include content, margins, padding, and min or max constraints.",
        "column"
    ],
    [
        "legacy-definite-main-size",
        "Definite main size and percentage children",
        "third_party/blink/web_tests/css3/flexbox/definite-main-size.html",
        "definite-percentages",
        "column-sizing",
        "layout-unit",
        "A definite flex container size resolves a percentage child against the correct axis.",
        "column"
    ],
    [
        "legacy-justify-content",
        "Main-axis distribution",
        "third_party/blink/web_tests/css3/flexbox/flex-justify-content.html",
        "main-axis-alignment",
        "alignment",
        "layout-unit",
        "Flex-end, center, and space distribution place fixed children at the expected main-axis offsets.",
        "justify"
    ],
    [
        "legacy-flex-align",
        "Cross-axis alignment and auto margins",
        "third_party/blink/web_tests/css3/flexbox/flex-align.html",
        "cross-axis-alignment",
        "alignment",
        "layout-unit",
        "Stretch, start, center, end, baseline, and auto cross margins produce different cross-axis geometry.",
        "align"
    ],
    [
        "legacy-flex-align-vertical-writing",
        "Alignment under vertical writing modes",
        "third_party/blink/web_tests/css3/flexbox/flex-align-vertical-writing-mode.html",
        "writing-modes",
        "alignment",
        "chrome-reference",
        "Vertical writing modes remap physical positions while preserving logical alignment.",
        "writing"
    ],
    [
        "legacy-flex-flow-orientations",
        "Direction and flow orientations",
        "third_party/blink/web_tests/css3/flexbox/flex-flow-orientations.html",
        "direction-reversal",
        "direction",
        "chrome-reference",
        "Direction, writing mode, row or column, and reverse flow change logical start and end positions.",
        "writing"
    ],
    [
        "legacy-flex-flow",
        "Flow, direction, padding, and constraints",
        "third_party/blink/web_tests/css3/flexbox/flex-flow.html",
        "direction-reversal",
        "direction",
        "chrome-reference",
        "Logical padding and margins remain attached to logical edges across reverse directions and axes.",
        "writing"
    ],
    [
        "legacy-multiline",
        "Wrapping across axes and directions",
        "third_party/blink/web_tests/css3/flexbox/multiline.html",
        "wrapping",
        "wrapping",
        "layout-unit",
        "Wrapping creates lines with independent cross sizes, including wrap-reverse and direction changes.",
        "wrap"
    ],
    [
        "legacy-multiline-align-content-column",
        "Align-content across wrapped columns",
        "third_party/blink/web_tests/css3/flexbox/multiline-align-content-horizontal-column.html",
        "multi-line-alignment",
        "wrapping",
        "layout-unit",
        "Align-content modes place wrapped column lines at distinct cross-axis offsets.",
        "wrap"
    ],
    [
        "legacy-flex-flow-auto-margins",
        "Logical auto margins in reverse flow",
        "third_party/blink/web_tests/css3/flexbox/flex-flow-auto-margins.html",
        "auto-margins",
        "direction",
        "chrome-reference",
        "Logical auto margins consume free space correctly through direction, writing mode, and column-reverse.",
        "margins"
    ],
    [
        "legacy-flex-align-baseline",
        "Baseline alignment across axes",
        "third_party/blink/web_tests/css3/flexbox/flex-align-baseline.html",
        "baseline",
        "alignment",
        "chrome-reference",
        "Items with different margins share the expected baseline across flows and writing modes.",
        "baseline"
    ],
    [
        "legacy-multiline-align-self",
        "Per-item alignment on multiple lines",
        "third_party/blink/web_tests/css3/flexbox/multiline-align-self.html",
        "multi-line-alignment",
        "alignment",
        "layout-unit",
        "Each line gets its own cross size before align-self applies start, center, end, baseline, or stretch.",
        "wrap"
    ],
    [
        "wpt-align-items-stretch",
        "Stretch alignment with margins",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_align-items-stretch-2.html",
        "cross-axis-alignment",
        "alignment",
        "layout-unit",
        "Stretch does not incorrectly override a flex item margin or explicit flex sizing.",
        "align"
    ],
    [
        "wpt-flex-basis-011",
        "Percentage flex basis in an indefinite column",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-basis-011.html",
        "flex-basis",
        "flex-sizing",
        "layout-unit",
        "A percentage flex basis in an indefinite nested column remains content-sized instead of becoming a definite full height.",
        "column"
    ],
    [
        "wpt-rtl-flow-reverse",
        "RTL column wrap reverse",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_rtl-flow-reverse.html",
        "direction-reversal",
        "direction",
        "chrome-reference",
        "RTL plus column wrap-reverse reverses logical column order without losing item order within a column.",
        "writing"
    ],
    [
        "wpt-flexbox-margin-auto",
        "Row auto margins",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_margin-auto.html",
        "auto-margins",
        "flex-sizing",
        "layout-unit",
        "Fixed children with auto margins split positive free space and remain vertically centered.",
        "margins"
    ],
    [
        "wpt-min-size-auto-overflow-clip",
        "Automatic minimum size with overflow clip",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/min-size-auto-overflow-clip.html",
        "minimum-size",
        "min-size",
        "chrome-reference",
        "The automatic minimum size preserves the content width even when the flex container is narrower.",
        "minmax"
    ],
    [
        "wpt-flow-row-wrap",
        "Row wrapping with margins",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_flow-row-wrap.html",
        "wrapping",
        "wrapping",
        "layout-unit",
        "Fixed item widths and margins create two rows with stable line assignments.",
        "wrap"
    ],
    [
        "wpt-gap-002-ltr",
        "Column gap with flexible children",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/gap-002-ltr.html",
        "gap",
        "wrapping",
        "layout-unit",
        "Column gap is counted between items and not added before the first or after the last item.",
        "gap"
    ],
    [
        "wpt-aspect-ratio-cross-size-002",
        "Nested aspect ratio and cross-size feedback",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-aspect-ratio-cross-size-002.html",
        "aspect-ratio",
        "intrinsic-sizing",
        "chrome-reference",
        "Nested flex sizing preserves the expected width and height instead of inflating the cross size.",
        "aspect"
    ],
    [
        "wpt-writing-mode-006",
        "Writing mode matrix",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox-writing-mode-006.html",
        "writing-modes",
        "direction",
        "chrome-reference",
        "Row, column, reverse, and wrap-reverse combinations preserve logical block ordering in vertical writing modes.",
        "writing"
    ],
    [
        "wpt-flex-item-percentage-abspos",
        "Percentage absolute child inside a flex item",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-item-and-percentage-abspos.html",
        "absolute-positioning",
        "positioning",
        "chrome-reference",
        "A percentage-sized absolute child fills its flex item without corrupting the item size.",
        "abspos"
    ],
    [
        "wpt-definite-sizes-002",
        "Definite size from min-height",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox-definite-sizes-002.html",
        "definite-percentages",
        "column-sizing",
        "layout-unit",
        "A minimum height establishes a definite percentage basis for a nested child.",
        "column"
    ],
    [
        "wpt-percentage-heights-005",
        "Percentage height in a column flex item",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/percentage-heights-005.html",
        "definite-percentages",
        "column-sizing",
        "layout-unit",
        "A height percentage fills the definite column flex item without leaving the parent visible.",
        "column"
    ],
    [
        "wpt-flex-minimum-width-aspect",
        "Automatic minimum width from aspect ratio",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-minimum-width-flex-items-010.html",
        "minimum-size",
        "min-size",
        "chrome-reference",
        "The automatic main-axis minimum uses the transferred aspect-ratio size.",
        "aspect"
    ],
    [
        "wpt-break-nested-float-print",
        "Flex content fragmentation in print",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/break-nested-float-in-flex-item-001-print.html",
        "fragmentation",
        "pagination",
        "golden-fixture",
        "A tall nested flex child fragments across three pages while preserving its width and offsets.",
        "print"
    ],
    [
        "wpt-auto-margins-column",
        "Auto margins and align-self in a column",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/auto-margins-003.html",
        "auto-margins",
        "alignment",
        "layout-unit",
        "Column auto margins and align-self center both center their items across the cross-axis.",
        "margins"
    ],
    [
        "wpt-column-reverse-multiline",
        "Column-reverse multiline item positions",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-column-reverse-multiline-item-position.html",
        "direction-reversal",
        "wrapping",
        "layout-unit",
        "Column-reverse lines pack from the main-start after wrapping and finalized line height.",
        "wrap"
    ],
    [
        "wpt-flex-factor-less-than-one",
        "Fractional flex grow and shrink factors",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-factor-less-than-one.html",
        "flex-sizing",
        "flex-sizing",
        "layout-unit",
        "Fractional grow and shrink factors distribute space proportionally in both row and column axes.",
        "grow"
    ],
    [
        "wpt-flex-item-compressible",
        "Replaced flex item automatic minimum",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-item-compressible-001.html",
        "minimum-size",
        "min-size",
        "chrome-reference",
        "An input flex item uses its specified-size suggestion correctly when the row must shrink it.",
        "replaced"
    ],
    [
        "wpt-flex-container-max-content",
        "Flex container max-content contribution",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-container-max-content-001.html",
        "intrinsic-sizing",
        "intrinsic-sizing",
        "chrome-reference",
        "The flex container measures outer item contributions including margins, padding, and borders.",
        "intrinsic"
    ],
    [
        "wpt-flex-cross-size-border-box",
        "Border-box cross-size for stretched items",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-cross-size-border-box-001.html",
        "cross-axis-sizing",
        "alignment",
        "layout-unit",
        "Border-box and content-box containers provide equivalent content cross sizes to stretched children.",
        "align"
    ],
    [
        "wpt-flex-base-size-max-width",
        "Flex base size before max-width clamp",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-base-size-ignores-max-width.html",
        "min-max-freeze",
        "flex-sizing",
        "layout-unit",
        "The flex base size uses content before the max-width clamp freezes the item and redistributes space.",
        "minmax"
    ],
    [
        "blink-replaced-aspect-ratio-precision",
        "Intrinsic replaced-element ratio precision",
        "third_party/blink/renderer/core/layout/flex/flex_layout_algorithm_test.cc",
        "intrinsic-sizing",
        "replaced",
        "layout-unit",
        "An auto-sized SVG keeps its intrinsic 29 by 22 size inside a 50px column item.",
        "replaced"
    ],
    [
        "blink-gap-decorations-basic",
        "Gap geometry with wrapping",
        "third_party/blink/renderer/core/layout/flex/flex_layout_algorithm_test.cc",
        "gap",
        "wrapping",
        "layout-unit",
        "Six fixed items wrap into two rows with main and cross gaps at known intersections.",
        "gap"
    ],
    [
        "blink-scrollbars-row-reverse-vrl",
        "Reverse flow with vertical writing and overflow",
        "third_party/blink/renderer/core/layout/layout_flexible_box_test.cc",
        "direction-reversal",
        "overflow",
        "chrome-reference",
        "Row-reverse with vertical-rl maps content and overflow into negative physical coordinates.",
        "writing"
    ],
    [
        "wpt-flex-container-min-content",
        "Flex container min-content contribution",
        "third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-container-min-content-001.html",
        "intrinsic-sizing",
        "intrinsic-sizing",
        "chrome-reference",
        "The flex container's min-content contribution reflects the largest constrained item contribution.",
        "intrinsic"
    ]
]

TEMPLATES = {
    "row": (
        ".case { display: flex; width: 300px; height: 100px; border: 1px solid #222; } "
        ".item { box-sizing: border-box; min-width: 0; flex: 1 1 0; height: 40px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div>'
        '<div class="item" style="background:#ffd6a5">C</div></div>',
    ),
    "grow": (
        ".case { display: flex; width: 100px; height: 100px; border: 1px solid #222; } "
        ".item { flex-basis: 0; height: 30px; }",
        '<div class="case"><div class="item" style="flex-grow:.5;background:#9ec5fe">A</div>'
        '<div class="item" style="flex-grow:.25;background:#b7e4c7">B</div></div>',
    ),
    "minmax": (
        ".case { display: flex; width: 600px; border: 1px solid #222; } "
        ".item { height: 30px; flex: 1 1 200px; min-width: 0; }",
        '<div class="case"><div class="item" style="background:#9ec5fe;max-width:100px">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div>'
        '<div class="item" style="background:#ffd6a5">C</div></div>',
    ),
    "margins": (
        ".case { display: flex; width: 300px; height: 80px; border: 1px solid #222; } "
        ".item { width: 40px; height: 30px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe;margin:auto">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div></div>',
    ),
    "column": (
        ".case { display: flex; flex-direction: column; width: 240px; height: 120px; border: 1px solid #222; } "
        ".item { min-height: 20px; padding: 4px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">Test Header</div>'
        '<div class="item" style="background:#b7e4c7">Test Subheader</div></div>',
    ),
    "justify": (
        ".case { display: flex; width: 300px; height: 80px; justify-content: center; border: 1px solid #222; } "
        ".item { width: 40px; height: 30px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div></div>',
    ),
    "align": (
        ".case { display: flex; width: 240px; height: 100px; align-items: center; border: 1px solid #222; } "
        ".item { width: 40px; height: 30px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7;align-self:flex-end">B</div></div>',
    ),
    "baseline": (
        ".case { display: flex; width: 240px; height: 100px; align-items: baseline; border: 1px solid #222; } "
        ".item { width: 80px; padding: 4px; }",
        '<div class="case"><div class="item" style="font-size:12px;background:#9ec5fe">small</div>'
        '<div class="item" style="font-size:24px;background:#b7e4c7">large</div></div>',
    ),
    "writing": (
        ".case { display: flex; width: 180px; height: 100px; writing-mode: vertical-rl; "
        "flex-flow: row wrap; border: 1px solid #222; } .item { width: 30px; height: 30px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div>'
        '<div class="item" style="background:#ffd6a5">C</div></div>',
    ),
    "wrap": (
        ".case { display: flex; flex-flow: row wrap; width: 100px; height: 80px; gap: 5px; border: 1px solid #222; } "
        ".item { width: 45px; height: 20px; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div>'
        '<div class="item" style="background:#ffd6a5">C</div></div>',
    ),
    "gap": (
        ".case { display: flex; flex-direction: column; width: 160px; height: 180px; gap: 20px; border: 1px solid #222; } "
        ".item { flex: 1 1 auto; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">A</div>'
        '<div class="item" style="background:#b7e4c7">B</div>'
        '<div class="item" style="background:#ffd6a5">C</div></div>',
    ),
    "aspect": (
        ".case { display: flex; flex-direction: column; width: 200px; border: 1px solid #222; } "
        ".item { width: 100px; aspect-ratio: 4 / 1; }",
        '<div class="case"><div class="item" style="background:#9ec5fe">ratio</div></div>',
    ),
    "abspos": (
        ".case { position: relative; display: flex; width: 100px; height: 100px; border: 1px solid #222; } "
        ".item { position: relative; width: 100px; height: 100px; } .fill { position:absolute;width:100%;height:100%;background:#b7e4c7; }",
        '<div class="case"><div class="item"><div class="fill"></div></div></div>',
    ),
    "print": (
        ".case { display: flex; flex-direction: column; width: 2in; min-height: 6in; border: 1px solid #222; } "
        ".item { min-height: 2in; }",
        '<div class="case"><div class="item" style="background:#b7e4c7">page one</div>'
        '<div class="item" style="background:#9ec5fe">page two</div>'
        '<div class="item" style="background:#ffd6a5">page three</div></div>',
    ),
    "replaced": (
        ".case { display: flex; flex-direction: column; width: 50px; border: 1px solid #222; } "
        ".item { width: 29px; height: 22px; background:#b7e4c7; }",
        '<div class="case"><div class="item">SVG ratio</div></div>',
    ),
    "intrinsic": (
        ".case { display: flex; width: max-content; border: 2px solid #222; } "
        ".item { margin: 5px; padding: 3px; border: 2px solid #9ec5fe; }",
        '<div class="case"><div class="item">X X</div><div class="item">LONG TEXT</div></div>',
    ),
    "minmax": (
        ".case { display: flex; width: 300px; border: 1px solid #222; } "
        ".item { flex: 1 1 300px; min-width: 0; height: 50px; }",
        '<div class="case"><div class="item" style="background:#b7e4c7;max-width:100px">A</div>'
        '<div class="item" style="background:#9ec5fe">B</div></div>',
    ),
}

def make_fixture(case):
    style, body = TEMPLATES[case["kind"]]
    source = case["source"]
    return f"""<!doctype html>
<meta charset="utf-8">
<title>{case["title"]}</title>
<style>
html, body {{ margin: 0; padding: 0; font: 12px sans-serif; }}
{style}
</style>
<!-- Source: {source} -->
<!-- Port status: scaffold. The next phase replaces this minimal case with a verified Go fixture. -->
<!-- Expected: {case["expected"]} -->
{body}
"""

def main():
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "cases").mkdir(parents=True, exist_ok=True)
    fields = ("id", "title", "source", "combination", "category", "goTarget", "expected", "kind")
    manifest = {
        "schema": 1,
        "purpose": "Chromium Flexbox behavior map and Go porting scaffolds",
        "source_root": "chromium/",
        "caseCount": len(CASES),
        "cases": [],
    }
    for row in CASES:
        case = dict(zip(fields, row, strict=True))
        entry = dict(case)
        entry["fixture"] = f"cases/{case['id']}.html"
        entry["status"] = "scaffold"
        manifest["cases"].append(entry)
        (OUT / entry["fixture"]).write_text(make_fixture(case), encoding="utf-8")
    (OUT / "manifest.json").write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )

if __name__ == "__main__":
    main()
