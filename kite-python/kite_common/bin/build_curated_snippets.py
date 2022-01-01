#!/usr/bin/env python
from __future__ import print_function

import json
import os
import sys
import zlib
import base64
import re
from pprint import pprint

from spooky import hash128

from kite.codeexamples import extract
from kite.emr import io


if __name__ == "__main__":
    total = 0
    error = 0
    missing = 0
    multiple = 0

    for _, curated_example in io.read(sys.stdin):
        total += 1

        try:
            example = json.loads(curated_example)
        except UnicodeDecodeError as ex:
            print("unicode decode error:", ex, file=sys.stderr)
            error += 1
            continue

        try:
            curated = example["Snippet"]
            code = '\n'.join([curated['prelude'],
                              curated['code'],
                              curated['postlude']])

            # Replace curation variables (e.g $HTTP_PORT$) with kite_var
            code = re.sub('\${1}([a-zA-Z_]+)\${1}', 'kite_var', code)

            snippets = extract.curated_snippets(code)
        except Exception as ex:
            print("snippet error:", ex, file=sys.stderr)
            error += 1
            continue

        if len(snippets) == 0:
            missing += 1

        if len(snippets) > 1:
            multiple += 1

        for snip in snippets:
            # This structure maps to pythoncuration.Snippet
            # in kiteco/kite-go/codeexample.
            obj = {
                'Curated': example,
                'Snippet': snip.to_json(),
            }

            try:
                snippetHash = str(hash128(snip.code.encode('utf-8')))
                io.emit(sys.stdout, snippetHash, json.dumps(obj, ensure_ascii=False))
            except Exception as ex:
                print("io error:", ex, file=sys.stderr)
                error += 1

    print(missing, "out of", total, "missing", file=sys.stderr)
    print(error, "errors", file=sys.stderr)
    print(multiple, "examples with multiple snippets", file=sys.stderr)
