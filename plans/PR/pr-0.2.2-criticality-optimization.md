## Summary

Closes the post-#45/#46 criticality and optimization follow-up for the shipped PDF 1.7 / 2.0 compliance surface. Fixes silent PDF/UA wiring holes (cloned page MCIDs, link identity, outline `/SD`, dest structure, single `/Document`, header/footer isolation), unifies profile aliases and conflict sentinels in `internal/pdfprofile`, isolates the default unclaimed PDF 1.4 path, and hoists the hot-path costs that the 0.2.2 review called out.

Also fixes the remaining PDF/UA-1 / PDF/UA-2 list-nesting failure on fixture-56: inline `<a>` inside `<li>` was emitting `L > Link`, which veraPDF and ISO 32000-1 / ISO 32005 reject. Links now live under `LI / LBody`. Regenerated the claimed and unclaimed 1.7 / 2.0 sample PDFs.

No new PDF/A or PDF/UA flavour. Default empty version + empty profile is still unclaimed PDF 1.4.

---

## Motivation / context

- **Plan:** [`plans/0.2.2/criticality-optimization-checklist.md`](../0.2.2/criticality-optimization-checklist.md) — status **complete** (phases 1–6)
- **Parent:** [`plans/0.2.2/README.md`](../0.2.2/README.md)
- **Reviewed predecessors:** [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45) (PDF 1.7 + PDF/A-3a + PDF/UA-1), [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46) (PDF 2.0 + PDF/A-4 + PDF/UA-2)
- **Epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) — newer PDF versions and compliance
- **Shipped version/profile issues:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31), [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32), [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33)
- **Constraint from the ledger:** do not add PDF/A or PDF/UA features; do not reopen completed 0.2.2 compliance ledgers; do not rewrite the layout engine
- **Validators:** in-tree veraPDF 1.30.2 (`-f 3a`, `-f ua1`, `-f 4`, `-f ua2`); unit structure-tree assertions

The product split after #45/#46 was already correct (`--pdf-version` is not a claim; `--pdf-profile` is). The follow-up is wiring, contracts, isolation, and cost — plus one list-hierarchy defect that made claimed fixture-56 fail UA-1 and UA-2.

---

## Changes

### Phase 1 — UA wiring (shipped contract)

| ID | Defect | Fix |
|----|--------|-----|
| C1 | `DuplicatePage` copied BDC/MCID bytes but dropped `page.mcids` and never appended `contentRef`s | Clone `src.mcids`; append `contentRef{page: clone, mcid}` on each owning `StructElem`; reset `annotRef` so finalize allocates a new annot + OBJR |
| C2 | Link fallback assumed `[OpLinkURI, OpText]`; layout emits `[OpText, OpLinkURI]` | `associateUnmappedOps` joins a preceding `OpText` with `OpLinkURI` onto the same `StructLink` |
| C3 | Outline `/SD` bound `HeadingStructElems()[i++]` | `emitOutline` matches outline items by `*outline.Heading` pointer identity |
| C4 | `Page.SetLinkDestStruct` had no call sites | `applyInternalLinks` and `applyTOCLinks` resolve the dest `StructElem` and call it |
| C5 | Each `PaintContext` created a new `/Document` | Reuse `StructTreeRoot.Children[0]` when it is already `StructDocument` |
| C6 | `" 1.4 "` + `a3a-ua1` bypassed the version conflict check | `ParsePDFVersion` first, then compare the parsed token against the profile base |
| 1.7 | Header/footer links joined the body structure tree | HF links are pagination artifacts; they no longer attach under `/Document` |

### Phase 2 — one source of truth

- New leaf package `internal/pdfprofile`: canonical constants, alias parse, and predicates (`IsPDFA3`, `IsPDFA4`, `IsPDFUA1`, `IsPDFUA2`, `IsPDFUA`)
- `WithPDFVersion` / `WithPDFProfile` store canonical tokens (`Get("pdfprofile")` after `"a3a-ua1"` is `PDF/A-3a+PDF/UA-1`)
- Duplicate alias tables and `Profile*` constants removed from `internal/settings` and `internal/pdf`
- Convert profile sentinels fold into `pdf.ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20`; public `api.go` aliases those
- `ErrProfilePDF20Unsupported` kept as a historical sentinel for source compatibility; never returned
- Strict alias parse rejects greedy fragments (`"a3"`, `"ua"`, `"a4+ua"`)

### Phase 3 / 4 — isolation and cost

- Default PDF 1.4 serialization stays unclaimed: no XMP, ICC, StructTree, ParentTree, named dests, `/Tabs /S`, or trailer `/ID`
- `/Tabs /S` gated on UA-1 or UA-2 only
- UA / A policy flags hoisted to `Document` and `pagePainter` (no per-op `Policy()` on the paint loop)
- Canonical profile stored as constants + boolean flags (no re-parse)
- ICC Flate bytes precomputed with `sync.OnceValue` in `internal/pdf/icc.go`
- Structure serialize uses `strings.Builder` and in-place `pruneEmptyStructElems`
- `finalize()` split into `embedICC` / `embedOutputIntents` / `embedMetadata` without a pipeline type
- Benchmark matrix: `BenchmarkWrite50Pages` and `BenchmarkWrite500Pages` for all 9 profiles (`default-1.4`, `pdf-1.7`, `pdf-2.0`, `pdfa-3a`, `pdfua-1`, `a3a-ua1`, `pdfa-4`, `pdfua-2`, `a4-ua2`)

### UA list nesting (veraPDF / octopdf)

Inline `<a href>` inside `<li>` never gets its own layout box. The list was tagged `L → LI → LBody`, then `associateUnmappedOps` attached leftover `OpLinkURI` as a sibling of `LI`:

```text
L
  LI
    Lbl
    LBody
  Link   ← illegal
```

ISO 32000-1 (UA-1) and ISO 32005 (UA-2) allow only `LI`, `L`, or `Caption` under `L`. That is the “List tag hierarchy is invalid” / “List tag hierarchy is invalid for ISO32005” report on fixture-56 (13 TOC links).

Fix:

- `tagListItem` maps `OpLinkURI` onto the existing `LBody`
- `mapSemanticOps` refuses to park content on structural containers (`L`, `Table`, `TR`, `Document`)
- `newLinkChild` / `ensureInlineParent` wrap `Link` in `LI / LBody` if the parent cannot hold inline content
- Regression: `TestStructureTreeListLinkHierarchy` (plain list of links + `columns: 2` TOC-style list)

Resulting tree:

```text
L
  LI
    Lbl
    LBody
      Link
```

### Samples and docs

- Regenerated:
  - `output/pdf-1.7/` — `--pdf-version 1.7` (unclaimed)
  - `output/pdf-1.7-compliance/` — `--pdf-profile a3a-ua1`
  - `output/pdf-2.0/` — `--pdf-version 2.0` (unclaimed)
  - `output/pdf-2.0-compliance/` — `--pdf-profile a4-ua2`
- Same two fixtures as the 1.7/2.0 smoke set: `fixture-21-detailed-report`, `fixture-56-architecture-diagram`
- `documentation/library-api.md`: `pdfprofile` added to the Global keys table; `pdfversion` wording no longer implies 2.0 is unfinished
- `doc.go`: names 1.4 / 1.7 / 2.0 and the A-3a/A-4 + UA-1/UA-2 profiles
- Architecture-diagram sample write path stays `output/` only (no golden PDF under `testdata/golden/api/`)

### API surface (no new flags)

| Surface | Change |
|---------|--------|
| `--pdf-version` / `WithPDFVersion` | Unchanged accepted set (`1.4`, `1.7`, `2.0`); stored canonical |
| `--pdf-profile` / `WithPDFProfile` | Unchanged accepted aliases; `Get` now returns the canonical name |
| `ErrConformanceRequiresPDF17` / `20` | Single sentinel from `internal/pdf`; `errors.Is` works from `api`, `convert`, and `pdf` |
| `ErrProfilePDF20Unsupported` | Documented historical; never returned |
| `ErrTitleRequired` / `ErrPDFUAMissingAlt` | Re-exported from `pdf` on the public API |

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Default 1.4 path is cheaper (no per-op UA policy, no per-doc ICC rebuild). Write50 `default-1.4` ≈ 3.42 ms/op; `a3a-ua1` ≈ 3.73 ms/op; `a4-ua2` ≈ 3.82 ms/op. Convert 500-page default ≈ 1.15 s/op. |
| **Memory** | Write50 default ≈ 1.00 MB/op; dual profiles ≈ 1.14–1.15 MB/op. ICC Flate caches are process-lifetime `sync.OnceValue`. |
| **Behavior / correctness** | Default unclaimed 1.4 unchanged. Claimed UA trees now clone MCIDs, keep one `/Document`, attach links/outlines/dests to the right struct elems, and keep `Link` out from under `L`. |
| **API / CLI** | No new flags. Builder `Get("pdfprofile")` now returns the canonical token (was the raw alias). Conflict errors compare equal via `errors.Is` across packages. |
| **Dependencies** | None. Still pure Go + allowlisted modules. veraPDF remains an optional external binary. |
| **Binary size / build time** | Unchanged aside from committed sample PDFs under `output/pdf-{1.7,2.0}*`. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None for default users | Empty version + empty profile is still unclaimed PDF 1.4 |
| `Get("pdfprofile")` after `WithPDFProfile("a3a-ua1")` | Now `"PDF/A-3a+PDF/UA-1"`, not `"a3a-ua1"`. Compare via `pdfprofile.Canonical` / `Parse`, or accept either alias |
| Callers that constructed `convert.ErrProfileRequiresPDF17` by value | Use `errors.Is(err, api.ErrConformanceRequiresPDF17)` (aliases still exist) |
| `ErrProfilePDF20Unsupported` | Still defined; never returned. PDF 2.0 profiles are supported |

---

## Test plan

- [x] `go test ./internal/layout ./internal/convert ./internal/pdf -count=1`
- [x] `TestStructureTreeListLinkHierarchy` (plain + multicol TOC list of links)
- [x] `TestStructureTreeListTagging`, `TestStructureTreeTableHierarchy`
- [x] `make build` → `bin/gowkhtmltopdf`
- [x] Regenerated claimed + unclaimed fixture-21 / fixture-56 for 1.7 and 2.0
- [x] veraPDF 1.30.2 on all four claimed fixtures — **PASS** (see sample output)
- [x] Ledger-recorded: `make lint` (golangci-lint v1.64.8, 0 issues), `make test`, `go test -race ./...`
- [ ] Optional: re-upload `output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf` and `output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf` to octopdf to confirm the list-hierarchy rules are gone

### Commands

```sh
go test ./internal/layout ./internal/convert ./internal/pdf -count=1
make build

./bin/gowkhtmltopdf --pdf-profile a3a-ua1 --enable-local-file-access \
  testdata/golden/fixture-56-architecture-diagram.html \
  output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf
./bin/gowkhtmltopdf --pdf-profile a4-ua2 --enable-local-file-access \
  testdata/golden/fixture-56-architecture-diagram.html \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf

./compliance/run_verapdf.sh --flavour 3a --flavour ua1 \
  output/pdf-1.7-compliance/fixture-21-detailed-report.pdf \
  output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf
./compliance/run_verapdf.sh --both \
  output/pdf-2.0-compliance/fixture-21-detailed-report.pdf \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
```

---

## Screenshots / sample output

```
veraPDF 1.30.2

======= 1.7 COMPLIANT (a3a-ua1) =======
==> output/pdf-1.7-compliance/fixture-21-detailed-report.pdf
PASS PDF/A-3a: compliant (155 rules, 37528 checks)
PASS PDF/UA-1: compliant (106 rules, 22207 checks)
==> output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf
PASS PDF/A-3a: compliant (155 rules, 460881 checks)
PASS PDF/UA-1: compliant (106 rules, 174759 checks)
PASSED: all checks

======= 2.0 COMPLIANT (a4-ua2) =======
==> output/pdf-2.0-compliance/fixture-21-detailed-report.pdf
PASS PDF/A-4: compliant (109 rules, 14386 checks)
PASS PDF/UA-2: compliant (1727 rules, 33392 checks)
==> output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
PASS PDF/A-4: compliant (109 rules, 101558 checks)
PASS PDF/UA-2: compliant (1727 rules, 156341 checks)
PASSED: all checks

Unclaimed 1.7 / 2.0 files: %PDF-1.7 / %PDF-2.0, no pdfaid / pdfuaid
Claimed files: pdfaid + pdfuaid present

Before (fixture-56 UA fail):
  L element contains Link … instead of L, LI or Caption
  <L> contains <Link>  (ISO 32005 Table 5. L-Link, 13×)

After:
  L → LI → LBody → Link

go test ./internal/layout ./internal/convert ./internal/pdf  — ok
```

---

## Related issues

- Relates to #29
- Relates to #31
- Relates to #32
- Relates to #33

Follow-up to merged PRs #45 and #46. Those issues are already closed; this PR does not reopen them and does not add a new compliance flavour.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`, `performance`)
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-0.2.2-criticality-optimization.md`

---

## Follow-ups (out of scope)

- Optional octopdf / Arlington human re-check of the regenerated fixture-56 claimed PDFs
- Prefer per-target structure Dest (heading/id StructElem) over first-page-MCID fallback for richer AT navigation
- PDF/A-4e / A-4f, multiple OutputIntents
- Byte-stable CLI creation times (determinism product work)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff (no new PDF/A or PDF/UA flavour; no layout-engine rewrite)
- [ ] Public API / CLI changes documented (`library-api.md`, `doc.go`)
- [ ] Default still unclaimed PDF 1.4 without profile/version flags
- [ ] List structure: `L` children are only `LI` / `L` / `Caption`; links sit under `LBody`
- [ ] `Get("pdfprofile")` returns the canonical token
- [ ] Sample PDFs under `output/pdf-{1.7,2.0}*` are intentional artifacts (not golden byte baselines)
- [ ] PR has assignee and labels
- [ ] Related issues use Relates (do not Closes already-closed #31/#32/#33)
- [ ] No secrets committed
