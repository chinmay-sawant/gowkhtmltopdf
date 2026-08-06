# Pending — Phase 6: Selector / cascade gaps for site skins

> **Parent:** [`README.md`](README.md)  
> **Status:** Partial (2026-08-05) — `^=`/`$=`/`|=` shipped; Selectors 4 remainder open  
> **Estimated effort:** 1–3 weeks  
> **Prior plan coverage:** **Partial** — element/class/id/sibling/` :has` lite shipped; many Selectors 4 gaps remain in matrix  

---

## Overview

Wiki/marketing skins use selectors we do not match → `display`/`color` never
apply. Expand only **high-evidence** gaps from Ana/marketing CSS dumps.

---

## Phase 6 checklist

### 6.1 Evidence

- [x] From live or saved wiki CSS, list top unmatched selector patterns (attribute ops, `:not()`, `:nth-*` variants, etc.)
- [x] Prioritize patterns that unlock `display:none` / link/typography rules

**Evidence (2026-08-05 Vector `site.styles`):**
- Attribute ops in skin: `*=` (already), `$=` (×2 `[href$=".pdf"]`), `~=` (already)
- `:not(` common; already supported
- No `:is(` / `:where(` in this sheet; `::before`/`::after` still unsupported (content gen)

### 6.2 Implement (evidence-gated)

- [x] Implement first prioritized gap with tests — `^=` / `$=` / `|=` (`TestAttrPrefixSuffixDash`)
- [~] Second gap deferred — `:first-of-type` / `:nth-of-type` / `::before` not evidenced as high-impact for Ana print density this pass
- [x] Stop when diminishing returns; document remainder as Partial

### 6.3 Gates

- [x] `make lint` → pass
- [x] `make test` → (css package + full suite at ledger close)
- [x] Status → Partial

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `:link` | Link-colored rules |
| Phase 4 media | Print rules visible to match |

---

## Out of scope

- Full Selectors 4
- Forgiving selector lists / complex Chrome edge cases unless evidenced
