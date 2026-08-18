# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 17 ms | 259 ms | 15.46x | 23,808 KiB | 44,192 KiB | 34,068 | 18,486 |
| 5 | 22 ms | 268 ms | 12.36x | 24,960 KiB | 44,716 KiB | 42,467 | 30,584 |
| 10 | 30 ms | 276 ms | 9.21x | 27,264 KiB | 45,992 KiB | 56,607 | 50,994 |
| 20 | 45 ms | 317 ms | 7.06x | 30,528 KiB | 47,464 KiB | 83,434 | 90,742 |
| 50 | 112 ms | 406 ms | 3.63x | 43,200 KiB | 51,856 KiB | 164,461 | 210,678 |
| 100 | 184 ms | 526 ms | 2.85x | 61,248 KiB | 59,048 KiB | 300,249 | 411,260 |
| 200 | 376 ms | 811 ms | 2.15x | 96,192 KiB | 74,192 KiB | 571,397 | 816,285 |
| 250 | 480 ms | 964 ms | 2.01x | 116,736 KiB | 81,740 KiB | 707,011 | 1,019,315 |
| 500 | 1.042 s | 1.671 s | 1.60x | 208,128 KiB | 123,080 KiB | 1,390,014 | 2,036,776 |
