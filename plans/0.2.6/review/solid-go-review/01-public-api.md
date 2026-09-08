# Review - Public API and adapters

> **Parent:** `skills/solid-go-review/SKILLS.md` - review workflow
> **Status:** Complete. All findings were implemented or closed by current-source proof.
> **Estimated effort:** Not estimated

---

## Overview

This packet covered the root library API, CLI parser and help, application adapters, settings, commands, examples, C bindings, and sentinel errors.

## Executive Summary

The public API copies caller-owned data at important boundaries and validates before rendering or opening output files. CLI flags use a shared descriptor table. The main weaknesses are contract drift in the C and CLI adapters, duplicated sentinel identity, and normalization that happens during validation but not mapping.

## Files and responsibilities

- `document.go:17-171` - public document and page models.
- `document_validate.go:32-188` - public input validation and sentinel errors.
- `internal/cli/cli.go:55-180` - command parsing and terminal actions.
- `internal/cli/flags.go:25-84` - reusable flag descriptors.
- `internal/app/pdf.go:37-105` and `internal/app/image.go:18-68` - CLI-to-engine adapters.
- `internal/settings/settings.go:408-598` and `internal/settings/reflect.go:84-135` - settings data and dotted dispatch.
- `bindings/c/options_image.go:14-95` and `bindings/c/classify.go:17-63` - C option mapping and status translation.

## SOLID and Go verdicts

| Area | Verdict | Evidence |
| --- | --- | --- |
| SRP | MIXED | Adapters keep validation and file opening ordered (`internal/app/pdf.go:73-100`), while C option mapping and status translation duplicate public contracts (`bindings/c/options_image.go:36-88`, `bindings/c/classify.go:43-63`). |
| OCP | MIXED | CLI flags share a descriptor table (`internal/cli/flags.go:25-84`), but terminal actions need a new conflict check for each output-like argument (`internal/cli/cli.go:324-338`). |
| LSP | N/A | No confirmed public subtype hierarchy in this packet. |
| ISP | PASS | The public API uses concrete data models, and the settings reflection descriptor has one focused read/write role (`internal/settings/reflect.go:84-135`). |
| DIP | MIXED | App adapters keep engine construction below the CLI boundary, but C status categories are maintained separately from public validation errors (`bindings/c/classify.go:43-63`). |

## Confirmed findings

### Phase 1: Correctness and API contracts

- [x] `A-01` `bindings/c/options_image.go:56-69` - zero-initialized C options now preserve unset markers and engine defaults. Proof: `TestZeroImageOptionsUseUnsetMarkers` and `TestCSharedImageOptionValidation` pass.
- [x] `A-02` `internal/cli/cli.go:324-334` - terminal XSL mode now rejects conflicting output arguments. Proof: `TestTerminalActionsAndModeValidation` passes.
- [x] `A-04` `bindings/c/classify.go:43-63` - image quality and crop validation errors now map to invalid arguments. Proof: `TestImageValidationErrorsAreInvalidArguments` passes.
- [x] `A-05` `document_validate.go:109-112` - orientation is trimmed once and the normalized value reaches PDF mapping. Proof: `TestDocumentOrientationValidationAndMappingUseTrimmedValue` passes.

### Phase 2: Error and reuse seams

- [x] `A-03` `internal/app/pdf.go:26-34` - one canonical nil-command sentinel is shared by the app adapter and lower-level errors package. Proof: `TestNilCommandUsesCanonicalSentinel` passes.

## Hypotheses

- [x] `A-H1` `internal/app/image.go:30-38` - the suspected format retention was not reproducible; repeated PNG then JPEG execution resolves each output extension independently. Proof: `TestRunImageResolvesFormatPerExecution` passes.
- [x] `A-H2` `document_validate.go:102-127` - numeric boundaries now reject negative, NaN, and infinite values before mapping. Proof: root document and image validation tests pass.

## Area score

**7.0/10.** The API boundaries and ownership copies are good. Boundary contract duplication lowers reuse and makes C and CLI behavior drift from root validation.

## Validation

Focused tests and the full `make test` passed. `make lint`, `make claim-scan`, and `make golden` passed. The CLI terminal-mode probe reproduced `exit 0` with 455 bytes on stdout.

## Dependencies

Fix the C and CLI contract findings before changing public types. Each fix needs focused adapter tests and the full suite.
