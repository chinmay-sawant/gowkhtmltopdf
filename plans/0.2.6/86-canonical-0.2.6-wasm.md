# 0.2.6 - Browser WASM conversion, PDF preview, and image output

> **Parent:** `plans/README.md` - next versioned implementation ledger
> **Status:** Complete. Phases 86-93 passed their implementation and validation gates on 2026-09-09.
> **Estimated effort:** multi-phase feature; estimate after Phase 86 contract review

---

## Overview

This ledger ports the current Go document engine into a browser-facing
WebAssembly artifact and adds a frontend conversion page. The first browser
slice accepts HTML text, renders PDF, PNG, or JPEG output in a worker, and
displays the result in an embedded preview with a download action. PDF preview
is the primary document view; image mode has its own raster preview contract.

The plan uses the existing root `Document` API and conversion pipeline. It does
not create a second layout or PDF engine. The browser adapter owns JavaScript
marshalling, worker messages, browser resource rules, and the preview lifecycle.

### Current source evidence

- The release record now names the shipped `GOOS=js` / `GOARCH=wasm` target,
  `bindings/wasm`, and the `syscall/js` bridge (`RELEASE.md:34-49`).
- The pure-Go tree already compiles for the browser target. On 2026-09-09,
  `GOOS=js GOARCH=wasm go list ./...`, `GOOS=js GOARCH=wasm go build ./...`,
  and `GOOS=js GOARCH=wasm go test -c . -o /tmp/gowk-root-wasm.test.wasm`
  all exited 0. These are compile checks, not browser-runtime proof.
- The public API accepts owned in-memory HTML through `Content.HTML` and
  returns owned PDF bytes through `Document.PDF` (`document.go:18-50`,
  `document.go:242-260`). The sibling `ImageDocument` API returns owned PNG or
  JPEG bytes through `ImageDocument.Image` (`document.go:146-172`,
  `document.go:262-327`). These are the narrowest existing input/output seams
  for a browser adapter.
- `Document.PDF` maps the typed document into `convert.Request`, while
  `convert.Run` creates the loader, loads the embedded default font, creates
  the PDF writer, and runs the render lifecycle (`document.go:330-349`,
  `internal/convert/convert.go:402-456`).
- `ImageDocument.Image` maps the typed image document into `imageout.Request`,
  which reuses the render pipeline and selects PNG or JPEG output
  (`document.go:484-526`, `internal/imageout/imageout.go:106-150`).
- The loader treats inline HTML as an in-memory resource, but its file and HTTP
  paths use filesystem and network policy code (`internal/load/load.go:895-955`,
  `internal/load/load.go:1241-1287`, `internal/load/load.go:1494-1545`). The
  browser MVP must therefore make its allowed source kinds explicit rather than
  exposing the native `File` and `URL` fields without a browser policy.
- The PDF package embeds the default Liberation and DejaVu faces in the Go
  binary (`internal/pdf/assets/assets.go:1-27`), and the default font path is
  already available without system font discovery (`internal/pdf/faces.go:206-214`).
- The frontend is currently a React/Vite documentation site. Its router has no
  conversion page (`frontend/src/App.jsx:44-88`), its build copies `frontend/dist`
  into `docs/` (`frontend/scripts/copy-to-docs.mjs:1-27`), and its package has
  only static-site smoke checks today (`frontend/package.json:1-18`,
  `frontend/scripts/smoke-test.mjs:1-80`).
- There is no `testdata/wasm` directory. Existing HTML fixtures live under
  `testdata/golden` and are tested through the native conversion package, so a
  browser fixture and browser-specific expectations must be added separately.

## Executive summary

The implementation order is:

1. freeze the browser input, output, error, and resource contract;
2. add a small `bindings/wasm` executable that exposes conversion to JavaScript;
3. package the Go WASM runtime and versioned artifact for Vite;
4. run that artifact in a Web Worker;
5. add the HTML editor, sample loader, PDF and image previews, and download
   flow;
6. prove the bridge and the real browser workflow with fixture data;
7. update Makefile, CI/release checks, and user documentation after the code
   and browser tests pass.

The default browser MVP supports PDF, PNG, and JPEG output from inline HTML. It
keeps native file access disabled, does not promise arbitrary cross-origin
resource loading, and does not add JavaScript execution to the document engine.
The PDF preview and image preview are separate output modes backed by the
existing `Document` and `ImageDocument` APIs.

## Phase 86: Browser contract and boundary audit

### 86.1 Define the browser request contract

- [x] **WASM-CONTRACT-01** Write the browser request schema beside the new
  adapter. It must accept one HTML string or UTF-8 byte sequence, optional
  document options needed by the page, and no filesystem path. Proof: a
  contract test rejects empty HTML, unsupported source fields, and malformed
  option values before conversion starts. Implemented in
  `bindings/wasm/contract.go`; `go test ./bindings/wasm -count=1` passed.
- [x] **WASM-CONTRACT-02** Define the browser response schema. Success must
  return output bytes plus a MIME type for PDF, PNG, or JPEG; image results must
  also expose their pixel dimensions; failure must return a stable error code
  and human-readable message; progress messages must identify the current phase
  and percentage. Proof: native adapter tests cover each output mode, validation
  failure, conversion failure, and progress ordering. Implemented in
  `bindings/wasm/contract.go` and covered by the output, cancellation, error,
  and progress tests; `go test ./bindings/wasm -count=1` passed.
- [x] **WASM-CONTRACT-03** Decide and document the first browser resource rule.
  The MVP permits inline HTML and resources already embedded in that HTML, such
  as data URLs. It rejects `File` and `URL` document sources and does not make a
  promise about arbitrary remote CSS, image, or font fetches. Proof: tests cover
  the accepted and rejected source forms, and `documentation/wasm.md` names the
  boundary. The request decoder rejects native `file` and `url` fields, while
  adapter tests verify empty network schemes, disabled local files, empty font
  paths, and disabled system fonts in `bindings/wasm/contract_test.go`. The
  `documentation/wasm.md` now states the inline-only boundary and its deferred
  remote-resource contract.
- [x] **WASM-CONTRACT-04** Set browser resource limits independently from the
  native CLI defaults. Include an HTML byte limit, output byte limit for both
  document and image results, image dimension limits, and a worker request
  lifetime. Proof: oversized input, oversized output, oversized images, and
  timed-out work return bounded errors without leaving a worker request pending.
  The limits are implemented in `bindings/wasm/contract.go` and
  `bindings/wasm/main_js_wasm.go`; native boundary tests cover input and image
  limits, and the browser harness covers cancellation and restart.

### 86.2 Confirm package and artifact ownership

- [x] **WASM-OWNERSHIP-01** Choose `bindings/wasm` as the browser adapter and
  keep engine work in the root `Document` API plus `internal/convert`. The
  adapter must not import CLI parsing or duplicate layout and PDF code. Proof:
  package imports and a source review match the architecture DAG in
  `documentation/architecture.md:161-196`. The adapter imports only the public
  package plus standard library code, and the contract check passed.
- [x] **WASM-OWNERSHIP-02** Choose stable artifact names and paths, such as
  `frontend/public/wasm/gowkhtmltopdf.wasm` and a checked-in or copied
  Go-version-matched `wasm_exec.js`. Proof: the Makefile target, Vite build, and
  browser test all load the same paths.

## Phase 87: Go WASM adapter

### 87.1 Add the executable and JavaScript bridge

- [x] **WASM-GO-01** Add a `bindings/wasm` `main` package with a small
  JavaScript bridge. Register one conversion function, keep the Go runtime alive
  after registration, and make the bridge safe to call from the worker message
  loop. Proof: the package builds with `GOOS=js GOARCH=wasm` and its native
  helper tests compile without requiring `syscall/js`. Implemented in
  `bindings/wasm/main_js_wasm.go` with a host stub in
  `bindings/wasm/main_native.go`; `bash scripts/check-wasm-contract.sh` passed.
- [x] **WASM-GO-02** Map bridge input into `gowkhtmltopdf.Document` using
  `Content.HTML` and the existing typed options. Map PDF requests to
  `Document` and PNG or JPEG requests to `ImageDocument`. Set browser-safe
  defaults for local files, font paths, system fonts, network policy, and PDF
  profile. Proof: native tests inspect both mappings and confirm no native path
  or system-font setting can enter the browser MVP. Implemented in
  `bindings/wasm/contract.go`; `TestBrowserDocumentsUseInlineOnlyDefaults` and
  the contract test command passed.
- [x] **WASM-GO-03** Copy PDF, PNG, and JPEG bytes to JavaScript as a
  `Uint8Array` or an equivalent transferable buffer, without exposing Go-owned
  memory after the callback returns. Return the selected MIME type and image
  dimensions when applicable. Proof: browser tests check `%PDF-`, PNG, and JPEG
  signatures and verify the response metadata.
- [x] **WASM-GO-04** Marshal errors without panics. Preserve the public
  validation and conversion errors from `document_validate.go` and the existing
  PDF API, then convert them into the stable browser error schema. Proof:
  invalid input, unsupported options, and a render failure produce structured
  errors in both native and browser tests.
- [x] **WASM-GO-05** Forward phase and progress callbacks from `Document` and
  `ImageDocument` to the bridge without writing to stdout or stderr. Proof: a
  test records the callback sequence for PDF and image conversion and the
  browser UI receives progress updates in order.

### 87.2 Keep host and browser tests separate

- [x] **WASM-GO-06** Add native unit tests for request decoding, option mapping
  to both document APIs, response encoding, error mapping, and callback
  ordering. Proof: `go test ./bindings/wasm -count=1` passes on the host and
  covers PDF, PNG, JPEG, validation, cancellation, policy, response metadata,
  error codes, and progress ordering.
- [x] **WASM-GO-07** Add a JS/WASM compile test for the actual bridge package and
  record the command in the ledger. Proof: `GOOS=js GOARCH=wasm go build -o
  <artifact> ./bindings/wasm` exits 0 through
  `bash scripts/check-wasm-contract.sh`.

## Phase 88: Browser assets and sample data

### 88.1 Add the WASM fixture set

- [x] **WASM-FIXTURE-01** Add `testdata/wasm/README.md` describing the fixture
  contract and why it is separate from the native golden corpus. Added the
  fixture contract documentation and validated its referenced files through
  `TestWASMFixtureManifest`.
- [x] **WASM-FIXTURE-02** Add a representative HTML sample under
  `testdata/wasm`, with inline print CSS, headings, a table, a forced page break,
  and enough content to exercise a multi-page PDF and a useful raster image.
  Proof: the native and browser tests use the same named fixture for all output
  modes. Added `testdata/wasm/sample.html`; the native fixture test renders it
  in PDF, PNG, and JPEG modes.
- [x] **WASM-FIXTURE-03** Add machine-readable expectations under
  `testdata/wasm`, including PDF, PNG, and JPEG MIME types, the PDF page-count
  envelope, ordered text needles, image dimensions, and minimum output sizes.
  Proof: the shared fixture test reads the manifest and fails when any output is
  empty, has the wrong signature, or violates its structural expectations.
  Added `testdata/wasm/manifest.json`; `go test ./bindings/wasm -count=1` passed
  `TestWASMFixtureManifest` for all three output modes.
- [x] **WASM-FIXTURE-04** Keep fixture resources self-contained. If an image is
  needed, use a small checked-in asset or a data URL. Do not make browser tests
  depend on a network server or local filesystem access. Proof: the browser
  test runs with network requests disabled or intercepted and still passes.

### 88.2 Package runtime assets

- [x] **WASM-ASSET-01** Add the Go runtime loader script and generated WASM
  artifact to the Vite public asset path through a Makefile target. The loader
  script version must match the Go toolchain used to build the artifact. Proof:
  the asset check finds both files in `frontend/dist/wasm` after a production
  build.
- [x] **WASM-ASSET-02** Keep the generated artifact out of hand-edited `docs/`.
  Vite must copy it through the existing `frontend/scripts/copy-to-docs.mjs`
  path. Proof: the frontend build succeeds and the generated site contains the
  same runtime assets.

## Phase 89: Frontend conversion and PDF or image preview

### 89.1 Run conversion in a worker

- [x] **WASM-FRONT-01** Add a worker module that loads `wasm_exec.js`, starts
  the Go WASM instance once, and queues or rejects overlapping requests with a
  clear state. Proof: a browser test performs PDF and image conversions and
  confirms the worker returns complete responses without corrupting either
  result.
- [x] **WASM-FRONT-02** Add worker termination and restart handling. A cancel or
  timeout must terminate the stuck worker, revoke its pending request, and allow
  a later conversion to start a fresh worker. Proof: the browser test cancels a
  request, starts another PDF or image conversion, and receives a valid result.
- [x] **WASM-FRONT-03** Transfer only serializable data across the worker
  boundary. Use request IDs so late messages from a terminated worker cannot
  overwrite the current UI state. Proof: a focused worker test ignores a stale
  response and keeps the latest request state.

### 89.2 Add the conversion page

- [x] **WASM-UI-01** Add a route such as `/wasm` to `frontend/src/App.jsx` and
  expose it from `SiteNav.jsx`, the command palette, and an appropriate landing
  page action. Proof: route smoke coverage finds the page and navigation reaches
  it under the existing `HashRouter`.
- [x] **WASM-UI-02** Add an HTML editor with a sample-data action, reset action,
  clear validation errors, and a conversion button. The editor must preserve
  the raw HTML string instead of trying to render it with the browser DOM first.
  Add an explicit PDF, PNG, or JPEG output selector. Proof: browser interaction
  tests edit the text, load the fixture, select each output mode, and submit
  both valid and invalid content.
- [x] **WASM-UI-03** Add a PDF preview panel using a Blob URL with
  `application/pdf`, plus download and open-in-new-tab actions. Revoke old Blob
  URLs when a new result replaces them or the page unmounts. Proof: the browser
  test sees a non-empty PDF URL, downloads bytes beginning with `%PDF-`, and
  observes no stale URL after a second conversion.
- [x] **WASM-UI-06** Add an image preview panel for PNG and JPEG Blob URLs, with
  the selected MIME type, dimensions, download action, and URL cleanup matching
  the PDF preview lifecycle. Proof: the browser test renders both image formats,
  checks their signatures and dimensions, and observes no stale URL after a
  second conversion.
- [x] **WASM-UI-04** Add visible idle, loading, success, error, and cancelled
  states. Keep the preview area stable while conversion runs and show the phase
  and progress values supplied by Go. Proof: browser assertions cover every
  state through the real worker path.
- [x] **WASM-UI-05** Add plain CSS in a focused stylesheet. The layout must work
  on narrow and wide viewports, keep the editor and preview readable, expose
  keyboard focus, and respect reduced-motion preferences. Proof: the browser
  test checks responsive layout and focus-visible behavior; a human opens the
  generated page for visual review.

## Phase 90: WASM and browser verification

### 90.1 Test the Go boundary

- [x] **WASM-TEST-01** Add tests for valid inline HTML, empty input, invalid
  options, rejected file and URL sources, bounded input, and conversion errors.
  Proof: `go test ./bindings/wasm -count=1` exits 0.
- [x] **WASM-TEST-02** Add an actual JS/WASM test harness that starts the built
  artifact, calls the exported bridge for PDF, PNG, and JPEG, and checks the
  returned signatures, MIME types, PDF EOF marker, image dimensions, and fixture
  text needles. Proof: the harness runs in a real browser, not only in a Go
  compile step.
- [x] **WASM-TEST-03** Add a repeated-conversion test that checks worker reuse,
  worker restart, PDF and image output, and Blob URL cleanup. Proof: the browser
  harness completes the sequence and reports no pending request at the end.

### 90.2 Test the frontend path

- [x] **WASM-TEST-04** Extend the frontend smoke test for WASM asset presence,
  conversion route wiring, sample-data presence, and the PDF and image preview
  contracts. Proof: `npm --prefix frontend test` exits 0 after a production
  build.
- [x] **WASM-TEST-05** Add browser assertions for keyboard use, error recovery,
  reduced-motion CSS, and narrow viewport rendering. Proof: the real browser
  runner records each assertion and stores a reviewable screenshot when the
  runner supports it.
- [x] **WASM-TEST-06** Compare the WASM fixture output with the native
  `Document.PDF` and `ImageDocument.Image` outputs using structural checks, not
  byte identity. The PDF writer includes time and version-dependent metadata, so
  the PDF contract is validity, page envelope, and ordered semantic text. The
  image contract is format, dimensions, and non-empty encoded pixels. Proof:
  both paths pass the shared fixture manifest.

## Phase 91: Makefile, CI, and release integration

### 91.1 Add repeatable commands

- [x] **WASM-MAKE-01** Add version-stamped WASM build variables and a `wasm`
  target to the Makefile. The target must use `GOOS=js GOARCH=wasm`, build only
  `bindings/wasm`, copy the matching `wasm_exec.js`, and write the runtime and
  bridge to the documented frontend public path. Proof: `make wasm` exits 0 and
  produces the named assets used by PDF and image conversion.
- [x] **WASM-MAKE-02** Add a `wasm-test` target that runs the native adapter
  tests, the JS/WASM compile check, the frontend asset/build checks, and the real
  browser harness with bounded concurrency. Proof: the target exits 0 on a clean
  checkout and records each subcommand.
- [x] **WASM-MAKE-03** Decide whether `make build` includes the WASM artifact or
  depends on `make wasm`. Keep native binary outputs unchanged and make the
  release artifact choice explicit in the Makefile comments and release docs.
  Proof: a clean build produces exactly the documented native and browser
  artifacts.

### 91.2 Wire automation and release records

- [x] **WASM-CI-01** Add CI coverage for the WASM compile and browser test. Keep
  the existing capped native test and lint jobs intact. Proof: the workflow
  invokes `make wasm-test` or its documented equivalent.
- [x] **WASM-CI-02** Update release packaging to publish the WASM artifact and
  the runtime loader after the artifact passes its tests. Proof: the release
  workflow names the artifact and checks its `VERSION` stamp.

## Phase 92: Documentation and frontend copy

### 92.1 Document the supported browser path

- [x] **WASM-DOC-01** Add `documentation/wasm.md` with installation/build steps,
  the JavaScript API for PDF, PNG, and JPEG output, worker usage, PDF and image
  preview examples, browser support limits, resource policy, size and timeout
  limits, and the difference between native and browser inputs.
- [x] **WASM-DOC-02** Update `documentation/README.md`, `README.md`,
  `documentation/getting-started.md`, `documentation/library-api.md`, and
  `documentation/architecture.md` to link the browser path and identify the
  shared `Document` pipeline. Proof: every new claim points to the shipped
  artifact or a passing test.
- [x] **WASM-DOC-03** Update `RELEASE.md` only after the artifact and release
  gates pass. Replace the current "does not ship" statement with the exact
  artifact and browser support contract, without claiming native file access,
  arbitrary network loading, JavaScript execution, or Chrome parity.
- [x] **WASM-DOC-04** Update `documentation/deferred.md` and the knowledge-base
  summaries so deferred browser features remain explicit. Do not erase the
  native security rules or turn a browser preview into a general web renderer.

### 92.2 Keep the generated site honest

- [x] **WASM-DOC-05** Update frontend copy, command-palette metadata, and the
  landing page to describe the new `/wasm` conversion route. Keep long-form
  Markdown in `documentation/`; never hand-edit generated `docs/` output.
- [x] **WASM-DOC-06** Run `make claim-scan` after the copy changes and remove
  stale "no browser runtime" language only where the new implementation proves
  the narrower browser claim.

## Phase 93: Closure gates

- [x] **WASM-GATE-01** Run targeted Go tests for `bindings/wasm` and any touched
  engine package after each implementation slice.
- [x] **WASM-GATE-02** Run `make wasm-test` on the final tree and record exit 0.
- [x] **WASM-GATE-03** Run `make test` and `make lint` on the final tree. The
  phase-wise checklist skill requires both for non-documentation changes.
- [x] **WASM-GATE-04** Run `make claim-scan` and `make golden` after the final
  documentation and source edits. Golden output remains the native rendering
  contract; the WASM fixture manifest covers the browser adapter.
- [x] **WASM-GATE-05** Run the frontend production build and real browser smoke
  test. Confirm the generated `docs/` tree receives the WASM assets through the
  build script and no source of truth is hand-edited.
- [x] **WASM-GATE-06** Run `make test-race` if the implementation changes
  `internal/convert`, `internal/load`, `internal/layout`, `internal/pdf`, or
  `internal/imageout`. Record skipped gates with the exact reason.
- [x] **WASM-GATE-07** Re-scan this ledger for `[ ]` and `[~]` rows before
  declaring the feature complete. Every `[x]` must name its matching source and
  validation evidence.

## Dependencies

```text
WASM-CONTRACT-01..04
        |
        v
WASM-OWNERSHIP-01..02 --> WASM-GO-01..07 --> WASM-FIXTURE-01..04
                                      |                  |
                                      v                  v
                              WASM-ASSET-01..02 --> WASM-FRONT-01..03
                                                         |
                                                         v
                                                   WASM-UI-01..06
                                                         |
                                                         v
                                                   WASM-TEST-01..06
                                                         |
                                                         v
                                                   WASM-MAKE-01..03
                                                         |
                                                         v
                                                   WASM-CI-01..02
                                                         |
                                                         v
                                                   WASM-DOC-01..06
                                                         |
                                                         v
                                                   WASM-GATE-01..07
```

## Explicit non-goals for this ledger

- No JavaScript execution inside converted documents. The engine currently
  does not provide a JavaScript runtime, and browser JavaScript only controls
  the adapter UI.
- No native filesystem access from the browser page.
- No promise of arbitrary cross-origin resource loading. Any later resource
  bridge must define CORS, origin, size, timeout, and cancellation behavior in
  a new ledger amendment.
- No Chrome or WebKit rendering parity claim.
- No byte-identical PDF claim between native and WASM. Compare valid structure,
  page bounds, and semantic text instead.
- No additional raster formats beyond PNG and JPEG in this ledger. Adding one
  requires its API, fixture, preview, and browser gates.

## Closure evidence

The following source and command evidence closes the ledger. The native Go
adapter suite and fixture test pass in `make wasm-test`; the compile script
passes `GOOS=js GOARCH=wasm go build`; the frontend smoke test passes asset,
route, CSS, and preview contracts; and the Chrome harness passes PDF, PNG,
JPEG, validation, cancellation, restart, text, signature, MIME, dimension,
focus, mobile, and Blob URL assertions. `Makefile`, `.github/workflows/ci.yml`,
`.github/workflows/release.yml`, and `RELEASE.md` contain the repeatable build,
CI, and release paths. `documentation/wasm.md` and the linked documentation
pages contain the browser contract and its native boundary.

## Plan evidence boundary

This file records the shipped source map and its proof. Every checklist row is
closed only after its matching source exists and the relevant test or source
inspection is recorded above.
