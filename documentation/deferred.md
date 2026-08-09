# Deferred: workload prioritisation (temporary response)

> Temporary note captured from the workload discussion; replace this with a
> sourced roadmap decision after the product-scope review.

The dominant practical workload for an HTML-to-PDF tool is backend-generated
business documents, not arbitrary public websites:

```text
application data
  -> server-side HTML template
  -> HTML/CSS
  -> PDF
```

Typical documents are invoices, receipts, reports, statements, purchase
orders, contracts, certificates, and shipping documents. Python, PHP, Ruby,
and other integrations commonly expose both HTML/string/file input and URL
input. The template path is usually the primary path because it avoids an
HTTP loopback, authentication and cookie problems, and makes assets and
testing more predictable.

URL input is still valuable when it points to a server-rendered internal page,
such as `/orders/123/print`: it reuses an existing web view, CSS, data loading,
and localisation. It should not be conflated with a client-rendered SPA URL.
The dossier URL is an SPA: its initial HTML is an empty root element and
React constructs the document in JavaScript. That is a lower-frequency,
browser-rendering workload and is not representative of the core invoice or
report path.

For gowkhtmltopdf, the highest-impact order is therefore:

1. rendered HTML strings and stdin;
2. local HTML template files;
3. server-rendered internal URLs;
4. modern JavaScript-heavy SPA URLs.

The core product should prioritise tables, pagination, headers and footers,
images, fonts, CSS layout, page breaks, and repeated sections. SPA execution
should remain a separate capability rather than the definition of the main
HTML-to-PDF workload.
