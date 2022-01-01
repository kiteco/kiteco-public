#!/usr/bin/env python

import sys
import gzip
import json
import argparse

from docutils import core

from kite.ioutils.stream import loadjson


def html_string(input_string):
    overrides = {'input_encoding': 'unicode',
                 'doctitle_xform': 1,
                 'initial_header_level': 1,
                 'report_level': 5,  # suppress errors for unresolved roles and directives
                 'halt_level': 5  # never halt on an error
                 }
    return core.publish_string(source=input_string, writer_name='html', settings_overrides=overrides)


def trim_leading_spaces(text):
    """Remove unnecessary indent from each line in the text.
    """
    modified = ""
    min_spaces = sys.maxsize
    split = text.splitlines()
    # count indentation of each line and track minimum (if positive, reflects
    # unnecessary indentation in text)
    for line in split[1:]:
        if len(line) == 0:
            continue
        spaces = len(line) - len(line.lstrip(' '))
        if spaces < min_spaces:
            min_spaces = spaces

    if min_spaces == sys.maxsize:
        min_spaces = 0

    # shift each line to the left by minimum
    modified = text
    if min_spaces > 0:
        l = []
        for i, line in enumerate(split):
            if i == 0 or len(line) == 0:
                l.append(line)
                continue
            l.append(line[min_spaces:])
        if len(l) > 0:
            modified = ''.join(l)

    return modified

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--input', required=True, help="map of node id to docstrings")
    parser.add_argument('--output', required=True,
                        help="path to the output map of node id to html descriptions")
    args = parser.parse_args()

    in_file = gzip.open(args.input)
    out_file = gzip.open(args.output, 'wt')
                         # json module produces str output (instead of unicode)
                         # so open in text mode

    print("Converting docstrings to structured html...")
    for obj in loadjson(in_file):
        docstring = obj["docstring"]
        html = ""
        if len(docstring) > 0:
            clean = trim_leading_spaces(
                docstring)  # docutils complains if incorrectly indented
            clean = u".. role:: func\n.. role:: class\n.. role:: meth\n.. role:: mod\n.. role:: attr\n.. role:: data\n.. role:: const\n.. role:: exc\n.. role:: obj\n.. role:: ref\n" + clean
            html = html_string(clean)  # docutils conversion
            html = html.decode()  # utf-8 to unicode
        converted = {
            "node_id": obj["node_id"],
            "description": html
        }
        json.dump(converted, out_file)

    in_file.close()
    out_file.close()

    print("Done")
