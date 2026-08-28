# Phase 68: Ignored inventory + policy amend

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 68
> **Status:** complete (inventory + reopen policy only; no engine promotions)
> **Estimated effort:** S
> **Owner:** catalog
> **Depends on:** Phase 67 honesty baseline
> **Unblocks:** Phase 69
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Lock the 247 former Ignored names and keep them on the work list (`goal: implement`, `engine_status: unsupported` until real code). This phase does **not** implement behavior.

## Delivered

- `plans/0.2.6/ignored-inventory.json` bucket ownership for phases 69-77
- Policy amendment on the canonical ledger for browser-level print reopen

## Checklist

- [x] 68.1.1 Inventory of 247 names with bucket ownership.
- [x] 68.1.2 Ledger amendment for reopen (not permanent ignore).
- [x] 68.1.3 After 2026-08-28 honesty revert: names are `unsupported` (not fake Implemented). `filter` is Partial (opacity-only).

## Agent warning

Do not “complete” Phase 69-77 by flipping these 247 to Implemented. Follow each phase **Work order** and `HONESTY-GATES.md`.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Handoff

Phase 69 vendor-prefix aliases.
