# Ponytail packet 1 - public boundaries

> **Scope:** root API, commands, CLI, settings, loader, HTML, outline, app, SVG, and small support packages
> **Canonical rows:** `PT26-BND-01` through `PT26-BND-12` in [0.2.6-ponytail-audit.md](0.2.6-ponytail-audit.md)
> **Method:** current caller search and focused source reading. No files changed and no tests ran.

## Active evidence

`internal/load/load.go:L494-522`: **delete** `NewLoader`. It is a deprecated constructor with no current caller. `NewLoaderWithError` preserves the loader creation path and reports invalid proxies at setup. Estimated cut: 29 lines. Prove with `go test ./internal/load`.

`internal/cli/flags.go:L71,L87-94,L476`: **delete** doc-flag registration, `addDocFlags`, and `nopFlag`. `internal/cli/cli.go:L185-188` recognizes the four flags before table lookup and `internal/cli/help.go:L97-99` does not use the registrations for help text. Estimated cut: 10 lines. Prove with `go test ./internal/cli`.

`internal/cli/cli.go:L270-274`: **delete** `ParseMode`. It only forwards to `Parse` and has no caller. Estimated cut: 5 lines. Prove with `go test ./internal/cli`.

`internal/cli/cli.go:L427-431,L478-499`: **shrink** one-consumer `boolFlagState`. Pass its three primitive values directly to `parseBoolFlag`. Estimated cut: 8 lines. Prove with `go test ./internal/cli`.

`internal/cli/cli.go:L197,L230-233`: **shrink** one-consumer `isShortFlag`. Inline its `strings.HasPrefix(arg, "-") && len(arg) == 2` condition in `step`. Estimated cut: 3 lines. Prove with `go test ./internal/cli`.

`internal/settings/settings.go:L92-125`, `internal/settings/reflect.go:L394-405`: **yagni** remove `ColorMode`, its constants, parser, and formatter. Its only effect is setting one `Grayscale` bool. Accept the same two strings in that setter and preserve the existing error. Estimated cut: 19 lines. Prove with `go test ./internal/settings`.

`internal/settings/reflect.go:L425-440,L698-703`: **shrink** remove `marginValue`. It repeats the four-edge switch already owned by `marginEdgePtr` and has one production caller. Dereference the returned pointer in the getter. Estimated cut: 14 lines. Prove with `go test ./internal/settings`.

`internal/settings/reflect.go:L1046-1064`: **shrink** fold `ApplyImageKeyNormalized` into its sole caller `ApplyImageKey`. Normalize there and retain the existing `background` routing. Estimated cut: 6 lines. Prove with `go test ./internal/settings`.

`internal/settings/object_roles.go:L3-25`: **shrink** inline `StampEmptyHFOverride` into `StampCover`. Its only non-test consumer is `StampCover`. Estimated cut: 12 lines. Prove with `go test ./internal/settings`.

`document.go:L415,L425-426,L496-497,L580,L632,L637-643`: **stdlib** replace `documentCloneStrings` and `documentCloneStringMap` with `slices.Clone` and `maps.Clone`. Every caller is in this file. Estimated cut: 6 lines. Prove with `go test .`.

`api.go:L198-207`: **stdlib** replace the local nil-preserving `cloneBytes` with `slices.Clone` at the two callers in `document.go`. Estimated cut: 9 lines. Prove with `go test .`.

`internal/html/html.go:L587-596`: **delete** `isTrimSpace`. It has no caller after `endTagName` moved to `strings.ToLower(strings.TrimSpace(...))`. Estimated cut: 10 lines. Prove with `go test ./internal/html`.

## Rejected lookalikes

- `internal/cli/cli.go:L501-508`: `splitFlag` is a one-caller name, but it already uses `strings.Cut`; the small rename is not a useful row.
- `internal/app/pdf.go:L18-23`, `cmd/gowkhtmltopdf/main.go:L48-53`: keep `DefaultTOCXSL`. It keeps the command main on the documented app and CLI boundary instead of importing conversion code.
- `internal/svg/raster.go:L272-301`: keep the bounded case-insensitive SVG probe. It retains the `errNotSVG` contract and the input-size limit without an allocation.
- `internal/outline/outline.go:L264-494`: keep the local-page helpers. `outline_test.go:L541-544` exercises their nil `PageOf` fallback.

Packet estimate: about -130 source lines before test changes.
