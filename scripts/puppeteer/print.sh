#!/usr/bin/env bash
# print.sh <input.html> <output.pdf>
#
# Renders an HTML file to PDF through headless Chrome using puppeteer-core.
# This is the Puppeteer command side of the bench-external process comparison
# (scripts/bench-external.sh); the bench invokes this script directly so the
# measured command is exactly what this file runs. The puppeteer-core module
# lives in scripts/puppeteer/node_modules (npm ci --prefix scripts/puppeteer).
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

exec node "$script_dir/../puppeteer_print.js" "$1" "$2"
