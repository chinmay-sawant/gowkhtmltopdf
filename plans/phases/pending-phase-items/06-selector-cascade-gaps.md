# Pending — Phase 6: Selector / cascade gaps for site skins

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** 1–3 weeks  
> **Prior plan coverage:** **Partial** — element/class/id/sibling/` :has` lite shipped; many Selectors 4 gaps remain in matrix  

---

## Overview

Wiki/marketing skins use selectors we do not match → `display`/`color` never
apply. Expand only **high-evidence** gaps from Ana/marketing CSS dumps.

---

## Phase 6 checklist

### 6.1 Evidence

- [ ] From live or saved wiki CSS, list top unmatched selector patterns (attribute ops, `:not()`, `:nth-*` variants, etc.)
- [ ] Prioritize patterns that unlock `display:none` / link/typography rules

### 6.2 Implement (evidence-gated)

- [ ] Implement first prioritized gap with tests
- [ ] Implement second if still high impact
- [ ] Stop when diminishing returns; document remainder as Partial

### 6.3 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Status → done / Partial

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
