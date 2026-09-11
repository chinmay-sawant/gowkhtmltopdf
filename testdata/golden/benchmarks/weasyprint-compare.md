# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.5.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; weasyprint used `-q` (quiet).
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 21 ms | 653 ms | 31.13x | 25,152 KiB | 81,932 KiB | 34,209 | 15,586 |
| 10 | 36 ms | 1.431 s | 39.80x | 28,416 KiB | 111,444 KiB | 57,231 | 45,172 |
| 50 | 124 ms | 5.482 s | 44.07x | 48,000 KiB | 252,448 KiB | 167,442 | 190,545 |
| 100 | 245 ms | 11.119 s | 45.29x | 65,856 KiB | 427,880 KiB | 306,144 | 372,867 |
