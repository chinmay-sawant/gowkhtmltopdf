# Load layer: URLs, I/O & ACL

## 1. Responsibility & position in the pipeline

`internal/load` is the **resource-fetching seam** of gowkhtmltopdf. Its own
package doc (`internal/load/doc.go`) states it plainly: it "reimplements the
MultiPageLoader orchestration layer: URL guessing, HTTP(S)/file fetching,
cookies, proxy, auth, local ACL, and POST bodies. Not a browser: it hands raw
bytes to the HTML/CSS/layout pipeline."

The conversion pipeline is

```text
load → parse → style → layout → paginate → paint → write (PDF 1.4 / 1.7 / 2.0 or PNG/JPEG)
```

`internal/load` owns the **first stage** and the **two resource seams** the
rest of the engine depends on:

1. **Primary page load** — one `Loader.Load` call per body object (page,
   cover, TOC) and per header/footer HTML document. It turns a user-supplied
   string (path, URL, `data:`, or in-memory HTML) into a `Resource` of raw
   bytes plus a base URL for relative resolution.
2. **Subresource load** — `Loader.FetchSub` (wrapped in
   `load.ResourceContext.Fetch`) resolves and fetches every document-relative
   CSS `<link>`, image, and `@font-face` URL against the loaded document's
   base URL.

Everything below this package is downstream of bytes: `internal/html` parses
the body, `internal/css` parses stylesheets, `internal/layout` lays out, and
`internal/pdf` / `internal/imageout` paint. The loader never interprets the
bytes it returns; it only fetches, caps, and (for documents) validates the
charset.

The package sits **low in the import graph**: it depends only on
`internal/settings` (for the load policy types) and the Go standard library.
It is consumed by `internal/convert` (orchestration), `internal/convert/prepare`
(shared load/parse/resource phase), `internal/convert/hf.go` (header/footer
HTML), and `internal/imageout` (image mode). Nothing below it imports it.

Because it is the trust boundary between "input strings" and "bytes entering
the engine", the package also carries the **security posture** of the whole
product: the local-file ACL, network timeouts, redirect caps, and body-size
caps described in `documentation/THREAT-MODEL.md`.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|----------------|
| `internal/load/load.go` | URL guessing, HTTP(S)/file/`data:` fetching, cookie jar, proxy, auth, POST, local-file ACL, charset gate, body/redirect/timeout caps, subresource resolution | 1322 |
| `internal/load/load_test.go` | External-package tests: httptest-based HTTP behaviour, ACL matrix (deny/prefix/enable/symlink/traversal), subresource fetching, body caps, timeouts, cancellation, inline/data sources, charset gate | 1117 |
| `internal/load/doc.go` | Package doc (quoted above) | 3 |

The entire domain is two files plus the package doc — a deliberate
single-package design. Contrast with `internal/layout` (~150 files): the load
layer's surface is small because it is a narrow I/O front door, not a
computational engine.

## 3. Key types, functions & entry points

### 3.1 Public types

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Kind` (`KindUnknown`, `KindHTTP`, `KindFile`, `KindInline`) | `load.go:73-81` | Classifies a resolved input after `GuessURL`. |
| `Resource` | `load.go:83-95` | One fetched document: `Kind`, final `URL` (after redirects), `Base` (for relative resolution), `Body` (`[]byte`), `ContentType`, `StatusCode`, and `Skip` (set when the load-error policy is `skip`). |
| `ResourceContext` | `load.go:97-105` | The narrow consumer seam for subresources: binds a loader, a document base, and a per-page load policy. Constructed via `Loader.ForResource`. |
| `AccessController` | `load.go:131-140` | The local-file ACL: `AllowPrefixes` plus `Allowed(path)`. Default deny. |
| `Loader` | `load.go:264-280` | The fetch engine: `*http.Client`, `settings.LoadGlobal` policy snapshot, `io.Writer` log, `MaxBodySize`, `MaxRedirects`, plus compatibility fields `Allow` / `EnableLocalFileAccess` (effective ACL state) and a private `initErr`. |

### 3.2 Constructors

| Symbol | Location | Purpose |
|--------|----------|---------|
| `NewLoader(global settings.LoadGlobal)` | `load.go:282` | Historical constructor shape: builds the loader but defers proxy-validation failure to the first `Load`/`FetchSub` call (recorded in `initErr`). Exists for existing internal callers. |
| `NewLoaderWithError(global settings.LoadGlobal) (*Loader, error)` | `load.go:308` | Fail-fast constructor: validates proxy config and installs the HTTP transport before returning. **This is the one new callers use** — `convert.Run` and `imageout` construct the loader at the request boundary so invalid policy fails before any pipeline state is built. |

### 3.3 Primary load entry points

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Load(ctx, input string, pageLoad settings.LoadPage) (*Resource, error)` | `load.go:394` | Primary document load. In-memory HTML (`pageLoad.InlineHTML`) short-circuits `GuessURL` entirely; otherwise `GuessURL` classifies the input and dispatches to file/HTTP/inline. Every returned document passes `checkDocumentCharset`. |
| `GuessURL(input string) (Kind, string, error)` | `load.go:191` | Mirrors wkhtmltopdf's `guessUrlFromString`: inline HTML → `KindInline`; `http(s)://` passthrough; `file://` → `KindFile`; `data:` → `KindInline`; `host:port` → `http://host:port`; an existing local path → `file://` URL; anything else defaults to `http://<input>`. |
| `IsHTML(s string) bool` | `load.go:182` | Detects inline markup (leading `<` or a UTF-8 BOM followed by `<`). Used by header/footer loading (`hf.go:227`) and mirrored in `internal/html` BOM stripping. |
| `FetchSub(ctx, base, ref string, pageLoad settings.LoadPage) (*Resource, error)` | `load.go:1016` | Subresource (CSS/image/font) fetch: resolves `ref` against `base`, then routes by scheme — `file`/`""` → ACL-gated file read, `http(s)` → HTTP fetch, `data:` → capped decode; anything else → `errUnsupportedScheme`. |
| `ResourceContext.Fetch(ctx, ref string)` | `load.go:117` | The narrow seam consumers actually call; delegates to `Loader.FetchSub` with the bound base and policy. |

### 3.4 Internal machinery (selected)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `initClient()` | `load.go:327` | Builds the `http.Client`: standard-library cookie jar, `ForceAttemptHTTP2` transport, `net.Dialer` with 30 s connect timeout + 30 s keep-alive, optional proxy, and a `CheckRedirect` that fails after `MaxRedirects`. |
| `parseProxy(raw)` | `load.go:369` | Validates proxy config: absolute URL with scheme and host, `http`/`https` only. |
| `loadFile` | `load.go:734` | ACL-checked, context-aware file read with the body cap (probe byte + overflow check). Sets `Base` to the file's directory. |
| `loadHTTP` | `load.go:820` | HTTP fetch: request build, per-page timeout, `>= 400` policy routing (`loadErrorResponse`), Content-Length short-circuit and read-side body cap, final URL after redirects as `Resource.URL`/`Base`. |
| `buildHTTPRequest` | `load.go:940` | Assembles method (GET vs POST with url-encoded form), User-Agent `gowkhtmltopdf/0.1`, basic auth, custom headers, and per-page cookies. |
| `loadErrorResponse` | `load.go:989` | Implements `--load-error-handling`: `abort` → `settings.HttpStatusError`; `skip` → `Resource{Skip:true}` (no body); `ignore` → `Resource` with empty body. |
| `fileAccessAllowed` | `load.go:811` | The frozen security policy: allow-prefix match wins; otherwise `EnableLocalFileAccess && !BlockLocalFileAccess`. |
| `AccessController.Allowed` | `load.go:142` | Real-path (symlink-resolved) prefix comparison with a directory-separator boundary. |
| `filePathFromURL` | `load.go:786` | Extracts the local path from a `file://` URL; refuses non-empty hosts other than `localhost`. |
| `resolveReference` | `load.go:1068` | Resolves a subresource reference against the document base; absolute refs pass through, relative refs need a base (`errNoDocumentBase` otherwise). |
| `checkDocumentCharset` / `charsetSupported` | `load.go:498` / `load.go:513` | Enforces the bytes→runes seam: only UTF-8/ASCII accepted, from Content-Type charset or a `<meta charset>` scan of the first 1 KiB. |
| `readFileBody` | `load.go:696` | Reads a file up to the cap while honouring context cancellation via a watcher goroutine that closes the file on `ctx.Done()` (plain `os.File` reads do not observe contexts). |
| `decodeDataURLLimited` | `load.go:1101` | `data:` URL decode (base64 or percent-escaped) under the same body cap; counts base64 payload compactly before allocating. |
| `cloneLoadPage` + helpers | `load.go:881-938` | Deep-copies `LoadPage` policy on entry so callers cannot mutate loader-owned state mid-flight. |

### 3.5 Error surface

Public sentinels (matchable with `errors.Is`):

- `ErrAccessDenied` (`load.go:45`) — local-file ACL blocked a path.
- `ErrInvalidProxy` (`load.go:51`) — proxy config rejected by `parseProxy`.

Package-private wrapped sentinels (`load.go:55-71`) — `errNilLoader`,
`errNilContext`, `errCannotLoad`, `errUnsupportedCharset`,
`errBlockedFileAccess`, `errNoDocumentBase`, `errUnsupportedScheme`,
`errMalformedDataURL`, `errInvalidDataURL`, `errTooManyRedirects`,
`errBodyTooLarge`, `errInvalidBodyLimit`, `errInvalidRedirects`,
`errUninitializedLoader` — are always wrapped with `fmt.Errorf("%w: ...")` so
dynamic messages wrap a static sentinel and stay matchable with `errors.Is`
from outside the package.

Cross-package errors: HTTP `>= 400` with `abort` policy returns
`*settings.HttpStatusError` (`internal/settings/httperror.go:16`), which maps
status → wkhtmltopdf-compatible exit codes: 404 → 2, 401 → 3, everything else
→ 1 (see `settings.HttpErrorCode`, `httperror.go:39`).

## 4. Data & control flow

### 4.1 Primary document load (PDF mode)

```text
convert.Run (internal/convert/convert.go:320)
  → load.NewLoaderWithError(req.Global.Load)      // fail-fast policy validation
  → prepare.Document (internal/convert/prepare/prepare.go:140)
      → loader.Load(ctx, page, loadPage)
          → GuessURL(input)  → Kind{HTTP,File,Inline}
          → loadByKind → loadFile | loadHTTP | inline/data decode
          → checkDocumentCharset(res)              // UTF-8/ASCII gate
      → Resource{Body, Base, Kind, StatusCode, Skip}
  → html.ParseDocument(res.Body)                   // if !res.Skip
  → prepare.ResourceContext (wraps loader.ForResource)   // subresource seam
  → Prepared{Resource, Root, Resources, Sheets, Registry}
```

The `Resource.Skip` flag is the skip-policy seam: when
`--load-error-handling skip` fires, `prepare.Document` returns a `Prepared`
with a nil `Root` and the object is skipped by the caller rather than aborted.

### 4.2 Subresource loads (CSS, images, fonts)

```text
prepare.ResourceContext.Fetch (prepare.go)
  → load.ResourceContext.Fetch (load.go:117)
      → Loader.FetchSub(ctx, base, ref, pageLoad)  (load.go:1016)
          → resolveReference(base, ref)            (load.go:1068)
          → file  → filePathFromURL + ACL → loadFile
          → http(s) → loadHTTP
          → data: → decodeDataURLLimited
```

Consumers of this seam:

- **Stylesheets** — `internal/convert/prepare/styles.go:37,180`
  (`CollectSheets` walks the DOM for `<link rel="stylesheet">` and `@import`
  and fetches each via the seam).
- **Fonts** — `@font-face` `url(...)` bodies are fetched through the same
  `ResourceContext` in `prepare.go` (`mergeFontFaces`), under the same ACL and
  body cap; WOFF1 decompression caps are applied by `internal/pdf` after the
  bytes arrive.

### 4.3 Header/footer HTML

`internal/convert/hf.go:227-243` (`loadHTMLHF`) loads header/footer HTML via
`loader.Load(ctx, rawOrURL, lineP)` — **resolved like a top-level page, not a
subresource** (CWD-relative / absolute / http(s)), because resolving against
the body document's base would break CLI paths like
`--header-html testdata/golden/fixture-36-header.html`. Raw markup in the
value is detected by `load.IsHTML` and skipped with a warning (upstream
`looksLikeHtmlAndNotAUrl` behaviour). Header/footer HTML passes through the
same ACL, body cap, and charset gate as body documents.

### 4.4 Image mode

`internal/imageout/imageout.go:1124` builds its loader with
`load.NewLoader(imageLoadGlobal(req.Global, *req.Image))`. `imageLoadGlobal`
(`imageout.go:1341`) merges the ACL homes: `Image.Load` is image-mode policy,
while `Global.Load.Allow` / `Global.Load.EnableLocalFileAccess` are OR'd in so
the CLI/library ACL flags apply to both modes. Everything else (parse, CSS,
layout) is shared with PDF mode via `internal/convert/prepare`.

### 4.5 Policy cloning invariant

`Load` and `FetchSub` both start by `cloneLoadPage(pageLoad)` deep-copying the
per-page policy (maps for headers/cookies, slice for POST items, bytes for
inline HTML). The loader never mutates caller-owned state, and a caller cannot
mutate the loader's snapshot after construction.

## 5. Cross-package dependencies

### Imports of `internal/load`

- `internal/settings` — the only internal dependency. Consumed types:
  - `settings.LoadGlobal` (`settings.go:206`): `Proxy`, `Allow` (ACL
    prefixes), `EnableLocalFileAccess`.
  - `settings.LoadPage` (`settings.go:214`): per-page policy — zoom,
    `BlockLocalFileAccess`, `LoadErrorHandling`, `Username`/`Password`,
    `CustomHeaders`, `Cookies`, `Post`, `MediaType`, `PrintMediaType`,
    `Timeout`, `InlineHTML`, `InlineBase`.
  - `settings.PostItem` (`settings.go:234`), `settings.HttpStatusError`
    (`httperror.go:16`).
- Standard library only otherwise: `net/http` (+ `cookiejar`), `net/url`,
  `mime`, `os`, `path/filepath`, `encoding/base64`, `strings`, `context`,
  `time`, `io`, `errors`, `fmt`.

### Consumers of `internal/load`

| Consumer | Usage |
|----------|-------|
| `internal/convert/convert.go:320` | `load.NewLoaderWithError(req.Global.Load)` at the request boundary; `loader.Log = log`. |
| `internal/convert/prepare/prepare.go` | `loader.Load` (primary), `loader.ForResource` → `ResourceContext` for all subresources. |
| `internal/convert/prepare/styles.go` | `loader.ForResource` for stylesheet collection. |
| `internal/convert/hf.go:227,243` | `load.IsHTML` + `loader.Load` for header/footer HTML. |
| `internal/imageout/imageout.go:1124` | `load.NewLoader(imageLoadGlobal(...))` for image mode. |
| `internal/html/html.go:291-295` | Comment-level mirror of `load.IsHTML` BOM handling. |
| `internal/settings/reflect.go` | Registers dotted keys that land on `LoadGlobal`/`LoadPage` (`enablelocalfileaccess` at `reflect.go:498`, `allow` at `reflect.go:527`, `load.*` object keys via `registerLoadPageKeys` at `reflect.go:701`). |

### Import-direction rule

`load` is one of the **lowest** internal packages: it may depend on
`internal/settings` but never on `html`/`css`/`layout`/`pdf`/`convert`.
Consumers sit above it. This keeps the trust boundary (network + filesystem
I/O) isolated from the compute layers, and keeps `internal/errs` (canonical
sentinels) unused here — `load` keeps its own sentinels because its errors are
I/O-specific and its package-private sentinels are wrapped, not shared.

## 6. Design decisions & trade-offs

### 6.1 Pure-Go, no-cgo, no browser

`internal/load` uses only the Go standard library `net/http` stack (with the
stdlib cookie jar and `net.Dialer`). There is no Qt/WebKit network layer, no
`os/exec` anywhere in the tree, and no JavaScript engine: the package doc and
THREAT-MODEL §1 both state the JS-compat flags (`--enable-javascript`,
`--javascript-delay`, `--run-script`, …) are accepted for CLI compatibility
but no code path evaluates scripts.

**Documentation drift to flag:** THREAT-MODEL.md §1 and §8 reference
`load.WaitJSDelay` (implements `--javascript-delay` as a sleep) and
`load.WarnJSStubs` as living in `internal/load`. **Neither symbol exists in
the current source** (`grep` across the repo returns nothing). Either they
were removed during the architecture overhauls (`git log` shows
"10/10 codebase architecture, safety, and performance overhaul") or the threat
model was written against an earlier tree. The security *claim* (no JS
execution) still holds — but THREAT-MODEL.md needs a line-number/name
reconciliation pass.

### 6.2 wkhtmltopdf work-alike behaviour

- `GuessURL` is a direct port of wkhtmltopdf's `guessUrlFromString` semantics
  (`load.go:191`): existing path → file, `host:port` → `http://`, bare input
  → `http://<input>`.
- `LoadErrorHandling` (`abort|skip|ignore`) mirrors `--load-error-handling`
  with the same default (`abort`).
- HTTP status → exit-code mapping (404→2, 401→3, else 1) mirrors
  wkhtmltopdf's `utilities.cc` convention via `settings.HttpStatusError`.
- `LoadPage` defaults match `loadsettings.cc` (`settings.go:401`:
  `BlockLocalFileAccess: true`, `LoadErrorHandling: LoadErrorAbort`).

The divergence from wkhtmltopdf is deliberate and product-shaped: wkhtmltopdf
runs a real JS engine and reads arbitrary encodings; gowkhtmltopdf refuses
non-UTF-8/ASCII documents at the load seam and never executes JS. The
controlled-report scope (invoices, statements, tables, TOCs) makes this
acceptable — see `documentation/fidelity.md` for the claims language.

### 6.3 Security posture shapes the code

The ACL, caps, and timeouts are not bolted on; they are baked into the
control flow (see §8). The two-constructor design (`NewLoader` vs
`NewLoaderWithError`) exists so policy validation happens **at the request
boundary** — `convert.Run` deliberately constructs the loader before fonts,
layout state, or document output are initialized (`convert.go:320-323`).

### 6.4 Trade-offs worth knowing

- **Single package, two files**: simplicity and auditability over
  decomposition. The domain is small enough that a single file is navigable.
- **Charset gate at load, not decode**: only UTF-8/ASCII are accepted; other
  encodings are refused with a clear error rather than silently garbled
  (upstream wkhtmltopdf accepts many encodings). This is a fidelity ceiling
  documented in `documentation/fidelity.md`.
- **Reject, don't truncate, oversized bodies**: `bodyReadLimit` adds one
  probe byte (`load.go:684`) so a body *at* the limit is distinguishable from
  one *over* it; oversized bodies are errors, never silently truncated.
- **Data-URL decode allocates cautiously**: base64 payload length is counted
  compactly before `DecodedLen` allocation (`load.go:1142`), and percent-
  escape decoding preallocates only within the cap — anti-DoS for embedded
  resources.
- **File reads are context-aware via a watcher goroutine** (`load.go:696`):
  `os.File` reads don't observe contexts, so a blocked local read on a hung
  filesystem is aborted at the same request boundary as an HTTP read.

## 7. Notable patterns & invariants

1. **Default-deny ACL with allow-prefix expansion** — `AccessController.Allowed`
   (`load.go:142`) resolves both the candidate path and every prefix to their
   real, symlink-free locations (`filepath.EvalSymlinks`) before the prefix
   comparison, with a directory-separator boundary check so `prefix-evil`
   never matches `prefix`. Non-existent paths fall back to their cleaned
   absolute form (the subsequent read fails anyway).
2. **Wrapped sentinels, matchable errors** — every dynamic failure wraps a
   static package-level sentinel with `%w`, so `errors.Is` works across the
   package boundary without string matching.
3. **Clone-on-entry policy** — `Load`/`FetchSub` deep-copy `LoadPage`
   (`cloneLoadPage`, `load.go:881`) so maps/slices in the policy can't be
   mutated by callers or race across concurrent conversions
   (`TestConcurrentLoads` guards this).
4. **One narrow resource seam** — `load.ResourceContext` (`load.go:97`) is
   the only sanctioned way to fetch subresources; it carries the base URL and
   per-page policy so CSS/images/fonts inherit the same ACL, caps, and
   timeouts as the primary document. `prepare.ResourceContext` wraps it
   (`prepare.go:31`).
5. **`Skip` as a first-class result state** — instead of a sentinel error,
   the `skip` load-error policy returns a bodyless `Resource{Skip:true}` so
   orchestration can continue with the remaining objects.
6. **`file://` host restriction** — remote file hosts are refused outright in
   both the primary path (`filePathFromURL`, `load.go:786`) and subresource
   resolution (`FetchSub` host check), closing the classic `file://evilhost/`
   SSRF-family hole.
7. **Defaults as named constants** — `DefaultConnectTimeout` (30 s),
   `DefaultResponseTimeout` (60 s), `DefaultMaxBodySize` (100 MiB),
   `DefaultMaxRedirects` (10) are exported constants (`load.go:37-42`), and
   per-page `--timeout 0` selects the response default (`requestTimeout`,
   `load.go:978`).
8. **Inline HTML bypasses URL guessing entirely** — `LoadPage.InlineHTML`
   (the library `SetBody` path) skips `GuessURL`; subresources resolve against
   `InlineBase` when set (`load.go:394-433`). `TestEmptyInlineBaseRejectsRelativeSubresources`
   pins the no-base failure mode.

## 8. Security considerations

`documentation/THREAT-MODEL.md` is the normative security document; this
section is the load-layer summary with pointers.

**Trust boundary (THREAT-MODEL §1).** HTML is *semi-trusted*: it can cause
network egress (matching upstream) and — only with operator opt-in — local
file reads. It cannot execute code (no JS engine, no `os/exec`).

**Local-file ACL (§3).** Default deny. The decision matrix:

| Global enable | Object block | Read allowed |
|---|---|---|
| false | true (default) | no |
| false | false | no |
| true | true | no |
| true | false | yes |

With an `--allow` prefix A, path P is readable iff `realpath(P)` is under
`realpath(A)` — independent of both flags. Implementation:
`AccessController.Allowed` (`load.go:142`), `fileAccessAllowed` (`load.go:811`),
`resolvePath` (`load.go:168`). Known limitation (acknowledged in THREAT-MODEL
§3, §6): the check is at read time, so the usual TOCTOU window between check
and open exists, and it cannot prevent reads by other processes.

**Network behaviour (§4).** Connect timeout 30 s; whole-request timeout
default 60 s (per-page `--timeout` overrides) via `http.Client.Timeout`;
context cancellation threaded into every request
(`http.NewRequestWithContext`); redirect hard cap 10 (`CheckRedirect` in
`initClient`); body cap 100 MiB enforced on both HTTP (Content-Length
short-circuit *and* read-side probe) and file reads; `data:` URLs bounded by
the size of the embedding document; TLS verification on by default.

**Exfiltration channels (§5).** Any URL in the HTML can be fetched, including
`http://localhost` and RFC1918 addresses — upstream behaviour, intentionally
not restricted (`TestHTTPLocalhostAllowedByDesign` pins this). The only
sensitive channel is local file reads, gated by the ACL. `@font-face` TTF/OTF/
WOFF1 bytes are untrusted parse input under the same ACL; WOFF2 is rejected
(Brotli not allowlisted); remote `https://` `@font-face` is not fetched
(product policy).

**Credential hygiene (§5).** Custom headers follow cross-host redirects while
`net/http` strips `Authorization`/`Cookie` on redirects — operators must not
combine credential-bearing loads with untrusted HTML. `buildHTTPRequest`
(`load.go:940`) attaches operator-configured basic auth, cookies, and custom
headers only to the requests the operator configured them for.

**Embedding in web apps (§7.1).** The engine becomes the server's HTTP client
(and optionally file reader) on behalf of whoever controls the input. The
preferred pattern is converting *trusted, server-generated* HTML; "convert any
user URL" without host allowlists and network isolation is an SSRF
anti-pattern. Full scenarios in `documentation/integration-security.md`.

**Recommended defaults for untrusted input (§7):** convert in an isolated
container, keep `--allow-local-files` off, no `--allow`, rely on the
built-in timeouts and 100 MiB cap, sanitise HTML or author it yourself.

## 9. Testing & verification

The package is tested by an **external test package** (`package load_test`,
`load_test.go:1`) using `httptest.NewServer` handlers and `t.Parallel()`
throughout — 30 test functions, no golden files (the loader returns bytes, so
behavioural tests suffice). Coverage themes map one-to-one onto the
responsibilities:

| Theme | Tests |
|-------|-------|
| URL guessing / inline detection | `TestGuessURL`, `TestIsHTML` |
| HTTP basics, auth, headers, POST, cookies | `TestLoadHTTPBasic`, `TestLoadHTTPCustomHeadersAndAuth`, `TestLoadHTTPPost`, `TestLoadCookies` |
| Error-status policy | `TestLoadHTTPErrorCodes` |
| ACL matrix | `TestACLDefaultDeny`, `TestACLAllowPrefix`, `TestACLEnableLocalFileAccess`, `TestACLFileURL` |
| ACL hardening (traversal, symlinks, subresources) | `TestACLPathTraversal`, `TestACLSymlinkEscape`, `TestSubresourceFileACL` |
| Body caps | `TestMaxBodySizeHTTP`, `TestMaxBodySizeFile`, `TestDataURLHonorsBodyLimitForPrimaryAndSubresource`, `TestInlineHTMLHonorsBodyLimit` |
| Timeouts & cancellation | `TestSlowServerTimeout`, `TestContextCancelAbortsBodyRead` |
| Redirects | `TestRedirectLimit`, `TestRedirectLimitExact` |
| Subresource seam | `TestSubresourceFetch`, `TestResourceContextBindsBaseAndPolicy`, `TestEmptyInlineBaseRejectsRelativeSubresources` |
| Inline HTML | `TestLoadInlineHTML`, `TestInlineHTMLHonorsBodyLimit` |
| Concurrency | `TestConcurrentLoads` |
| Charset gate | `TestLoadCharsetContentType`, `TestLoadCharsetMetaDecl` |
| SSRF posture | `TestHTTPLocalhostAllowedByDesign` |

These tests are the enforcement point for the THREAT-MODEL controls inventory
(§8): every row in that table has a corresponding test above.

## 10. Known limitations, deferred items & open questions

1. **Non-UTF-8 documents are refused** (`checkDocumentCharset`, `load.go:498`).
   `--load.default-encoding` is accepted but inert (registered as ignored in
   `settings/reflect.go:180`). Reintroducing Windows-1252/ISO-8859-1 support
   would be the highest-value encoding follow-up; see
   `documentation/deferred.md` and the compatibility matrix for the encoding
   stance.
2. **No JavaScript, ever** — `--enable-javascript`, `--javascript-delay`,
   `--run-script`, `--window-status`, `--debug-javascript` are accepted for
   CLI compatibility and routed to `Ignored` (Policy A), never consumed here.
   This is a permanent product boundary, not a deferred item.
3. **THREAT-MODEL.md references `load.WaitJSDelay` / `load.WarnJSStubs`
   which no longer exist in source** — documentation drift (§6.1). The
   underlying claim (JS flags are no-ops) is still true; the doc needs a
   symbol reconciliation pass. **Open question: were these ever implemented
   and removed, or only planned?** (`git log` for `internal/load` shows a
   "10/10 codebase architecture, safety, and performance overhaul" commit,
   consistent with removal.)
4. **TOCTOU window** in the ACL (check at read time) — acknowledged in
   THREAT-MODEL §3/§6; accepted residual risk.
5. **No network-egress restrictions** — SSRF posture is "input trust, not
   network filtering" (THREAT-MODEL §5/§6); host allowlisting is left to the
   embedding application (`documentation/integration-security.md`).
6. **`http.Client.Timeout` is a whole-request timeout** — it cannot
   distinguish connect vs body-read stall; a slow trickle can hold the
   connection until the full timeout. Acceptable for the report workload, but
   a read-header deadline plus per-read deadlines would harden hostile-server
   cases further (see `internal/settings` `--timeout` semantics).
7. **Proxy support is `http`/`https` only** (`parseProxy`, `load.go:369`) —
   no `socks5`; matches upstream scope but worth documenting for operators
   who need SOCKS.
8. **`data:` URLs are bounded by document size** rather than an independent
   budget — a document that embeds many large `data:` images can push total
   memory above the per-resource cap. The per-resource cap still applies, so
   the total is bounded by document size × count.

## 11. Related documents

- [../architecture.md](../architecture.md) — high-level package map and pipeline (this document is its deep-dive for the load stage).
- [../THREAT-MODEL.md](../THREAT-MODEL.md) — normative security model; §3 (ACL), §4 (network), §5 (exfiltration), §8 (controls inventory) map onto `internal/load`.
- [../integration-security.md](../integration-security.md) — Gin/web-app embedding scenarios, SSRF and local-file guidance.
- [../fidelity.md](../fidelity.md) — encoding/feature fidelity tiers (why the charset gate and no-JS stance are claims, not bugs).
- [../deferred.md](../deferred.md) — deferred items incl. encoding support.
- [../compatibility-matrix.md](../compatibility-matrix.md) — per-flag/per-feature support including the accepted-but-inert load flags.
- [../cli.md](../cli.md) — CLI flags that feed `LoadGlobal`/`LoadPage` (`--allow-local-files`, `--allow`, `--load-error-handling`, `--proxy`, `--timeout`, `--custom-header`, `--cookie`, `--username/--password`, POST flags).
- [../library-api.md](../library-api.md) — `ObjectSettings.SetBody` / inline HTML path (`LoadPage.InlineHTML`/`InlineBase`).

Sibling architecture deep-dives in this directory:

- [01-entrypoints-cli.md](01-entrypoints-cli.md) — CLI flag wiring that feeds the load policy.
- [02-library-api.md](02-library-api.md) — `SetBody`/inline-HTML entry into the loader.
- [03-settings.md](03-settings.md) — `LoadGlobal`/`LoadPage` definitions and dotted-key registration.
- [05-html-parser.md](05-html-parser.md) — consumes `Resource.Body` after this layer.
- [08-convert-pipeline.md](08-convert-pipeline.md) — orchestrates `Loader.Load`, header/footer loads, and the `prepare` resource seam.
- [10-imageout-svg.md](10-imageout-svg.md) — image-mode loader construction and ACL merge (`imageLoadGlobal`).
