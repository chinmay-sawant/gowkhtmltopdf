# Phase 94.9: Closure

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** S
> **Depends on:** 94.0 through 94.8
> **Unblocks:** nothing else in this ledger

---

## Overview

This is the only phase that runs the full integration commands. Earlier phases
stay on one package and one fixture-test batch. Closure remains open while
fixtures 56 and 64 have committed-versus-fresh page-operation differences.

`make lint` stays out. The owner runs lint after this ledger closes.

## Checklist

- [ ] 94.9.1 Every new `.go` file from this ledger is at or under 2000 lines. Record `wc -l` for `internal/pdf/page_ops.go`, `internal/pdf/page_ops_test.go`, `internal/convert/output_ops_test.go`, `internal/convert/output_fixture_shape_test.go`, and every `internal/convert/output_fixture_*_test.go` file.
- [ ] 94.9.2 `make size-check` exits 0. No allowlist edit.
- [ ] 94.9.3 Targeted suite, exit 0:

```bash
go test ./internal/pdf -run 'TestTextPositions' -count=1
go test ./internal/convert -run 'TestOutputFixture' -count=1
```

- [ ] 94.9.4 `make test` exits 0. Record the exit code here.
- [ ] 94.9.5 `make golden` exits 0. The reader is test-only, so this is a regression check that semantic parsing still walks the corpus. Record the exit code here.
- [ ] 94.9.6 Parent canonical status line moves from `not started` to `complete`, and `plans/0.2.7/README.md` says this ledger is complete.
- [ ] 94.9.7 Knowledge-base page `knowledge-base/wiki/concepts/output-position-validator.md` flips from planned to shipped, with the test file names and the closure commands. Append `knowledge-base/wiki/log.md`.

## Proof template

```text
make size-check: exit 0
go test ./internal/pdf -run 'TestTextPositions' -count=1: exit 0
go test ./internal/convert -run 'TestOutputPos' -count=1: exit 0
make test: exit 0
make golden: exit 0
file lines: text_positions.go=N, each output_pos_*_test.go=N
```
