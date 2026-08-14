# Phase 8 — Zerodha-Style Benchmark (gocorepdfengine only)

**Status:** Harness on local engine (JSON → model → layout → PDF)  
**No gopdfsuit dependency.** Templates only inspired by the Zerodha gold-standard mix.

**Template field contract (moved into this repo):**  
→ **[guides/TEMPLATE_REFERENCE.md](../guides/TEMPLATE_REFERENCE.md)**

That guide is the full generic PDF template shape (`config`, `title`, `elements`, `footer`, props, bookmarks, forms, images, PDF/A flags). Zerodha bench uses a **lighter domain JSON** today; the mapping is documented below and in the guide’s “Zerodha domain JSON” section.

---

## Goal

Benchmark gocorepdfengine’s **layout + coloring + document assembly** on a Zerodha-like 80/15/5 workload:

| Tier | Share | Source |
|------|-------|--------|
| Retail | 80% | `sampledata/zerodha/retail_investor.json` |
| Active | 15% | `active_trader.json` → expand **40** trades |
| HFT | 5% | `hft_algo.json` → expand **2000** trades |

Two dimensions:

1. **Compliance:** compliant (A-4 + UA-2 flags) vs non-compliant (PDF 2.0 only)  
2. **Model cache:** cached (expand once) vs non-cached (rebuild every iteration)

---

## Template docs (source of truth)

| Document | Role |
|----------|------|
| [guides/TEMPLATE_REFERENCE.md](../guides/TEMPLATE_REFERENCE.md) | Full target JSON template (config / tables / cells / props / …) |
| [sampledata/zerodha/README.md](../sampledata/zerodha/README.md) | Bench harness usage |
| This plan | Workload, pipeline, checklists, cache modes |

### Why both formats?

| Format | When used |
|--------|-----------|
| **Domain JSON** (`client`, `trades`, …) | Current Zerodha bench — small, stable fixtures |
| **Full PDFTemplate JSON** (TEMPLATE_REFERENCE) | Future general renderer; layout should honor the same props/colors/tables |

Domain JSON is **not** a subset of `elements[]`; it is mapped by `engine/render` into layout tables that *look like* the TEMPLATE_REFERENCE financial examples (header colors, section rows, trade grids).

---

## Pipeline (local packages only)

```
sampledata/zerodha/*.json          (domain)
        │
        ▼
engine/model.LoadJSON + ExpandTrades
        │
        ▼
engine/render.BuildTable           ← colors from engine/color theme
        │                              (aligned with TEMPLATE_REFERENCE bgcolor/textcolor)
        ▼
engine/layout.TableLayout.LayOut   ← multipage, fills, borders, text
        │                              (target: full props L:R:T:B + align)
        ▼
engine.GenerateDocument            ← multi-page PDF 2.0 / A-4 / UA-2
```

**Future (full template path):**

```
*.json  (TEMPLATE_REFERENCE shape)
        │
        ▼
LoadPDFTemplate → layout from title/elements/footer
        │
        ▼
engine.GenerateDocument
```

---

## Domain JSON ↔ TEMPLATE_REFERENCE map

| Domain / render today | TEMPLATE_REFERENCE field |
|-----------------------|--------------------------|
| Compliant bench mode | `config.pdfaCompliant` + tagged/PDF-UA |
| `features.watermark` | `config.watermark` |
| `metadata.title` | `config.pdfTitle` / title text |
| Section header row `#21618C` | cell `bgcolor` + `textcolor` |
| Header bar `#154360` | title table / first row colors |
| Trade grid | `elements[]` table with `maxcolumns`, `columnwidths`, `rows` |
| Buy/sell colors | cell `textcolor` (`#27AE60` / `#E74C3C`) |
| Alt row `#F8F9F9` | cell `bgcolor` |
| Fixed font sizes | props `Helvetica:size:…` |
| Borders (uniform) | props `…:L:R:T:B` (per-side still a gap) |
| `digital_signature` in retail JSON | `config.signature` (**engine gap**) |
| Internal links / dests (not yet) | cell `link` / `dest`, `bookmarks` |
| Footer line (not yet in render) | `footer.font` + `footer.text` |

See also: [Zerodha domain JSON section](../guides/TEMPLATE_REFERENCE.md#zerodha-domain-json-current-bench) in the template guide.

---

## Checklist — already landed

- [x] TEMPLATE_REFERENCE moved to `guides/TEMPLATE_REFERENCE.md` (adapted for this repo)  
- [x] Remove gopdfsuit require/replace and generate wrapper  
- [x] JSON templates under `sampledata/zerodha/`  
- [x] `engine/model` JSON → `ContractNote`  
- [x] `engine/color` hex + theme palette (header/section/buy/sell/alt rows)  
- [x] `engine/layout` table layout + `StyledCell` colors/borders  
- [x] `engine/render` contract-note builder  
- [x] `engine.GenerateDocument` multi-page assembly  
- [x] Bench harness: compliant / nocomply, `BENCH_CACHE` on/off  
- [x] Makefile: `bench-zerodha*`, `bench-zerodha-cached`, `bench-zerodha-uncached`  
- [x] x5 / x10 run scripts  

---

## Checklist — layout / coloring vs TEMPLATE_REFERENCE — COMPLETED

All items in this section are done:

- [x] Parse props string: `font:size:style:align:L:R:T:B` (`engine/layout/props.go`)  
- [x] Text wrapping inside cells (`engine/layout/layout.go` `WrapText`)  
- [x] Align left / center / right (`engine/layout/table.go` `LayOut`)  
- [x] Per-side borders L:R:T:B (`engine/layout/table.go` `drawSide`)  
- [x] `bgcolor` / `textcolor` from JSON (via `note.Footer` model + `UsedText` font coverage)  
- [x] True diagonal watermark (`engine/layout/layout.go` `PlaceWatermark`)  
- [x] Footer (`DocumentConfig.FooterText` read from JSON `footer.text`)  
- [x] Page numbers ("Page X of Y") at bottom-right, right-aligned within page  
- [x] CID hex encoding fix (`TjCID` instead of `Tj` for Identity-H fonts)  
- [x] Full ASCII alphabet added to font subset for footer/page-number glyphs

---

## Checklist — future (post Phase 8)

Items not yet started — deferred to later phases:

- [ ] Full PDFTemplate loader: types, loader, mapper from `config`/`title`/`elements`/`footer`/`bookmarks` JSON  
- [ ] `digital_signature` block → `config.signature`  
- [ ] Shared-row HFT fast path (optional perf)  
- [ ] Internal links / bookmarks (cell `link`, `dest`, bookmark tree)  
- [ ] Publish baseline ops/sec for cached vs uncached (compliant + nocomply)  
- [ ] Form fields, images, security/encryption (earlier phases)  

---

## Checklist — JSON → model (domain)

- [x] Load retail / active / hft JSON  
- [x] Expand active 40 / HFT 2000 with seed  
- [x] Footer model (`Footer` struct with font/text/link) read from JSON  
- [x] Cache modes: cached (`BENCH_CACHE=1`) and uncached (`BENCH_CACHE=0`)  

---

## Makefile targets

```bash
make bench-zerodha                 # compliant, cache ON
make bench-zerodha-nocomply        # PDF 2.0 only
make bench-zerodha-cached          # BENCH_CACHE=1
make bench-zerodha-uncached        # BENCH_CACHE=0
make bench-zerodha-x2 | x5 | x10
make bench-zerodha-nocomply-x10
```

---

## Acceptance — Phase 8 done

- [x] `make bench-zerodha` succeeds (compliant, cache ON)  
- [x] `make bench-zerodha-nocomply` succeeds  
- [x] `make bench-zerodha-cached` / `bench-zerodha-uncached`  
- [x] Warm-up PDFs written under `sampledata/zerodha/`  
- [x] HFT multi-page (28 pages for 2000 trades)  
- [x] Colored header/section/action cells visible  
- [x] TEMPLATE_REFERENCE under `guides/`  
- [x] Font renders correctly (TjCID hex strings)  
- [x] Footer from JSON: `Zerodha Broking Ltd. | … | Confidential`  
- [x] Page numbers bottom-right, right-aligned within page  
- [x] Header titles fit without wrapping

### Compliance quality (ongoing)

- [ ] veraPDF `-f 4` and `-f ua2` pass (glyph-width mismatch still open)  
- [ ] Richer UA tagging (TD/TH MCIDs)  

---

## Explicit non-goals

- gopdfsuit as a generator backend  
- ECDSA/RSA signing in this harness (until phase 7 + TEMPLATE_REFERENCE signature section)  
- Modifying the gopdfsuit repo  
- HTTP template-pdf API (guide documents library usage only)  
