#!/usr/bin/env python
import sys
import json

from spooky import hash128

from kite.codeexamples import inference
from kite.emr import io


MAX_CODE_BYTES = 1 << 20


def main():
    for path, code in io.read(sys.stdin):
        if len(code) > MAX_CODE_BYTES:
            print >>sys.stderr, path, "exceedes code size limit of %d, was %d" % (
                MAX_CODE_BYTES, len(code))
            continue

        try:
            usages = inference.get_obj_usages(code)
        except Exception as ex:
            print >>sys.stderr, "inference error:", ex
            continue

        # Skip anything that yielding no obj incantations
        if len(usages) == 0:
            continue

        codeHash = str(hash128(code))
        usages_json = [x.to_json() for x in usages]
        try:
            io.emit(sys.stdout, codeHash, json.dumps(usages_json))
        except Exception as ex:
            print >>sys.stderr, "io error:", ex


if __name__ == "__main__":
    main()
