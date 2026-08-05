# Pending — Phase 10: JavaScript / hydration

> **Parent:** [`README.md`](README.md)  
> **Status:** in progress (reopened — execute here, do not only point at Phase 22)  
> **Estimated effort:** see also [`../phase-22-javascript.md`](../phase-22-javascript.md)  
> **Prior plan coverage:** **Yes** — Phase 22 staged JS  

---

## Overview

Ship a **minimal** print-oriented JS subset so skins that gate content on
`document.documentElement.classList` / simple DOM prep can render. Full SPA
hydration remains Phase 22 depth, but this ledger owns a first working slice.

---

## Phase 10 checklist

- [ ] Parse and run a tiny JS subset (or embed a pure-Go interpreter under allowlist amendment)
- [ ] Apply `classList` / `getElementById` / `querySelector` lite mutations before layout
- [ ] Strip remaining scripts safely when unsupported
- [ ] Tests: fixture HTML that requires one class toggle to reveal content
- [ ] Cross-link Phase 22 for deeper work; do **not** leave this row `[~]` only

### Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Status → done / Partial with shipped subset named
