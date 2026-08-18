# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 17 ms | 257 ms | 15.56x | 24,192 KiB | 44,300 KiB | 34,068 | 18,486 |
| 5 | 22 ms | 265 ms | 11.86x | 24,768 KiB | 45,024 KiB | 42,467 | 30,584 |
| 10 | 29 ms | 282 ms | 9.60x | 27,264 KiB | 45,672 KiB | 56,607 | 50,994 |
| 20 | 46 ms | 307 ms | 6.72x | 30,144 KiB | 47,224 KiB | 83,434 | 90,742 |
| 50 | 94 ms | 393 ms | 4.18x | 42,816 KiB | 51,772 KiB | 164,461 | 210,678 |
| 100 | 181 ms | 533 ms | 2.95x | 60,480 KiB | 59,104 KiB | 300,249 | 411,260 |
| 200 | 380 ms | 838 ms | 2.21x | 96,000 KiB | 74,032 KiB | 571,397 | 816,285 |
| 250 | 493 ms | 969 ms | 1.97x | 117,504 KiB | 81,624 KiB | 707,011 | 1,019,315 |
| 500 | 1.020 s | 1.707 s | 1.67x | 209,664 KiB | 122,876 KiB | 1,390,014 | 2,036,776 |
