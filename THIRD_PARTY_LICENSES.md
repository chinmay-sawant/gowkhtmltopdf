# Third-party licenses and notices

The gowkhtmltopdf application is MIT-licensed; see [`LICENSE`](LICENSE). The
MIT license does not cover the third-party font files embedded in the binary or
the pure-Go modules used for shaping and SVG rasterization. This file records
the direct runtime dependencies and bundled font provenance for release review.

The direct module allowlist is intentionally small and is checked by
`internal/pdf.TestDirectModuleAllowlist`. The transitive Go module graph is
resolved by Go from these direct modules; its source distributions remain under
their own licenses and should be retained when a source or vendor archive is
prepared.

## Bundled fonts

### Liberation fonts

The embedded `LiberationSans-*`, `LiberationSerif-*`, and `LiberationMono-*`
TrueType files in `internal/pdf/assets/` are identified by the package source as
copyright Red Hat / Ascender Corp. and licensed under the SIL Open Font License
(OFL) 1.1. The application embeds them as default PDF faces. The OFL text
already retained in this repository is [`testdata/fonts/OFL.txt`](testdata/fonts/OFL.txt).

### DejaVu Sans fallback

`DejaVuSans-Regular.ttf` and `DejaVuSans-Bold.ttf` in
`internal/pdf/assets/` are the embedded Unicode fallback faces. The embedded
bytes match the corresponding DejaVu Sans files from the DejaVu font package;
the applicable upstream notice identifies the original Bitstream Vera
copyright and places the DejaVu changes in the public domain. The relevant
Bitstream Vera terms are reproduced below. Modified versions must follow the
font renaming condition.

> Copyright (c) 2003 by Bitstream, Inc. All Rights Reserved. Bitstream Vera is
> a trademark of Bitstream, Inc. DejaVu changes are in the public domain.
>
> Permission is hereby granted, free of charge, to any person obtaining a copy
> of the fonts and associated documentation files (the “Font Software”), to
> reproduce and distribute the Font Software, including without limitation the
> rights to use, copy, merge, publish, distribute, and/or sell copies, and to
> permit persons to whom the Font Software is furnished to do so, subject to
> inclusion of this copyright and permission notice.
>
> The Font Software may be modified, altered, or added to only if the fonts
> are renamed to names not containing “Bitstream” or “Vera”. The Font Software
> may be sold as part of a larger software package but not by itself.
>
> THE FONT SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED. IN NO EVENT SHALL BITSTREAM OR THE GNOME FOUNDATION BE
> LIABLE FOR ANY CLAIM, DAMAGES, OR OTHER LIABILITY ARISING FROM THE USE OF
> THE FONT SOFTWARE.

The DejaVu source/license sidecar should remain paired with these assets in any
future asset refresh. Do not add another font family without adding its
provenance and notice here.

### Test-only Noto subset

`testdata/fonts/NotoSansKR-HangulSubset.ttf` is a test fixture, not embedded in
the default binary. Its repository documentation identifies it as an OFL
subset; the full retained license text is [`testdata/fonts/OFL.txt`](testdata/fonts/OFL.txt).

## Go modules

### `github.com/go-text/typesetting` v0.3.4

Used for pure-Go OpenType shaping when the selected font has the required
tables. The module's upstream `LICENSE` declares:

`SPDX-License-Identifier: Unlicense OR BSD-3-Clause`

Copyright: The go-text authors. The full license text is distributed with the
module source and must be retained in source/vendor release archives.

### `github.com/tdewolff/canvas`

Used only for SVG-as-image rasterization through the allowlisted canvas
rasterizer. The pinned pseudo-version is recorded in `go.mod`. Its upstream
`LICENSE.md` grants the MIT permission notice and attributes copyright to Taco
de Wolff. Retain that upstream license text in source/vendor release archives.

## Runtime boundary

Both direct modules are pure Go. Releases are built with `CGO_ENABLED=0` and do
not link to or launch Qt, WebKit, Chrome, FreeType, HarfBuzz C, or another
native/browser rendering process. The project uses the pure-Go shaping path in
`go-text/typesetting`; “HarfBuzz” references in rendering documentation mean
that pure-Go port, not the CGO library.
