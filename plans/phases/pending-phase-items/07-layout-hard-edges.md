# Pending — Phase 7: Layout hard edges (flex / grid / float depth)

> **Parent:** [`README.md`](README.md)  
> **Status:** Partial stop (2026-08-05) — no new slice; prior flex/grid Partial stands  
> **Estimated effort:** weeks+  
> **Prior plan coverage:** **Yes** — `tier-2-pending-3/flex-grid-remaining.md` Partial; Chrome parity non-goal  

---

## Overview

Deeper flex/grid/float only where Phase 2/6 evidence shows marketing or wiki
gadgets still broken. Prefer Partial honesty over endless Chrome chasing.

---

## Phase 7 checklist

### 7.1 Scope control

- [x] List concrete broken samples (fixture or URL) needing layout depth
- [x] Reject “fix all of Flexbox/Grid” as success criterion

**Samples considered:**
- `testdata/web/marketing-landing.html` / golden flex-grid fixtures — already covered by pending-3 Partial
- Live Ana density (~93pp) is dominated by cascade/chrome/typography residuals, not a single missing flex algorithm knob evidenced this pass
- True joint-intrinsic subgrid / full masonry / deep nested flex multi-pass remain documented Partial in `flex-grid-remaining.md`

### 7.2 Implement slices

- [~] No new flex/grid/float code slice this ledger pass (diminishing returns vs Phases 1–6 wins)
- [~] Optional second slice — deferred until a concrete failing fixture is filed
- [x] Matrix Partial updated (already via pending-3; no change required here)

### 7.3 Gates

- [x] `make lint` → n/a code (docs/ledger only this pass)
- [x] `make test` → suite green at ledger close
- [x] Status → Partial stop

---

## Out of scope

- Chrome layout-test parity (→ Phase 11 / Phase 23)
