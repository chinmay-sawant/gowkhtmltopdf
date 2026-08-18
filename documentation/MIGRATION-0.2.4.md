# Migrating to 0.2.4

This guide describes the intentional pre-1.0 hard break from the 0.2.3
wkhtml-shaped library and CLI to the 0.2.4 Document model.

> **Availability note:** the hard break is complete in v0.2.4. The root package
> exports the `Document` / `ImageDocument` model and the CLI uses the document
> grammar described below. The old symbols are shown only as migration input;
> they are not available from the v0.2.4 root package.

## At a glance

| 0.2.3 | 0.2.4 target |
|---|---|
| Converter | Document |
| ImageConverter | ImageDocument |
| PDFRequest / RunPDF | Document.WritePDF / Document.PDF |
| ImageRequest / RunImage | ImageDocument.WriteImage / ImageDocument.Image |
| GlobalSettings / ObjectSettings | Fields on Document and Page |
| ImageSettings | Fields on ImageDocument |
| Dotted public Set / Get | Named struct fields |
| SetPage(path) | Content{File: path} or File(...) |
| SetBody(html, base) | Content{HTML: html, Base: base} or HTML(...) |
| Global + object local-file ACL pair | AllowLocalFiles: true |
| NewTOCObject() | Document.TOC |
| NewCoverObject() | Document.Cover |
| PDF/image output in converter state | Explicit writer or byte-returning method |

The document order is fixed: Cover → TOC → Pages.

## Library migration

### 1. Converter to Document

Old:

~~~go
c := gowkhtmltopdf.NewConverter()
c.AddObject(gowkhtmltopdf.NewObjectSettings().SetPage("report.html"))
if err := c.Convert(ctx); err != nil {
    return err
}
return os.WriteFile("report.pdf", c.Output(), 0o644)
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{File: "report.html"},
    }},
    AllowLocalFiles: true,
}

out, err := os.Create("report.pdf")
if err != nil {
    return err
}
defer out.Close()

return doc.WritePDF(ctx, out)
~~~

Use Document.PDF(ctx) when the application needs bytes rather than a
caller-owned writer.

### 2. Dotted settings to fields

Old:

~~~go
global := gowkhtmltopdf.NewGlobalSettings()
if err := global.Set("size.pagesize", "A4"); err != nil {
    return err
}
if err := global.Set("margin.top", "15"); err != nil {
    return err
}
if err := global.Set("title", "Invoice"); err != nil {
    return err
}
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    PageSize: "A4",
    Margin: gowkhtmltopdf.Margin{Top: 15},
    Title: "Invoice",
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{HTML: []byte("<h1>Invoice</h1>")},
    }},
}
~~~

The 0.2.4 public API intentionally does not provide a generic dotted-key
fallback. A setting that matters to users must have a typed field or remain an
internal engine setting.

Common mappings:

| Old key / builder | Target field |
|---|---|
| size.pagesize | Document.PageSize |
| size.width / size.height | WidthMM / HeightMM |
| orientation | Orientation |
| margin.top/right/bottom/left | Margin |
| title | Title |
| pdfversion | PDFVersion |
| pdfprofile | PDFProfile |
| copies / collate | Copies / Collate |
| outline / outline.depth | Outline / OutlineDepth |
| web.background | Background |
| smart shrinking settings | SmartShrinking |
| load.* network policy | Network and named security fields |
| fontpath / system-font setting | FontPaths / UseSystemFonts |

### 3. SetBody to Content

Old:

~~~go
body := gowkhtmltopdf.NewObjectSettings().SetBody(html, base)
c.AddObject(body)
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{
            HTML: html,
            Base: base,
        },
    }},
}
~~~

Content makes the source kind explicit. HTML bytes are copied at the
ownership boundary. Do not prefix bytes with inline: or data: and do not
send them through URL guessing.

### 4. SetPage to File or URL

Old:

~~~go
page := gowkhtmltopdf.NewObjectSettings().SetPage(input)
~~~

Target for a local file:

~~~go
page := gowkhtmltopdf.Page{
    Source: gowkhtmltopdf.Content{File: input},
}
~~~

Target for a remote page:

~~~go
page := gowkhtmltopdf.Page{
    Source: gowkhtmltopdf.Content{URL: "https://example.test/report"},
}
~~~

An empty source is invalid. File and URL are mutually exclusive, and Base is
only valid with HTML.

### 5. Local-file access

Old:

~~~go
_ = global.Set("enablelocalfileaccess", "true")
_ = page.Set("load.blocklocalfileaccess", "false")
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    AllowLocalFiles: true,
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{File: "report.html"},
    }},
}
~~~

The one public bool maps to the internal global and page ACL decisions. Keep
the default false for untrusted input.

### 6. PDFRequest and RunPDF to WritePDF and PDF

Old:

~~~go
err := gowkhtmltopdf.RunPDF(ctx, &gowkhtmltopdf.PDFRequest{
    Global:  global,
    Objects: []*gowkhtmltopdf.ObjectSettings{body},
    Output:  out,
})
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{Source: gowkhtmltopdf.Content{
        HTML: html,
    }}},
}
err := doc.WritePDF(ctx, out)
~~~

For bytes:

~~~go
pdfBytes, err := doc.PDF(ctx)
~~~

WritePDFOutline(ctx, pdfWriter, outlineWriter) replaces configurations that
used PDFRequest.OutlineOutput. The outline sink is explicit and can no
longer be silently mixed with PDF bytes.

### 7. TOC and cover objects to document fields

Old:

~~~go
c.AddObject(gowkhtmltopdf.NewCoverObject().SetPage("cover.html"))
c.AddObject(gowkhtmltopdf.NewTOCObject())
c.AddObject(gowkhtmltopdf.NewObjectSettings().SetPage("chapter.html"))
~~~

Target:

~~~go
doc := gowkhtmltopdf.Document{
    Cover: &gowkhtmltopdf.Page{
        Source: gowkhtmltopdf.Content{File: "cover.html"},
    },
    TOC: &gowkhtmltopdf.TOC{
        Caption: "Contents",
    },
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{File: "chapter.html"},
    }},
    AllowLocalFiles: true,
}
~~~

A TOC-only document is invalid. A cover is excluded from the outline and does
not inherit document headers or footers unless configured explicitly.

### 8. ImageConverter to ImageDocument

Old:

~~~go
image := gowkhtmltopdf.NewImageConverter()
image.Set("width", "1024")
image.Set("format", "png")
image.AddObject("badge.html")
if err := image.Convert(ctx); err != nil {
    return err
}
return os.WriteFile("badge.png", image.Output(), 0o644)
~~~

Target:

~~~go
imageDoc := gowkhtmltopdf.ImageDocument{
    Source: gowkhtmltopdf.Content{File: "badge.html"},
    Width:  1024,
    Format: "png",
}

pngBytes, err := imageDoc.Image(ctx)
if err != nil {
    return err
}
return os.WriteFile("badge.png", pngBytes, 0o644)
~~~

Use WriteImage(ctx, writer) for a streaming sink. Width, height, format,
quality, smart width, transparency, and crop are fields; PDF-only concepts
such as TOC and outline do not apply.

## Validation changes

The target model validates before engine work:

- Exactly one of Content.HTML, Content.File, and Content.URL is set.
- Empty HTML is rejected.
- A PDF needs at least one valid body page or cover.
- A TOC alone is not renderable.
- Copies below one are rejected when explicitly configured.
- An image document has exactly one valid source.
- Nil document receivers return typed sentinels.
- Invalid values return errors from Validate / Write*; they are not
  programmer panics.

Use errors.Is with the package sentinels. See the validation table in
[library-api.md](library-api.md#validation-and-errors).

## CLI migration

Old:

~~~sh
gowkhtmltopdf --enable-local-file-access \
  cover cover.html toc page chapter.html old.pdf
~~~

Target:

~~~sh
gowkhtmltopdf --allow-local-files \
  --cover cover.html --toc -o new.pdf chapter.html
~~~

| 0.2.3 CLI | 0.2.4 target |
|---|---|
| Final positional output | Required -o OUTPUT / --output OUTPUT |
| page input.html | Positional input.html |
| cover cover.html | --cover cover.html |
| toc | --toc |
| inline:<html>...</html> | --html '<html>...</html>' |
| URL-looking positional source | --url URL |
| --enable-local-file-access | --allow-local-files |
| Dotted / generic settings | Named flags only |

Use [cli.md](cli.md) for the target flag groups and exit codes.

## Suggested migration order

1. Replace source construction with Content.
2. Move global settings onto Document fields.
3. Move each object to a Page; put cover and TOC on their dedicated fields.
4. Replace local-file ACL pairs with AllowLocalFiles.
5. Replace RunPDF / RunImage with writer-first methods.
6. Add Validate checks and errors.Is assertions around user input.
7. Migrate CLI invocations to -o, explicit source flags, --cover, and --toc.
8. Update examples after the root package and CLI migration gates are green.

There is no source-compatible adapter package in 0.2.4. Keep the migration in
an application-owned compatibility layer if downstream callers need a
gradual rollout.
