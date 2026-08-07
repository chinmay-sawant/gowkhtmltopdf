# Phase 5 — Validation and closure

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** baseline recorded; implementation closure not started
> **Depends on:** Phases 1–4

## Baseline evidence

- [x] Current-source `make lint` passed on 2026-08-07 (`go vet ./...`; no gofmt
  output).
- [x] Current-source `make test` passed on 2026-08-07 (`go test ./...`).
- [ ] Re-run both gates after every non-documentation phase diff; old passing output
  cannot close a later diff.

## Checklist

- [ ] Add a cross-mode contract matrix covering PDF/image settings, mode-valid flags,
  output sinks, inline HTML, ACL, media, simplify profiles, and error routing.
- [ ] Add regression coverage for all high-severity rows before any rating increase:
  output bypass, document preparation drift, container convergence, display-list
  identity, PDF resource collision, and paint traversal parity.
- [ ] **X-02** — carry cancellation through layout, paint, and rasterization with
  measured checkpoints. Proof: mid-work cancellation tests, overhead measurement,
  `go test ./...`, and `go test -race ./...`.
- [ ] Record release-vs-debug benchmark commands separately. Include fixture path,
  toolchain, OS, cold/warm cache state, concurrency, and metric; do not call tests
  performance proof.
- [ ] Recompute the weighted architecture rating from the table in `rating.md`; do
  not raise the headline without updating every area score and arithmetic.
- [ ] If a row is intentionally deferred, mark `[~]` with the reason, owner boundary,
  and next validation gate; do not leave duplicate active rows in another ledger.
