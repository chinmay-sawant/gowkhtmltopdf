# Plans - v0.2.7 real-sites wave

> **Parent:** [`../README.md`](../README.md) (v0.2.7 real-world conversion)
> **Status:** wave 1 fix complete 2026-09-16 (ledger [`01-canonical-0.2.7-real-sites.md`](01-canonical-0.2.7-real-sites.md)). Audit wave 2 ledger [`02-canonical-0.2.7-real-sites-audit2.md`](02-canonical-0.2.7-real-sites-audit2.md): pre-gate and deferred clusters closed 2026-09-17 (LookupRune wiring, root overflow, flex pseudo, ANA-17/18, LCO-13, body wash, ICO, sprite `<use>`, 0-word warn; w3schools-6 upstream pin, no `go.mod` replace). Phase 7 gates green. Phase 8 regen+verify green (`verify.py` 23/23 PASS). Working tree uncommitted for user review.
> **Estimated effort:** delivered in one day with 10 agents across 5 fix waves plus 3 gate agents
> **Depends on:** [`../learncpp/02-canonical-0.2.7-learncpp-visual.md`](../learncpp/02-canonical-0.2.7-learncpp-visual.md) (method), `scripts/real_site_drill.sh`, `scripts/pdf_page_forensics.py`
> **Evidence:** per-site `evidence/` folders (before and after forensics, findings JSON, reports, probes, crops); post-fix artifacts are `2026-09-16-forensics-gowk-after.{json,md}` in each folder.

---

## Overview

The LearnCpp wave proved the method: render the same URL with `bin/gowkhtmltopdf`
and with `wkhtmltopdf 0.12.6.1 (with patched qt)`, then measure every difference
with PyMuPDF and the per-page forensics script. This wave applied the same method
to seven real sites converted on 2026-09-16, one read-only comparison sub-agent per
site pair, then fixed the confirmed engine defects and regenerated every artifact.

No git command was run by the orchestrator. One test-gate agent ran a read-only
`git status` by mistake; it changed nothing.

## Sites

| Slug | URL | gowk artifact (`evidence/`) | wk reference (`evidence/`) | Findings (engine / probe / ref) | After fix |
|------|-----|---------------------------|--------------------------|---------------------------------|-----------|
| ana-de-armas | `https://en.wikipedia.org/wiki/Ana_de_Armas` | `evidence/ana-de-armas.pdf` (22 p) | none, wk segfaults; Chrome fallback | 12 (6 / 4 / 2) | 22 p, 978 annots, 0 corrupted URIs |
| gobyexample | `https://gobyexample.com/` | `evidence/gobyexample.pdf` (3 p) | `evidence/gobyexample_wk.pdf` (2 p) | 6 (2 / 0 / 4) | 3 p, regular h2, wrap fixed |
| learn-cpp-org | `https://www.learn-cpp.org/` | `evidence/learn-cpp-org.pdf` (2 p) | `evidence/learn-cpp-org_wk.pdf` (2 p) | 10 (6 / 1 / 3) | 2 p, buttons trimmed, dock labels fixed |
| cplusplus-tutorial | `https://cplusplus.com/doc/tutorial/` | `evidence/cplusplus-tutorial.pdf` (4 p) | `evidence/cplusplus-tutorial_wk.pdf` (2 p) | 7 (7 / 0 / 0) | 2 p, breadcrumb visible, no phantom pages |
| programiz-cpp | `https://www.programiz.com/cpp-programming` | `evidence/programiz-cpp.pdf` (15 p) | `evidence/programiz-cpp_wk.pdf` (8 p) | 10 (9 / 1 / 0) | 10 p, accordion collapsed, page-1 annots 8 |
| tutorialspoint-cpp | `https://www.tutorialspoint.com/cplusplus/` | `evidence/tutorialspoint-cpp.pdf` (9 p) | `evidence/tutorialspoint-cpp_wk.pdf` (7 p) | 8 (4 / 0 / 4) | 8 p, 1219 stray borders -> 104 drawings |
| geeksforgeeks-cpp | `https://www.geeksforgeeks.org/c-plus-plus/` | `evidence/geeksforgeeks-cpp.pdf` (6 p shell) | `evidence/geeksforgeeks-cpp_wk.pdf` (5 p blank, JS off) | 7 (1 / 2 / 4) | blank shell, matching wk; site-side wall |
| w3schools (cross-site) | `https://www.w3schools.com/cpp/` | not in the original 7 | not produced | 1 blocker | fixed: 7-page PDF, 469 KB |

Housekeeping 2026-09-16: the site PDFs moved from the repo root into each site's
`evidence/` folder (`real-sites/learncpp/evidence/` for the earlier LearnCpp
artifacts; root logs stay in the repo root). The whole real-sites `evidence/`
folder is ignored in `.gitignore` (PDF, JSON, PNG, Markdown, probe HTML/CSS,
logs, text dumps), and root `*.pdf`/`*.log` are ignored too. Probe fixtures live
under each site's `evidence/2026-09-16-probes/` folder (gobyexample wrap/font
probes and contact sheets, Wikipedia q-probes, learn-cpp.org nav probes, cplusplus
header/visibility/marker probes, Programiz A/B mirrors and base capture,
tutorialspoint bullet probe); generate future probes there only, never in the repo
root.

Per-site evidence: [`ana-de-armas/evidence/`](ana-de-armas/evidence/),
[`gobyexample/evidence/`](gobyexample/evidence/),
[`learn-cpp-org/evidence/`](learn-cpp-org/evidence/),
[`cplusplus-tutorial/evidence/`](cplusplus-tutorial/evidence/),
[`programiz-cpp/evidence/`](programiz-cpp/evidence/),
[`tutorialspoint-cpp/evidence/`](tutorialspoint-cpp/evidence/),
[`geeksforgeeks-cpp/evidence/`](geeksforgeeks-cpp/evidence/),
[`w3schools/evidence/`](w3schools/evidence/).

## Method

1. Convert each URL with `bin/gowkhtmltopdf` (print media default, images on) and
   with `wkhtmltopdf 0.12.6.1` (JS on, patched qt, screen media default).
2. Per PDF: `python3 scripts/pdf_page_forensics.py <pdf> --json <evidence>.json`,
   page PNGs at 150 dpi, PyMuPDF word and box probes, text diff after whitespace
   normalization, and a Chrome print render where media or JS questions needed a
   third opinion.
3. Classify each difference: engine defect, reference artifact (Qt WebKit limits or
   smart shrink), expected no-JS difference, data/media artifact (bot wall, missing
   asset), or needs probe.
4. Fix engine defects in package-owned waves with red-first tests; rerun the gates.
5. Regenerate every site artifact and compare before/after forensics.

Matched-media note: `wkhtmltopdf` defaults to screen media and `bin/gowkhtmltopdf`
to print, so matched print references were rendered for all six convertible sites
under `/tmp/opencode/real-sites/wk-print/`.

## Constraints

- Findings wave was read-only; the fix wave changed engine code and tests only.
- No git commands. No new third-party dependencies.
- Rows close on proof: every fixed row cites a named test and, where applicable, a
  regenerated artifact number.
- The catalogue JSON did not need a refresh: no CSS property coverage changed.

## Validation record

| Date | Phase | Command | Outcome |
|------|-------|---------|---------|
| 2026-09-16 | findings | `bin/gowkhtmltopdf --url <url> -o <slug>.pdf` | 7 artifacts; w3schools blocked (403 then empty-cmap fatal) |
| 2026-09-16 | findings | `wkhtmltopdf --quiet <url> <slug>_wk.pdf` | 5 clean; GeeksforGeeks hung with JS on, blank with JS off; Wikipedia segfaulted in five configurations |
| 2026-09-16 | findings | `google-chrome --headless=new --print-to-pdf` | Wikipedia Chrome fallback reference, exit 0 |
| 2026-09-16 | findings | 7 read-only comparison agents | 60 findings archived under `evidence/` |
| 2026-09-16 | fix | 10 agents across 5 waves | layout hidden boxes/border:0, inline measurement + q-quotes + font keywords + \r, image/page-1 annots, placeholder + pre break-inside + fixed chrome; pdf empty-cmap + URI encoding + bullet fold + face-keyed runes + weight faces; svg gradient/viewBox/font preprocess |
| 2026-09-16 | 0 | `wkhtmltopdf --print-media-type` x6 | all exit 0; print refs in `/tmp/opencode/real-sites/wk-print/` |
| 2026-09-16 | gates | `make lint` | exit 0 (size-check clean, frontend lint clean) |
| 2026-09-16 | gates | `make test` | exit 0; fixture-61 envelope [5,8] -> [5,9], verified by a fresh 9-page conversion; no needle edits |
| 2026-09-16 | gates | `make golden` / catalogue check / `make claim-scan` | exit 0 (71 pass) / exit 0 (240 arms) / clean |
| 2026-09-16 | regen | `bin/gowkhtmltopdf` for 7 sites + w3schools | all exit 0; before/after in per-site `evidence/2026-09-16-forensics-gowk-after.*` |
