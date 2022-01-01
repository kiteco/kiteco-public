#!/usr/bin/env python
import json
import os
import sys
import zlib
import base64
from pprint import pprint

import chardet
from spooky import hash128

from kite.codeexamples import extract
from kite.emr import io


MAX_CODE_BYTES = 1<<20


if __name__ == "__main__":
    for path, code in io.read(sys.stdin):
        if len(code) > MAX_CODE_BYTES:
            print >>sys.stderr, path, "exceedes code size limit of %d, was %d" % (
                MAX_CODE_BYTES, len(code))
            continue

        try:
            enc = chardet.detect(code)
            if enc['encoding'] == 'GB2312':
                code = code.decode('gb2312').encode('utf-8')
            if enc['encoding'] == 'ISO-8859-2':
                code = code.decode('iso-8859-1').encode('utf8')
        except Exception as ex:
            print >>sys.stderr, "encoding error:", ex

        try:
            snippets = extract.get_snippets(path, code)
        except Exception as ex:
            print >>sys.stderr, "snippet error:", ex
            continue

        for snip in snippets:
            snippetHash = str(hash128(snip.code))
            try:
                io.emit(sys.stdout, snippetHash, json.dumps(snip.to_json()))
            except Exception as ex:
                print >>sys.stderr, "io error:", ex, enc
