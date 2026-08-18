# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --enable-local-file-access`; weasyprint used `-q` (quiet).
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 21 ms | 726 ms | 33.78x | 23,616 KiB | 77,392 KiB | 34,068 | 15,583 |
| 10 | 38 ms | 1.584 s | 41.66x | 27,072 KiB | 106,000 KiB | 56,607 | 45,171 |
| 50 | 113 ms | 6.102 s | 53.77x | 42,240 KiB | 246,700 KiB | 164,461 | 190,543 |
| 100 | 231 ms | 13.833 s | 59.78x | 60,864 KiB | 422,944 KiB | 300,249 | 372,868 |
