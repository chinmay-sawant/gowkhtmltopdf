#!/usr/bin/env python3
"""Map frozen CSS catalogs onto engine apply arms.

Reads plans/0.2.6/catalog/{webref-css,w3c-all-properties,mdn-units,mdn-properties}.json
and greps apply-arm property names from:
  - internal/layout/style_properties.go  (case "foo":)
  - internal/layout/style_cascade.go     (applyFontProps raw["foo"])

Print-noop UI is permanent ignore for this print engine (no pointer, caret, or
form chrome in the PDF):
  cursor, caret-color, resize, user-select, pointer-events, touch-action,
  appearance

SVG presentation fill/stroke (and fill-* / stroke-*) is also goal=ignore.

Usage:
  python3 scripts/css-catalog-map.py --check
      Exit 0 if print-noop names have goal=ignore and every apply-arm property
      in mapping.json is implemented or partial (already-ignored arms such as
      filter stay ignored). Exit 1 with a short diff otherwise.
      Does not require byte-identical mapping regeneration.
  python3 scripts/css-catalog-map.py --write
      Reclassify print-noop UI and SVG fill/stroke, bump unsupported apply-arm
      names to partial (implemented/partial rows stay as they are), rewrite
      mapping.json summary plus coverage-summary.json counts.

Default with no flags is --check. No Makefile target; run the python command.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path
from typing import Any

PRINT_NOOP = (
    "cursor",
    "caret-color",
    "resize",
    "user-select",
    "pointer-events",
    "touch-action",
    "appearance",
)

APPLY_OK = frozenset({"implemented", "partial"})
QUOTED = re.compile(r'"([^"]+)"')
RAW_LOOKUP = re.compile(r'\braw\["([^"]+)"\]')
APPLY_FONT = re.compile(
    r"func applyFontProps\b.*?\nfunc ",
    re.S,
)
IDENT_BEFORE = re.compile(r"[A-Za-z0-9_]$")
STYLE_PROPERTIES = Path("internal/layout/style_properties.go")
STYLE_PAINT_PROPS = Path("internal/layout/style_paint_props.go")
STYLE_CASCADE = Path("internal/layout/style_cascade.go")
CATALOG_DIR = Path("plans/0.2.6/catalog")
ENGINE_PATH = "internal/layout/style_properties.go"


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def load_json(path: Path) -> Any:
    return json.loads(path.read_text())


def dump_json(path: Path, data: Any) -> None:
    path.write_text(json.dumps(data, indent=2) + "\n")


def is_svg_fill_stroke(name: str) -> bool:
    return name in {"fill", "stroke"} or name.startswith(("fill-", "stroke-"))


def mark_ignored(row: dict[str, Any]) -> bool:
    changed = False
    if row.get("goal") != "ignore":
        row["goal"] = "ignore"
        changed = True
    if row.get("engine_status") != "ignored":
        row["engine_status"] = "ignored"
        changed = True
    if row.get("print_relevant") is not False:
        row["print_relevant"] = False
        changed = True
    return changed


def catalog_property_names(webref: dict[str, Any]) -> set[str]:
    names: set[str] = set()
    for item in webref.get("properties", []):
        name = item.get("name")
        if isinstance(name, str) and name:
            names.add(name)
    return names


def apply_font_body(cascade_src: str) -> str:
    match = APPLY_FONT.search(cascade_src + "\nfunc ")
    return match.group(0) if match else ""


def switch_prop_case_names(src: str) -> set[str]:
    """Quoted names in case labels of switch prop { ... }, not nested value switches."""
    names: set[str] = set()
    brace_depth = 0
    switch_depth: int | None = None
    i = 0
    n = len(src)
    while i < n:
        if src.startswith("switch prop", i) and (i == 0 or not IDENT_BEFORE.match(src[i - 1])):
            j = i + len("switch prop")
            while j < n and src[j] in " \t\n":
                j += 1
            if j < n and src[j] == "{":
                switch_depth = brace_depth + 1
        ch = src[i]
        if ch == "{":
            brace_depth += 1
        elif ch == "}":
            if switch_depth is not None and brace_depth == switch_depth:
                switch_depth = None
            brace_depth -= 1
        elif (
            switch_depth is not None
            and brace_depth == switch_depth
            and src.startswith("case", i)
            and (i == 0 or not IDENT_BEFORE.match(src[i - 1]))
        ):
            colon = src.find(":", i)
            if colon != -1:
                names.update(QUOTED.findall(src[i:colon]))
        i += 1
    return names


def apply_arm_names(root: Path) -> set[str]:
    """Quoted case "foo" names plus applyFontProps raw lookups."""
    names = switch_prop_case_names((root / STYLE_PROPERTIES).read_text())
    paint = root / STYLE_PAINT_PROPS
    if paint.is_file():
        names.update(switch_prop_case_names(paint.read_text()))
    font_body = apply_font_body((root / STYLE_CASCADE).read_text())
    names.update(RAW_LOOKUP.findall(font_body))
    return names


def load_catalogs(root: Path) -> tuple[dict[str, Any], dict[str, Any], dict[str, Any], dict[str, Any]]:
    catalog = root / CATALOG_DIR
    webref = load_json(catalog / "webref-css.json")
    w3c = load_json(catalog / "w3c-all-properties.json")
    units = load_json(catalog / "mdn-units.json")
    mdn_props = load_json(catalog / "mdn-properties.json")
    if not isinstance(webref, dict) or "properties" not in webref:
        raise SystemExit("webref-css.json: missing properties")
    if not isinstance(w3c, list):
        raise SystemExit("w3c-all-properties.json: expected a list")
    if not isinstance(units, dict) or not isinstance(mdn_props, dict):
        raise SystemExit("mdn overlays: expected objects")
    return webref, w3c, units, mdn_props


def mapping_by_name(mapping: dict[str, Any]) -> dict[str, dict[str, Any]]:
    rows = {}
    for row in mapping.get("properties", []):
        name = row.get("property")
        if isinstance(name, str):
            rows[name] = row
    return rows


def recount_properties(mapping: dict[str, Any]) -> None:
    rows = mapping.get("properties", [])
    engine = Counter(row.get("engine_status") for row in rows)
    summary = mapping.setdefault("summary", {})
    summary["webref_properties"] = len(rows)
    summary["mapped_properties"] = len(rows)
    by_status = summary.setdefault("properties_by_engine_status", {})
    for key in ("ignored", "implemented", "partial", "unsupported"):
        by_status[key] = int(engine.get(key, 0))
    summary["properties_print_relevant"] = sum(1 for row in rows if row.get("print_relevant"))
    summary["properties_goal_implement"] = sum(1 for row in rows if row.get("goal") == "implement")


def sync_coverage_summary(mapping: dict[str, Any], coverage: dict[str, Any]) -> None:
    summary = mapping.get("summary", {})
    counts = coverage.setdefault("counts", {})
    for key in (
        "webref_properties",
        "mapped_properties",
        "properties_print_relevant",
        "properties_goal_implement",
    ):
        if key in summary:
            counts[key] = summary[key]
    if "properties_by_engine_status" in summary:
        counts["properties_by_engine_status"] = dict(summary["properties_by_engine_status"])


def apply_updates(
    mapping: dict[str, Any],
    arm_catalog: set[str],
) -> list[str]:
    """Reclassify print-noop/SVG; bump unsupported apply arms to partial."""
    notes: list[str] = []
    for row in mapping.get("properties", []):
        name = row.get("property")
        if not isinstance(name, str):
            continue
        if name in PRINT_NOOP or is_svg_fill_stroke(name):
            if mark_ignored(row):
                notes.append(f"{name}: goal=ignore engine_status=ignored print_relevant=false")
            continue
        if name not in arm_catalog:
            continue
        status = row.get("engine_status")
        if status in APPLY_OK:
            continue
        if status == "unsupported":
            row["engine_status"] = "partial"
            if not row.get("code_path"):
                row["code_path"] = ENGINE_PATH
            notes.append(f"{name}: engine_status unsupported -> partial")
    recount_properties(mapping)
    return notes


def check_invariants(
    mapping: dict[str, Any],
    webref_names: set[str],
    arm_catalog: set[str],
) -> list[str]:
    """Return problem lines. Empty means ok."""
    problems: list[str] = []
    rows = mapping_by_name(mapping)
    mapped_names = set(rows)

    missing = sorted(webref_names - mapped_names)
    extra = sorted(mapped_names - webref_names)
    if missing:
        preview = ", ".join(missing[:8])
        more = f" (+{len(missing) - 8})" if len(missing) > 8 else ""
        problems.append(f"mapping.json missing webref properties: {preview}{more}")
    if extra:
        preview = ", ".join(extra[:8])
        more = f" (+{len(extra) - 8})" if len(extra) > 8 else ""
        problems.append(f"mapping.json extra properties not in webref: {preview}{more}")

    for name in PRINT_NOOP:
        row = rows.get(name)
        if row is None:
            problems.append(f"{name}: print-noop missing from mapping.json")
            continue
        goal = row.get("goal")
        status = row.get("engine_status")
        print_rel = row.get("print_relevant")
        if goal != "ignore" or status != "ignored" or print_rel is not False:
            problems.append(
                f"{name}: print-noop want goal=ignore engine_status=ignored "
                f"print_relevant=false; got goal={goal!r} engine_status={status!r} "
                f"print_relevant={print_rel!r}"
            )

    for name in sorted(arm_catalog):
        row = rows.get(name)
        if row is None:
            problems.append(f"{name}: apply arm missing from mapping.json")
            continue
        status = row.get("engine_status")
        if status in APPLY_OK:
            continue
        if status == "ignored":
            # Permanent ignore (filter blur and similar) even if a parse arm exists.
            continue
        problems.append(
            f"{name}: apply arm engine_status={status!r} want implemented|partial"
        )
    return problems


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n", 1)[0])
    parser.add_argument(
        "--check",
        action="store_true",
        help="check invariants; default if no flags",
    )
    parser.add_argument(
        "--write",
        action="store_true",
        help="rewrite mapping.json and coverage-summary.json",
    )
    args = parser.parse_args(argv)
    do_write = args.write
    do_check = args.check or not args.write

    root = repo_root()
    catalog = root / CATALOG_DIR
    mapping_path = catalog / "mapping.json"
    coverage_path = catalog / "coverage-summary.json"

    webref, _w3c, _units, _mdn = load_catalogs(root)
    webref_names = catalog_property_names(webref)
    mapping = load_json(mapping_path)
    coverage = load_json(coverage_path)
    arms = apply_arm_names(root)
    arm_catalog = {name for name in arms if name in webref_names}

    if do_write:
        notes = apply_updates(mapping, arm_catalog)
        sync_coverage_summary(mapping, coverage)
        dump_json(mapping_path, mapping)
        dump_json(coverage_path, coverage)
        print(f"wrote {mapping_path.relative_to(root)} ({len(notes)} property edits)")
        print(f"wrote {coverage_path.relative_to(root)}")
        for line in notes:
            print(f"  {line}")

    if do_check:
        # Re-read after write so check sees what is on disk.
        mapping = load_json(mapping_path)
        problems = check_invariants(mapping, webref_names, arm_catalog)
        if problems:
            print("css-catalog-map: check failed", file=sys.stderr)
            for line in problems:
                print(f"  {line}", file=sys.stderr)
            return 1
        print(
            "css-catalog-map: check ok "
            f"({len(PRINT_NOOP)} print-noop ignored, "
            f"{len(arm_catalog)} apply arms mapped)"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
