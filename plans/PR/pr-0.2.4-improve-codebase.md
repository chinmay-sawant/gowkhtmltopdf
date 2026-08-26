## Summary

Close the post-v0.2.4 improve-codebase architecture ledger: deepen Document/CLI seams, collapse remaining convert/imageout forks, fix layout paint locality, and restore docs honesty. Health rating moves from **7.8 / 10** (audit) back to **8.8 / 10** after implementation. Also ships repo housekeeping: a `/feynman` agent skill and a README stability note about `master`.

---

## Motivation / context

- Plans: [`plans/0.2.4/improve-codebase/codebase-2026-08-20/phase-wise-checklist.md`](plans/0.2.4/improve-codebase/codebase-2026-08-20/phase-wise-checklist.md)
- Index: [`plans/0.2.4/improve-codebase/README.md`](plans/0.2.4/improve-codebase/README.md)
- Follows the Document hard break in #53 (`Document` / `ImageDocument`); this wave fixes incomplete mapper contracts, paint forks, and stale architecture prose that the hard break left behind
- Issues: see **Related issues**

---

## Changes

### Document / CLI seams

- Shared `settings.StampCover` / `StampEmptyHFOverride` / `StampTOC` so covers do not inherit document headers/footers (library matches CLI)
- Document `Allow []string` path prefixes (threat-model ACL), Policy-A knobs (`Grayscale`, `PageOffset`, `ExcludeFromOutline`, Zoom, HF `Replace`)
- `Collate *bool` mapped independently of `Copies`; `Copies` upper bound aligned with the engine
- OnError / nil-context / write-boundary ownership tests restored for Document and ImageDocument
- Document write path builds `*convert.Request` directly (no typed PDFRequest facade)

### Engine seams

- Shared `prepare.BuildOptions` for PDF and image; convert imports `prepare` directly (hub re-exports removed)
- `pdf.RegistryFromGlobal`; convert/imageout only log
- Dead post-validate `len(cmd.Objects)==0` guard removed
- `PdfGlobalOptions` builder removed; Document remains the typed public overlay
- CLI dead dual homes (`OutlineWriter`, unused dump-default field) cleaned up

### Layout / paint locality

- Drop fixture-56 `#2563eb` stroke-width color gate
- `PaintBand` shares body draw policy (radius / rotate / opacity)
- Delete wiki thumb hairline Paint stripper; fix emission at chrome/inline
- Emit `OpStrokeRect` + `StrokeMask*` for mixed rounded sides; remove imageout OpLine overlay rewrite
- Consume `table-layout: fixed` in column sizing

### Repo docs and agent tooling

- Add `/feynman` agent skill (`skills/feynman/SKILLS.md`): plain-words explanation loop with self-audit, used to explain repo behavior from source
- README: add a note that `master` tracks active development and stable builds come from the latest tagged release

### Docs honesty

- Compatibility matrix / deferred / architecture pages updated for Document, text-indent, writing-mode, and imageout→prepare DAG
- Settings package docs: dotted `Set` is CLI/engine; Document is the library overlay

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No intentional perf change; not re-benchmarked in this wave |
| **Memory** | Unchanged claim surface |
| **Behavior / correctness** | Cover HF no longer inherits globals; Collate/`Allow`/grayscale knobs reachable from Document; HF band paint matches body; thumb hairlines and accent stroke gating fixed |
| **API / CLI** | Additive Document fields (`Allow`, Policy-A knobs, `Collate *bool`); CLI cover/TOC stamps share settings helpers |
| **Dependencies** | None |
| **Binary size / build time** | Pure Go / no CGO unchanged |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `Document.Collate` is now `*bool` | Use `boolPtr(false)` / `boolPtr(true)` (or `&v`) instead of a plain bool; unset keeps the engine default |
| New Document fields | Optional: `Allow`, `Grayscale`, `PageOffset`, `ExcludeFromOutline`, page `Zoom`, HF `Replace` |

No reintroduction of public dotted `Set` / `Converter`.

---

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make claim-scan`
- [x] `go test . ./internal/cli ./internal/app ./internal/layout ./internal/convert/... ./internal/imageout ./internal/settings -count=1`

### Commands

```sh
make lint
make test
make claim-scan
```

---

## Screenshots / sample output

```
make lint   # clean
make test   # go test ./... green
make claim-scan  # clean
```

Ledger scorecard after implementation: **8.8 / 10** (pre-fix audit **7.8 / 10**).

---

## Related issues

- Relates to #53 (v0.2.4 Document API / CLI hard break)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-0.2.4-improve-codebase.md`

---

## Follow-ups (out of scope)

- Deeper ARC-03 / ARC-04 immutability / responsive stylesheet follow-up
- Parked layout items (private `Result.root` test surface, `beforeAlways` split, image progress hook plumbing)
- Pixel-diff merge gate / visual evidence classes

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
