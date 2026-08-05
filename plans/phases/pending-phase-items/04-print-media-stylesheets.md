# Pending — Phase 4: `@media print` + large stylesheet application

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** 3–10 days  
> **Prior plan coverage:** **Yes** — Phase 21 §21.3; layout already `Media: "print"`; matrix `@media` **feature queries** weak  

---

## Overview

Chrome print applies wiki print CSS aggressively. We filter `print`/`all` links
but still miss feature queries and may under-apply rules that hide chrome /
tighten type — driving page-count inflation.

---

## Phase 4 checklist

### 4.1 Audit

- [ ] Log which stylesheets apply on Ana fetch (print vs screen) — temporary debug or test harness OK
- [ ] List high-value `@media print` rules we skip (e.g. `(max-width)`, `print` + features)

### 4.2 Media matching

- [ ] Improve `@media` matching for print pipeline: at least `print` and `all`; document feature-query policy
- [ ] If feature queries remain ignored, treat unknown features as **false** or **true** consistently (pick one; document; add tests)
- [ ] Ensure `link rel=stylesheet media="print"` still loads

### 4.3 Volume / degrade

- [ ] Verify graceful degrade on huge CSS (already Phase 21) — add test only if gap found
- [ ] Optional: warn when rule count exceeds soft threshold (no hard fail)

### 4.4 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Ana smoke page-count note vs Phase 2 baseline
- [ ] Status → done / Partial

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 2 density work | Combined page-count improvement |
| `collectSheets` / css media | |

---

## Out of scope

- Full CSS Media Queries Level 4
- Screen-only interactive chrome fidelity
