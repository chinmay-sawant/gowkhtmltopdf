# Pending — Phase 5: URL-mode recommended flags (docs / defaults)

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** 0.5–1 day  
> **Prior plan coverage:** **Yes** — Phase 21 §21.5; `documentation/cli.md` URL mode  

---

## Overview

Document two recipes:

1. **Raw honesty smoke** (Ana artifact): no `--simplify-dom`
2. **Decent-print attempt**: fonts + optional `--simplify-dom`

Do not silently change Ana `make samples` to chrome-strip.

---

## Phase 5 checklist

### 5.1 Docs

- [ ] `documentation/cli.md`: URL-mode section with both recipes
- [ ] `documentation/samples.md`: Ana command stays raw; link to decent-print recipe
- [ ] `documentation/fonts.md`: cross-link system fonts for IPA/Unicode
- [ ] `output/README.md` / fidelity: one sentence on raw vs simplify

### 5.2 Samples Makefile

- [ ] Keep wiki smoke **without** `--simplify-dom`
- [ ] Optionally add `--use-system-fonts` to wiki smoke **only if** Phase 3 agrees (record decision)

### 5.3 Gates

- [ ] Docs-only: no required `make test` beyond link sanity
- [ ] Status → done

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 3 font decision | Samples flags |
| `--simplify-dom` shipped | Decent-print recipe |

---

## Out of scope

- Changing default global flags for invoice/report HTML
