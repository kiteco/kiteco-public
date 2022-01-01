#!/usr/bin/env bash

set -euo pipefail

function run {
  local cwd=""; cwd="$(pwd)"
  local dir="$1"

  # build the go program only once
  go build -o test_prettify_js ./main.go 2> /dev/null

  local count=$((0))
  # process all js files in the provided directory, recursively
  find "${dir}" -name "*.js" -print0 | while read -rd $'\0' file
  do
    # ignore any entry that is not a regular file
    if [[ ! -f "${file}" ]]; then
      continue
    fi

    # ignore @flow-annotated files
    if head -c20 "${file}" | grep -qe "@flow"; then
      continue
    fi

    # get the file size, can be useful to select a file for debugging
    local size; size=$(du -k "${file}" | cut -f1)

    count=$((count + 1))
    # shorten the output file name a bit if possible
    file="$(realpath --relative-to "${cwd}" "${file}")"
    echo -n "${count} - (${size}kb) ${file} ... "

    if ! ( ./test_prettify_js -semi -ignore-semi -ignore-empty -ignore-invalid-file -level 1 "${file}" > /dev/null); then
      echo "FAIL"
      continue
    fi
    echo "PASS"
  done
}

if [[ $# -lt 1 ]]; then
  echo "USAGE: diff.sh DIR"
  echo "example: diff.sh ~/github.com/facebook/react"
fi
run "$@"

