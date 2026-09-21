#!/usr/bin/env python3
"""Capture Chromium DOM rectangles for selected elements in a fixture."""

from __future__ import annotations

import argparse
import html
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile


START_MARKER = "__GOWKHTMLTOPDF_CHROME_RECTS_START__"
END_MARKER = "__GOWKHTMLTOPDF_CHROME_RECTS_END__"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Capture Chromium getBoundingClientRect values from an HTML fixture."
    )
    parser.add_argument("fixture", type=Path, help="HTML fixture to load")
    parser.add_argument(
        "--selector",
        action="append",
        required=True,
        help="CSS selector to capture; repeat for multiple selectors",
    )
    parser.add_argument(
        "--wait-ms",
        type=int,
        default=1000,
        help="Chromium virtual-time budget in milliseconds",
    )
    return parser.parse_args()


def chrome_binary() -> str:
    configured = os.environ.get("CHROME_BIN")
    if configured:
        return configured

    for candidate in ("google-chrome", "chromium", "chromium-browser"):
        found = shutil.which(candidate)
        if found:
            return found

    raise RuntimeError(
        "Chromium was not found; set CHROME_BIN or install google-chrome"
    )


def instrument_html(source: str, selectors: list[str]) -> str:
    selector_json = json.dumps(selectors)
    script = f"""
<script>
(() => {{
  const selectors = {selector_json};
  const rows = [];
  for (const selector of selectors) {{
    for (const [index, element] of [...document.querySelectorAll(selector)].entries()) {{
      const rect = element.getBoundingClientRect();
      rows.push({{selector, index, x: rect.x, y: rect.y, width: rect.width, height: rect.height}});
    }}
  }}
  const output = document.createElement('pre');
  output.id = 'gowkhtmltopdf-chrome-rects';
  output.style.cssText = 'position:fixed;left:-10000px;top:-10000px;width:1px;height:1px;overflow:hidden;';
  output.textContent = {json.dumps(START_MARKER)} + JSON.stringify(rows) + {json.dumps(END_MARKER)};
  (document.body || document.documentElement).appendChild(output);
}})();
</script>
"""

    body_match = re.search(r"</body\s*>", source, flags=re.IGNORECASE)
    if body_match:
        return source[: body_match.start()] + script + source[body_match.start() :]

    return source + script


def capture(fixture: Path, selectors: list[str], wait_ms: int) -> list[dict[str, object]]:
    source = fixture.read_text(encoding="utf-8")

    with tempfile.TemporaryDirectory(prefix="gowkhtmltopdf-chrome-rects-") as temp_dir:
        instrumented = Path(temp_dir) / fixture.name
        instrumented.write_text(instrument_html(source, selectors), encoding="utf-8")

        command = [
            chrome_binary(),
            "--headless=new",
            "--no-sandbox",
            "--disable-gpu",
            "--disable-dev-shm-usage",
            "--allow-file-access-from-files",
            "--dump-dom",
            f"--virtual-time-budget={wait_ms}",
            instrumented.as_uri(),
        ]
        result = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=max(10, wait_ms / 1000 + 10),
        )

    if result.returncode != 0:
        details = result.stderr.strip() or result.stdout.strip()
        raise RuntimeError(f"Chromium failed with exit code {result.returncode}: {details}")

    output = result.stdout
    start = output.rfind(START_MARKER)
    end = output.find(END_MARKER, start + len(START_MARKER))
    if start < 0 or end < 0:
        raise RuntimeError("Chromium output did not contain the rectangle marker")

    payload = html.unescape(output[start + len(START_MARKER) : end])
    rows = json.loads(payload)
    if not isinstance(rows, list):
        raise RuntimeError("Chromium rectangle payload was not a list")

    return rows


def main() -> int:
    args = parse_args()
    if not args.fixture.is_file():
        print(f"fixture does not exist: {args.fixture}", file=sys.stderr)
        return 2
    if args.wait_ms < 0:
        print("--wait-ms must be non-negative", file=sys.stderr)
        return 2

    try:
        rows = capture(args.fixture, args.selector, args.wait_ms)
    except (OSError, RuntimeError, json.JSONDecodeError, subprocess.SubprocessError) as error:
        print(f"chrome-rects: {error}", file=sys.stderr)
        return 1

    print(json.dumps(rows, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
