# gowkhtmltopdf

Pure-Go rewrite plan for [wkhtmltopdf](wkhtmltopdf/) — convert HTML to PDF (and images) **without third-party libraries** (Go **standard library only**: no module dependencies, no Chrome/WebKit, no cgo).

This repository currently contains:

| Path | Contents |
|------|----------|
| [`wkhtmltopdf/`](wkhtmltopdf/) | Upstream wkhtmltopdf 0.12.7-dev sources (analysis target) |
| [`plans/`](plans/) | Canonical phase-wise execution ledger + phase reports |
| [`skills/phase-wise-checklist/`](skills/phase-wise-checklist/) | Planning skill used for ledgers |
| **This README** | Scope, architecture summary, **time estimates** |

**Implementation has not started.** Plans are evidence-backed from a multi-agent source analysis of the C++/Qt tree.

---

## Critical reality check

**wkhtmltopdf is not a PDF engine.** It is a thin orchestration layer (~10.6k lines of application C++) on top of:

- **Qt WebKit** — HTML/CSS/JS layout and print pagination (`QWebPage`, patched `QWebPrinter`)
- **Qt print** — PDF emission (`QPrinter` / `QPainter`, including links, forms, outlines)

A pure-stdlib Go rewrite is therefore **not** “port the C++.” It is **build a document layout engine + PDF writer**, then re-home the CLI/settings/load pipeline.

| Goal | Feasible under stdlib-only? |
|------|------------------------------|
| CLI + settings + loader orchestration | Yes |
| PDF writer (text, images, links, outlines) | Yes (substantial) |
| Controlled invoice/report HTML → PDF (no JS, CSS subset) | Yes, **hard**, multi-year for one senior |
| Drop-in parity for arbitrary web pages / JS | **No** (not on a product timeline) |

Upstream maintainers already recommend WeasyPrint/Prince for controlled reports and Puppeteer for JS-heavy pages (`wkhtmltopdf/docs/status.md`). Our MVP aligns with the **controlled report** niche — implemented in pure Go.

---

## Time estimates

Assumptions:

- Engineers: senior, already comfortable with systems/PDF/fonts or willing to learn mid-flight  
- Constraint: **Go stdlib only** (inflates cost vs allowing pure-Go modules)  
- Full-time focus; ~20% schedule risk buffer not fully included in ranges  
- “Done” for MVP = golden corpus of controlled templates, not WebKit pixel parity  

### By product scope

| Scope | What you get | Person-months | Solo senior (calendar) | 2 seniors (calendar) |
|-------|--------------|---------------|------------------------|----------------------|
| **A. CLI + settings shell** | Flag-compatible CLI, settings structs, convert stub | 2–3 PM | **2–3 months** | **1–1.5 months** |
| **B. MVP reports** | Subset HTML/CSS → multi-page PDF, text HF, basic outline/TOC, loader, golden tests | **18–28 PM** | **18–30 months** | **10–18 months** |
| **C. Intermediate** | HTML HF, richer CSS (floats/partial flex), better tables/breaks, image CLI polished | **45–70 PM** | **~3.5–6 years** | **~2–3.5 years** |
| **D. Full wkhtmltopdf / WebKit-like** | JS, modern CSS, complex scripts, print edge cases | **200–500+ PM** (still incomplete) | **Not credible** | **Not credible** |

### By implementation phase (canonical plan)

See [`plans/00-canonical-pure-go-rewrite.md`](plans/00-canonical-pure-go-rewrite.md).

| Phase | Work | Solo calendar (order) |
|------:|------|------------------------|
| 0 | Scope freeze, allowlist, module scaffold | 0.5–1 month |
| 1 | Settings + full CLI parse surface | 1–1.5 months |
| 2 | Multi-resource loader / network / ACL | 0.75–1.5 months |
| 3 | PDF object model, fonts, images, links | 2–3 months |
| 4 | **HTML + CSS subset layout** (critical path) | **6–12 months** |
| 5 | Print pagination, multi-object assembly | 2–4 months |
| 6 | Headers/footers, TOC, outlines, links | 2–4 months |
| 7 | `wkhtmltoimage`-style raster CLI | 1–2 months |
| 8 | Idiomatic Go library API | 1–1.5 months |
| 9 | Golden corpus, security, release gates | 2–4 months (overlap) |

**MVP ship path:** Phases **0–6 + 9** (Phase 7–8 can trail).  
**Long pole:** Phase 4 (layout). Parallelize Phase 3 (PDF) with Phase 1–2.

### Component cost (stdlib-only)

| Component | Stdlib coverage | Effort (person-months) |
|-----------|-----------------|------------------------|
| HTTP / file / cookies / proxy / ACL | High (`net/http`, `crypto/tls`) | 0.5–2 |
| HTML subset parser | None useful | 1–6 |
| CSS parse + cascade + box/table layout | None | **8–14** (MVP) · 30–80+ (richer) |
| JavaScript + DOM | None | **Omit for MVP** · 24–48+ for toy ES5 |
| Font load + Latin subset embed | None | 3–6 |
| Complex script shaping (HarfBuzz-class) | None | 18–36+ (avoid) |
| JPEG/PNG/GIF | Yes (`image/*`) | 0.5–1 |
| SVG / WebP | No | Defer |
| PDF writer + compression | `compress/flate` only | 2–4 core · +3–6 fonts/links |
| Page fragmentation / print CSS | None | 2–4 (MVP) · 12–24+ (high fidelity) |
| HF / TOC / outline app logic | N/A | 2–5 |
| CLI + settings + exit codes | N/A | 2–3 |

### Effort if policy changed (reference only)

Not the current mandate, but useful for comparison:

| Policy | MVP calendar (solo) | Why different |
|--------|---------------------|---------------|
| **Stdlib only** (this project) | 18–30 months | Reimplement HTML/PDF/fonts |
| Pure-Go modules allowed (still no Chrome) | ~12–20 months | Reuse HTML parser/PDF/font pkgs; **layout still custom** |
| Headless Chrome / chromedp driver | ~1–3 months for CLI wrapper | Not a pure engine; not stdlib-only |

---

## Architecture (target)

```
cmd/gowkhtmltopdf ──► settings + CLI
                         │
                         ▼
                   convert (phases)
                    │         │
                    ▼         ▼
                  load      layout (HTML/CSS subset)
                    │         │
                    │         ▼
                    │      paginate + outline/HF/TOC
                    │         │
                    └────────►│
                              ▼
                         pdf writer ──► file / stdout / bytes
```

Upstream pipeline mirrored from `PdfConverterPrivate`:

1. Load pages (and measure HTML headers/footers if needed)  
2. Count pages (layout + pagination)  
3. Load TOC (optional)  
4. Resolve links  
5. Load headers/footers  
6. Print / write PDF  

---

## Plans index

| Document | Purpose |
|----------|---------|
| **[plans/00-canonical-pure-go-rewrite.md](plans/00-canonical-pure-go-rewrite.md)** | **Canonical live execution ledger** (checkboxes) |
| [plans/phases/phase-00-scope-foundations.md](plans/phases/phase-00-scope-foundations.md) | Scope, allowlist, scaffold |
| [plans/phases/phase-01-settings-cli.md](plans/phases/phase-01-settings-cli.md) | Settings + CLI |
| [plans/phases/phase-02-loader-network.md](plans/phases/phase-02-loader-network.md) | Loader / network |
| [plans/phases/phase-03-pdf-writer.md](plans/phases/phase-03-pdf-writer.md) | PDF writer |
| [plans/phases/phase-04-html-css-layout.md](plans/phases/phase-04-html-css-layout.md) | HTML/CSS layout |
| [plans/phases/phase-05-pagination-print.md](plans/phases/phase-05-pagination-print.md) | Pagination / multi-object |
| [plans/phases/phase-06-headers-toc-outline.md](plans/phases/phase-06-headers-toc-outline.md) | HF / TOC / outlines / links |
| [plans/phases/phase-07-image-converter.md](plans/phases/phase-07-image-converter.md) | Image CLI |
| [plans/phases/phase-08-library-api.md](plans/phases/phase-08-library-api.md) | Go API |
| [plans/phases/phase-09-hardening-closure.md](plans/phases/phase-09-hardening-closure.md) | Release gates |
| [plans/exploration/](plans/exploration/) | Multi-agent analysis notes |

Plan format follows [`skills/phase-wise-checklist/SKILLS.md`](skills/phase-wise-checklist/SKILLS.md).

---

## MVP product contract (recommended)

**Supported (goal):**

- Server-generated HTML (templates), **no JavaScript**
- Documented CSS subset (box model, fonts, colors, tables, basic text, page-break-*)
- Local files (with ACL) and HTTP(S) fetch
- Multi-page PDF: page size, margins, orientation, grayscale, compression
- Text headers/footers with `[page]`-style placeholders
- PDF outlines + simple TOC
- Internal/external links
- Single static binary (`CGO_ENABLED=0`)

**Explicitly out of MVP:**

- JS execution / SPAs / `window.status` real waits  
- Full CSS (Grid, modern Flex, filters, transforms, web fonts runtime)  
- Custom XSLT TOC stylesheets (no XSLT in stdlib; Go templates instead)  
- PDF encryption, PDF/A, duplex (also absent or irrelevant upstream)  
- Pixel-identical output vs Qt WebKit  
- Safe rendering of **untrusted** HTML (same warning as upstream)

---

## Exploration method

Five read-only explore subagents covered:

1. Architecture & conversion pipeline  
2. PDF converter, settings, outline/TOC  
3. MultiPageLoader, network, WebKit dependency surface  
4. CLI flags, C API, image converter  
5. Pure-Go stdlib feasibility & calendar estimates  

Findings are summarized under [`plans/exploration/`](plans/exploration/) and rolled into the canonical ledger.

---

## Working with the plans

1. Treat **`plans/00-canonical-pure-go-rewrite.md`** as the single source of status.  
2. Mark checklist rows `[x]` only with test/command evidence.  
3. Use `[~]` for intentional deferrals with a next gate.  
4. After non-doc code exists: run `make test` and `make lint` before closing a phase.  
5. Phase detail files are subordinate ledgers; do not leave duplicate active work.

---

## License note

Upstream wkhtmltopdf is LGPL. A clean-room pure-Go implementation should avoid copying non-trivial C++/Qt code; reimplement from observed **behavior and public CLI/API contracts**. Choose a license for new Go code when implementation starts.

---

## Bottom line

| Question | Answer |
|----------|--------|
| How long to rebuild **glue + CLI**? | ~2–3 months solo |
| How long for a **useful pure-Go report PDF tool** (stdlib only)? | **~1.5–2.5 years solo**, **~1–1.5 years with two seniors** |
| How long for **true wkhtmltopdf/WebKit parity** in pure Go? | **Not a realistic project goal** |

Start at **Phase 0** in the [canonical plan](plans/00-canonical-pure-go-rewrite.md): freeze the HTML/CSS allowlist before writing a layout engine.
