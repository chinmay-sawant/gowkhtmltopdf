#!/usr/bin/env python3
"""Report embedded PDF fonts against the matching golden HTML declarations."""
from __future__ import annotations

import argparse
import re
import sys
from collections import Counter
from pathlib import Path

try:
    import fitz  # PyMuPDF
except ImportError:
    print("need pymupdf: pip install pymupdf", file=sys.stderr)
    raise SystemExit(2)


SUBSET_PREFIX = re.compile(r"^[A-Z]{6}\+")
FONT_FAMILY_DECLARATION = re.compile(
    r"font-family\s*:\s*([^;{}]+)", re.IGNORECASE
)
STYLESHEET_LINK = re.compile(
    r"<link\b[^>]*href\s*=\s*[\"']([^\"']+\.css)[\"'][^>]*>",
    re.IGNORECASE,
)
CSS_VARIABLE = re.compile(r"(--[\w-]+)\s*:\s*([^;{}]+)")
CSS_VAR_REFERENCE = re.compile(r"var\(\s*(--[\w-]+)(?:\s*,[^)]*)?\s*\)")

GENERIC_FAMILIES = {"serif", "sans-serif", "monospace", "system-ui"}

# These are the aliases currently handled by the bundled FaceSet. The report
# intentionally calls them fallbacks, not errors: the renderer's default
# policy is deterministic and does not promise the proprietary named face.
ALIASES = {
    "georgia": "Liberation Serif",
    "times": "Liberation Serif",
    "times new roman": "Liberation Serif",
    "arial": "Liberation Sans",
    "helvetica": "Liberation Sans",
    "tahoma": "Liberation Sans",
    "verdana": "Liberation Sans",
    "calibri": "Liberation Sans",
    "courier": "Liberation Mono",
    "courier new": "Liberation Mono",
    "consolas": "Liberation Mono",
    "monaco": "Liberation Mono",
}


def normalize_family(value: str) -> str:
    return value.strip().strip("'\"").strip().lower()


def split_families(value: str) -> list[str]:
    return [part.strip().strip("'\"").strip() for part in value.split(",") if part.strip()]


def clean_pdf_font_name(value: str) -> str:
    return SUBSET_PREFIX.sub("", value)


def pdf_family_name(value: str) -> str:
    """Collapse a PDF BaseFont name to the renderer's visible family."""
    low = clean_pdf_font_name(value).lower().replace("-", "")
    if "liberationsans" in low:
        return "Liberation Sans"
    if "liberationserif" in low:
        return "Liberation Serif"
    if "liberationmono" in low:
        return "Liberation Mono"
    if "dejavusans" in low:
        return "DejaVu Sans"
    if "dejavuserif" in low:
        return "DejaVu Serif"
    if "dejavumono" in low:
        return "DejaVu Mono"
    return clean_pdf_font_name(value)


def stylesheet_text(html: Path) -> str:
    text = html.read_text(encoding="utf-8")
    parts = [text]
    for href in STYLESHEET_LINK.findall(text):
        linked = (html.parent / href).resolve()
        if linked.is_file() and linked.parent == html.parent.resolve():
            parts.append(linked.read_text(encoding="utf-8"))
    return "\n".join(parts)


def expand_variables(value: str, variables: dict[str, str]) -> str:
    for _ in range(4):
        expanded = CSS_VAR_REFERENCE.sub(
            lambda match: variables.get(match.group(1), match.group(0)), value
        )
        if expanded == value:
            break
        value = expanded
    return value


def declared_families(html: Path) -> tuple[list[str], list[list[str]]]:
    text = stylesheet_text(html)
    variables = {name: value.strip() for name, value in CSS_VARIABLE.findall(text)}
    declarations: list[list[str]] = []
    families: list[str] = []
    for match in FONT_FAMILY_DECLARATION.finditer(text):
        value = expand_variables(match.group(1), variables)
        declaration = split_families(value)
        if declaration:
            declarations.append(declaration)
            families.extend(declaration)
    return list(dict.fromkeys(families)), declarations


def embedded_families(pdf: Path) -> tuple[list[str], Counter[str]]:
    doc = fitz.open(pdf)
    names: Counter[str] = Counter()
    raw_names: Counter[str] = Counter()
    for page in doc:
        for font in page.get_fonts(full=True):
            raw = clean_pdf_font_name(font[3])
            raw_names[raw] += 1
            names[pdf_family_name(raw)] += 1
    return list(names), raw_names


def fallback_findings(declarations: list[list[str]], embedded: list[str]) -> list[str]:
    embedded_keys = {normalize_family(name) for name in embedded}
    findings: list[str] = []
    for declaration in declarations:
        for family in declaration:
            key = normalize_family(family)
            if key in embedded_keys:
                break
            if key in GENERIC_FAMILIES:
                break
            fallback = ALIASES.get(key)
            if fallback and normalize_family(fallback) in embedded_keys:
                findings.append(f"{family} -> {fallback}")
    return list(dict.fromkeys(findings))


def inspect(output_dir: Path, html_dir: Path) -> int:
    pdfs = sorted(output_dir.glob("fixture-*.pdf"))
    if not pdfs:
        print(f"no fixture PDFs found under {output_dir}", file=sys.stderr)
        return 1

    print("fixture | pages | embedded fonts | declared families | named fallbacks")
    print("--- | ---: | --- | --- | ---")
    total_fallbacks: Counter[str] = Counter()
    missing_html = 0
    empty_declarations = 0

    for pdf in pdfs:
        html = html_dir / f"{pdf.stem}.html"
        doc = fitz.open(pdf)
        embedded, raw_names = embedded_families(pdf)
        if html.exists():
            declared, declarations = declared_families(html)
            findings = fallback_findings(declarations, embedded)
            if not declared:
                empty_declarations += 1
        else:
            declared = []
            declarations = []
            findings = []
            missing_html += 1
        total_fallbacks.update(findings)

        if declared:
            declared_text = ", ".join(declared)
        elif html.exists():
            declared_text = "<no font-family declaration>"
        else:
            declared_text = "<no matching HTML>"
        embedded_text = ", ".join(embedded) if embedded else "<none>"
        finding_text = ", ".join(findings) if findings else "-"
        print(
            f"{pdf.stem} | {doc.page_count} | {embedded_text} | "
            f"{declared_text} | {finding_text}"
        )
        if raw_names:
            print(f"  PDF BaseFont names: {', '.join(sorted(raw_names))}")

    print("\nSummary of named-family fallbacks:")
    if total_fallbacks:
        for finding, count in total_fallbacks.most_common():
            print(f"- {finding}: {count} fixture(s)")
    else:
        print("- none")
    if missing_html:
        print(f"- {missing_html} PDF(s) had no matching golden HTML")
    if empty_declarations:
        print(f"- {empty_declarations} matching HTML file(s) had no font-family declaration")
    print("\nInterpretation: generic-family mappings are expected; named-family "
          "fallbacks require visual review when exact browser-family parity matters.")
    return 0


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, default=root / "output")
    parser.add_argument("--html-dir", type=Path, default=root / "testdata" / "golden")
    args = parser.parse_args()
    return inspect(args.output_dir, args.html_dir)


if __name__ == "__main__":
    raise SystemExit(main())
