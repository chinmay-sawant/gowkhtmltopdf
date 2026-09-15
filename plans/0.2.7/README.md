# Plans - v0.2.7 (LearnCpp real-world conversion)

| File / Folder | Role |
|---------------|------|
| [learncpp/01-canonical-0.2.7-learncpp.md](learncpp/01-canonical-0.2.7-learncpp.md) | Canonical v0.2.7 execution ledger, phases 1-8: learncpp.com conversion fixes |

Workflow: [`../../skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [`../0.2.6/48-canonical-0.2.6-css-coverage.md`](../0.2.6/48-canonical-0.2.6-css-coverage.md) (CSS coverage, released 2026-09-13).

## Scope in one line

Fix the pipeline defects exposed by converting `https://www.learncpp.com/` with the CLI: image payload safety, paint z-order, stylesheet base URLs, font sources, flex/calc width resolution, media selection, and PDF artifact polish. Code wave first; docs and frontend follow as a separate ledger.

## Verification

Baseline captured 2026-09-16 on `master` (VERSION 0.2.6):

- `bin/gowkhtmltopdf --url https://www.learncpp.com/ -o learncpp.pdf` -> exit 1 after 89s, 0-byte PDF (`learncpp_conversion.log`)
- `bin/gowkhtmltopdf --no-images --url https://www.learncpp.com/ -o learncpp_noimages.pdf` -> exit 0 after 82s, 37 pages, text layer complete but 34/37 pages visually blank under the content pane

After the fix wave (2026-09-16, same VERSION 0.2.6 tree plus the 0.2.7 changes):

- `scripts/real_site_drill.sh https://www.learncpp.com/ learncpp.pdf learncpp_v027.log --baseline ...` -> exit 0, 24s, 269,184 bytes, 19 pages, 0 low-ink pages, tagged Open Sans + Liberation subsets embedded (`learncpp_v027.log`)
- Gates run 2: `make test` TEST_EXIT=0, `make lint` LINT_EXIT=0, `make golden` GOLDEN_EXIT=0
- Evidence: [learncpp/evidence/](learncpp/evidence/) (baseline and after reports, contact sheet)
