#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-master}"
tab=$(printf '\t')

git diff "$base_ref"...HEAD --numstat |
  awk -F '\t' '
    function extension(path, name, part_count, parts) {
      if (path ~ / => /) {
        sub(/^.* => /, "", path)
        sub(/}$/, "", path)
      }

      part_count = split(path, parts, "/")
      name = parts[part_count]
      if (name !~ /\./ || name ~ /^\.[^.]+$/) {
        return "No extension"
      }

      sub(/^.*\./, ".", name)
      return name
    }

    {
      ext = extension($3)
      files[ext]++
      if ($1 == "-") {
        binary[ext]++
      } else {
        insertions[ext] += $1
        deletions[ext] += $2
        total_insertions += $1
        total_deletions += $2
      }
      total_files++
    }

    END {
      for (ext in files) {
        printf "%s\t%d\t%d\t%d\t%d\n", ext, files[ext], insertions[ext] + 0, deletions[ext] + 0, binary[ext] + 0
      }
      printf "__TOTAL__\t%d\t%d\t%d\t0\n", total_files, total_insertions + 0, total_deletions + 0
    }
  ' |
  sort -t "$tab" -k1,1 |
  awk -F '\t' '
    BEGIN {
      print "| Extension | Files | Insertions | Deletions |"
      print "| --- | ---: | ---: | ---: |"
    }

    $1 == "__TOTAL__" {
      total_files = $2
      total_insertions = $3
      total_deletions = $4
      next
    }

    $5 > 0 {
      if ($1 == "No extension") {
        printf "| No extension | %d | Binary | Binary |\n", $2
      } else {
        printf "| `%s` | %d | Binary | Binary |\n", $1, $2
      }
      next
    }

    {
      if ($1 == "No extension") {
        printf "| No extension | %d | %d | %d |\n", $2, $3, $4
      } else {
        printf "| `%s` | %d | %d | %d |\n", $1, $2, $3, $4
      }
    }

    END {
      printf "| **Total** | **%d** | **%d** | **%d** |\n", total_files, total_insertions, total_deletions
    }
  '
