# Phase 4 — Output resources, paint traversal, and fonts

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** not started
> **Depends on:** Phase 3

## Goal

Use the real PDF/raster adapters to deepen resource ownership and rendering policy,
while preserving PDF pagination, annotations, raster alpha, and font compatibility.

## Checklist

- [ ] **PDF-01** — centralize PDF page resource naming and cloned-page ownership;
  prevent body/header/footer `F0`/`I0` collisions. Proof: body + HTML header/footer
  images, copy/collate, embedded-font, and PDF resource dictionary checks.
- [ ] **PAINT-01** — share display-list traversal policy for order, transforms,
  opacity, and resources across PDF body, HTML header/footer, and raster adapters.
  Keep destination-specific pagination, annotations, and alpha behavior in adapters.
  Proof: transformed/stacked/transparent PDF-vs-PNG fixtures and existing goldens.
- [ ] **FONT-01** — give every loaded face a stable identity independent of family
  display name; centralize shaped font runs and advances before PDF/raster adapters.
  Proof: same-family regular/bold, Arabic, CJK, Type0/ToUnicode, and cross-output
  advance tests.

## Required gate

- [ ] Run `make lint` and `make test`; run the PDF resource benchmark and raster/PDF
  parity suite with exact command, fixture, OS, cold/warm state, concurrency, and
  before/after metrics.
