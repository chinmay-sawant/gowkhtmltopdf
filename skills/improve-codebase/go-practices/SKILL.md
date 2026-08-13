---
name: improve-codebase-practices
description: >
  Audit Go code against production conventions: error sentinels, wrapping,
  context, ownership cloning, tests, concurrency, lint suppressions, and
  public-API contracts. Emits structured findings for a phase-wise
  checklist. Use when reviewing Go best practices, idioms, test quality,
  API contracts, or running /improve-codebase-practices. Do not reopen
  decided product tradeoffs, run perf-review, or delete code for leanness
  (ponytail).
---

# Improve-codebase — Go practices

Audit **conventions that keep a Go library honest**. This is not a
style pass (`gofmt` already ran) and not a performance pass.

Do not implement.

Read first: `../references/finding-schema.md`. If the module path is
`gowkhtmltopdf`, also read `../references/gowkhtmltopdf.md`. Calibrate
to `.golangci.yml`, `Makefile`, and existing tests before filing.

## 1. Law vs taste

**Law** (file if broken):

- `make lint` (`enable-all` here; `gofumpt`/`tenv` stay disabled).
- `%w` wrapping. No new `nolint:wrapcheck` outside tests.
- Canonical sentinels aliased, not re-constructed.
- Public clone-on-intake for maps/slices/bytes the caller still owns.
- Nil context rejected at every exported cancellation-aware entry with
  the canonical `ErrNilContext`.
- Tests use `errors.Is`, `t.Parallel()`, `t.Context()` / `t.TempDir()`.
- `CGO_ENABLED=0`. Direct deps stay on the allowlist.
- Lint suppressions are local and name the reason.

**Taste** (do not file): line wrapping, comment voice, whether a
helper is 8 lines or 12, `:=` vs `var` unless it hides a nil map.

## 2. Error and API contracts

Walk exported functions and the engine entries they call.

| Pattern | Finding when |
|---|---|
| Sentinel | Same condition, different `errors.New` text/value |
| Prefix | Public errors lose the `gowkhtmltopdf:` prefix; internal wrap drops `%w` |
| `OnError` / hooks | Preflight fails and the hook is skipped, or the hook swallows the return |
| Panic vs error | Fluent programmer-error panics are policy; `Set`/`Convert`/`Run*` must not panic on user input |
| Distinct nils | `ErrNilConverter` and `ErrNilPDFRequest` (etc.) collapsed into one value |
| Validate-before-write | `Validate*` / `Run*` writes bytes, then fails |

Do not demand a sealed option interface. A compatibility dotted `Set`
plus a typed overlay is a product, not a smell.

## 3. Context and cancellation

- Thread `ctx` as a **parameter**.
- One documented `containedctx` engine that polls at recursion
  boundaries is allowed. A second struct that stores ctx is a finding
  unless it is the same engine.
- `ctx.Err()` belongs between expensive stages and at load/fetch
  seams, not on every CSS declaration.
- Adapters named `Layout` / `Paint` that call `context.Background()`
  are legacy names, not bugs, if `*Context` variants exist.

## 4. Ownership and concurrency

- Clone at the public boundary. Prove with a test that mutates the
  caller's buffer after `AddObject` / `SetBody` / `Convert`.
- Single-goroutine engines do **not** get a "just in case" mutex.
  File the mutex as `defect` if someone added one on `Document` or
  layout without a second goroutine that actually writes.
- `sync.Pool`: no `&slice` escape; copy-on-put when the buffer is
  retained. `sync.Once` for lazy immutable tables is correct.
- `Converter` is not concurrent. Do not add internal locking to make
  `Convert` re-entrant; the docs already say one converter per run.

## 5. Tests that lie

File `defect` or `friction` when a test claims a contract it does not
exercise:

- Golden / structural walker treated as a pixel or overlap oracle.
- Semantic oracle that builds a `pdf.Document` by hand and is sold as
  HTML→PDF proof.
- Byte-identical PDF assertions (forbidden product claim here).
- Performance budgets run under `-race` and treated as release numbers.
- Live network inside `make test`.
- White-box `//nolint:testpackage` with no reason.
- A matrix **Implemented** row with no test and no fixture.

Prefer adding the missing assertion over inventing a new test
framework.

## 6. Refuse list (do not nag)

If this is `gowkhtmltopdf`, the refuse list is
`../references/gowkhtmltopdf.md` § Do not nag — do not restate it.

In any other repo, refuse: a second settings system, pixel-diff merge
gates, mutexes on a documented single-goroutine engine, context on
every struct, live network in unit tests, and re-filing a closed
ledger row without a current-source regression.

## 7. Emit findings

Finding-schema records. Cap a solo run at **15**. Prefer `PRAC-` IDs
after dedupe. Quote current `file:line`. Name the sentinel, test, or
clone helper that should own the fix.

If invoked **by** `/improve-codebase`, return the finding list only.

If invoked **alone**, continue to
`skills/phase-wise-checklist/SKILLS.md` and write:

`plans/reviews/improve-codebase/practices-<YYYY-MM-DD>/phase-wise-checklist.md`

Phase order for this lens: security/ACL and output integrity → public
sentinel/clone contracts → context → tests that lie → lint
suppressions / docs. Closure gates stay in the checklist skill.
