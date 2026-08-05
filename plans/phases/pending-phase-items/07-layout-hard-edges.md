# Pending — Phase 7: Layout hard edges (flex / grid / float depth)

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** weeks+  
> **Prior plan coverage:** **Yes** — `tier-2-pending-3/flex-grid-remaining.md` Partial; Chrome parity non-goal  

---

## Overview

Deeper flex/grid/float only where Phase 2/6 evidence shows marketing or wiki
gadgets still broken. Prefer Partial honesty over endless Chrome chasing.

---

## Phase 7 checklist

### 7.1 Scope control

- [ ] List concrete broken samples (fixture or URL) needing layout depth
- [ ] Reject “fix all of Flexbox/Grid” as success criterion

### 7.2 Implement slices

- [ ] One evidenced flex/grid/float fix with test
- [ ] Optional second slice
- [ ] Matrix Partial updated

### 7.3 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Status → done / Partial stop

---

## Out of scope

- Chrome layout-test parity (→ Phase 11 / Phase 23)
