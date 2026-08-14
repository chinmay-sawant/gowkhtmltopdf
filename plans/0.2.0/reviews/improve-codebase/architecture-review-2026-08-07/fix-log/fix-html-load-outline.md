# fix-html-load-outline — remediation log (2026-08-07)

Owner: `internal/html/*`, `internal/load/*`, `internal/outline/*`
Source of truth: `phases/phase-02-document-prep-and-pipeline.md` anchors p2-02 / p2-04 / p2-05 / p2-07 / p2-08 / p2-09, plus `fix-contract.md` cross-package contracts #5 (Outline), #6 (Loader policy), #8 (In-memory HTML).

## Per-CID status

| CID | Status | Notes |
|---|---|---|
| P2-02 (outline side) | **done** | `Heading.DocPage int` added with the contract comment ("Page stays object-local forever; DocPage is set exactly once during assembly and never mutated afterwards"). Convert-side consumers (collectObjectHeadings, emitOutline, toc/links/hf rebase reads) are fix-convert's row — API additive, nothing to do at my call sites. |
| P2-05 (outline side) | **done** | Exported `SortHeadings(hs []*Heading)` and `SectionOf(hs []*Heading, page int) (section, subsection string)` per the phase Future snippet. `BuildTree` now calls `SortHeadings(sel)`; the duplicate inline comparator in `outline.go` (the package's own copy) was deleted. Convert-side (`flatHeadings`, deleting `sectionOf` in hf.go) is fix-convert's row — they consume the new exports. |
| P2-06 | **n/a (fully owned by fix-convert)** | All P2-06 locations (`pageRect`/`destPoint`/`locationOf` in `internal/convert/links.go`, `emitOutline` y-flip in `internal/convert/outline.go`) are fix-convert files. No action taken; logged per instructions. |
| P2-07 (load side) | **done (code) / partial (repo)** | `NewLoader` now applies `Allow: g.Allow, EnableLocalFileAccess: g.EnableLocalFileAccess` per the Future snippet and the Loader-struct comment was updated. `settings.LoadGlobal` gains those fields via fix-settings-cli (in flight — see Validation). `Client *http.Client`: no caller outside `load` replaces it (only `initClient`/`loadHTTP` use it; grep of the whole repo found zero external writes) — kept exported for now per instructions; unexporting is optional and deferred (convert/imageout pokes are being removed in parallel). |
| P2-08 | **done** | `(*Node).Walk` (pre-order) and `(*Node).TextContentOf(name)` added to `internal/html` per the Future snippet. `outline.CollectHeadings` and the `headNodes` helper in `outline_test.go` refactored onto `Walk`. Convert/imageout call sites (`docTitle`, `collectSheets` walks) are fix-convert's / wave-2 rows. |
| P2-09 | **done** | `html.ParseDocument(body []byte)` added per the Future snippet (BOM strip, then `Parse`); the stale "callers handle charset detection beforehand" comment on `Parse` repaired. Load-side charset rule **implemented at the load seam** (chosen over a FIX-REVIEW marker): `Load` checks the Content-Type charset parameter and, when absent, a `<meta charset>` / `<meta http-equiv="content-type">` declaration in the first 1 KiB; any non-UTF-8/ASCII charset returns `unsupported charset: %s (only UTF-8/ASCII)`. `--default-encoding` untouched (fix-settings-cli's marker). |
| P2-04 (load side) | **done (code) / partial (repo)** | `load.Load` checks `lp.InlineHTML` first and skips `GuessURL` entirely; `Resource.Base` = `lp.InlineBase` when set, so subresources resolve against it. `settings.LoadPage.InlineHTML []byte` + `InlineBase string` are being landed by fix-settings-cli (grep confirmed absent today; written against the contract). Unit test added (`TestLoadInlineHTML`): inline source short-circuits a garbage input, returns KindInline, body, and Base. |
| Cross-contract #5 | **done** | `outline.AssignAnchors` stays in `internal/outline` (I own it); unchanged. |

## Files changed

- `internal/html/html.go` — `Walk`, `TextContentOf`, `ParseDocument` added; `Parse` doc comment repaired (P2-08, P2-09).
- `internal/html/html_test.go` — `TestWalkPreOrder`, `TestTextContentOf`, `TestParseDocument` (P2-08, P2-09).
- `internal/outline/outline.go` — `Heading.DocPage` + contract comment; `CollectHeadings` on `Walk`; `SortHeadings` + `SectionOf` exported; `BuildTree` uses `SortHeadings`; duplicate comparator deleted (P2-02, P2-05, P2-08).
- `internal/outline/outline_test.go` — `headNodes` on `Walk`; `TestSortHeadings`, `TestSectionOf` (P2-05, P2-08).
- `internal/load/load.go` — `NewLoader` applies `g.Allow`/`g.EnableLocalFileAccess`; `Load` InlineHTML-first + charset seam; charset helpers (`checkDocumentCharset`, `charsetSupported`, `charsetFromContentType`, `metaCharset`, `metaTagCharset`, `charsetInContent`, `attrValue`) (P2-07, P2-04, P2-09).
- `internal/load/load_test.go` — `TestLoadInlineHTML`, `TestLoadCharsetContentType`, `TestLoadCharsetMetaDecl`; `net/url` import (P2-04, P2-09).

## Validation

- `gofmt -l internal/html internal/load internal/outline` → clean (no files listed).
- `go build ./internal/html/...` → **OK**; `go vet` → **OK**; `go test` → **OK** (`ok gowkhtmltopdf/internal/html`).
- `go build/vet/test ./internal/outline/...` → **OK at time of change** (`ok gowkhtmltopdf/internal/outline`). On later reruns it is **blocked by fix-pdf-fonts' in-flight edits**: all 12 error lines are inside `internal/pdf` (objRef/String mismatch in content.go/fontpdf.go/fonttype0.go) — transitive dep of layout, zero errors in my packages.
- `go build ./internal/load/...` → **blocked by fix-settings-cli's in-flight edits**: (a) the contract fields `settings.LoadGlobal.Allow`, `settings.LoadGlobal.EnableLocalFileAccess`, `settings.LoadPage.InlineHTML`, `settings.LoadPage.InlineBase` do not exist yet (expected parallel-wave state — my code is written against the contract signatures); (b) `internal/settings/reflect.go:470-476` mid-refactor compile errors (setter signature change, in flight).
- **Contract verification**: copied `internal/settings` + `internal/load` into a throwaway module (`/tmp/opencode/chk`, deleted afterwards) with the four contract fields shimmed onto `LoadGlobal`/`LoadPage`; `go build ./...`, `go vet ./...`, `go test ./...` all green there, including the new InlineHTML + charset tests (two test-setup iterations: `DefaultLoadPage().BlockLocalFileAccess` defaults true, and `mime.TypeByExtension(".html")` already carries `charset=utf-8`, so the meta-decl test was moved to the HTTP path with a bare `text/html` — the load-side behavior itself was correct).
- Testdata scan: every fixture declares `charset="utf-8"`/`UTF-8`, so the new load seam breaks no fixtures.

## Remaining markers

- None added: every deferred piece of my rows lives in another agent's package with the contract already exported (`SortHeadings`, `SectionOf`, `DocPage`, `Walk`, `TextContentOf`, `ParseDocument`), so no `// FIX-REVIEW:` marker is required at any call site in my files. No `// ponytail:` markers touched.

## Notes for the orchestrator

- `settings.LoadGlobal` / `settings.LoadPage` must land the contract fields exactly as named (`Allow []string`, `EnableLocalFileAccess bool`; `InlineHTML []byte`, `InlineBase string`) for `internal/load` to compile; the shim build proves my code is correct against those signatures.
- `Loader.Client` kept exported (zero external writers); unexporting can be a follow-up once convert/imageout pokes are gone.
- Wave-2 rows for others: convert/imageout `html.Parse` call sites → `html.ParseDocument(res.Body)`; convert `docTitle`/`collectSheets` walks → `Walk`/`TextContentOf`; convert `flatHeadings`/`sectionOf` → `outline.SortHeadings`/`outline.SectionOf` + DocPage rebase; convert/imageout loader pokes → drop (NewLoader applies policy).
