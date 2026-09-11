# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 17 ms | 258 ms | 14.95x | 24,576 KiB | 44,716 KiB | 34,209 | 18,486 |
| 5 | 23 ms | 266 ms | 11.44x | 26,496 KiB | 45,068 KiB | 42,791 | 30,584 |
| 10 | 34 ms | 286 ms | 8.29x | 28,608 KiB | 46,024 KiB | 57,231 | 50,994 |
| 20 | 53 ms | 306 ms | 5.79x | 33,984 KiB | 47,772 KiB | 84,654 | 90,742 |
| 50 | 122 ms | 394 ms | 3.23x | 47,616 KiB | 52,240 KiB | 167,442 | 210,678 |
| 100 | 229 ms | 541 ms | 2.36x | 69,120 KiB | 59,380 KiB | 306,144 | 411,260 |
| 200 | 468 ms | 830 ms | 1.77x | 113,280 KiB | 74,492 KiB | 583,231 | 816,285 |
| 250 | 599 ms | 988 ms | 1.65x | 139,584 KiB | 81,884 KiB | 721,739 | 1,019,315 |
| 500 | 1.288 s | 1.753 s | 1.36x | 240,960 KiB | 123,172 KiB | 1,419,234 | 2,036,776 |
