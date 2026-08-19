#!/usr/bin/env bash
# print.sh <input.html> <output.pdf>
#
# Renders an HTML file to PDF through the WeasyPrint CLI. This is the
# WeasyPrint command side of the bench-external process comparison
# (scripts/bench-external.sh); the bench invokes this script directly so the
# measured command is exactly what this file runs.
set -euo pipefail

exec weasyprint -q "$1" "$2"
