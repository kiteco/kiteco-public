#!/usr/bin/env python
import sys
from chardet import detect
from base64 import b64decode

from kite.emr import io


MAX_CODE_BYTES = 1<<20

valid_encodings = ["ascii", "utf-8", "ISO-8859-2"]

# TODO(juan): move this into go
if __name__ == "__main__":
    for path, code in io.read(sys.stdin):
        try:
            name = b64decode(path)
        except Exception as ex:
            print >>sys.stderr, path, " had decoding error:", ex
            continue
        
        if len(code) > MAX_CODE_BYTES:
            print >>sys.stderr, name, "exceedes code size limit of %d, was %d" % (
                MAX_CODE_BYTES, len(code))
            continue
        
        if not name.endswith(".py"):
            continue

        enc = detect(code)
        if enc['encoding'] not in valid_encodings:
            continue

        try:
            if enc['encoding'] != 'utf-8':
                code = code.decode(enc['encoding']).encode('utf-8')
        except Exception as ex:
            print >>sys.stderr, name, "had encoding error:", ex
            continue

        try:
            io.emit(sys.stdout, path, code)
        except Exception as ex:
            print >>sys.stderr, name, " had emit error:", ex
