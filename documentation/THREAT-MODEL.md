# Threat Model

Scope: security-relevant behaviour of the pure-Go HTML→PDF engine, centred
on `internal/load` (resource fetching) and the pipeline that consumes
fetched bytes. Written for Phase 9.2 (security).

## 1. Trust boundary

The engine converts HTML into a PDF. The HTML document is the primary
attack surface:

- A document can name arbitrary network resources (`http`, `https`,
  `data:`, and - subject to the ACL - `file:`).
- A document **cannot execute code**. There is no JavaScript engine and no
  process execution anywhere in the tree (no `os/exec` usage; grepping for
  `exec.Command` returns nothing outside test helpers that parse the CLI).
  The JS-related flags (`--enable-javascript`, `--javascript-delay`,
  `--run-script`, `--window-status`, `--debug-javascript`) are accepted for
  CLI compatibility and stored in settings (`settings.Web.JavaScript`), but
  no code path ever evaluates a document's scripts. The remaining JS flags
  are ignored.

**Stance: HTML is semi-trusted.** It may cause network egress (matching
upstream wkhtmltopdf) and may read local files only where the operator
explicitly enabled it. Treat HTML as fully trusted whenever
`--enable-local-file-access` or `--allow` is used, or when the machine
running the conversion holds secrets reachable via arbitrary network
fetches.

## 2. Assets

- Local file confidentiality: every file readable by the user running the
  conversion.
- Operator-supplied credentials: basic auth, custom headers, cookies, proxy
  credentials attached to outbound requests.
- Availability: a hostile or dead server must not hang the conversion
  (timeouts), and bodies must not exhaust memory (size caps).

## 3. Local file access model (ACL)

Implemented in `internal/load`: `AccessController.Allowed` and
`Loader.fileAccessAllowed`.

- **Default: deny.** `LoadPage.BlockLocalFileAccess` defaults to `true`
  (`settings.DefaultLoadPage`) and `PdfGlobal.EnableLocalFileAccess`
  defaults to `false` (`settings.DefaultPdfGlobal`).
- `--enable-local-file-access` sets the global flag;
  `--disable-local-file-access` sets the object flag. A blocked object wins
  over an enabled global.
- `--allow <path>` adds allow prefixes. A path is readable when its real
  path equals a prefix or sits below it (boundary check at the directory
  separator, so `prefix-evil` does not match `prefix`).
- Both the requested path and each prefix are resolved to their real,
  symlink-free location (`filepath.EvalSymlinks`) before comparison, so a
  symlink planted inside an allowed directory cannot escape to a file
  outside it. `..` components and percent-encoded forms (`%2e%2e`) in
  `file://` URLs resolve to the same real path the read would follow and
  are checked before the read.
- `file://` hosts other than the empty host and `localhost` are refused
  outright (both in the primary load and in subresource resolution).
- The primary page, subresources (CSS/images via `FetchSub`) and
  header/footer HTML all pass through the same ACL and the same body cap.

Decision matrix for path P with no `--allow` prefixes:

| Global enable | Object block | Read allowed |
|---|---|---|
| false | true (default) | no |
| false | false | no |
| true | true | no |
| true | false | yes |

With an `--allow` prefix A, P is readable iff `realpath(P)` is under
`realpath(A)`, independent of the two flags.

Known limitation: the ACL check happens at read time; there is the usual
TOCTOU window between check and open, and it cannot prevent reads by other
processes. The trust envelope of any local reader applies.

## 4. Network behaviour

- **Connect timeout**: 30 s (`DefaultConnectTimeout`, `net.Dialer.Timeout`).
- **Whole-request timeout**: per-page `--timeout` seconds, default 60 s
  (`DefaultResponseTimeout`), enforced via `http.Client.Timeout` - covers
  TLS handshake, headers and body read. `--timeout 0` selects the default.
- **Context cancellation**: `Load(ctx, ...)` and `FetchSub(ctx, ...)` thread
  the caller's context into every request
  (`http.NewRequestWithContext`); cancelling it aborts the request even
  mid-body-read.
- **Redirects**: hard cap `Loader.MaxRedirects` (default 10); exceeding it
  fails the load (`CheckRedirect`). `net/http` strips `Authorization` and
  `Cookie` headers on cross-host redirects.
- **Body size**: `Loader.MaxBodySize` (default 100 MiB) is enforced on both
  HTTP and file reads; oversized bodies are rejected with an error, never
  silently truncated. A declared `Content-Length` over the cap is rejected
  before any body bytes are read; chunked/unknown-length bodies are capped
  on the read side. `data:` URLs are bounded by the size of the document
  that embeds them.
- **TLS**: certificate verification on by default. There is no `--insecure`
  / `InsecureSkipVerify` switch. Proxy, client certificates, cookies and
  POST bodies are operator-supplied configuration.
- **NetworkPolicy**: `CompatibleNetworkPolicy` is the default when no
  policy is set (CLI without `--restrict-network`). `RestrictedNetworkPolicy`
  / `--restrict-network` blocks private destinations and cross-host
  redirects. Restricted dials pin the resolved IP (no second DNS lookup).
  Exact `--allow-host` entries may skip the private-IP check; wildcards do
  not. `GlobalSettings.SetNetworkPolicy` is the library seam.

## 5. Data exfiltration channels

- Every URL referenced in the HTML - images, stylesheets, the primary page
  itself - can be fetched. A document can cause fetches to any reachable
  host, **including `http://localhost` and RFC1918 addresses** unless
  Restricted network policy is enabled. Compatible mode matches historical
  wkhtmltopdf URL behavior. Restricted mode is the recommended default for
  untrusted HTML in a service.
- Local file reads are the only sensitive channel and are gated by the ACL
  (section 3). With default flags, no document-reachable path reads any
  local file.
- **Fonts:** TTF/OTF/WOFF1 bytes loaded via `@font-face` `url(...)` are
  untrusted parse input under the same ACL as other subresources; WOFF1
  decompress uses size caps (table count, per-table / reconstructed SFNT
  limits, overlap rejection) before `ParseTTF`. `--font-path` /
  `--use-system-fonts` are operator-controlled discovery (not HTML ACL).
  Remote `https://` `@font-face` is not fetched (product policy). WOFF2 is
  rejected (Brotli not allowlisted).
- Operator credentials (custom headers, basic auth, cookies) are attached
  to the requests the operator configured them for. Cross-host auth and
  cookie headers are stripped by `net/http` on redirects, but custom
  headers follow. Operators should not combine credential-bearing loads
  with untrusted HTML.

## 6. Explicitly out of scope

- **No JavaScript**: `--enable-javascript` is a no-op (section 1).
- **No sandbox for rendering**: the document cannot execute code, but HTML
  parsing, CSS processing and layout run in-process; parser bugs are the
  residual risk. A crafted document can consume CPU/memory up to the body
  cap.
- **No network egress restrictions** (SSRF posture, section 5).
- **TOCTOU** between the ACL check and the file open (section 3).

## 7. Recommendations for untrusted HTML

- Convert untrusted documents in an isolated container: no access to
  sensitive filesystems or networks, no host credentials, no
  `--username/--password`, `--custom-header`, `--cookie` or `--proxy`
  credentials aimed at non-public hosts.
- Keep the defaults: `--enable-local-file-access` off, no `--allow`.
- Rely on the built-in timeouts (30 s connect / 60 s response default) and
  the 100 MiB body cap; both are on by default.
- Sanitise HTML before conversion, or convert only HTML you author.

## 7.1 Embedding in web apps (Gin / similar) - short scenarios

Full write-up: **[integration-security.md](integration-security.md)**.

**Happy path people assume:** user hits Gin → converter fetches a URL → PDF
returned. That is fine only when **you** control the URL/HTML. The issue is
not “displaying PDF”; it is that the **server** becomes an HTTP client (and
optionally a file reader) on behalf of whoever controls the input.

| Pattern | Risk | Preferred? |
|---------|------|------------|
| Gin renders **your** template → convert that HTML/path | Low | **Yes** |
| Gin passes `c.Query("url")` (arbitrary) into convert | **High (SSRF)** - server can hit localhost, cloud metadata, RFC1918 | No |
| HTML references extra `img`/`link` URLs | Server fetches them too (second-hop SSRF) | Avoid untrusted HTML |
| Local file access on + user-influenced path/`file:` | **High (file read)** into PDF | Keep default deny |
| Many concurrent converts / huge pages | DoS (CPU/RAM) | Rate-limit + timeouts |

**Preferred:** generate HTML server-side → convert **trusted** bytes/path →
return `application/pdf`. Do **not** expose “convert any URL” without host
allowlists and network isolation.

**Same for upstream wkhtmltopdf:** the same SSRF / local-file classes apply
if the app design is “user URL → convert.” wkhtmltopdf also runs a real
JS engine (additional surface); gowkhtmltopdf does not. Neither tool is a
substitute for not letting strangers drive server-side fetches.

## 8. Controls inventory

| Control | Location |
|---|---|
| ACL: default deny, allow prefixes, symlink/traversal resolution | `internal/load/load.go` - `AccessController.Allowed`, `resolvePath`, `fileAccessAllowed` |
| `file://` host restriction | `internal/load/load.go` - `filePathFromURL` (used by `loadFile`; `FetchSub` host check) |
| Body cap, HTTP (header + read side) | `internal/load/load.go` - `loadHTTP` |
| Body cap, file | `internal/load/load.go` - `loadFile` |
| Redirect cap | `internal/load/load.go` - `initClient` / `CheckRedirect` |
| Connect timeout | `internal/load/load.go` - `DefaultConnectTimeout`, `net.Dialer` |
| Response timeout | `internal/load/load.go` - `loadHTTP` `client.Timeout`, `DefaultResponseTimeout` |
| Context cancellation | `internal/load/load.go` - `http.NewRequestWithContext` in `loadHTTP` |
| No JS / no exec | whole repo - no `os/exec`; JS flags accepted and ignored |
| NetworkPolicy | `internal/load` + `GlobalSettings.SetNetworkPolicy`; CLI `--restrict-network` / `--allow-host` |
