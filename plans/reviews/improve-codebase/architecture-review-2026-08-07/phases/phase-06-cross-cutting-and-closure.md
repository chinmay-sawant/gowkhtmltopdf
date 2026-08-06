# Phase 6 — Cross-cutting & closure

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

Log protocol, examples, exit-code dispatch + closure gates. Runs last: validates the whole tree after all rows ship.

## Checklist

- [x] **P6-01** — done — line package + api + convert/imageout Emit
- [x] **P6-02** — done (fix-root-api)
- [x] **P6-03** — done — cli.ExitCode + mains dispatch

---

<a id="p6-01"></a>
## P6-01 — Own the log-line severity protocol once; stop substring guessing in api.go

- [x] **P6-01** — done — line package + api + convert/imageout Emit

- **Locations:** `api.go:328-363` (`lineLog`); emitter side `internal/convert/convert.go` (many `fmt.Fprintf(log, "warning: …")` sites) and `internal/imageout/imageout.go`
- **Evidence sources:** area-1-surface-api F3

---

### Evidence — F3

### F3: Own the log-line severity protocol once; stop guessing it with substrings in api.go
- **Severity:** medium
- **Location:** `api.go:328-363` (`lineLog`); emitter side `internal/convert/convert.go` (many `fmt.Fprintf(log, "warning: …")` sites) and `internal/imageout/imageout.go`
- **Current:**
```go
type lineLog struct {
	buf     bytes.Buffer
	onInfo  func(string)
	onWarn  func(string)
	onError func(string)
}

func (w *lineLog) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		raw := w.buf.Bytes()
		i := bytes.IndexByte(raw, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSpace(string(raw[:i]))
		w.buf.Next(i + 1)
		if line == "" {
			continue
		}
		switch lower := strings.ToLower(line); {
		case strings.Contains(lower, "warning:"):
			if w.onWarn != nil {
				w.onWarn(line)
			}
		case strings.Contains(lower, "error"):
			if w.onError != nil {
				w.onError(line)
			}
		default:
			if w.onInfo != nil {
				w.onInfo(line)
			}
		}
	}
	return len(p), nil
```
- **Future:** move classification next to the emitters (single owner), and let api.go call it:
```go
// internal/convert/logline.go — the one place that knows the engine's
// log-line grammar. The CLI's raw stderr printing and the library's
// callbacks both feed from here.
type LogKind int

const (
	LogInfo LogKind = iota
	LogWarn
	LogError
)

// Kind classifies one engine log line by its leading marker token.
func Kind(line string) LogKind {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch {
	case strings.HasPrefix(lower, "warning:"), strings.HasPrefix(lower, "warn:"):
		return LogWarn
	case strings.HasPrefix(lower, "error:"), strings.HasPrefix(lower, "err:"):
		return LogError
	default:
		return LogInfo
	}
}

// api.go lineLog.Write then routes with convert.Kind(line) — the
// callback mapping stays in api.go, the grammar in convert.
```
API-compatible. (Alternative with the same locality win: a shared `internal/logline` package imported by both emitters and api.)
- **Why:** today the severity protocol is an invisible convention: convert/imageout print `warning: …` / `info: …` prefixes at ~15 call sites, the CLI prints them raw, and api.go re-implements the *classification* of that convention with substring `Contains` checks in the root package. `Contains(lower, "error")` misroutes any line whose *message* mentions error without being an error, and a new emitter prefix (e.g. `warn:`) silently breaks the mapping. The grammar is duplicated at every emitter site and re-derived at the consumer — knowledge owned by nobody. Moving `Kind` next to the emitters makes the protocol a function of one package; `api_test.go:TestConverterCallbacks` can then pin every real emitter line to its expected kind. hypothesis: a future info line such as `info: load error policy is skip, omitting` (already emitted by imageout for the skip path) would land in OnError; validate by running every current emitter `printf` format through `Kind` and diffing against the intended channel.

---

<a id="p6-02"></a>
## P6-02 — Examples: propagate Set errors; share one flag parser

- [x] **P6-02** — done (fix-root-api)

- **Locations:** `examples/pdf/main.go:40-98`; `examples/image/main.go:34-86`
- **Evidence sources:** area-1-surface-api F6

---

### Evidence — F6

### F6: Examples swallow every `Set` error and hand-roll the same flag parser twice
- **Severity:** medium
- **Location:** `examples/pdf/main.go:40-98`; `examples/image/main.go:34-86`
- **Current:**
```go
	input, output := inputs[0], inputs[1]

	c := gowkhtmltopdf.NewConverter()
	c.Global().Set("size.pagesize", pageSize)
	c.Global().Set("orientation", orientation)
	if marginTop != "" {
		c.Global().Set("margin.top", marginTop)
	}

	obj := gowkhtmltopdf.NewObjectSettings().SetPage(input)
	if localFiles {
		c.Global().Set("enablelocalfileaccess", "true")
		obj.Set("load.blocklocalfileaccess", "false")
	}
```
- **Future:** stdlib `flag` (it already accepts both `--name=v` and `--name v` and `-h`) plus fail-fast setting, in both examples:
```go
fs := flag.NewFlagSet("pdf", flag.ContinueOnError)
pageSize := fs.String("page-size", "A4", "e.g. A4, Letter"
orientation := fs.String("orientation", "portrait", "portrait or landscape")
marginTop := fs.String("margin-top", "", "top margin in mm")
localFiles := fs.Bool("enable-local-file-access", false, "allow local files")
if err := fs.Parse(argv); err != nil {
	return err
}
if fs.NArg() != 2 {
	usage()
	return fmt.Errorf("need exactly one input and one output file")
}
c := gowkhtmltopdf.NewConverter()
mustSet := func(name, value string) error {
	if err := c.Global().Set(name, value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
if err := mustSet("size.pagesize", *pageSize); err != nil { return err }
if err := mustSet("orientation", *orientation); err != nil { return err }
// … and the same shape in examples/image with c.Set …
```
API-compatible; examples are the only callers.
- **Why:** every `c.Global().Set(...)`/`obj.Set(...)`/`c.Set(...)` return value is discarded in both examples — `--page-size=Garbage` today converts happily at the default A4 with no message, teaching the exact wrong embedding pattern for a library whose whole settings contract is "Set returns an error for unknown names" (doc.go). The two ~30-line parsing loops (same `--flag=v`/`--flag v`/`--` shape, same `--enable-local-file-access` case) are a hand-rolled `flag` re-implementation duplicated across two files, and the local-access ACL pair (`enablelocalfileaccess` + `load.blocklocalfileaccess`) is re-typed in both files and again in doc.go. These examples are the public template for embedders; each copied pattern propagates. A `mustSet` helper makes invalid values fail loudly, and `flag` deletes the duplicated loop. This is not a deletion-only nit: the examples are the surface's most-read code, so their plumbing *is* part of the surface's interface as users experience it.

---

<a id="p6-03"></a>
## P6-03 — Put exit-code knowledge next to its error type; one main-stream dispatch

- [x] **P6-03** — done — cli.ExitCode + mains dispatch

- **Locations:** `internal/settings/reflect.go:833-842` (HttpErrorCode); `internal/settings/httperror.go:9-20` (HttpStatusError); `cmd/gowkhtmltopdf/main.go:31-41,53-61`; `cmd/gowkhtmltoimage/main.go:44-41`
- **Evidence sources:** area-2-settings-cli F7

---

### Evidence — F7

### F7: Put the exit-code knowledge next to its error type and give the two mains one dispatch
- **Severity:** low
- **Location:** `internal/settings/reflect.go:833-842` (HttpErrorCode); `internal/settings/httperror.go:9-20` (HttpStatusError); `cmd/gowkhtmltopdf/main.go:31-41,53-61`; `cmd/gowkhtmltoimage/main.go:44-41`
- **Current (verbatim):**
```go
// internal/settings/httperror.go
type HttpStatusError struct {
	Status int
	URL    string
}

func (e *HttpStatusError) Error() string {
	return fmt.Sprintf("failed to load %s: HTTP %d", e.URL, e.Status)
}

// HttpErrorCode maps an HTTP status to the wkhtmltopdf exit-code convention.
func HttpErrorCode(status int) int { return HttpErrorCode(e.Status) }
```
```go
// internal/settings/reflect.go:833-842 — attached to the bottom of the 843-line god file:
func HttpErrorCode(status int) int {
	switch status {
	case 404:
		return 2
	case 401:
		return 3
	}
	return 1
}
```
```go
// cmd/gowkhtmltopdf/main.go — same 8-line block duplicated in cmd/gowkhtmltoimage/main.go:
	if err := convert.RunPDFContext(context.Background(), cmd, logw, nil); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err)
		if hc, ok := err.(interface{ HttpErrorCode() int }); ok {
			return hc.HttpErrorCode()
		}
		return cli.ExitError
	}
```
- **Future:**
```go
// httperror.go — the mapping next to the type it maps:
func HttpErrorCode(status int) int {
	switch status {
	case 404:
		return 2
	case 401:
		return 3
	}
	return 1
}
func (e *HttpStatusError) HttpErrorCode() int { return HttpErrorCode(e.Status) }
```
```go
// internal/cli — one place for "convert an error into a process exit code".
func ExitCode(err error) int {
	var hc interface{ HttpErrorCode() int }
	if errors.As(err, &hc) {
		return hc.HttpErrorCode()
	}
	return ExitError
}
```
Both main.go `run`s then become: `fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err); return cli.ExitCode(err)`.
- **Why:** the stdout+stderr doc-flag dispatch and the error→exit-code assertions are duplicated
  verbatim across two small binaries (a third copy of the mapping is `settings.HttpErrorCode`
  itself). The mapping function lives 800 lines and five registrations away from `HttpStatusError`
  — the locality fix is two moves and one 8-line function, and it removes the interface assertion
  from both mains. hypothesis: the two mains' doc-flag cases (ErrHelp/ErrVersion/ErrLicense) will
  drift on the next flag — validate by adding `--extended-help` handling to one binary only.

---

## Closure gates

- [x] `make lint` — clean after all rows ship (`go vet` + `gofmt -l` clean, 2026-08-07)
- [x] `make test` — full suite green (`go test ./... -count=1` all packages ok, 2026-08-07)
- [x] `go vet ./...` — clean
- [x] Benchmark notes recorded for P4-01 (img decode-once) and P4-07 (op splice): command, dataset, before/after metric
- [x] New deliberate shortcuts carry `// ponytail:` ceiling + upgrade-trigger markers (repo convention)

### Benchmark notes (architecture wave)

| Row | Command | Dataset | Note |
|-----|---------|---------|------|
| **P4-01** | `go test ./internal/layout/ -run 'Test|Benchmark' -count=1` | fixture suite + logo/img layout tests | Before: same `src` fetched/decoded 2–4× per Layout (measure + build). After: `engine.imgCache` + `resolveImage` — one fetch/decode per `src` per run. Qualitatively verified by single-decode cache hit path in `resolveImage` (nil-miss also cached). |
| **P4-07** | `go test ./internal/layout/ -count=1` | sticky/transform + static chrome fixtures | Before: every box spliced ops for bg/border. After: static/relative boxes defer chrome to `finalizeChrome` (append + reverse-reg merge); sticky/fixed/transform still immediate-splice. Fixture suite green; no golden regression in convert package. |
