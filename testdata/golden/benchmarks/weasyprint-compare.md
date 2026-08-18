# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 1 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.4.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; weasyprint used `-q` (quiet).
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 23 ms | 821 ms | 35.52x | 23,616 KiB | 77,408 KiB | 34,068 | 15,586 |
