# Browser WASM conversion

The repository includes an opt-in WebAssembly adapter for browser previews.
It runs the existing Go document pipeline in a Web Worker. The adapter accepts
inline HTML and returns a PDF, PNG, or JPEG byte buffer.

The browser adapter does not run document JavaScript. It does not read browser
files, fetch arbitrary remote resources, or replace a full browser renderer.
Use the native [Go API](library-api.md) when you need local files, URL input,
custom font paths, or native network policy.

## Build the browser assets

Use Go 1.26 or a compatible Go toolchain. Run the following command from the
repository root:

```sh
make wasm
```

The target builds `bindings/wasm` with `GOOS=js GOARCH=wasm`, stamps the
artifact from `VERSION`, and copies these files into `frontend/public/wasm/`:

- `gowkhtmltopdf.wasm`
- `wasm_exec.js` from the active Go toolchain
- `sample.html` and `manifest.json` from `testdata/wasm/`

Build the site after the WASM target:

```sh
npm --prefix frontend run build
```

Vite copies the public files into `docs/`. Do not edit the generated `docs/`
tree by hand.

Run `make wasm` again whenever the Go request contract changes. The site loads
`frontend/public/wasm/gowkhtmltopdf.wasm`. If that file is older than the Go
source, the browser can reject a field such as `padding` even when the current
request type accepts it.

Run the complete adapter and browser checks with:

```sh
make wasm-test
```

That target runs the native adapter tests, the JavaScript and WASM compile
check, the frontend smoke tests, and the real Chrome browser harness.

## JavaScript bridge

The WASM program registers `globalThis.gowkhtmltopdfWASM` after
`wasm_exec.js` starts the Go runtime. The function accepts a JSON request and
an optional progress callback:

```js
const response = globalThis.gowkhtmltopdfWASM(
  JSON.stringify({
    html: '<h1>Invoice</h1>',
    mode: 'pdf',
    pageSize: 'A4',
    orientation: 'Portrait',
  }),
  (phase, percent) => console.log(phase, percent),
)
```

The success response has this shape:

```js
{
  ok: true,
  mode: 'pdf',
  mime: 'application/pdf',
  version: '<VERSION value>',
  width: 0,
  height: 0,
  bytes: Uint8Array,
}
```

PNG and JPEG responses use `image/png` and `image/jpeg`. They also return the
encoded image width and height. The error response has a stable code and
message:

```js
{
  ok: false,
  error: { code: 'invalid_request', message: 'invalid request: HTML is empty' },
}
```

The frontend worker in `frontend/src/workers/wasmWorker.js` loads the runtime,
starts the Go program once, sends request IDs, transfers result bytes, and
reports progress. `frontend/src/lib/wasmClient.js` rejects pending requests
when the worker is cancelled.

## Request modes

The request uses these fields:

| Field | PDF | PNG or JPEG |
|-------|-----|-------------|
| `html` | Required inline HTML string | Required inline HTML string |
| `mode` | `pdf` or empty for the default | `png` or `jpeg` |
| `pageSize` | Optional page size such as `A4` | Not allowed |
| `orientation` | Optional `Portrait` or `Landscape` | Not allowed |
| `width` | Not allowed | Optional pixel width |
| `height` | Not allowed | Optional pixel height |
| `padding` | Not allowed | Optional non-negative whole-pixel inset added to all four image edges |
| `quality` | Not allowed | Optional JPEG quality from 0 to 100 |

For example, this request adds 12 pixels of transparent padding around PNG
output:

```js
JSON.stringify({
  html: '<h1>Badge</h1>',
  mode: 'png',
  padding: 12,
})
```

The adapter rejects `padding` in PDF mode. It also rejects negative, fractional,
or oversized padding values. The frontend shows the padding field only for PNG
and JPEG output. It defaults the field to 20 pixels, and Reset restores that
20-pixel default.

The adapter rejects unknown JSON fields, empty HTML, unsupported modes, and
options that belong to another output mode. HTML input is limited to 4 MiB.
Output is limited to 32 MiB. Image dimensions are limited to 4096 pixels per
axis. The worker gives one conversion 60 seconds before its context expires.
These limits are defined in `bindings/wasm/contract.go` and
`bindings/wasm/main_js_wasm.go`.

## Preview PDF and images

The `/live-demo` page in the generated site provides an HTML editor, a sample
loader, a reset action, output selection, image padding, and download and open
actions. Image padding uses output pixels and expands the PNG or JPEG canvas by
the same amount on every side. PNG keeps transparent pixels. JPEG composites
transparent pixels onto white. For PNG and JPEG, the browser first gets the
paginated PDF from the local Go engine, then rasterizes every PDF page into its
selected image format. The preview shows every image page in a scrollable
panel. A multi-page result provides a ZIP containing one image per page and an
Open action that opens a standalone image gallery. Single-page results keep
the direct image download and open actions. PDF output uses the bundled PDF.js
viewer in `frontend/src/components/PdfViewer.jsx`. It renders each page into a
canvas inside a fixed-height, vertically scrollable viewport. The page revokes
the previous Blob URLs when a result is replaced or the page unmounts.

The shared fixture in `testdata/wasm/` exercises a two-page PDF, a data-URL
image, a table, inline print CSS, and both image formats. Native tests read the
fixture manifest. The browser harness reads the built page and checks the
output signatures, MIME types, dimensions, PDF trailer, and preview cleanup.

## Resource and security boundary

The browser MVP accepts inline HTML and resources embedded in that HTML, such
as data URLs. It disables local file access, system font discovery, and
network schemes in the document policy. It rejects native `file` and `url`
request fields before conversion starts.

The adapter does not promise arbitrary remote CSS, image, or font loading. A
future resource bridge needs its own origin, CORS, size, timeout, and
cancellation contract. The native loader keeps its separate file and HTTP
policy described in [the security model](THREAT-MODEL.md).

## Native and browser inputs

Native `Document` and `ImageDocument` callers can use explicit HTML, file, or
URL sources and can configure native policies. Browser calls use only the
inline `html` field. Both paths reuse the root document APIs and the internal
load, parse, style, layout, paginate, paint, and write pipeline.

The browser adapter does not add a JavaScript runtime to converted documents,
and it does not claim Chrome or WebKit rendering parity.
