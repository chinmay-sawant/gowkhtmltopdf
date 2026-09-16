# Diagnose fixture picture - learncpp no-images PDF, evidence file

Date: 2026-09-16. Diagnosis only, no fixes, no repo changes.

## Artifacts

- Target: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/learncpp_noimages.pdf` (19 pages, print media, `--no-images`, regenerated 2026-09-16; 0 low-ink pages)
- Companion: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/learncpp.pdf` (19 pages, print media, images on)
- Our screen-media render: `/tmp/opencode/learncpp/wk/learncpp_gowk_screen.pdf` (13 pages)
- wkhtmltopdf reference (screen media default): `/tmp/opencode/learncpp/wk/learncpp_wk.pdf` (16 pages, 7s)
- Page PNGs (150 dpi):
  - `/tmp/opencode/learncpp/diag_noimages/learncpp_noimages-p01.png` ... `-p19.png`
  - `/tmp/opencode/learncpp/wk/pages/learncpp_wk-p01.png` ... (reference)
  - `/tmp/opencode/learncpp/wk/pages_screen/learncpp_gowk_screen-p01.png` ...
- Web mirror for CSS/HTML grounding: `/tmp/opencode/learncpp/home.html`, `/tmp/opencode/learncpp/css/*.css`, `/tmp/opencode/learncpp/inline_style_0.css`

Commands used:

- `wkhtmltopdf https://www.learncpp.com/ learncpp_wk.pdf` -> exit 0, 7s, 581503 bytes, 16 pages
- `bin/gowkhtmltopdf --media-type screen --url https://www.learncpp.com/ -o learncpp_gowk_screen.pdf` -> exit 0, 7s, 270221 bytes, 13 pages, OpenSans embedded
- `python3 scripts/pdf_page_forensics.py <pdf>` for per-page numbers
- PyMuPDF word/drawing probes for geometry

Content parity is close: words 2132 (ours screen) vs 2129 (wk); links 358 vs 359.

## Measured comparison, page 1

| Component | wkhtmltopdf (reference) | gowk screen | gowk print / no-images |
|---|---|---|---|
| Header title "LEARN" | x0=62.1 y0=34.5 x1=93.0 (compact bar) | not found on page 1 | x0=39.3 y0=68.9 x1=84.8 |
| Tagline "Skill up..." | one line under the title (visible in `/tmp/opencode/learncpp/wk/pages/learncpp_wk-p01.png`) | fragments clipped at page top ("our", "fre", "e", "s" at y<40pt) | 13pt-wide column, 17 lines, y=86.6..252.1; underlines under every word |
| Chapter title "Introduction..." | x0=42.3 y0=270.3 | x0=55.5 y0=433.6 | x0=58.6 y0=538.2 |
| "Chapter" badge word | x0=464.8 y0=267.4 x1=496.8 (then "0" follows inside the badge) | x0=521.8 y0=429.6 x1=562.1 | x0=516.7 y0=534.2 x1=563.7 |
| Badge rect | [460.0, 265.0, 509.2, 279.4] fill #6daaf3, ends inside the card | [515.8, 426.9, 566.9, 445.0], right edge == card clip edge 566.9 | [510.7, 531.7, 566.9, 551.5], right edge == card clip edge 566.9 |
| Chapter number "0" | inside the badge: "Chapter 0" | not found near the badge | [567.3, 534.2, 574.7, 548.9] = 0.4pt past the card edge, clipped |
| Row 0.1 structure | stacked (badge line then title line), dy=18.0pt | stacked dy=25.9pt | stacked dy=35.8pt (print adds the URL line) |

Note: the "stacked" numbers above compare the chapter title word to the row badge, not the lesson row itself. Lesson rows in gowk render badge and title side by side on one line; wk rows stack badge above title at its 1024px viewport.

## CSS facts (mirror paths)

- `/tmp/opencode/learncpp/css/10_style.css:680` `#site-text { position: relative; top: 50%; display: inline-block; float: left; transform: translateY(-50%); }`
- `:738` `#site-description { display: none; clear: left; float: left; font-size: .9em; line-height: 1.3; opacity: .5; }`
- `:6480` `#masthead.cryout #site-text { transform: none; text-decoration: underline; }`. Critic correction (2026-09-16): this rule sits inside `@media print` (block around 6404-6558), not in the `max-width: 1152px` block. Chrome at 718px: screen keeps `transform: translateY(-50%)`; print drops it and adds the underline.
- `:6487` `#site-title a span { color: inherit; font-weight: 400; padding: 0; text-decoration: underline; }`
- `:5967` starts `@media (max-width: 1152px)`; `:609` starts `@media (min-width: 1152px)`.
- `/tmp/opencode/learncpp/css/14_18062.css:464` `.lessontable-row:nth-child(odd){ background:#f4f6fd }` (the rule that fed the old fatal)
- `:442-503` `.lessontable-row`, `.lessontable-row-number { width:46px; min-width:46px }`, `.lessontable-row-title`, `.lessontable-row-tag { margin-left:auto }`
- Chapter header markup: `<div class=lessontable-header-chapter>` blocks; grep the full markup in `home.html`.

## Open questions for the council

1. Why does the tagline column collapse to min-content (~13pt) instead of one/few lines? Is it `#site-text` float/inline-block shrink-to-fit, the float chain (`float:left` + `clear:left`), or the `top:50%`/transform handling after the Phase 5 inset changes?
2. Which media branch does the engine apply for the header (>=1152 desktop rules like `top:50%`/`display:none`, or <=1152 mobile rules like `transform:none`/underline)? Both symptoms appear mixed in one render.
3. Why is the "Chapter N" number word at x=567.3 (past the card's 566.9 clip edge) while the badge ends at 566.9? Is the chapter header a flex row whose right item overflows (same class as the 12.7pt text-overflow warning) or a wrong item order?
4. Should lesson rows stack (wk at 1024px) or sit side by side at our ~718px CSS viewport? Find the media rule that decides this and whether the engine applies it.

## Constraints

- Read-only. No repo edits, no fixes, no `git` commands.
- Scratch under `/tmp/opencode/learncpp/council/` if needed.
- Evidence over memory: measure with PyMuPDF or quote CSS/HTML.

## Council verdicts (2026-09-16)

Three read-only agents (Analyst, Interpreter, Critic) ran against this evidence.
Independent reproduction: the measurements re-derived to 0.1pt with PyMuPDF.

| Claim | Verdict | Basis |
|---|---|---|
| Tagline collapses to a ~13pt column | Real engine defect | Overrides on the live page: `#site-text{float:none}` or any explicit width on `#site-text`/`#branding` flips 12 fragments (max 12.65pt) to one line (125.7pt). Chrome at the same 718px and media renders one line (155.4px screen, 180.9px print). |
| Screen render clips the header | Real engine defect, same root cause | Same collapse plus the screen-only `transform: translateY(-50%)`; killing the transform moves the title back to y approx 69. Print media drops the transform, so only screen clips. |
| Chapter badge clipped, `0` invisible | Real engine defect | Badge needs ~68pt but gets 56.2pt; right edge laid out at x=580.7 while the card clip is 566.9; the white `0` glyph survives past the clip on the white margin. HTML text is one NBSP run `Chapter\xa00`. |
| Lesson rows side by side vs wk stacked | Killed: reference artifact | wkhtmltopdf 0.12.6.1 Qt WebKit ignores unprefixed `display:flex`, so it stacks. Chrome at 718px and 1024px puts badge and title side by side, matching our engine. No media rule changes row direction. |
| Page 19 row 21.2 "empty" | Real engine defect, print-only | Number painted at y=178.2 in the gap between bands (21.1 band ends 172.8, 21.2 band starts 219.3), badge background empty, title at y=222.4. Screen render is correct. Mechanism needs a dedicated probe. |
| Tagline underline on a float | Real engine defect | CSS 2.1 16.3.1 excludes floats from `text-decoration` propagation. A/B: `text-decoration:none` on `#site-description` drops 12 underline strokes to 0. |
| Title "LEARN C++" 6.1pt overlap | Real engine defect, trigger not isolated | The second word advance matches untransformed lowercase text while uppercase is painted. Needs its own probe. |
| Print appends `(url)` to 356 links | Real engine defect | Site rule is `.cryout p a::after`; anchors sit in `<div>`s. Home has 6 `<p>` opens and 0 closes; Chrome implicitly closes `<p>` at block starts, our parser does not, so every anchor matches. Chrome print adds no URL text. |
| Mobile menu icon renders as `()` | Data/media artifact | The icon font data is absent in the offline no-images run; not a layout op. |

Reproducibility trap: probes must use the live URL or an HTTP-served mirror.
The mirror served from `file://` does not load the protocol-relative `18062.css`
and therefore does not reproduce the collapse.
