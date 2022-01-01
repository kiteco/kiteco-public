from __future__ import print_function
import sys
import zlib
import base64
import json

_base64_url_charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
_base64_padding = "="


def emit(fd, key, value):
    # in python3, strings need to be cast into bytes to be passed
    # into zlib.compress
    # the output of base64.urlsafe_b64encode is converted to a string
    s = "%s\t%s" % (key,
            base64.urlsafe_b64encode(zlib.compress(
                value.encode('utf-8'))).decode('utf-8'))
    print(s, file=fd)

# --

def read_as_json(fd):
    for key, val in read(fd):
        yield key, json.loads(val.decode())

def read(fd):
    for parts in _read_base64_zlib(fd):
        yield parts

def _read_base64_zlib(fd):
    for parts in _read(fd):
        key = parts[0].strip()
        value = parts[1].strip()

        try:
            value = zlib.decompress(
                base64.urlsafe_b64decode(_filter_base64(value)))
        except Exception as ex:
            s = "error decoding key: %s, err: %s" % (key, ex)
            print(s, file=sys.stderr)
            continue

        yield key, value

def _filter_base64(contents):
    return ''.join(x for x in contents if x in _base64_url_charset
                   or x == _base64_padding)

def _read(fd):
    for line in fd:
        parts = line.split('\t')
        if len(parts) != 2:
            raise Exception("expected record to have 2 parts, but has %d" % len(parts))
        yield parts
