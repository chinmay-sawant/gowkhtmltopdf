# WASM fixture data

This directory contains the small, self-contained input used by the browser
adapter tests. It stays separate from `testdata/golden` because the golden
corpus checks the native PDF corpus, while this manifest is shared by PDF,
PNG, JPEG, and future browser-runtime tests.

- `sample.html` is inline HTML with print CSS, a table, a forced page break,
  and a data-URL image. It does not require a file or network request.
- `manifest.json` defines the request for each output mode and its structural
  expectations.

The native adapter test and the browser test must read these files instead of
copying the sample into separate test code.
