# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.6.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; weasyprint used `-q` (quiet).
Gowk source: baseline rows from testdata/golden/benchmarks/cli-compare-results.csv (the same session's make bench-cli-compare run); engine rows are measured in this session.
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 639 ms | 49.18x | 19,584 KiB | 81,744 KiB | 34,210 | 15,584 |
| 10 | 24 ms | 1.435 s | 59.78x | 24,384 KiB | 110,976 KiB | 57,239 | 45,174 |
| 50 | 67 ms | 5.496 s | 82.04x | 29,376 KiB | 251,804 KiB | 167,525 | 190,544 |
| 100 | 124 ms | 10.953 s | 88.33x | 35,520 KiB | 427,372 KiB | 306,321 | 372,868 |
