# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Both binaries used `--quiet --enable-local-file-access` on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 22 ms | 273 ms | 12.24x | 23,232 KiB | 44,500 KiB | 34,068 | 18,486 |
| 5 | 29 ms | 276 ms | 9.47x | 24,384 KiB | 45,000 KiB | 42,467 | 30,584 |
| 10 | 38 ms | 283 ms | 7.38x | 26,496 KiB | 45,848 KiB | 56,607 | 50,994 |
| 20 | 46 ms | 312 ms | 6.74x | 29,568 KiB | 47,508 KiB | 83,434 | 90,742 |
| 50 | 102 ms | 403 ms | 3.93x | 42,432 KiB | 51,700 KiB | 164,461 | 210,678 |
| 100 | 200 ms | 540 ms | 2.70x | 62,400 KiB | 59,004 KiB | 300,249 | 411,260 |
| 200 | 425 ms | 847 ms | 1.99x | 97,152 KiB | 74,004 KiB | 571,397 | 816,285 |
| 250 | 540 ms | 993 ms | 1.84x | 111,936 KiB | 81,712 KiB | 707,011 | 1,019,315 |
| 500 | 1.109 s | 1.829 s | 1.65x | 206,592 KiB | 122,928 KiB | 1,390,014 | 2,036,776 |
