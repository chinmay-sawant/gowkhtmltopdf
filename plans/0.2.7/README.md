# Plans - v0.2.7 (real-world conversion)

| File / Folder | Role |
|---------------|------|
| [learncpp/01-canonical-0.2.7-learncpp.md](learncpp/01-canonical-0.2.7-learncpp.md) | Canonical v0.2.7 execution ledger, phases 1-8: learncpp.com conversion fixes |
| [learncpp/02-canonical-0.2.7-learncpp-visual.md](learncpp/02-canonical-0.2.7-learncpp-visual.md) | Canonical visual ledger, phases 9-15: wkhtmltopdf comparison defects (header branding, chapter badge, print URL spam, pagination paint) plus the mandatory catalogue JSON refresh. Complete 2026-09-16; gates green, 3 deferred rows |
| [real-sites/01-canonical-0.2.7-real-sites.md](real-sites/01-canonical-0.2.7-real-sites.md) | Real-sites comparison wave, 7 sites vs wkhtmltopdf 0.12.6.1 and Chrome: findings complete and fix wave landed 2026-09-16 - 60 findings (35 engine defects, 8 probes, 17 reference artifacts), 30+ defects fixed with red-first tests, 8 rows deferred with next gates, all site PDFs regenerated, gates green (lint / test / golden / catalogue / claim-scan) |
| [real-sites/README.md](real-sites/README.md) | Wave index: site table, method, media caveat, evidence map, validation record |

Workflow: [`../../skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [`../0.2.6/48-canonical-0.2.6-css-coverage.md`](../0.2.6/48-canonical-0.2.6-css-coverage.md) (CSS coverage, released 2026-09-13).

## Scope in one line

Fix the pipeline defects exposed by converting `https://www.learncpp.com/` with the CLI: image payload safety, paint z-order, stylesheet base URLs, font sources, flex/calc width resolution, media selection, and PDF artifact polish. Code wave first; docs and frontend follow as a separate ledger.

Visual wave (phases 9-15, `learncpp/02-canonical-0.2.7-learncpp-visual.md`): a
wkhtmltopdf side-by-side found header branding, chapter badge, print URL spam, and
pagination paint defects that artifact checks cannot see. That ledger ends with a
mandatory catalogue JSON refresh (`plans/0.2.6/catalog/`). Complete 2026-09-16:
tagline on one line, `Chapter N` badge inside the card with its number, zero URL
suffixes in print (19 -> 13 pages), row 21.2 inside its band, catalogue re-pointed
with no status moves, and `make test` / `make lint` / `make golden` green.

Real-sites wave (`real-sites/01-canonical-0.2.7-real-sites.md`): the same
picture-diagnosis method applied to seven new sites (Wikipedia Ana de Armas, Go by
Example, learn-cpp.org, cplusplus.com tutorial, Programiz, TutorialsPoint,
GeeksforGeeks), one read-only comparison agent per pair against wkhtmltopdf
0.12.6.1 and Chrome print. Findings complete and fixed 2026-09-16: 60 findings
(35 engine defects, 8 probes, 17 reference artifacts), 30+ defects fixed across
`internal/layout`, `internal/pdf`, and `internal/svg` with red-first tests, 8 rows
deferred with next gates, all seven site PDFs regenerated plus a new w3schools PDF.
Gates: `make lint` exit 0, `make test` exit 0, `make golden` exit 0 (71 pass),
catalogue check exit 0, claim-scan clean.

## Verification

Baseline captured 2026-09-16 on `master` (VERSION 0.2.6):

- `bin/gowkhtmltopdf --url https://www.learncpp.com/ -o learncpp.pdf` -> exit 1 after 89s, 0-byte PDF (`learncpp_conversion.log`)
- `bin/gowkhtmltopdf --no-images --url https://www.learncpp.com/ -o learncpp_noimages.pdf` -> exit 0 after 82s, 37 pages, text layer complete but 34/37 pages visually blank under the content pane

After the fix wave (2026-09-16, same VERSION 0.2.6 tree plus the 0.2.7 changes):

- `scripts/real_site_drill.sh https://www.learncpp.com/ learncpp.pdf learncpp_v027.log --baseline ...` -> exit 0, 24s, 269,184 bytes, 19 pages, 0 low-ink pages, tagged Open Sans + Liberation subsets embedded (`learncpp_v027.log`)
- Gates run 2: `make test` TEST_EXIT=0, `make lint` LINT_EXIT=0, `make golden` GOLDEN_EXIT=0
- Evidence: [learncpp/evidence/](learncpp/evidence/) (baseline and after reports, contact sheet)

After the visual wave (2026-09-16, final tree):

- Print `real-sites/learncpp/evidence/learncpp.pdf` 226,441 bytes / 13 pages (was 19), no-images 13 pages, screen render 13 pages; all with 0 URL suffixes, 0 literal `()`, 0 covered-text pages. (PDFs moved out of the repo root on 2026-09-16; the command lines above keep their historical output names.)
- Picture verification (V8/V9): tagline one line 125.65pt, `Chapter 0` digit 6.0pt inside the pill, all 310 row numbers inside their bands, 55 headings / 310 rows intact, fonts subset-tagged
- Gates run 4 (post-lint): TEST_EXIT=0, LINT_EXIT=0, GOLDEN_EXIT=0
- Catalogue: `scripts/css-catalog-map.py --check` exit 0, counts unchanged 354/0/464/0
- Deferred: 9.4c fixed-header chain offset, catalogue map-scan extension (240 of 354 arms)

Real-sites wave (findings only, 2026-09-16):

- Seven gowk artifacts and six wkhtmltopdf references produced; Wikipedia wk segfaults (exit 139, five configs) and Chrome print is the fallback reference; GeeksforGeeks is a bot wall on both sides
- 60 findings with per-site reports, findings JSON, forensics JSON, probes, and crops under [real-sites/](real-sites/)
- No engine code changed, no builds, no test or lint runs in this wave
